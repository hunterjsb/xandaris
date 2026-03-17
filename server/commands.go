package server

import (
	"fmt"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/utils"
)

// remoteForwardedCommands lists command types that should be forwarded in remote mode.
var remoteForwardedCommands = map[game.CommandType]bool{
	game.CmdTrade: true, game.CmdBuild: true, game.CmdBuildShip: true,
	game.CmdMoveShip: true, game.CmdUpgrade: true, game.CmdRefuel: true,
	game.CmdCargoLoad: true, game.CmdCargoUnload: true, game.CmdColonize: true,
	game.CmdFleetMove: true, game.CmdFleetCreate: true, game.CmdFleetDisband: true,
	game.CmdFleetAddShip: true, game.CmdFleetRemoveShip: true,
	game.CmdWorkforceAssign: true, game.CmdCancelConstruction: true,
	game.CmdDockShip: true, game.CmdUndockShip: true, game.CmdSellAtDock: true,
	game.CmdDemolish: true,
	game.CmdTransferFuel: true,
}

// executeCommand processes a single game command via the registry.
func (gs *GameServer) executeCommand(cmd game.GameCommand) {
	// In remote mode, forward gameplay commands to the remote server
	if gs.remoteSync != nil && remoteForwardedCommands[cmd.Type] {
		gs.forwardCommandToRemote(cmd)
		return
	}

	if err := gs.cmdRegistry.Execute(cmd); err != nil {
		fmt.Printf("[Server] %v\n", err)
	}
}

// initCommandRegistry registers all command handlers.
func (gs *GameServer) initCommandRegistry() {
	cr := NewCommandRegistry()

	cr.Register(game.CmdSave, func(cmd game.GameCommand) {
		if playerName, ok := cmd.Data.(string); ok {
			// Already under gs.mu from DrainCommands — use locked variant to avoid deadlock
			if err := gs.saveGameLocked(playerName); err != nil {
				fmt.Printf("[Server] Save failed: %v\n", err)
			}
		}
	})
	cr.Register(game.CmdSetSpeed, func(cmd game.GameCommand) {
		if speed, ok := cmd.Data.(systems.TickSpeed); ok {
			gs.TickManager.SetSpeed(speed)
		}
	})
	cr.Register(game.CmdTogglePause, func(cmd game.GameCommand) {
		gs.TickManager.TogglePause()
	})
	cr.Register(game.CmdTrade, gs.handleTradeCommand)
	cr.Register(game.CmdCargoLoad, gs.handleCargoCommand)
	cr.Register(game.CmdCargoUnload, gs.handleCargoCommand)
	cr.Register(game.CmdBuild, gs.handleBuildCommand)
	cr.Register(game.CmdBuildShip, gs.handleBuildShipCommand)
	cr.Register(game.CmdMoveShip, gs.handleMoveShipCommand)
	cr.Register(game.CmdUpgrade, gs.handleUpgradeCommand)
	cr.Register(game.CmdRefuel, gs.handleRefuelCommand)
	cr.Register(game.CmdColonize, gs.handleColonizeCommand)
	cr.Register(game.CmdRegisterPlayer, gs.handleRegisterPlayerCommand)
	cr.Register(game.CmdWorkforceAssign, gs.handleWorkforceAssignCommand)
	cr.Register(game.CmdCancelConstruction, gs.handleCancelConstructionCommand)
	cr.Register(game.CmdStandingOrder, gs.handleStandingOrderCommand)
	cr.Register(game.CmdCancelOrder, gs.handleCancelOrderCommand)
	cr.Register(game.CmdFleetMove, gs.handleFleetMoveCommand)
	cr.Register(game.CmdFleetCreate, gs.handleFleetCreateCommand)
	cr.Register(game.CmdFleetDisband, gs.handleFleetDisbandCommand)
	cr.Register(game.CmdFleetAddShip, gs.handleFleetAddShipCommand)
	cr.Register(game.CmdFleetRemoveShip, gs.handleFleetRemoveShipCommand)
	cr.Register(game.CmdDockShip, gs.handleDockShipCommand)
	cr.Register(game.CmdUndockShip, gs.handleUndockShipCommand)
	cr.Register(game.CmdSellAtDock, gs.handleSellAtDockCommand)
	cr.Register(game.CmdDemolish, gs.handleDemolishCommand)
	cr.Register(game.CmdTransferFuel, gs.handleTransferFuelCommand)

	gs.cmdRegistry = cr
}

// resolvePlayer finds the player for a command. Uses cmd.PlayerName if set,
// otherwise falls back to HumanPlayer (backwards compat for local play).
func (gs *GameServer) resolvePlayer(cmd game.GameCommand) *entities.Player {
	if cmd.PlayerName != "" {
		for _, p := range gs.State.Players {
			if p != nil && p.Name == cmd.PlayerName {
				return p
			}
		}
	}
	return gs.State.HumanPlayer
}

func (gs *GameServer) handleBuildCommand(cmd game.GameCommand) {
	bd, ok := cmd.Data.(game.BuildCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid build data"))
		return
	}

	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}

	planet := gs.CargoCommander.FindPlanetByID(bd.PlanetID)
	if planet == nil {
		sendResult(cmd, fmt.Errorf("planet not found"))
		return
	}
	if planet.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("not your planet"))
		return
	}

	// Tech level gate
	techReq := entities.GetTechRequirement(bd.BuildingType)
	if techReq > 0 && planet.TechLevel < techReq {
		sendResult(cmd, fmt.Errorf("%s requires tech level %.1f (planet has %.1f)", bd.BuildingType, techReq, planet.TechLevel))
		return
	}

	// Look up build cost from entity generator (single source of truth)
	cost := game.GetBuildingCost(bd.BuildingType)
	if cost == 0 {
		sendResult(cmd, fmt.Errorf("unknown building type: %s", bd.BuildingType))
		return
	}
	if human.Credits < cost {
		sendResult(cmd, fmt.Errorf("insufficient credits (need %d, have %d)", cost, human.Credits))
		return
	}

	// For mines, determine attachment
	attachmentID := fmt.Sprintf("%d", planet.GetID())
	if bd.BuildingType == entities.BuildingMine {
		if bd.ResourceID > 0 {
			attachmentID = fmt.Sprintf("%d", bd.ResourceID)
		} else {
			sendResult(cmd, fmt.Errorf("mine requires a resource_id"))
			return
		}
	}

	// Deduct cost and queue construction
	human.Credits -= cost

	// Build time scales with cost: 1 tick per 2 credits, minimum 100 ticks
	buildTicks := cost / 2
	if buildTicks < 100 {
		buildTicks = 100
	}

	item := &tickable.ConstructionItem{
		ID:             fmt.Sprintf("api_%d_%d", bd.PlanetID, gs.TickManager.GetCurrentTick()),
		Type:           "Building",
		Name:           bd.BuildingType,
		Location:       attachmentID,
		Owner:          human.Name,
		Progress:       0,
		TotalTicks:     buildTicks,
		RemainingTicks: buildTicks,
		Cost:           cost,
		Started:        gs.TickManager.GetCurrentTick(),
	}

	if cs := tickable.GetConstructionSystem(); cs != nil {
		cs.AddToQueue(attachmentID, item)
	}

	sendSuccess(cmd, map[string]interface{}{
		"building":  bd.BuildingType,
		"planet_id": bd.PlanetID,
		"cost":      cost,
		"ticks":     buildTicks,
	})
}

func (gs *GameServer) handleBuildShipCommand(cmd game.GameCommand) {
	sd, ok := cmd.Data.(game.ShipBuildCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid ship build data"))
		return
	}

	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}

	planet := gs.CargoCommander.FindPlanetByID(sd.PlanetID)
	if planet == nil {
		sendResult(cmd, fmt.Errorf("planet not found"))
		return
	}
	if planet.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("not your planet"))
		return
	}

	if !planet.HasOperationalBuilding(entities.BuildingShipyard) {
		sendResult(cmd, fmt.Errorf("no operational shipyard on this planet"))
		return
	}

	shipType := entities.ShipType(sd.ShipType)
	cost := entities.GetShipBuildCost(shipType)
	if human.Credits < cost {
		sendResult(cmd, fmt.Errorf("insufficient credits (need %d, have %d)", cost, human.Credits))
		return
	}

	// Check resource requirements
	requirements := entities.GetShipResourceRequirements(shipType)
	for resType, amount := range requirements {
		if !planet.HasStoredResource(resType, amount) {
			sendResult(cmd, fmt.Errorf("need %d %s, have %d", amount, resType, planet.GetStoredAmount(resType)))
			return
		}
	}

	// Deduct credits and resources
	human.Credits -= cost
	for resType, amount := range requirements {
		planet.RemoveStoredResource(resType, amount)
	}

	// Queue construction
	buildTime := entities.GetShipBuildTime(shipType)
	location := fmt.Sprintf("planet_%d", planet.GetID())
	item := &tickable.ConstructionItem{
		ID:             fmt.Sprintf("ship_%s_%d", shipType, gs.TickManager.GetCurrentTick()),
		Type:           "Ship",
		Name:           string(shipType),
		Location:       location,
		Owner:          human.Name,
		Progress:       0,
		TotalTicks:     buildTime,
		RemainingTicks: buildTime,
		Cost:           cost,
		Started:        gs.TickManager.GetCurrentTick(),
	}

	if cs := tickable.GetConstructionSystem(); cs != nil {
		cs.AddToQueue(location, item)
	}

	sendSuccess(cmd, map[string]interface{}{
		"ship_type": sd.ShipType,
		"cost":      cost,
		"ticks":     buildTime,
		"resources": requirements,
	})
}

func (gs *GameServer) handleMoveShipCommand(cmd game.GameCommand) {
	md, ok := cmd.Data.(game.ShipMoveCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid move data"))
		return
	}

	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	ship := game.FindShipByID(gs.State.Players, md.ShipID)
	if ship == nil {
		sendResult(cmd, fmt.Errorf("ship not found"))
		return
	}
	if ship.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("not your ship"))
		return
	}

	helper := tickable.NewShipMovementHelper(gs.GetSystemsMap(), gs.State.Hyperlanes)
	if helper.StartJourney(ship, md.TargetSystemID) {
		sendSuccess(cmd, map[string]interface{}{
			"ship_id": md.ShipID,
			"target":  md.TargetSystemID,
			"status":  "moving",
		})
	} else {
		sendResult(cmd, fmt.Errorf("cannot move to system %d (no route or insufficient fuel)", md.TargetSystemID))
	}
}

func (gs *GameServer) handleUpgradeCommand(cmd game.GameCommand) {
	ud, ok := cmd.Data.(game.UpgradeCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid upgrade data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}

	planet := gs.CargoCommander.FindPlanetByID(ud.PlanetID)
	if planet == nil || planet.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("planet not found or not owned"))
		return
	}

	if ud.BuildingIndex < 0 || ud.BuildingIndex >= len(planet.Buildings) {
		sendResult(cmd, fmt.Errorf("invalid building index"))
		return
	}

	building, ok := planet.Buildings[ud.BuildingIndex].(*entities.Building)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid building"))
		return
	}
	if !building.CanUpgrade() {
		sendResult(cmd, fmt.Errorf("building cannot be upgraded (max level or not operational)"))
		return
	}

	cost := building.GetUpgradeCost()
	if human.Credits < cost {
		sendResult(cmd, fmt.Errorf("insufficient credits (need %d, have %d)", cost, human.Credits))
		return
	}

	human.Credits -= cost
	building.Upgrade()

	sendSuccess(cmd, map[string]interface{}{
		"building":  building.BuildingType,
		"new_level": building.Level,
		"cost":      cost,
	})
}

func (gs *GameServer) handleDemolishCommand(cmd game.GameCommand) {
	dd, ok := cmd.Data.(game.DemolishCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid demolish data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}

	planet := gs.CargoCommander.FindPlanetByID(dd.PlanetID)
	if planet == nil || planet.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("planet not found or not owned"))
		return
	}

	if dd.BuildingIndex < 0 || dd.BuildingIndex >= len(planet.Buildings) {
		sendResult(cmd, fmt.Errorf("invalid building index"))
		return
	}

	building, ok := planet.Buildings[dd.BuildingIndex].(*entities.Building)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid building"))
		return
	}

	// Can't demolish the Base
	if building.BuildingType == entities.BuildingBase {
		sendResult(cmd, fmt.Errorf("cannot demolish the Base"))
		return
	}

	// Refund 25% of build cost
	refund := game.GetBuildingCost(building.BuildingType) / 4

	// Remove from planet and rebalance workforce
	planet.Buildings = append(planet.Buildings[:dd.BuildingIndex], planet.Buildings[dd.BuildingIndex+1:]...)
	planet.RebalanceWorkforce()
	human.Credits += refund

	sendSuccess(cmd, map[string]interface{}{
		"demolished": building.BuildingType,
		"level":      building.Level,
		"refund":     refund,
	})
}

func (gs *GameServer) handleRefuelCommand(cmd game.GameCommand) {
	rd, ok := cmd.Data.(game.ShipRefuelCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid refuel data"))
		return
	}

	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	ship := game.FindShipByID(gs.State.Players, rd.ShipID)
	if ship == nil {
		sendResult(cmd, fmt.Errorf("ship not found"))
		return
	}
	if ship.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("not your ship"))
		return
	}

	planet := gs.CargoCommander.FindPlanetByID(rd.PlanetID)
	if planet == nil || planet.Owner != ship.Owner {
		sendResult(cmd, fmt.Errorf("planet not found or not owned"))
		return
	}

	// Check ship is at this planet
	if ship.Status == entities.ShipStatusMoving {
		sendResult(cmd, fmt.Errorf("ship is moving"))
		return
	}

	needed := ship.MaxFuel - ship.CurrentFuel
	if needed <= 0 {
		sendResult(cmd, fmt.Errorf("ship fuel is already full"))
		return
	}

	amount := rd.Amount
	if amount <= 0 || amount > needed {
		amount = needed
	}

	available := planet.GetStoredAmount(entities.ResFuel)
	if available <= 0 {
		sendResult(cmd, fmt.Errorf("no Fuel on planet"))
		return
	}
	if amount > available {
		amount = available
	}

	planet.RemoveStoredResource(entities.ResFuel, amount)
	ship.Refuel(amount)

	sendSuccess(cmd, map[string]interface{}{
		"ship_id":  rd.ShipID,
		"refueled": amount,
		"fuel_now": ship.CurrentFuel,
		"fuel_max": ship.MaxFuel,
	})
}

func (gs *GameServer) handleColonizeCommand(cmd game.GameCommand) {
	cd, ok := cmd.Data.(game.ColonizeCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid colonize data"))
		return
	}

	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}

	// Find the colony ship
	ship := game.FindShipByID(gs.State.Players, cd.ShipID)
	if ship == nil || ship.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("ship not found or not owned"))
		return
	}
	if ship.ShipType != entities.ShipTypeColony {
		sendResult(cmd, fmt.Errorf("not a colony ship"))
		return
	}
	if ship.Colonists <= 0 {
		sendResult(cmd, fmt.Errorf("no colonists on ship"))
		return
	}
	if ship.Status == entities.ShipStatusMoving {
		sendResult(cmd, fmt.Errorf("ship is moving"))
		return
	}

	// Find the target planet
	planet := gs.CargoCommander.FindPlanetByID(cd.PlanetID)
	if planet == nil {
		sendResult(cmd, fmt.Errorf("planet not found"))
		return
	}
	if planet.Owner != "" {
		sendResult(cmd, fmt.Errorf("planet already claimed by %s", planet.Owner))
		return
	}
	if !planet.IsHabitable() {
		sendResult(cmd, fmt.Errorf("planet is not habitable"))
		return
	}

	systemID := gs.CargoCommander.GetSystemForPlanet(planet)
	game.ColonizePlanet(planet, ship, human, systemID)

	// Log the event
	gs.LogEvent("colonize", human.Name, fmt.Sprintf("%s colonized %s!", human.Name, planet.Name))

	sendSuccess(cmd, map[string]interface{}{
		"planet":    planet.Name,
		"planet_id": planet.GetID(),
		"system_id": systemID,
		"colonists": planet.Population,
	})
}

func (gs *GameServer) handleRegisterPlayerCommand(cmd game.GameCommand) {
	rd, ok := cmd.Data.(game.RegisterPlayerCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid register data"))
		return
	}

	// Check name isn't already a player
	for _, p := range gs.State.Players {
		if p != nil && p.Name == rd.Name {
			sendResult(cmd, fmt.Errorf("player name already exists in game"))
			return
		}
	}

	// Create new player with unique ID and color
	playerID := len(gs.State.Players)
	colors := utils.GetAIPlayerColors()
	playerColor := colors[playerID%len(colors)]
	newPlayer := entities.NewPlayer(playerID, rd.Name, playerColor, entities.PlayerTypeHuman)

	// Initialize with a homeworld
	entities.InitializePlayer(newPlayer, gs.State.Systems)
	game.PrepareHomeworld(newPlayer, false)

	// Extra starting resources
	if newPlayer.HomePlanet != nil {
		newPlayer.HomePlanet.AddStoredResource(entities.ResFuel, 200)
		newPlayer.HomePlanet.AddStoredResource(entities.ResOil, 150)
	}

	gs.State.Players = append(gs.State.Players, newPlayer)

	// Update account with player ID
	if gs.Registry != nil {
		if acc := gs.Registry.GetAccount(rd.Name); acc != nil {
			acc.PlayerID = playerID
		}
	}

	// Log event
	if gs.Events != nil {
		gs.Events.Add(gs.TickManager.GetCurrentTick(), gs.TickManager.GetGameTimeFormatted(),
			game.EventType("join"), rd.Name, fmt.Sprintf("%s joined the galaxy!", rd.Name))
	}

	fmt.Printf("[Server] New player registered: %s (id=%d)\n", rd.Name, playerID)

	sendSuccess(cmd, playerID)
}

func (gs *GameServer) handleWorkforceAssignCommand(cmd game.GameCommand) {
	wd, ok := cmd.Data.(game.WorkforceAssignCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid workforce data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	planet := gs.CargoCommander.FindPlanetByID(wd.PlanetID)
	if planet == nil || planet.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("planet not found or not owned"))
		return
	}
	if wd.BuildingIndex < 0 || wd.BuildingIndex >= len(planet.Buildings) {
		sendResult(cmd, fmt.Errorf("invalid building index"))
		return
	}
	building, ok := planet.Buildings[wd.BuildingIndex].(*entities.Building)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid building"))
		return
	}
	building.SetDesiredWorkers(wd.Workers)
	planet.RebalanceWorkforce()
	sendSuccess(cmd, map[string]interface{}{
		"building":  building.BuildingType,
		"desired":   building.DesiredWorkers,
		"assigned":  building.WorkersAssigned,
		"required":  building.WorkersRequired,
		"staffing":  building.GetStaffingRatio(),
	})
}

func (gs *GameServer) handleCancelConstructionCommand(cmd game.GameCommand) {
	cd, ok := cmd.Data.(game.CancelConstructionCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid cancel construction data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}

	cs := tickable.GetConstructionSystem()
	if cs == nil {
		sendResult(cmd, fmt.Errorf("construction system not found"))
		return
	}

	// Find the construction item
	items := cs.GetConstructionsByOwner(human.Name)
	var target *tickable.ConstructionItem
	for _, item := range items {
		if item.ID == cd.ConstructionID {
			target = item
			break
		}
	}
	if target == nil {
		sendResult(cmd, fmt.Errorf("construction not found: %s", cd.ConstructionID))
		return
	}

	// Refund partial cost based on progress
	target.Mutex.RLock()
	progress := target.Progress
	cost := target.Cost
	location := target.Location
	target.Mutex.RUnlock()

	refund := int(float64(cost) * (1.0 - float64(progress)/100.0))
	human.Credits += refund
	cs.RemoveFromQueue(location, cd.ConstructionID)

	sendSuccess(cmd, map[string]interface{}{
		"cancelled": cd.ConstructionID,
		"refund":    refund,
		"progress":  progress,
	})
}

func (gs *GameServer) handleDockShipCommand(cmd game.GameCommand) {
	dd, ok := cmd.Data.(game.DockShipCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid dock data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	ship := game.FindShipByID(gs.State.Players, dd.ShipID)
	if ship == nil || ship.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("ship not found or not owned"))
		return
	}
	planet := gs.CargoCommander.FindPlanetByID(dd.PlanetID)
	if planet == nil {
		sendResult(cmd, fmt.Errorf("planet not found"))
		return
	}
	if err := gs.CargoCommander.DockShip(ship, planet); err != nil {
		sendResult(cmd, err)
		return
	}
	sendSuccess(cmd, map[string]interface{}{
		"ship_id":   dd.ShipID,
		"planet_id": dd.PlanetID,
		"status":    "docked",
	})
}

func (gs *GameServer) handleUndockShipCommand(cmd game.GameCommand) {
	ud, ok := cmd.Data.(game.UndockShipCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid undock data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	ship := game.FindShipByID(gs.State.Players, ud.ShipID)
	if ship == nil || ship.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("ship not found or not owned"))
		return
	}
	if err := gs.CargoCommander.UndockShip(ship); err != nil {
		sendResult(cmd, err)
		return
	}
	sendSuccess(cmd, map[string]interface{}{
		"ship_id": ud.ShipID,
		"status":  "undocked",
	})
}

func (gs *GameServer) handleSellAtDockCommand(cmd game.GameCommand) {
	sd, ok := cmd.Data.(game.SellAtDockCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid sell-at-dock data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	ship := game.FindShipByID(gs.State.Players, sd.ShipID)
	if ship == nil || ship.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("ship not found or not owned"))
		return
	}

	buyPrice := gs.State.Market.GetBuyPrice(sd.Resource)
	sold, credits, err := gs.CargoCommander.SellAtDock(ship, sd.Resource, sd.Quantity, buyPrice, human)
	if err != nil {
		sendResult(cmd, err)
		return
	}

	// Find the TP owner and apply docking fee (goes to planet owner as revenue)
	dockingFee := 0
	if ship.DockedAtPlanet != 0 {
		planet := gs.CargoCommander.FindPlanetByID(ship.DockedAtPlanet)
		if planet != nil && planet.Owner != "" && planet.Owner != human.Name {
			// Fee: 5% base, reduced by TP level (L1=5%, L2=4%, L3=3%, L4=2%, L5=1%)
			feeRate := 0.05
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost {
					feeRate = 0.06 - float64(b.Level)*0.01
					if feeRate < 0.01 {
						feeRate = 0.01
					}
					break
				}
			}
			dockingFee = int(float64(credits) * feeRate)
			// Pay the fee to the planet owner
			for _, p := range gs.State.Players {
				if p != nil && p.Name == planet.Owner {
					p.Credits += dockingFee
					break
				}
			}
			credits -= dockingFee
		}
	}

	// Credit the ship owner (after fee deduction)
	human.Credits += credits

	// Log to trade history
	if gs.State.TradeExec != nil {
		gs.State.Market.AddTradeVolume(sd.Resource, sold, false)
		if gs.State.TradeExec.OnTrade != nil {
			gs.State.TradeExec.OnTrade(economy.TradeRecord{
				Tick:      gs.TickManager.GetCurrentTick(),
				Player:    human.Name,
				Resource:  sd.Resource,
				Quantity:  sold,
				Action:    "sell_at_dock",
				UnitPrice: buyPrice,
				Total:     credits,
			})
		}
	}

	sendSuccess(cmd, map[string]interface{}{
		"ship_id":     sd.ShipID,
		"resource":    sd.Resource,
		"sold":        sold,
		"credits":     credits,
		"docking_fee": dockingFee,
	})
}

func (gs *GameServer) handleTransferFuelCommand(cmd game.GameCommand) {
	td, ok := cmd.Data.(game.TransferFuelCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid transfer data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}

	fromShip := game.FindShipByID(gs.State.Players, td.FromShipID)
	toShip := game.FindShipByID(gs.State.Players, td.ToShipID)
	if fromShip == nil || toShip == nil {
		sendResult(cmd, fmt.Errorf("ship not found"))
		return
	}
	if fromShip.Owner != human.Name || toShip.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("not your ship"))
		return
	}

	// Both ships must be in the same system and not moving
	if fromShip.Status == entities.ShipStatusMoving || toShip.Status == entities.ShipStatusMoving {
		sendResult(cmd, fmt.Errorf("cannot transfer fuel while a ship is moving"))
		return
	}
	if fromShip.CurrentSystem != toShip.CurrentSystem {
		sendResult(cmd, fmt.Errorf("ships must be in the same system"))
		return
	}

	// Calculate transfer amount
	available := fromShip.CurrentFuel
	needed := toShip.MaxFuel - toShip.CurrentFuel
	amount := td.Amount
	if amount <= 0 || amount > needed {
		amount = needed
	}
	if amount > available {
		amount = available
	}
	if amount <= 0 {
		sendResult(cmd, fmt.Errorf("no fuel to transfer"))
		return
	}

	fromShip.CurrentFuel -= amount
	toShip.CurrentFuel += amount

	sendSuccess(cmd, map[string]interface{}{
		"from_ship":     fromShip.Name,
		"to_ship":       toShip.Name,
		"transferred":   amount,
		"from_fuel_now": fromShip.CurrentFuel,
		"to_fuel_now":   toShip.CurrentFuel,
	})
}

func sendResult(cmd game.GameCommand, err error) {
	if cmd.Result != nil {
		cmd.Result <- err
		close(cmd.Result)
	}
}

func sendSuccess(cmd game.GameCommand, data interface{}) {
	if cmd.Result != nil {
		cmd.Result <- data
		close(cmd.Result)
	}
}
