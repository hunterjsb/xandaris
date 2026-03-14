package api

import (
	"fmt"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/tickable"
)

// GameStateProvider is an interface the App implements to give the API read access
// to game state without importing core (avoiding circular deps).
type GameStateProvider interface {
	GetSystems() []*entities.System
	GetHyperlanes() []entities.Hyperlane
	GetPlayers() []*entities.Player
	GetHumanPlayer() *entities.Player
	GetSeed() int64
	GetMarket() *economy.Market
	GetTradeExecutor() *economy.TradeExecutor
	GetCargoCommander() *game.CargoCommandExecutor
	GetTickInfo() (tick int64, gameTime string, speed string, paused bool)
	GetCommandChannel() chan game.GameCommand
}

// --- handler logic (pure functions, no net/http) ---

func handleGetMarket(p GameStateProvider) interface{} {
	market := p.GetMarket()
	if market == nil {
		return []MarketCommodity{}
	}
	snap := market.GetSnapshot()
	result := make([]MarketCommodity, 0, len(snap.Resources))
	for name, rm := range snap.Resources {
		result = append(result, MarketCommodity{
			Resource:      name,
			BasePrice:     rm.BasePrice,
			CurrentPrice:  rm.CurrentPrice,
			BuyPrice:      rm.BuyPrice,
			SellPrice:     rm.SellPrice,
			TotalSupply:   rm.TotalSupply,
			TotalDemand:   rm.TotalDemand,
			PriceVelocity: rm.PriceVelocity,
		})
	}
	return result
}

func handleGetTradeHistory(p GameStateProvider, limit int) interface{} {
	exec := p.GetTradeExecutor()
	if exec == nil {
		return []TradeHistoryEntry{}
	}
	records := exec.GetHistory(limit)
	result := make([]TradeHistoryEntry, len(records))
	for i, r := range records {
		result[i] = TradeHistoryEntry{
			Tick:      r.Tick,
			Player:    r.Player,
			Resource:  r.Resource,
			Quantity:  r.Quantity,
			Action:    r.Action,
			UnitPrice: r.UnitPrice,
			Total:     r.Total,
		}
	}
	return result
}

func handleGetGalaxy(p GameStateProvider) interface{} {
	systems := p.GetSystems()
	hyperlanes := p.GetHyperlanes()

	// Build adjacency map
	links := make(map[int][]int)
	for _, hl := range hyperlanes {
		links[hl.From] = append(links[hl.From], hl.To)
	}

	result := make([]SystemSummary, 0, len(systems))
	for _, sys := range systems {
		starType := ""
		planets := 0
		for _, e := range sys.Entities {
			switch e.(type) {
			case *entities.Star:
				if s, ok := e.(*entities.Star); ok {
					starType = s.StarType
				}
			case *entities.Planet:
				planets++
			}
		}
		result = append(result, SystemSummary{
			ID:       sys.ID,
			Name:     sys.Name,
			X:        sys.X,
			Y:        sys.Y,
			StarType: starType,
			Planets:  planets,
			Links:    links[sys.ID],
		})
	}
	return result
}

func handleGetSystem(p GameStateProvider, id int) (interface{}, bool) {
	for _, sys := range p.GetSystems() {
		if sys.ID == id {
			return buildSystemDetail(sys), true
		}
	}
	return nil, false
}

func buildSystemDetail(sys *entities.System) interface{} {
	type systemDetail struct {
		ID      int            `json:"id"`
		Name    string         `json:"name"`
		X       float64        `json:"x"`
		Y       float64        `json:"y"`
		Planets []PlanetDetail `json:"planets"`
	}

	planets := make([]PlanetDetail, 0)
	for _, e := range sys.Entities {
		if planet, ok := e.(*entities.Planet); ok {
			planets = append(planets, buildPlanetDetail(planet, sys.ID))
		}
	}
	return systemDetail{
		ID:      sys.ID,
		Name:    sys.Name,
		X:       sys.X,
		Y:       sys.Y,
		Planets: planets,
	}
}

func handleGetPlanet(p GameStateProvider, id int) (interface{}, bool) {
	for _, sys := range p.GetSystems() {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok {
				if planet.GetID() == id {
					return buildPlanetDetail(planet, sys.ID), true
				}
			}
		}
	}
	return nil, false
}

func buildPlanetDetail(planet *entities.Planet, systemID int) PlanetDetail {
	stored := make(map[string]int)
	for resType, s := range planet.StoredResources {
		if s != nil {
			stored[resType] = s.Amount
		}
	}

	buildings := make([]BuildingInfo, 0)
	for i, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok {
			buildings = append(buildings, BuildingInfo{
				Index:         i,
				Type:          b.BuildingType,
				Level:         b.Level,
				MaxLevel:      b.MaxLevel,
				IsOperational: b.IsOperational,
				Staffing:      b.GetStaffingRatio(),
				UpgradeCost:   b.GetUpgradeCost(),
			})
		}
	}

	// Resource deposits (minable nodes)
	deposits := make([]ResourceDeposit, 0)
	for _, resEntity := range planet.Resources {
		if res, ok := resEntity.(*entities.Resource); ok {
			resIDStr := fmt.Sprintf("%d", res.GetID())
			hasMine := false
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					if b.BuildingType == "Mine" && b.AttachedTo == resIDStr {
						hasMine = true
						break
					}
				}
			}
			deposits = append(deposits, ResourceDeposit{
				ID:             res.GetID(),
				ResourceType:   res.ResourceType,
				Abundance:      res.Abundance,
				ExtractionRate: res.ExtractionRate,
				HasMine:        hasMine,
			})
		}
	}

	return PlanetDetail{
		ID:              planet.GetID(),
		Name:            planet.Name,
		PlanetType:      planet.PlanetType,
		Population:      planet.Population,
		PopulationCap:   planet.GetTotalPopulationCapacity(),
		Habitability:    planet.Habitability,
		Owner:           planet.Owner,
		StoredResources: stored,
		ResourceDeposits: deposits,
		Buildings:       buildings,
		SystemID:        systemID,
	}
}

func handleGetPlayers(p GameStateProvider) interface{} {
	players := p.GetPlayers()
	result := make([]PlayerInfo, 0, len(players))
	for _, pl := range players {
		if pl == nil {
			continue
		}
		pType := "ai"
		if pl.IsHuman() {
			pType = "human"
		}
		result = append(result, PlayerInfo{
			ID:      pl.ID,
			Name:    pl.Name,
			Type:    pType,
			Credits: pl.Credits,
			Planets: len(pl.OwnedPlanets),
			Ships:   len(pl.OwnedShips),
			Fleets:  len(pl.OwnedFleets),
		})
	}
	return result
}

func handleGetPlayerMe(p GameStateProvider) interface{} {
	human := p.GetHumanPlayer()
	if human == nil {
		return nil
	}

	type playerMe struct {
		Name    string         `json:"name"`
		Credits int            `json:"credits"`
		Planets []PlanetDetail `json:"planets"`
		Ships   []ShipInfo     `json:"ships"`
	}

	planets := make([]PlanetDetail, 0)
	for _, planet := range human.OwnedPlanets {
		if planet == nil {
			continue
		}
		// Find system ID
		sysID := 0
		for _, sys := range p.GetSystems() {
			for _, e := range sys.Entities {
				if pl, ok := e.(*entities.Planet); ok && pl.GetID() == planet.GetID() {
					sysID = sys.ID
					break
				}
			}
		}
		planets = append(planets, buildPlanetDetail(planet, sysID))
	}

	ships := make([]ShipInfo, 0)
	for _, ship := range human.OwnedShips {
		if ship == nil {
			continue
		}
		cargo := make(map[string]int)
		for k, v := range ship.CargoHold {
			cargo[k] = v
		}
		ships = append(ships, ShipInfo{
			ID:            ship.GetID(),
			Name:          ship.Name,
			Type:          string(ship.ShipType),
			Owner:         ship.Owner,
			Status:        string(ship.Status),
			SystemID:      ship.CurrentSystem,
			TargetSystem:  ship.TargetSystem,
			FuelCurrent:   ship.CurrentFuel,
			FuelMax:       ship.MaxFuel,
			HealthCurrent: ship.CurrentHealth,
			HealthMax:     ship.MaxHealth,
			CargoUsed:     ship.GetTotalCargo(),
			CargoMax:      ship.MaxCargo,
			CargoHold:     cargo,
		})
	}

	return playerMe{
		Name:    human.Name,
		Credits: human.Credits,
		Planets: planets,
		Ships:   ships,
	}
}

func handleGetGame(p GameStateProvider) interface{} {
	tick, gameTime, speed, paused := p.GetTickInfo()
	return GameInfo{
		Tick:     tick,
		GameTime: gameTime,
		Speed:    speed,
		Paused:   paused,
		Systems:  len(p.GetSystems()),
		Players:  len(p.GetPlayers()),
		Seed:     p.GetSeed(),
	}
}

func handleGetShips(p GameStateProvider) interface{} {
	result := make([]ShipInfo, 0)
	for _, player := range p.GetPlayers() {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil {
				continue
			}
			cargo := make(map[string]int)
			for k, v := range ship.CargoHold {
				cargo[k] = v
			}
			result = append(result, ShipInfo{
				ID:            ship.GetID(),
				Name:          ship.Name,
				Type:          string(ship.ShipType),
				Owner:         ship.Owner,
				Status:        string(ship.Status),
				SystemID:      ship.CurrentSystem,
				TargetSystem:  ship.TargetSystem,
				FuelCurrent:   ship.CurrentFuel,
				FuelMax:       ship.MaxFuel,
				HealthCurrent: ship.CurrentHealth,
				HealthMax:     ship.MaxHealth,
				CargoUsed:     ship.GetTotalCargo(),
				CargoMax:      ship.MaxCargo,
				CargoHold:     cargo,
			})
		}
	}
	return result
}

func handleGetFleets(p GameStateProvider) interface{} {
	result := make([]FleetInfo, 0)
	for _, player := range p.GetPlayers() {
		if player == nil {
			continue
		}
		for _, fleet := range player.OwnedFleets {
			if fleet == nil {
				continue
			}
			ships := make([]ShipInfo, 0, len(fleet.Ships))
			for _, ship := range fleet.Ships {
				if ship == nil {
					continue
				}
				cargo := make(map[string]int)
				for k, v := range ship.CargoHold {
					cargo[k] = v
				}
				ships = append(ships, ShipInfo{
					ID:            ship.GetID(),
					Name:          ship.Name,
					Type:          string(ship.ShipType),
					Owner:         ship.Owner,
					Status:        string(ship.Status),
					SystemID:      ship.CurrentSystem,
					TargetSystem:  ship.TargetSystem,
					FuelCurrent:   ship.CurrentFuel,
					FuelMax:       ship.MaxFuel,
					HealthCurrent: ship.CurrentHealth,
					HealthMax:     ship.MaxHealth,
					CargoUsed:     ship.GetTotalCargo(),
					CargoMax:      ship.MaxCargo,
					CargoHold:     cargo,
				})
			}
			result = append(result, FleetInfo{
				ID:    fleet.ID,
				Owner: fleet.GetOwner(),
				Size:  fleet.Size(),
				Ships: ships,
			})
		}
	}
	return result
}

func handleGetEconomy(p GameStateProvider) interface{} {
	overview := EconomyOverview{
		Resources: make(map[string]ResourceSummary),
	}

	// Aggregate population, credits, and resource stock
	for _, player := range p.GetPlayers() {
		if player == nil {
			continue
		}
		overview.TotalCredits += player.Credits
		for _, planet := range player.OwnedPlanets {
			if planet == nil {
				continue
			}
			overview.TotalPopulation += planet.Population
			for resType, storage := range planet.StoredResources {
				if storage == nil {
					continue
				}
				rs := overview.Resources[resType]
				rs.TotalSupply += storage.Amount
				overview.Resources[resType] = rs
			}
		}
	}

	// Add market price data
	market := p.GetMarket()
	if market != nil {
		snap := market.GetSnapshot()
		for name, rm := range snap.Resources {
			rs := overview.Resources[name]
			rs.BuyPrice = rm.BuyPrice
			rs.SellPrice = rm.SellPrice
			rs.BasePrice = rm.BasePrice
			rs.Demand = rm.TotalDemand
			rs.Trend = rm.PriceVelocity
			overview.Resources[name] = rs
		}
	}

	return overview
}

func handleGetPlanetStorage(p GameStateProvider, planetID int) (interface{}, bool) {
	for _, sys := range p.GetSystems() {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.GetID() == planetID {
				result := make([]PlanetStorageInfo, 0)
				for resType, storage := range planet.StoredResources {
					if storage != nil {
						result = append(result, PlanetStorageInfo{
							Resource: resType,
							Amount:   storage.Amount,
							Capacity: storage.Capacity,
						})
					}
				}
				return result, true
			}
		}
	}
	return nil, false
}

func handleGetPlanetRates(p GameStateProvider, planetID int) (interface{}, bool) {
	for _, sys := range p.GetSystems() {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.GetID() != planetID {
				continue
			}

			production := make(map[string]float64)
			consumption := make(map[string]float64)
			netFlow := make(map[string]float64)

			// Calculate mine production
			for _, resEntity := range planet.Resources {
				res, ok := resEntity.(*entities.Resource)
				if !ok || res.Abundance <= 0 {
					continue
				}
				resIDStr := fmt.Sprintf("%d", res.GetID())
				multiplier := 0.0
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok {
						if b.BuildingType == "Mine" && b.AttachedTo == resIDStr && b.IsOperational {
							multiplier += b.GetStaffingRatio() * b.ProductionBonus
						}
					}
				}
				if multiplier > 0 {
					abundanceFactor := float64(res.Abundance) / 70.0
					if abundanceFactor > 1.0 {
						abundanceFactor = 1.0
					}
					if abundanceFactor < 0.1 {
						abundanceFactor = 0.1
					}
					amt := 8.0 * res.ExtractionRate * multiplier * abundanceFactor
					production[res.ResourceType] += amt
				}
			}

			// Calculate refinery production
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == "Refinery" && b.IsOperational {
					levelMult := 1.0 + float64(b.Level-1)*0.3
					production["Fuel"] += 3.0 * levelMult
					consumption["Oil"] += 2.0 * levelMult
				}
			}

			// Population consumption (from economy.PopulationConsumption)
			for _, rate := range economy.PopulationConsumption {
				consumption[rate.ResourceType] += float64(planet.Population) / rate.PopDivisor * rate.PerPopulation
			}

			// Building upkeep (from economy.BuildingResourceUpkeep)
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.IsOperational {
					if upkeeps, found := economy.BuildingResourceUpkeep[b.BuildingType]; found {
						for _, u := range upkeeps {
							consumption[u.ResourceType] += float64(u.Amount)
						}
					}
				}
			}

			// Net flow
			allRes := make(map[string]bool)
			for r := range production {
				allRes[r] = true
			}
			for r := range consumption {
				allRes[r] = true
			}
			for r := range allRes {
				netFlow[r] = production[r] - consumption[r]
			}

			return PlanetRates{
				PlanetID:    planet.GetID(),
				PlanetName:  planet.Name,
				Population:  planet.Population,
				Production:  production,
				Consumption: consumption,
				NetFlow:     netFlow,
			}, true
		}
	}
	return nil, false
}

func handleGetConstructionQueue(p GameStateProvider) interface{} {
	constructionSystem := tickable.GetSystemByName("Construction")
	if constructionSystem == nil {
		return []ConstructionQueueItem{}
	}
	cs, ok := constructionSystem.(*tickable.ConstructionSystem)
	if !ok {
		return []ConstructionQueueItem{}
	}

	allQueues := cs.GetAllQueues()
	result := make([]ConstructionQueueItem, 0)
	for _, items := range allQueues {
		for _, item := range items {
			progress := 0
			if item.TotalTicks > 0 {
				progress = 100 - (item.RemainingTicks*100)/item.TotalTicks
			}
			result = append(result, ConstructionQueueItem{
				ID:             item.ID,
				Name:           item.Name,
				Location:       item.Location,
				Progress:       progress,
				RemainingTicks: item.RemainingTicks,
				TotalTicks:     item.TotalTicks,
			})
		}
	}
	return result
}
