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

	// Look up build cost
	costs := map[string]int{
		"Mine": 500, "Trading Post": 1200, "Refinery": 1500,
		"Habitat": 800, "Shipyard": 2000,
	}
	cost, valid := costs[bd.BuildingType]
	if !valid {
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

	item := &tickable.ConstructionItem{
		ID:             fmt.Sprintf("api_%d_%d", bd.PlanetID, gs.TickManager.GetCurrentTick()),
		Type:           "Building",
		Name:           bd.BuildingType,
		Location:       attachmentID,
		Owner:          human.Name,
		Progress:       0,
		TotalTicks:     600,
		RemainingTicks: 600,
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
			"ticks":     600,
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

func sendResult(cmd game.GameCommand, err error) {
	if cmd.Result != nil {
		cmd.Result <- err
		close(cmd.Result)
	}
}
