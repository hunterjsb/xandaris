package api

import (
	"fmt"
	"math"
	"strings"

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
	GetFleetManagementSystem() *game.FleetManagementSystem
	GetEventLog() *game.EventLog
	GetChatLog() *game.ChatLog
	GetRegistry() *game.PlayerRegistry
	GetTickInfo() (tick int64, gameTime string, speed string, paused bool)
	GetCommandChannel() chan game.GameCommand
	GetStandingOrders(player string) []*game.StandingOrder
	GetDeliveryManager() *economy.DeliveryManager
	GetShippingManager() *game.ShippingManager
	GetCreditLedger() *economy.CreditLedger
	GetOrderBook() *economy.OrderBook
	GetContractManager() *economy.ContractManager
	RemovePlayer(name string) bool
}

// findPlayer returns the player matching the given name, or falls back to the human player.
func findPlayer(p GameStateProvider, name string) *entities.Player {
	if name != "" {
		for _, player := range p.GetPlayers() {
			if player != nil && strings.EqualFold(player.Name, name) {
				return player
			}
		}
	}
	return p.GetHumanPlayer()
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
		owner := ""
		var totalPop int64
		resSet := make(map[string]bool)
		for _, e := range sys.Entities {
			switch v := e.(type) {
			case *entities.Star:
				starType = v.StarType
			case *entities.Planet:
				planets++
				totalPop += v.Population
				if v.Owner != "" && owner == "" {
					owner = v.Owner
				}
				for _, resEntity := range v.Resources {
					if res, ok := resEntity.(*entities.Resource); ok {
						resSet[res.ResourceType] = true
					}
				}
			}
		}
		resources := make([]string, 0, len(resSet))
		for r := range resSet {
			resources = append(resources, r)
		}
		summary := SystemSummary{
			ID:       sys.ID,
			Name:     sys.Name,
			X:        sys.X,
			Y:        sys.Y,
			StarType: starType,
			Planets:  planets,
			Links:    links[sys.ID],
		}
		if owner != "" {
			summary.Owner = owner
		}
		if totalPop > 0 {
			summary.Population = totalPop
		}
		if len(resources) > 0 {
			summary.Resources = resources
		}
		result = append(result, summary)
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
				ExtractionRate: math.Round(res.ExtractionRate*10) / 10,
				HasMine:        hasMine,
			})
		}
	}

	return PlanetDetail{
		ID:                planet.GetID(),
		Name:              planet.Name,
		PlanetType:        planet.PlanetType,
		Population:        planet.Population,
		PopulationCap:     planet.GetTotalPopulationCapacity(),
		Habitability:      planet.Habitability,
		Happiness:         math.Round(planet.Happiness*100) / 100,
		ProductivityBonus: math.Round(planet.ProductivityBonus*100) / 100,
		TechLevel:         math.Round(planet.TechLevel*100) / 100,
		TechEra:           entities.TechEraName(planet.TechLevel),
		StorageCapacity:   int(float64(entities.DEFAULT_RESOURCE_CAPACITY) * (1.0 + planet.TechLevel*0.2)),
		PowerGenerated:    math.Round(planet.PowerGenerated*10) / 10,
		PowerConsumed:     math.Round(planet.PowerConsumed*10) / 10,
		PowerRatio:        math.Round(planet.GetPowerRatio()*100) / 100,
		Owner:             planet.Owner,
		StoredResources:   stored,
		ResourceDeposits:  deposits,
		Buildings:         buildings,
		SystemID:          systemID,
	}
}

func handleGetPowerGrid(p GameStateProvider) interface{} {
	type planetPower struct {
		PlanetID       int       `json:"planet_id"`
		PlanetName     string    `json:"planet_name"`
		Owner          string    `json:"owner"`
		Generated      float64   `json:"generated_mw"`
		Consumed       float64   `json:"consumed_mw"`
		Ratio          float64   `json:"ratio"`
		Generators     int       `json:"generators"`
		FusionReactors int       `json:"fusion_reactors"`
		FuelStored     int       `json:"fuel_stored"`
		He3Stored      int       `json:"he3_stored"`
		History        []float64 `json:"history,omitempty"`
	}
	var result []planetPower
	for _, player := range p.GetPlayers() {
		if player == nil {
			continue
		}
		for _, planet := range player.OwnedPlanets {
			if planet == nil {
				continue
			}
			gens, reactors := 0, 0
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					if b.BuildingType == "Generator" {
						gens++
					} else if b.BuildingType == "Fusion Reactor" {
						reactors++
					}
				}
			}
			result = append(result, planetPower{
				PlanetID:       planet.GetID(),
				PlanetName:     planet.Name,
				Owner:          player.Name,
				Generated:      math.Round(planet.PowerGenerated*10) / 10,
				Consumed:       math.Round(planet.PowerConsumed*10) / 10,
				Ratio:          math.Round(planet.GetPowerRatio()*100) / 100,
				Generators:     gens,
				FusionReactors: reactors,
				FuelStored:     planet.GetStoredAmount("Fuel"),
				He3Stored:      planet.GetStoredAmount("Helium-3"),
				History:        planet.PowerHistory,
			})
		}
	}
	return result
}

func handleGetLeaderboard(p GameStateProvider) interface{} {
	players := p.GetPlayers()

	entries := make([]LeaderboardEntry, 0, len(players))
	for _, pl := range players {
		if pl == nil {
			continue
		}
		pType := "ai"
		if pl.IsHuman() {
			pType = "human"
		}

		var pop int64
		bldgs := 0
		stockValue := 0
		maxTech := 0.0
		for _, planet := range pl.OwnedPlanets {
			if planet == nil {
				continue
			}
			pop += planet.Population
			bldgs += len(planet.Buildings)
			if planet.TechLevel > maxTech {
				maxTech = planet.TechLevel
			}
			// Use BASE prices for stable scoring (not volatile market prices)
			for resType, s := range planet.StoredResources {
				if s != nil {
					stockValue += int(float64(s.Amount) * economy.GetBasePrice(resType))
				}
			}
		}

		// Score: credits + stock + pop/10 + buildings*200 + ships*500 + planets*2000 + tech*1000
		techScore := int(maxTech * 1000)
		score := pl.Credits + stockValue + int(pop/10) + bldgs*200 + len(pl.OwnedShips)*500 + len(pl.OwnedPlanets)*2000 + techScore

		entries = append(entries, LeaderboardEntry{
			Name:       pl.Name,
			Type:       pType,
			Score:      score,
			Credits:    pl.Credits,
			Population: pop,
			Planets:    len(pl.OwnedPlanets),
			Ships:      len(pl.OwnedShips),
			Buildings:  bldgs,
			StockValue: stockValue,
			MaxTech:    math.Round(maxTech*100) / 100,
			TechEra:    entities.TechEraName(maxTech),
		})
	}

	// Sort by score descending
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].Score > entries[i].Score {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	for i := range entries {
		entries[i].Rank = i + 1
	}

	return entries
}

func handleRemovePlayer(p GameStateProvider, name string) (interface{}, error) {
	if p.RemovePlayer(name) {
		return map[string]string{"removed": name}, nil
	}
	return nil, fmt.Errorf("player not found: %s", name)
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
		mines := 0
		bldgs := 0
		stock := 0
		var pop int64
		for _, planet := range pl.OwnedPlanets {
			if planet == nil {
				continue
			}
			pop += planet.Population
			bldgs += len(planet.Buildings)
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == "Mine" {
					mines++
				}
			}
			for _, s := range planet.StoredResources {
				if s != nil {
					stock += s.Amount
				}
			}
		}
		result = append(result, PlayerInfo{
			ID:         pl.ID,
			Name:       pl.Name,
			Type:       pType,
			Credits:    pl.Credits,
			Planets:    len(pl.OwnedPlanets),
			Ships:      len(pl.OwnedShips),
			Fleets:     len(pl.OwnedFleets),
			Mines:      mines,
			Buildings:  bldgs,
			Population: pop,
			Stock:      stock,
		})
	}
	return result
}

func handleGetPlayerMe(p GameStateProvider, authPlayer string) interface{} {
	human := findPlayer(p, authPlayer)
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
		// Find the authoritative planet from system entities (OwnedPlanets may be stale)
		sysID := 0
		livePlanet := planet
		for _, sys := range p.GetSystems() {
			for _, e := range sys.Entities {
				if pl, ok := e.(*entities.Planet); ok && pl.GetID() == planet.GetID() {
					sysID = sys.ID
					livePlanet = pl // use the system entity version (has current buildings)
					break
				}
			}
			if sysID != 0 {
				break
			}
		}
		planets = append(planets, buildPlanetDetail(livePlanet, sysID))
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
			CargoUsed:      ship.GetTotalCargo(),
			CargoMax:       ship.MaxCargo,
			CargoHold:      cargo,
			TravelProgress: ship.TravelProgress,
			RoutePath:      ship.RoutePath,
		})
	}

	return playerMe{
		Name:    human.Name,
		Credits: human.Credits,
		Planets: planets,
		Ships:   ships,
	}
}

func handleGetStatus(p GameStateProvider, authPlayer string) interface{} {
	tick, gameTime, speed, paused := p.GetTickInfo()

	// Player info
	var playerStatus PlayerStatus
	human := findPlayer(p, authPlayer)
	if human != nil {
		playerStatus.Name = human.Name
		playerStatus.Credits = human.Credits
		playerStatus.Ships = len(human.OwnedShips)
		playerStatus.Planets = make([]PlanetBrief, 0)

		for _, planet := range human.OwnedPlanets {
			if planet == nil {
				continue
			}
			// Find the authoritative planet from system entities
			livePlanet := planet
			sysID := 0
			for _, sys := range p.GetSystems() {
				for _, e := range sys.Entities {
					if pl, ok := e.(*entities.Planet); ok && pl.GetID() == planet.GetID() {
						sysID = sys.ID
						livePlanet = pl
						break
					}
				}
				if sysID != 0 {
					break
				}
			}
			storage := make(map[string]int)
			for resType, s := range livePlanet.StoredResources {
				if s != nil {
					storage[resType] = s.Amount
				}
			}
			mines := 0
			bldgCount := 0
			for _, be := range livePlanet.Buildings {
				bldgCount++
				if b, ok := be.(*entities.Building); ok && b.BuildingType == "Mine" {
					mines++
				}
			}
			playerStatus.Planets = append(playerStatus.Planets, PlanetBrief{
				ID:              planet.GetID(),
				Name:            planet.Name,
				SystemID:        sysID,
				Population:      planet.Population,
				Storage:         storage,
				StorageCapacity: int(float64(entities.DEFAULT_RESOURCE_CAPACITY) * (1.0 + planet.TechLevel*0.2)),
				Buildings:       bldgCount,
				Mines:           mines,
				TechLevel:       math.Round(planet.TechLevel*100) / 100,
				TechEra:         entities.TechEraName(planet.TechLevel),
			})
		}
	}

	// Economy
	econ := handleGetEconomy(p).(EconomyOverview)

	// Generate actionable hints based on state
	hints := generateHints(human, &playerStatus, &econ)

	return GameStatus{
		Tick:     tick,
		GameTime: gameTime,
		Speed:    speed,
		Paused:   paused,
		Player:   playerStatus,
		Economy:  econ,
		Hints:    hints,
	}
}

func generateHints(human *entities.Player, player *PlayerStatus, econ *EconomyOverview) []string {
	var hints []string
	if human == nil {
		return hints
	}

	// Check if player has no mines
	totalMines := 0
	for _, p := range player.Planets {
		totalMines += p.Mines
	}
	if totalMines == 0 {
		hints = append(hints, "Build mines on resource deposits to start producing")
	}

	// Check for critical/depleted resources
	for name, r := range econ.Resources {
		if r.Scarcity == "Critical" || r.Scarcity == "Depleted" {
			hints = append(hints, fmt.Sprintf("%s is %s — find deposits and build mines", name, r.Scarcity))
		}
	}

	// Check buildings
	hasShipyard := false
	hasRefinery := false
	hasFuel := false
	hasOil := false
	for _, planet := range human.OwnedPlanets {
		if planet == nil {
			continue
		}
		if planet.GetStoredAmount("Fuel") > 0 {
			hasFuel = true
		}
		if planet.GetStoredAmount("Oil") > 20 {
			hasOil = true
		}
		for _, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok {
				if b.BuildingType == "Shipyard" {
					hasShipyard = true
				}
				if b.BuildingType == "Refinery" {
					hasRefinery = true
				}
			}
		}
	}
	// Find highest tech level across planets
	maxTech := 0.0
	hasElectronics := false
	hasFactory := false
	for _, planet := range human.OwnedPlanets {
		if planet == nil {
			continue
		}
		if planet.TechLevel > maxTech {
			maxTech = planet.TechLevel
		}
		if planet.GetStoredAmount("Electronics") > 0 {
			hasElectronics = true
		}
		for _, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok && b.BuildingType == "Factory" {
				hasFactory = true
			}
		}
	}

	// Tech progression hints
	if maxTech < 0.5 && !hasElectronics {
		hints = append(hints, "Buy Electronics from the market to grow tech level — POST /api/market/trade {resource: \"Electronics\", quantity: 10, action: \"buy\"}")
	}
	if maxTech >= 0.5 && !hasRefinery && hasOil {
		hints = append(hints, "Tech 0.5 reached — build a Refinery to convert Oil into Fuel")
	} else if !hasRefinery && hasOil && !hasFuel && maxTech < 0.5 {
		hints = append(hints, "Refinery unlocks at Tech 0.5 — buy Electronics to advance")
	}
	if maxTech >= 1.0 && !hasShipyard && human.Credits > 2000 {
		hints = append(hints, "Tech 1.0 reached — build a Shipyard for ships")
	} else if !hasShipyard && maxTech < 1.0 {
		hints = append(hints, fmt.Sprintf("Shipyard unlocks at Tech 1.0 (current: %.1f) — stockpile Electronics", maxTech))
	}
	if maxTech >= 1.0 && !hasFactory {
		hints = append(hints, "Build a Factory to produce Electronics locally (no more buying!)")
	}
	if maxTech >= 2.5 {
		hasLab := false
		for _, planet := range human.OwnedPlanets {
			if planet == nil {
				continue
			}
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingResearchLab {
					hasLab = true
				}
			}
		}
		if !hasLab {
			hints = append(hints, "Tech 2.5 reached — build a Research Lab for passive Electronics generation")
		}
	}

	// Post-infrastructure progression hints
	if hasShipyard && len(human.OwnedShips) <= 1 && len(human.OwnedFleets) == 0 {
		hints = append(hints, "Build a Cargo ship at your Shipyard — POST /api/ships/build {ship_type: \"Cargo\"}")
	}

	if human.Credits > 50000 {
		hints = append(hints, "Excess credits — invest in mine upgrades or new buildings")
	}

	// Check low fuel on ships
	for _, ship := range human.OwnedShips {
		if ship != nil && ship.CurrentFuel < ship.MaxFuel/4 {
			hints = append(hints, fmt.Sprintf("Ship %s is low on fuel — orbit a planet with Fuel to refuel", ship.Name))
			break
		}
	}

	// Check for colony ships ready to colonize
	for _, ship := range human.OwnedShips {
		if ship != nil && ship.ShipType == entities.ShipTypeColony && ship.Colonists > 0 {
			hints = append(hints, "Colony ship ready — move to an unclaimed habitable planet and POST /api/colonize")
			break
		}
	}

	// Resource depletion warnings
	for _, planet := range human.OwnedPlanets {
		if planet == nil {
			continue
		}
		for _, resEntity := range planet.Resources {
			if res, ok := resEntity.(*entities.Resource); ok && res.Abundance > 0 && res.Abundance < 15 {
				hints = append(hints, fmt.Sprintf("%s deposit on %s nearly depleted (abundance %d)", res.ResourceType, planet.Name, res.Abundance))
			}
		}
	}

	// Price-driven investment hints
	if totalMines > 0 {
		for name, r := range econ.Resources {
			if r.PriceRatio > 2.0 {
				hints = append(hints, fmt.Sprintf("%s at %.0fx base price — build mines or sell stock", name, r.PriceRatio))
				break
			}
		}
	}

	return hints
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
				CargoUsed:      ship.GetTotalCargo(),
				CargoMax:       ship.MaxCargo,
				CargoHold:      cargo,
				TravelProgress: ship.TravelProgress,
				RoutePath:      ship.RoutePath,
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
					CargoUsed:      ship.GetTotalCargo(),
					CargoMax:       ship.MaxCargo,
					CargoHold:      cargo,
					TravelProgress: ship.TravelProgress,
					RoutePath:      ship.RoutePath,
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

	// Add market price data and trade volume
	market := p.GetMarket()
	if market != nil {
		overview.TradeVolume = market.GetTradeVolume()
	}
	if market != nil {
		snap := market.GetSnapshot()
		for name, rm := range snap.Resources {
			rs := overview.Resources[name]
			rs.BuyPrice = rm.BuyPrice
			rs.SellPrice = rm.SellPrice
			rs.BasePrice = rm.BasePrice
			rs.Demand = rm.TotalDemand
			rs.Trend = rm.PriceVelocity
			rs.ImportFee = economy.ComputeImportFee(rm.TotalSupply, rm.TotalDemand)
			if rm.BasePrice > 0 {
				rs.PriceRatio = rm.CurrentPrice / rm.BasePrice
			}

			rs.Scarcity = economy.ComputeScarcity(rm.TotalSupply, rm.TotalDemand)
			if len(rm.PriceHistory) > 0 {
				rs.PriceHistory = rm.PriceHistory
			}
			overview.Resources[name] = rs
		}
	}

	// GDP: total production value per interval (production × base price)
	gdp := 0.0
	totalPlanets := 0
	for _, player := range p.GetPlayers() {
		if player == nil {
			continue
		}
		totalPlanets += len(player.OwnedPlanets)
	}
	// Use flows data for GDP calculation
	if market != nil {
		snap := market.GetSnapshot()
		for _, rm := range snap.Resources {
			// GDP contribution = production rate × base price
			// (we approximate from supply/demand signals)
			if rm.TotalDemand > 0 {
				gdp += rm.TotalDemand * rm.BasePrice
			}
		}
		_ = snap
	}
	overview.GDP = math.Round(gdp * 10) / 10
	overview.TotalPlanets = totalPlanets

	// Logistics stats
	if dm := p.GetDeliveryManager(); dm != nil {
		overview.ActiveDeliveries = len(dm.GetActiveDeliveries())
	}
	if sm := p.GetShippingManager(); sm != nil {
		overview.ActiveRoutes = len(sm.GetRoutes(""))
	}

	return overview
}

func handleGetSystemPrices(p GameStateProvider) interface{} {
	market := p.GetMarket()
	if market == nil {
		return []SystemPrices{}
	}

	// Only include systems with owned planets (active markets)
	systemHasPlanets := make(map[int]bool)
	for _, player := range p.GetPlayers() {
		if player == nil {
			continue
		}
		for _, planet := range player.OwnedPlanets {
			if planet == nil {
				continue
			}
			for _, sys := range p.GetSystems() {
				for _, e := range sys.Entities {
					if pl, ok := e.(*entities.Planet); ok && pl.GetID() == planet.GetID() {
						systemHasPlanets[sys.ID] = true
					}
				}
			}
		}
	}

	result := make([]SystemPrices, 0)
	snap := market.GetSnapshot()

	for _, sys := range p.GetSystems() {
		if !systemHasPlanets[sys.ID] {
			continue
		}
		prices := make(map[string]float64)
		for name := range snap.Resources {
			prices[name] = market.GetLocalBuyPrice(name, sys.ID)
		}
		result = append(result, SystemPrices{
			SystemID:   sys.ID,
			SystemName: sys.Name,
			Prices:     prices,
		})
	}
	return result
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

			techBonus := 1.0 + planet.TechLevel*0.03
			powerFactor := 0.25 + 0.75*planet.GetPowerRatio()

			// Calculate mine production (matches resource_accumulation.go)
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
					amt := 8.0 * res.ExtractionRate * multiplier * abundanceFactor * techBonus * powerFactor
					production[res.ResourceType] += amt
				}
			}

			// Calculate refinery production (matches refinery_production.go)
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == "Refinery" && b.IsOperational {
					levelMult := 1.0 + float64(b.Level-1)*0.3
					combined := levelMult * powerFactor * techBonus
					production["Fuel"] += 3.0 * combined
					consumption["Oil"] += 2.0 * combined
				}
			}

			// Calculate factory production (matches factory_production.go)
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == "Factory" && b.IsOperational {
					levelMult := 1.0 + float64(b.Level-1)*0.3
					staffing := b.GetStaffingRatio()
					combined := levelMult * staffing * powerFactor * techBonus
					production["Electronics"] += 2.0 * combined
					consumption["Rare Metals"] += 2.0 * combined
					consumption["Iron"] += 1.0 * combined
				}
			}

			// Research Lab production (matches research_production.go)
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == "Research Lab" && b.IsOperational {
					levelMult := 1.0 + float64(b.Level-1)*0.3
					staffing := b.GetStaffingRatio()
					combined := levelMult * staffing * powerFactor * techBonus
					production["Electronics"] += 1.0 * combined
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
				PlanetID:          planet.GetID(),
				PlanetName:        planet.Name,
				Population:        planet.Population,
				Happiness:         math.Round(planet.Happiness*100) / 100,
				ProductivityBonus: math.Round(planet.ProductivityBonus*100) / 100,
				TechLevel:         math.Round(planet.TechLevel*100) / 100,
				TechEra:           entities.TechEraName(planet.TechLevel),
				Production:        production,
				Consumption:       consumption,
				NetFlow:           netFlow,
			}, true
		}
	}
	return nil, false
}

func handleGetCatalog() interface{} {
	buildingTypes := []struct {
		name        string
		description string
		maxLevel    int
		workers     int
		produces    map[string]int
		consumes    map[string]int
	}{
		{"Mine", "Extracts resources from planetary deposits", 5, 80, nil, nil},
		{"Trading Post", "Enables market access and generates trade revenue", 5, 150, nil, nil},
		{"Refinery", "Converts Oil into Fuel (2 Oil -> 3 Fuel per interval)", 5, 250,
			map[string]int{"Fuel": 3}, map[string]int{"Oil": 2}},
		{"Factory", "Converts Rare Metals + Iron into Electronics (2 RM + 1 Iron -> 2 Elec)", 5, 300,
			map[string]int{"Electronics": 2}, map[string]int{"Rare Metals": 2, "Iron": 1}},
		{"Generator", "Burns Fuel to produce 50 MW power (3 Fuel/interval)", 5, 100, nil, nil},
		{"Fusion Reactor", "Helium-3 fusion produces 200 MW power (1 He-3/interval)", 5, 200, nil, nil},
		{"Habitat", "Provides housing for population (+700 capacity per level)", 10, 200, nil, nil},
		{"Shipyard", "Enables ship construction", 5, 400, nil, nil},
		{"Research Lab", "Generates 1 Electronics/interval passively (no inputs)", 5, 200,
			map[string]int{"Electronics": 1}, nil},
	}

	buildings := make([]CatalogBuilding, 0, len(buildingTypes))
	for _, bt := range buildingTypes {
		resUpkeep := make(map[string]int)
		if upkeeps, found := economy.BuildingResourceUpkeep[bt.name]; found {
			for _, u := range upkeeps {
				resUpkeep[u.ResourceType] = u.Amount
			}
		}
		creditUpkeep := 0
		if cu, found := economy.BuildingCreditUpkeep[bt.name]; found {
			creditUpkeep = cu
		}
		buildings = append(buildings, CatalogBuilding{
			Type:           bt.name,
			Description:    bt.description,
			Cost:           game.GetBuildingCost(bt.name),
			TechRequired:   entities.GetTechRequirement(bt.name),
			MaxLevel:       bt.maxLevel,
			Workers:        bt.workers,
			CreditUpkeep:   creditUpkeep,
			ResourceUpkeep: resUpkeep,
			Produces:       bt.produces,
			Consumes:       bt.consumes,
		})
	}

	shipTypes := []entities.ShipType{
		entities.ShipTypeScout,
		entities.ShipTypeCargo,
		entities.ShipTypeColony,
		entities.ShipTypeFrigate,
		entities.ShipTypeDestroyer,
		entities.ShipTypeCruiser,
	}

	ships := make([]CatalogShip, 0, len(shipTypes))
	for _, st := range shipTypes {
		ships = append(ships, CatalogShip{
			Type:         string(st),
			Cost:         entities.GetShipBuildCost(st),
			BuildTime:    entities.GetShipBuildTime(st),
			TechRequired: entities.GetShipTechRequirement(st),
			Resources:    entities.GetShipResourceRequirements(st),
			MaxFuel:      entities.GetShipMaxFuel(st),
			MaxCargo:     entities.GetShipMaxCargo(st),
			MaxHealth:    entities.GetShipMaxHealth(st),
		})
	}

	// Population consumption rates
	popConsumption := make([]PopConsumptionRate, 0, len(economy.PopulationConsumption))
	for _, rate := range economy.PopulationConsumption {
		popConsumption = append(popConsumption, PopConsumptionRate{
			Resource:      rate.ResourceType,
			PerPopulation: rate.PerPopulation,
			PopDivisor:    rate.PopDivisor,
		})
	}

	// Resource catalog
	resources := []CatalogResource{
		{"Iron", economy.GetBasePrice("Iron"), "mining"},
		{"Water", economy.GetBasePrice("Water"), "mining"},
		{"Oil", economy.GetBasePrice("Oil"), "mining"},
		{"Rare Metals", economy.GetBasePrice("Rare Metals"), "mining"},
		{"Helium-3", economy.GetBasePrice("Helium-3"), "mining"},
		{"Fuel", economy.GetBasePrice("Fuel"), "refining"},
		{"Electronics", economy.GetBasePrice("Electronics"), "manufacturing"},
	}

	return Catalog{
		Buildings:             buildings,
		Ships:                 ships,
		Resources:             resources,
		PopulationConsumption: popConsumption,
	}
}

func handleGetWorkforce(p GameStateProvider, planetID int) (interface{}, bool) {
	for _, sys := range p.GetSystems() {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.GetID() != planetID {
				continue
			}

			buildings := make([]WorkforceEntry, 0)
			for i, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					buildings = append(buildings, WorkforceEntry{
						Index:    i,
						Type:     b.BuildingType,
						Level:    b.Level,
						Assigned: int64(b.WorkersAssigned),
						Required: int64(b.WorkersRequired),
						Staffing: b.GetStaffingRatio(),
						Online:   b.IsOperational,
					})
				}
			}

			return WorkforceInfo{
				PlanetID:       planet.GetID(),
				PlanetName:     planet.Name,
				Population:     planet.Population,
				PopulationCap:  planet.GetTotalPopulationCapacity(),
				WorkforceTotal: planet.WorkforceTotal,
				WorkforceUsed:  planet.WorkforceUsed,
				WorkforceFree:  planet.GetAvailableWorkforce(),
				Buildings:      buildings,
			}, true
		}
	}
	return nil, false
}

func handleGetDeposits(p GameStateProvider, filterResource string, filterUnmined bool, filterOwner string) interface{} {
	type DepositInfo struct {
		SystemID     int     `json:"system_id"`
		SystemName   string  `json:"system_name"`
		PlanetID     int     `json:"planet_id"`
		PlanetName   string  `json:"planet_name"`
		Owner        string  `json:"owner"`
		ResourceType string  `json:"resource_type"`
		ResourceID   int     `json:"resource_id"`
		Abundance    int     `json:"abundance"`
		Rate         float64 `json:"extraction_rate"`
		HasMine      bool    `json:"has_mine"`
	}

	deposits := make([]DepositInfo, 0)
	for _, sys := range p.GetSystems() {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok {
				continue
			}
			for _, resEntity := range planet.Resources {
				res, ok := resEntity.(*entities.Resource)
				if !ok || res.Abundance <= 0 {
					continue
				}
				hasMine := false
				resIDStr := fmt.Sprintf("%d", res.GetID())
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok {
						if b.BuildingType == "Mine" && b.AttachedTo == resIDStr {
							hasMine = true
							break
						}
					}
				}
				// Apply filters
				if filterResource != "" && !strings.EqualFold(res.ResourceType, filterResource) {
					continue
				}
				if filterUnmined && hasMine {
					continue
				}
				if filterOwner != "" && !strings.EqualFold(planet.Owner, filterOwner) {
					continue
				}
				deposits = append(deposits, DepositInfo{
					SystemID:     sys.ID,
					SystemName:   sys.Name,
					PlanetID:     planet.GetID(),
					PlanetName:   planet.Name,
					Owner:        planet.Owner,
					ResourceType: res.ResourceType,
					ResourceID:   res.GetID(),
					Abundance:    res.Abundance,
					Rate:         math.Round(res.ExtractionRate*10) / 10,
					HasMine:      hasMine,
				})
			}
		}
	}
	return deposits
}

func handleGetGalaxyFlows(p GameStateProvider) interface{} {
	production := make(map[string]float64)
	consumption := make(map[string]float64)
	var totalPop int64

	for _, player := range p.GetPlayers() {
		if player == nil {
			continue
		}
		for _, planet := range player.OwnedPlanets {
			if planet == nil {
				continue
			}
			totalPop += planet.Population

			// Mine production
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
					af := float64(res.Abundance) / 70.0
					if af > 1.0 {
						af = 1.0
					}
					if af < 0.1 {
						af = 0.1
					}
					production[res.ResourceType] += 8.0 * res.ExtractionRate * multiplier * af
				}
			}

			// Refinery production
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == "Refinery" && b.IsOperational {
					lm := 1.0 + float64(b.Level-1)*0.3
					production["Fuel"] += 3.0 * lm
					consumption["Oil"] += 2.0 * lm
				}
			}

			// Factory production
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == "Factory" && b.IsOperational {
					lm := 1.0 + float64(b.Level-1)*0.3
					production["Electronics"] += 2.0 * lm
					consumption["Rare Metals"] += 2.0 * lm
					consumption["Iron"] += 1.0 * lm
				}
			}

			// Population consumption
			for _, rate := range economy.PopulationConsumption {
				consumption[rate.ResourceType] += float64(planet.Population) / rate.PopDivisor * rate.PerPopulation
			}

			// Building upkeep
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.IsOperational {
					if upkeeps, found := economy.BuildingResourceUpkeep[b.BuildingType]; found {
						for _, u := range upkeeps {
							consumption[u.ResourceType] += float64(u.Amount)
						}
					}
				}
			}
		}
	}

	// Net flow
	netFlow := make(map[string]float64)
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

	return GalaxyFlows{
		Production:  production,
		Consumption: consumption,
		NetFlow:     netFlow,
		Population:  totalPop,
	}
}

func handleGetConstructionQueue(p GameStateProvider) interface{} {
	cs := tickable.GetConstructionSystem()
	if cs == nil {
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
				Owner:          item.Owner,
				Progress:       progress,
				RemainingTicks: item.RemainingTicks,
				TotalTicks:     item.TotalTicks,
			})
		}
	}
	return result
}

// handleGetDiagnostics returns diagnostic info about tickable systems
func handleGetDiagnostics(p GameStateProvider) interface{} {
	type diag struct {
		ShipsInSystems    int      `json:"ships_in_systems"`
		ShipsInPlayers    int      `json:"ships_in_players"`
		MovingShips       int      `json:"moving_ships"`
		OrphanedShips     int      `json:"orphaned_ships"`
		OrphanedNames     []string `json:"orphaned_names"`
		SystemCount       int      `json:"system_count"`
	}
	d := diag{}
	
	// Count ships in system entities
	systemShipIDs := make(map[int]bool)
	for _, sys := range p.GetSystems() {
		for _, e := range sys.Entities {
			if ship, ok := e.(*entities.Ship); ok {
				systemShipIDs[ship.GetID()] = true
				d.ShipsInSystems++
			}
		}
	}
	d.SystemCount = len(p.GetSystems())

	// Count ships in player ownership + collect moving ship details
	type movingDetail struct {
		Name     string  `json:"name"`
		Status   string  `json:"status"`
		System   int     `json:"sys"`
		Target   int     `json:"target"`
		Progress float64 `json:"progress"`
		Fuel     int     `json:"fuel"`
		Speed    float64 `json:"speed"`
		InSysEnt bool    `json:"in_sys_ent"`
	}
	var movingDetails []movingDetail

	for _, player := range p.GetPlayers() {
		if player == nil { continue }
		for _, ship := range player.OwnedShips {
			if ship == nil { continue }
			d.ShipsInPlayers++
			if ship.Status == entities.ShipStatusMoving {
				d.MovingShips++
				movingDetails = append(movingDetails, movingDetail{
					Name:     ship.Name,
					Status:   string(ship.Status),
					System:   ship.CurrentSystem,
					Target:   ship.TargetSystem,
					Progress: ship.TravelProgress,
					Fuel:     ship.CurrentFuel,
					Speed:    ship.Speed,
					InSysEnt: systemShipIDs[ship.GetID()],
				})
			}
			if !systemShipIDs[ship.GetID()] {
				d.OrphanedShips++
				d.OrphanedNames = append(d.OrphanedNames, ship.Name)
			}
		}
	}

	// Tickable system diagnostics
	smTicks, smShips, smMoving := tickable.GetShipMovementDiag()
	smPlayerMoving := tickable.GetShipMovementPlayers()

	type result struct {
		diag
		MovingDetail    []movingDetail `json:"moving_detail"`
		SMTickCount     int64          `json:"sm_tick_count"`
		SMShipsFound    int            `json:"sm_ships_found"`
		SMMovingFound   int            `json:"sm_moving_found"`
		SMPlayerMoving  int            `json:"sm_player_moving"`
	}
	return result{d, movingDetails, smTicks, smShips, smMoving, smPlayerMoving}
}

// handleGetDefense returns per-system military defense ratings.
func handleGetDefense(p GameStateProvider) interface{} {
	type systemDefense struct {
		SystemID    int    `json:"system_id"`
		SystemName  string `json:"system_name"`
		Power       int    `json:"defense_power"`  // total military AttackPower
		Ships       int    `json:"military_ships"` // count of military ships present
		Safe        bool   `json:"safe"`           // power >= 10 (piracy immune)
		Owner       string `json:"owner,omitempty"`
	}

	defensePower := make(map[int]int)
	defenseShips := make(map[int]int)
	for _, player := range p.GetPlayers() {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil {
				continue
			}
			if ship.ShipType == entities.ShipTypeFrigate ||
				ship.ShipType == entities.ShipTypeDestroyer ||
				ship.ShipType == entities.ShipTypeCruiser {
				if ship.Status != entities.ShipStatusMoving {
					defensePower[ship.CurrentSystem] += ship.AttackPower
					defenseShips[ship.CurrentSystem]++
				}
			}
		}
	}

	result := make([]systemDefense, 0)
	for _, sys := range p.GetSystems() {
		power := defensePower[sys.ID]
		ships := defenseShips[sys.ID]
		owner := ""
		for _, e := range sys.Entities {
			if pl, ok := e.(*entities.Planet); ok && pl.Owner != "" {
				owner = pl.Owner
				break
			}
		}
		if power > 0 || owner != "" {
			result = append(result, systemDefense{
				SystemID:   sys.ID,
				SystemName: sys.Name,
				Power:      power,
				Ships:      ships,
				Safe:       power >= 10,
				Owner:      owner,
			})
		}
	}
	return result
}

// handleGetStations returns all stations across the galaxy.
func handleGetStations(p GameStateProvider) interface{} {
	type stationInfo struct {
		ID          int      `json:"id"`
		Name        string   `json:"name"`
		Type        string   `json:"type"`
		SystemID    int      `json:"system_id"`
		SystemName  string   `json:"system_name"`
		Owner       string   `json:"owner"`
		Population  int      `json:"population"`
		Capacity    int      `json:"capacity"`
		DockingFee  int      `json:"docking_fee"`
		Services    []string `json:"services"`
		DefenseLevel int     `json:"defense_level"`
	}

	result := make([]stationInfo, 0)
	for _, sys := range p.GetSystems() {
		for _, e := range sys.Entities {
			if station, ok := e.(*entities.Station); ok {
				result = append(result, stationInfo{
					ID:          station.GetID(),
					Name:        station.Name,
					Type:        station.StationType,
					SystemID:    sys.ID,
					SystemName:  sys.Name,
					Owner:       station.Owner,
					Population:  station.CurrentPop,
					Capacity:    station.Capacity,
					DockingFee:  station.GetDockingFee(),
					Services:    station.Services,
					DefenseLevel: station.DefenseLevel,
				})
			}
		}
	}
	return result
}
