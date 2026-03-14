package server

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/tickable"
)

// executeCommand processes a single game command.
func (gs *GameServer) executeCommand(cmd game.GameCommand) {
	switch cmd.Type {
	case "save":
		if playerName, ok := cmd.Data.(string); ok {
			if err := gs.SaveGame(playerName); err != nil {
				fmt.Printf("[Server] Save failed: %v\n", err)
			}
		}

	case "set_speed":
		if speed, ok := cmd.Data.(systems.TickSpeed); ok {
			gs.TickManager.SetSpeed(speed)
		}

	case "toggle_pause":
		gs.TickManager.TogglePause()

	case "trade":
		gs.handleTradeCommand(cmd)

	case "cargo_load", "cargo_unload":
		gs.handleCargoCommand(cmd)

	case "build":
		gs.handleBuildCommand(cmd)

	case "build_ship":
		gs.handleBuildShipCommand(cmd)

	case "move_ship":
		gs.handleMoveShipCommand(cmd)

	case "upgrade":
		gs.handleUpgradeCommand(cmd)

	case "refuel":
		gs.handleRefuelCommand(cmd)

	case "colonize":
		gs.handleColonizeCommand(cmd)
	}
}

func (gs *GameServer) handleTradeCommand(cmd game.GameCommand) {
	td, ok := cmd.Data.(game.TradeCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid trade data"))
		return
	}
	exec := gs.State.TradeExec
	human := gs.State.HumanPlayer
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

	if cmd.Result != nil {
		if err != nil {
			cmd.Result <- err
		} else {
			cmd.Result <- qty
		}
		close(cmd.Result)
	}
}

func (gs *GameServer) handleBuildCommand(cmd game.GameCommand) {
	bd, ok := cmd.Data.(game.BuildCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid build data"))
		return
	}

	human := gs.State.HumanPlayer
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}

	// Find the planet
	planet := gs.CargoCommander.FindPlanetByID(bd.PlanetID)
	if planet == nil {
		sendResult(cmd, fmt.Errorf("planet not found"))
		return
	}
	if planet.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("planet not owned by player"))
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
	if bd.BuildingType == "Mine" {
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

	if cmd.Result != nil {
		cmd.Result <- map[string]interface{}{
			"building":  bd.BuildingType,
			"planet_id": bd.PlanetID,
			"cost":      cost,
			"ticks":     buildTicks,
		}
		close(cmd.Result)
	}
}

func (gs *GameServer) handleBuildShipCommand(cmd game.GameCommand) {
	sd, ok := cmd.Data.(game.ShipBuildCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid ship build data"))
		return
	}

	human := gs.State.HumanPlayer
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

	// Check for shipyard
	hasShipyard := false
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok {
			if b.BuildingType == "Shipyard" && b.IsOperational {
				hasShipyard = true
				break
			}
		}
	}
	if !hasShipyard {
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

	if cmd.Result != nil {
		cmd.Result <- map[string]interface{}{
			"ship_type": sd.ShipType,
			"cost":      cost,
			"ticks":     buildTime,
			"resources": requirements,
		}
		close(cmd.Result)
	}
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
	if ship.Owner != gs.State.HumanPlayer.Name {
		sendResult(cmd, fmt.Errorf("not your ship"))
		return
	}

	helper := tickable.NewShipMovementHelper(gs.GetSystemsMap(), gs.State.Hyperlanes)
	if helper.StartJourney(ship, md.TargetSystemID) {
		if cmd.Result != nil {
			cmd.Result <- map[string]interface{}{
				"ship_id": md.ShipID,
				"target":  md.TargetSystemID,
				"status":  "moving",
			}
			close(cmd.Result)
		}
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
	human := gs.State.HumanPlayer
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

	if cmd.Result != nil {
		cmd.Result <- map[string]interface{}{
			"building":  building.BuildingType,
			"new_level": building.Level,
			"cost":      cost,
		}
		close(cmd.Result)
	}
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
	if ship.Owner != gs.State.HumanPlayer.Name {
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

	if cmd.Result != nil {
		cmd.Result <- map[string]interface{}{
			"ship_id":  rd.ShipID,
			"refueled": amount,
			"fuel_now": ship.CurrentFuel,
			"fuel_max": ship.MaxFuel,
		}
		close(cmd.Result)
	}
}

func (gs *GameServer) handleColonizeCommand(cmd game.GameCommand) {
	cd, ok := cmd.Data.(game.ColonizeCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid colonize data"))
		return
	}

	human := gs.State.HumanPlayer
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

	if cmd.Result != nil {
		cmd.Result <- map[string]interface{}{
			"planet":    planet.Name,
			"planet_id": planet.GetID(),
			"system_id": systemID,
			"colonists": planet.Population,
		}
		close(cmd.Result)
	}
}

func sendResult(cmd game.GameCommand, err error) {
	if cmd.Result != nil {
		cmd.Result <- err
		close(cmd.Result)
	}
}
