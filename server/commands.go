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
			if err := gs.SaveGame(playerName); err != nil {
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

func (gs *GameServer) handleTradeCommand(cmd game.GameCommand) {	td, ok := cmd.Data.(game.TradeCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid trade data"))
		return
	}
	exec := gs.State.TradeExec
	human := gs.resolvePlayer(cmd)
	if exec == nil || human == nil {
		sendResult(cmd, fmt.Errorf("game not initialized"))
		return
	}

	// Resolve optional planet
	var tradePlanet *entities.Planet
	if td.PlanetID > 0 && gs.CargoCommander != nil {
		tradePlanet = gs.CargoCommander.FindPlanetByID(td.PlanetID)
	}

	var result interface{}
	var err error
	if td.Buy {
		result, err = exec.Buy(human, gs.State.Players, td.Resource, td.Quantity, tradePlanet)
	} else {
		result, err = exec.Sell(human, gs.State.Players, td.Resource, td.Quantity, tradePlanet)
	}

	if cmd.Result != nil {
		if err != nil {
			cmd.Result <- err
		} else {
			cmd.Result <- result
			// Log trade event
			if record, ok := result.(economy.TradeRecord); ok && gs.Events != nil {
				action := "bought"
				if record.Action == "sell" {
					action = "sold"
				}
				_, gt, _, _ := gs.GetTickInfo()
				gs.Events.Addf(gs.TickManager.GetCurrentTick(), gt, game.EventTrade, record.Player,
					"%s %s %d %s @ %.0f = %dcr", record.Player, action, record.Quantity, record.Resource, record.UnitPrice, record.Total)
			}
		}
		close(cmd.Result)
	}
}

func (gs *GameServer) handleCargoCommand(cmd game.GameCommand) {
	cd, ok := cmd.Data.(game.CargoCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid cargo data"))
		return
	}
	if gs.CargoCommander == nil {
		sendResult(cmd, fmt.Errorf("cargo system not initialized"))
		return
	}

	ship := game.FindShipByID(gs.State.Players, cd.ShipID)
	if ship == nil {
		sendResult(cmd, fmt.Errorf("ship not found"))
		return
	}
	planet := gs.CargoCommander.FindPlanetByID(cd.PlanetID)
	if planet == nil {
		sendResult(cmd, fmt.Errorf("planet not found"))
		return
	}

	var qty int
	var err error
	if cd.Load {
		qty, err = gs.CargoCommander.LoadCargo(ship, planet, cd.Resource, cd.Quantity)
	} else {
		qty, err = gs.CargoCommander.UnloadCargo(ship, planet, cd.Resource, cd.Quantity)
	}

	if err != nil {
		sendResult(cmd, err)
	} else {
		sendSuccess(cmd, qty)
	}
}

func (gs *GameServer) handleBuildCommand(cmd game.GameCommand) {
	bd, ok := cmd.Data.(game.BuildCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid build data"))
		return
	}

	// Find the planet and its owner
	planet := gs.CargoCommander.FindPlanetByID(bd.PlanetID)
	if planet == nil {
		sendResult(cmd, fmt.Errorf("planet not found"))
		return
	}
	if planet.Owner == "" {
		sendResult(cmd, fmt.Errorf("planet is unclaimed"))
		return
	}

	// Find the player who owns this planet
	var human *entities.Player
	for _, p := range gs.State.Players {
		if p != nil && p.Name == planet.Owner {
			human = p
			break
		}
	}
	if human == nil {
		sendResult(cmd, fmt.Errorf("planet owner not found"))
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

	if constructionSystem := tickable.GetSystemByName("Construction"); constructionSystem != nil {
		if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
			cs.AddToQueue(attachmentID, item)
		}
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

	if constructionSystem := tickable.GetSystemByName("Construction"); constructionSystem != nil {
		if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
			cs.AddToQueue(location, item)
		}
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

	ship := game.FindShipByID(gs.State.Players, md.ShipID)
	if ship == nil {
		sendResult(cmd, fmt.Errorf("ship not found"))
		return
	}
	if ship.Owner != gs.resolvePlayer(cmd).Name {
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

func (gs *GameServer) handleRefuelCommand(cmd game.GameCommand) {
	rd, ok := cmd.Data.(game.ShipRefuelCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid refuel data"))
		return
	}

	ship := game.FindShipByID(gs.State.Players, rd.ShipID)
	if ship == nil {
		sendResult(cmd, fmt.Errorf("ship not found"))
		return
	}
	if ship.Owner != gs.resolvePlayer(cmd).Name {
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

	available := planet.GetStoredAmount("Fuel")
	if available <= 0 {
		sendResult(cmd, fmt.Errorf("no Fuel on planet"))
		return
	}
	if amount > available {
		amount = available
	}

	planet.RemoveStoredResource("Fuel", amount)
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

	// Colonize: transfer colonists, claim planet, set up base
	planet.Owner = human.Name
	planet.Population = int64(ship.Colonists)
	planet.SetBaseOwner(human.Name)
	human.AddOwnedPlanet(planet)

	// Mark all resources on the planet as owned
	for _, resEntity := range planet.Resources {
		if res, ok := resEntity.(*entities.Resource); ok {
			res.Owner = human.Name
		}
	}

	// Seed initial resources and add Trading Post
	systemID := gs.CargoCommander.GetSystemForPlanet(planet)
	game.PrepareHomeworld(human, false)

	// Consume the colony ship (colonists are now on the planet)
	ship.Colonists = 0
	ship.Status = entities.ShipStatusOrbiting

	// Rebalance workforce
	planet.RebalanceWorkforce()

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
		newPlayer.HomePlanet.AddStoredResource("Fuel", 200)
		newPlayer.HomePlanet.AddStoredResource("Oil", 150)
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

	constructionSystem := tickable.GetSystemByName("Construction")
	if constructionSystem == nil {
		sendResult(cmd, fmt.Errorf("construction system not found"))
		return
	}
	cs, ok := constructionSystem.(*tickable.ConstructionSystem)
	if !ok {
		sendResult(cmd, fmt.Errorf("construction system type error"))
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

func (gs *GameServer) handleFleetMoveCommand(cmd game.GameCommand) {
	fd, ok := cmd.Data.(game.FleetMoveCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid fleet move data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	fleet, owner := game.FindFleetByID(gs.State.Players, fd.FleetID)
	if fleet == nil || owner != human {
		sendResult(cmd, fmt.Errorf("fleet not found or not owned"))
		return
	}
	success, fail := gs.FleetCmdExecutor.MoveFleetToSystem(fleet, fd.TargetSystemID)
	if success == 0 {
		sendResult(cmd, fmt.Errorf("no ships could move (no route or insufficient fuel)"))
		return
	}
	sendSuccess(cmd, map[string]interface{}{
		"fleet_id": fd.FleetID,
		"target":   fd.TargetSystemID,
		"moved":    success,
		"failed":   fail,
	})
}

func (gs *GameServer) handleFleetCreateCommand(cmd game.GameCommand) {
	fd, ok := cmd.Data.(game.FleetCreateCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid fleet create data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	ship := game.FindShipByID(gs.State.Players, fd.ShipID)
	if ship == nil || ship.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("ship not found or not owned"))
		return
	}
	fleet, err := gs.FleetMgmtSystem.CreateFleetFromShip(ship, human)
	if err != nil {
		sendResult(cmd, err)
		return
	}
	sendSuccess(cmd, map[string]interface{}{
		"fleet_id": fleet.ID,
		"ship_id":  fd.ShipID,
		"size":     fleet.Size(),
	})
}

func (gs *GameServer) handleFleetDisbandCommand(cmd game.GameCommand) {
	fd, ok := cmd.Data.(game.FleetDisbandCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid fleet disband data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	fleet, owner := game.FindFleetByID(gs.State.Players, fd.FleetID)
	if fleet == nil || owner != human {
		sendResult(cmd, fmt.Errorf("fleet not found or not owned"))
		return
	}
	shipCount := len(fleet.Ships)
	err := gs.FleetMgmtSystem.DisbandFleet(fleet, human)
	if err != nil {
		sendResult(cmd, err)
		return
	}
	sendSuccess(cmd, map[string]interface{}{
		"fleet_id":       fd.FleetID,
		"ships_released": shipCount,
	})
}

func (gs *GameServer) handleFleetAddShipCommand(cmd game.GameCommand) {
	fd, ok := cmd.Data.(game.FleetAddShipCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid fleet add ship data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	ship := game.FindShipByID(gs.State.Players, fd.ShipID)
	if ship == nil || ship.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("ship not found or not owned"))
		return
	}
	fleet, owner := game.FindFleetByID(gs.State.Players, fd.FleetID)
	if fleet == nil || owner != human {
		sendResult(cmd, fmt.Errorf("fleet not found or not owned"))
		return
	}
	err := gs.FleetMgmtSystem.AddShipToFleet(ship, fleet, human)
	if err != nil {
		sendResult(cmd, err)
		return
	}
	sendSuccess(cmd, map[string]interface{}{
		"fleet_id": fd.FleetID,
		"ship_id":  fd.ShipID,
		"size":     fleet.Size(),
	})
}

func (gs *GameServer) handleFleetRemoveShipCommand(cmd game.GameCommand) {
	fd, ok := cmd.Data.(game.FleetRemoveShipCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid fleet remove ship data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	ship := game.FindShipByID(gs.State.Players, fd.ShipID)
	if ship == nil || ship.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("ship not found or not owned"))
		return
	}
	fleet, owner := game.FindFleetByID(gs.State.Players, fd.FleetID)
	if fleet == nil || owner != human {
		sendResult(cmd, fmt.Errorf("fleet not found or not owned"))
		return
	}
	err := gs.FleetMgmtSystem.RemoveShipFromFleet(ship, fleet, human)
	if err != nil {
		sendResult(cmd, err)
		return
	}
	sendSuccess(cmd, map[string]interface{}{
		"fleet_id": fd.FleetID,
		"ship_id":  fd.ShipID,
	})
}

func (gs *GameServer) handleStandingOrderCommand(cmd game.GameCommand) {
	data, ok := cmd.Data.(game.StandingOrderCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid standing order data"))
		return
	}

	order := &game.StandingOrder{
		Player:    "", // resolved below from planet ownership
		PlanetID:  data.PlanetID,
		Resource:  data.Resource,
		Action:    data.Action,
		Quantity:  data.Quantity,
		Threshold: data.Threshold,
		MaxPrice:  data.MaxPrice,
		MinPrice:  data.MinPrice,
	}

	// If no player name from auth context, try to find from planet ownership
	if order.Player == "" {
		for _, p := range gs.State.Players {
			for _, planet := range p.OwnedPlanets {
				if planet != nil && planet.GetID() == data.PlanetID {
					order.Player = p.Name
					break
				}
			}
		}
	}

	id := gs.State.AddStandingOrder(order)
	fmt.Printf("[Server] Standing order #%d: %s %s %d %s on planet %d (threshold %d)\n",
		id, order.Player, order.Action, order.Quantity, order.Resource, order.PlanetID, order.Threshold)

	sendSuccess(cmd, map[string]interface{}{
		"order_id": id,
		"action":   order.Action,
		"resource": order.Resource,
		"quantity": order.Quantity,
	})
}

func (gs *GameServer) handleCancelOrderCommand(cmd game.GameCommand) {
	data, ok := cmd.Data.(game.CancelOrderCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid cancel order data"))
		return
	}
	if gs.State.RemoveStandingOrder(data.OrderID) {
		sendSuccess(cmd, map[string]interface{}{"cancelled": data.OrderID})
	} else {
		sendResult(cmd, fmt.Errorf("order #%d not found", data.OrderID))
	}
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
