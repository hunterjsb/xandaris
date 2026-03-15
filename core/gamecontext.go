package core

import (
	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/views"
)

// GameContext interface implementation — delegates to embedded server

func (a *App) GetSystems() []*entities.System       { return a.Server.GetSystems() }
func (a *App) GetHyperlanes() []entities.Hyperlane   { return a.Server.GetHyperlanes() }
func (a *App) GetPlayers() []*entities.Player        { return a.Server.GetPlayers() }
func (a *App) GetHumanPlayer() *entities.Player      { return a.Server.GetHumanPlayer() }
func (a *App) GetSeed() int64                        { return a.Server.GetSeed() }
func (a *App) GetMarket() *economy.Market            { return a.Server.GetMarket() }
func (a *App) GetTradeExecutor() *economy.TradeExecutor { return a.Server.GetTradeExecutor() }

func (a *App) GetCargoCommander() *game.CargoCommandExecutor {
	return a.Server.GetCargoCommander()
}
func (a *App) GetEventLog() *game.EventLog {
	return a.Server.GetEventLog()
}
func (a *App) GetRegistry() *game.PlayerRegistry {
	return a.Server.GetRegistry()
}

func (a *App) GetSaveLoad() views.SaveLoadInterface { return a }

// GetMarketEngine implements tickable.MarketProvider (for backward compat)
func (a *App) GetMarketEngine() *economy.Market { return a.Server.GetMarketEngine() }

// GetTickInfo implements api.GameStateProvider
func (a *App) GetTickInfo() (tick int64, gameTime string, speed string, paused bool) {
	return a.Server.GetTickInfo()
}

// GetCommandChannel implements api.GameStateProvider
func (a *App) GetCommandChannel() chan game.GameCommand {
	return a.Server.GetCommandChannel()
}

// GetSystemsMap returns systems indexed by ID (for UI components)
func (a *App) GetSystemsMap() map[int]*entities.System {
	return a.Server.GetSystemsMap()
}

// Fleet commands delegate to server
func (a *App) MoveFleetToSystem(fleet *entities.Fleet, targetSystemID int) (int, int) {
	return a.Server.MoveFleetToSystem(fleet, targetSystemID)
}
func (a *App) MoveFleetToPlanet(fleet *entities.Fleet, targetPlanet *entities.Planet) (int, int) {
	return a.Server.MoveFleetToPlanet(fleet, targetPlanet)
}
func (a *App) MoveFleetToStar(fleet *entities.Fleet) (int, int) {
	return a.Server.MoveFleetToStar(fleet)
}
func (a *App) GetConnectedSystems(fromSystemID int) []int {
	return a.Server.GetConnectedSystems(fromSystemID)
}
func (a *App) GetSystemByID(systemID int) *entities.System {
	return a.Server.GetSystemByID(systemID)
}

// Cargo operations delegate to server (for tickable.CargoOperator)
func (a *App) LoadCargo(ship *entities.Ship, planet *entities.Planet, resource string, qty int) (int, error) {
	return a.Server.LoadCargo(ship, planet, resource, qty)
}
func (a *App) UnloadCargo(ship *entities.Ship, planet *entities.Planet, resource string, qty int) (int, error) {
	return a.Server.UnloadCargo(ship, planet, resource, qty)
}

// LogEvent delegates to server (for tickable.EventLogger)
func (a *App) LogEvent(eventType string, player string, message string) {
	a.Server.LogEvent(eventType, player, message)
}

// StartShipJourney delegates to server (for tickable.ShipJourney)
func (a *App) StartShipJourney(ship *entities.Ship, targetSystemID int) bool {
	return a.Server.StartShipJourney(ship, targetSystemID)
}
