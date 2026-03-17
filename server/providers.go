package server

// providers.go implements the interface contracts that GameServer satisfies:
//   - tickable.GameProvider (game logic for tick systems)
//   - api.GameStateProvider (read-only state for REST/WebSocket)
//   - economy.ShipDispatcher (cargo delivery ship dispatch)
//
// Keeping these separate from server.go means adding a new provider method
// doesn't require touching the core server lifecycle code.

import (
	"fmt"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/tickable"
)

// --- tickable.GameProvider ---

// Compile-time check: GameServer must implement GameProvider.
var _ tickable.GameProvider = (*GameServer)(nil)

func (gs *GameServer) AIBuildOnPlanet(planet *entities.Planet, buildingType string, owner string, systemID int) {
	game.AddBuildingToPlanet(planet, buildingType, owner, systemID)
}

func (gs *GameServer) ColonizePlanet(planet *entities.Planet, ship *entities.Ship, player *entities.Player, systemID int) {
	game.ColonizePlanet(planet, ship, player, systemID)
}

func (gs *GameServer) GetMarketEngine() *economy.Market { return gs.State.Market }

func (gs *GameServer) LoadCargo(ship *entities.Ship, planet *entities.Planet, resource string, qty int) (int, error) {
	if gs.CargoCommander == nil {
		return 0, fmt.Errorf("cargo system not initialized")
	}
	return gs.CargoCommander.LoadCargo(ship, planet, resource, qty)
}

func (gs *GameServer) UnloadCargo(ship *entities.Ship, planet *entities.Planet, resource string, qty int) (int, error) {
	if gs.CargoCommander == nil {
		return 0, fmt.Errorf("cargo system not initialized")
	}
	return gs.CargoCommander.UnloadCargo(ship, planet, resource, qty)
}

func (gs *GameServer) LogEvent(eventType string, player string, message string) {
	if gs.Events != nil {
		gs.Events.Add(gs.TickManager.GetCurrentTick(), gs.TickManager.GetGameTimeFormatted(),
			game.EventType(eventType), player, message)
	}
}

func (gs *GameServer) StartShipJourney(ship *entities.Ship, targetSystemID int) bool {
	helper := tickable.NewShipMovementHelper(gs.GetSystemsMap(), gs.State.Hyperlanes)
	return helper.StartJourney(ship, targetSystemID)
}

func (gs *GameServer) GetSystemsMap() map[int]*entities.System {
	return gs.State.GetSystemsMap()
}

func (gs *GameServer) GetConnectedSystems(fromSystemID int) []int {
	return gs.FleetCmdExecutor.GetConnectedSystems(fromSystemID)
}

func (gs *GameServer) GetStandingOrderInfos() []tickable.StandingOrderInfo {
	result := make([]tickable.StandingOrderInfo, 0, len(gs.State.StandingOrders))
	for _, o := range gs.State.StandingOrders {
		result = append(result, tickable.StandingOrderInfo{
			ID: o.ID, Player: o.Player, PlanetID: o.PlanetID,
			Resource: o.Resource, Action: o.Action, Quantity: o.Quantity,
			Threshold: o.Threshold, Active: o.Active,
		})
	}
	return result
}

func (gs *GameServer) ExecuteStandingOrderTrade(order tickable.StandingOrderInfo, player *entities.Player) error {
	if gs.State.TradeExec == nil || gs.State.Market == nil {
		return fmt.Errorf("market not available")
	}

	// Price check from original order (if set)
	for _, o := range gs.State.StandingOrders {
		if o.ID == order.ID {
			if o.Action == "buy" && o.MaxPrice > 0 {
				price := gs.State.Market.GetBuyPrice(order.Resource)
				if int(price) > o.MaxPrice {
					return fmt.Errorf("price too high")
				}
			}
			if o.Action == "sell" && o.MinPrice > 0 {
				price := gs.State.Market.GetSellPrice(order.Resource)
				if int(price) < o.MinPrice {
					return fmt.Errorf("price too low")
				}
			}
			break
		}
	}

	var planet *entities.Planet
	for _, p := range player.OwnedPlanets {
		if p != nil && p.GetID() == order.PlanetID {
			planet = p
			break
		}
	}
	if planet == nil {
		return fmt.Errorf("planet not found")
	}

	var err error
	if order.Action == "buy" {
		_, err = gs.State.TradeExec.Buy(player, gs.State.Players, order.Resource, order.Quantity, planet)
	} else {
		_, err = gs.State.TradeExec.Sell(player, gs.State.Players, order.Resource, order.Quantity, planet)
	}
	return err
}

func (gs *GameServer) GetDeliveryManager() *economy.DeliveryManager {
	return gs.DeliveryMgr
}
func (gs *GameServer) GetShippingManager() *game.ShippingManager {
	return gs.ShippingMgr
}

func (gs *GameServer) DockShip(ship *entities.Ship, planet *entities.Planet) error {
	if gs.CargoCommander == nil {
		return fmt.Errorf("cargo system not initialized")
	}
	return gs.CargoCommander.DockShip(ship, planet)
}

func (gs *GameServer) UndockShip(ship *entities.Ship) error {
	if gs.CargoCommander == nil {
		return fmt.Errorf("cargo system not initialized")
	}
	return gs.CargoCommander.UndockShip(ship)
}

func (gs *GameServer) SellAtDock(ship *entities.Ship, resource string, qty int) (int, int, error) {
	if gs.CargoCommander == nil {
		return 0, 0, fmt.Errorf("cargo system not initialized")
	}
	buyPrice := 0.0
	if gs.State.Market != nil {
		buyPrice = gs.State.Market.GetBuyPrice(resource)
	}
	sold, credits, err := gs.CargoCommander.SellAtDock(ship, resource, qty, buyPrice, nil)
	if err != nil {
		return 0, 0, err
	}

	// Docking fee goes to planet owner (incentivizes TP upgrades)
	if ship.DockedAtPlanet != 0 {
		planet := gs.CargoCommander.FindPlanetByID(ship.DockedAtPlanet)
		if planet != nil && planet.Owner != "" && planet.Owner != ship.Owner {
			feeRate := planet.GetDockingFeeRate()
			dockingFee := int(float64(credits) * feeRate)
			for _, p := range gs.State.Players {
				if p != nil && p.Name == planet.Owner {
					p.Credits += dockingFee
					break
				}
			}
			credits -= dockingFee
		}
	}

	// Credit the ship owner (after fee)
	for _, p := range gs.State.Players {
		if p != nil && p.Name == ship.Owner {
			p.Credits += credits
			break
		}
	}
	if gs.State.Market != nil {
		gs.State.Market.AddTradeVolume(resource, sold, false)
	}
	return sold, credits, nil
}

func (gs *GameServer) GetCreditLedger() *economy.CreditLedger {
	return gs.CreditLedger
}

func (gs *GameServer) GetOrderBook() *economy.OrderBook {
	return gs.OrderBook
}

func (gs *GameServer) GetContractManager() *economy.ContractManager {
	return gs.ContractMgr
}

func (gs *GameServer) GetDiplomacyManager() *economy.DiplomacyManager {
	return gs.DiplomacyMgr
}

func (gs *GameServer) GetEspionageManager() *economy.EspionageManager {
	return gs.EspionageMgr
}

func (gs *GameServer) GetBountyBoard() *economy.BountyBoard {
	return gs.BountyBoard
}

func (gs *GameServer) GetBlackMarket() *economy.BlackMarket {
	return gs.BlackMarket
}

func (gs *GameServer) GetAuctionHouse() *economy.AuctionHouse {
	return gs.AuctionHouse
}

func (gs *GameServer) GetShippingRoutes() []tickable.ShippingRouteInfo {
	if gs.ShippingMgr == nil {
		return nil
	}
	routes := gs.ShippingMgr.GetRoutes("")
	result := make([]tickable.ShippingRouteInfo, 0, len(routes))
	for _, r := range routes {
		result = append(result, tickable.ShippingRouteInfo{
			ID:            r.ID,
			Owner:         r.Owner,
			SourcePlanet:  r.SourcePlanet,
			DestPlanet:    r.DestPlanet,
			Resource:      r.Resource,
			Quantity:      r.Quantity,
			ShipID:        r.ShipID,
			Active:        r.Active,
			TripsComplete: r.TripsComplete,
		})
	}
	return result
}

func (gs *GameServer) CompleteShippingTrip(routeID int) {
	if gs.ShippingMgr != nil {
		gs.ShippingMgr.CompleteTrip(routeID)
	}
}

func (gs *GameServer) AssignShipToRoute(routeID, shipID int) {
	if gs.ShippingMgr != nil {
		gs.ShippingMgr.AssignShip(routeID, shipID)
	}
}

func (gs *GameServer) CancelShippingRoute(routeID int) {
	if gs.ShippingMgr != nil {
		gs.ShippingMgr.CancelRoute(routeID)
	}
}

// --- api.GameStateProvider ---

func (gs *GameServer) GetSystems() []*entities.System     { return gs.State.Systems }
func (gs *GameServer) GetHyperlanes() []entities.Hyperlane { return gs.State.Hyperlanes }
func (gs *GameServer) GetPlayers() []*entities.Player      { return gs.State.Players }
func (gs *GameServer) GetHumanPlayer() *entities.Player    { return gs.State.HumanPlayer }
func (gs *GameServer) GetSeed() int64                      { return gs.State.Seed }
func (gs *GameServer) GetMarket() *economy.Market          { return gs.State.Market }
func (gs *GameServer) GetTradeExecutor() *economy.TradeExecutor {
	return gs.State.TradeExec
}
func (gs *GameServer) GetCargoCommander() *game.CargoCommandExecutor {
	return gs.CargoCommander
}
func (gs *GameServer) GetFleetManagementSystem() *game.FleetManagementSystem {
	return gs.FleetMgmtSystem
}
func (gs *GameServer) GetEventLog() *game.EventLog {
	return gs.Events
}
func (gs *GameServer) GetRegistry() *game.PlayerRegistry {
	return gs.Registry
}
func (gs *GameServer) GetChatLog() *game.ChatLog {
	return gs.Chat
}
func (gs *GameServer) GetCommandChannel() chan game.GameCommand {
	return gs.State.Commands
}
func (gs *GameServer) GetStandingOrders(player string) []*game.StandingOrder {
	return gs.State.GetStandingOrders(player)
}
func (gs *GameServer) GetTickInfo() (tick int64, gameTime string, speed string, paused bool) {
	if gs.TickManager == nil {
		return 0, "0:00", "1x", false
	}
	return gs.TickManager.GetCurrentTick(),
		gs.TickManager.GetGameTimeFormatted(),
		gs.TickManager.GetSpeedString(),
		gs.TickManager.IsPaused()
}

// --- Fleet delegation ---

func (gs *GameServer) MoveFleetToSystem(fleet *entities.Fleet, targetSystemID int) (int, int) {
	return gs.FleetCmdExecutor.MoveFleetToSystem(fleet, targetSystemID)
}
func (gs *GameServer) MoveFleetToPlanet(fleet *entities.Fleet, targetPlanet *entities.Planet) (int, int) {
	return gs.FleetCmdExecutor.MoveFleetToPlanet(fleet, targetPlanet)
}
func (gs *GameServer) MoveFleetToStar(fleet *entities.Fleet) (int, int) {
	return gs.FleetCmdExecutor.MoveFleetToStar(fleet)
}
func (gs *GameServer) GetSystemByID(systemID int) *entities.System {
	return gs.FleetCmdExecutor.GetSystemByID(systemID)
}

// --- economy.ShipDispatcher ---

func (gs *GameServer) FindAvailableCargoShip(owner string, systemID int) *entities.Ship {
	for _, p := range gs.State.Players {
		if p == nil || p.Name != owner {
			continue
		}
		// Prefer ships in the requested system
		for _, ship := range p.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeCargo {
				continue
			}
			if ship.Status == entities.ShipStatusMoving || ship.DeliveryID != 0 {
				continue
			}
			if ship.CurrentSystem == systemID {
				return ship
			}
		}
		// Fallback: any idle cargo ship
		for _, ship := range p.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeCargo {
				continue
			}
			if ship.Status == entities.ShipStatusMoving || ship.DeliveryID != 0 {
				continue
			}
			return ship
		}
	}
	return nil
}

func (gs *GameServer) DispatchShipToSystem(ship *entities.Ship, targetSystemID int) bool {
	return gs.StartShipJourney(ship, targetSystemID)
}

func (gs *GameServer) AreSystemsConnected(fromID, toID int) bool {
	helper := tickable.NewShipMovementHelper(gs.GetSystemsMap(), gs.State.Hyperlanes)
	return helper.AreSystemsConnected(fromID, toID)
}

func (gs *GameServer) FindPath(fromID, toID int) []int {
	helper := tickable.NewShipMovementHelper(gs.GetSystemsMap(), gs.State.Hyperlanes)
	return helper.FindPath(fromID, toID)
}

// --- Admin operations ---

// RemovePlayer fully removes a player: releases planets, removes ships from systems, compacts the players slice.
func (gs *GameServer) RemovePlayer(name string) bool {
	idx := -1
	for i, pl := range gs.State.Players {
		if pl != nil && pl.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	pl := gs.State.Players[idx]

	// Release all owned planets
	for _, planet := range pl.OwnedPlanets {
		if planet != nil {
			planet.Owner = ""
			planet.Population = 0
		}
	}

	// Remove ships from system entities
	for _, ship := range pl.OwnedShips {
		if ship == nil {
			continue
		}
		for _, sys := range gs.State.Systems {
			sys.RemoveEntity(ship.GetID())
		}
	}

	// Remove standing orders for this player
	remaining := make([]*game.StandingOrder, 0)
	for _, o := range gs.State.StandingOrders {
		if o != nil && o.Player != name {
			remaining = append(remaining, o)
		}
	}
	gs.State.StandingOrders = remaining

	// Compact nil out of players slice
	gs.State.Players = append(gs.State.Players[:idx], gs.State.Players[idx+1:]...)

	fmt.Printf("[Admin] Removed player %q: planets released, ships deleted, orders purged\n", name)
	return true
}

// --- serverSystemContext implements tickable.SystemContext ---

type serverSystemContext struct {
	server *GameServer
}

func (ssc *serverSystemContext) GetGame() tickable.GameProvider { return ssc.server }
func (ssc *serverSystemContext) GetPlayers() []*entities.Player {
	return ssc.server.State.Players
}
func (ssc *serverSystemContext) GetTick() int64 {
	return ssc.server.TickManager.GetCurrentTick()
}
