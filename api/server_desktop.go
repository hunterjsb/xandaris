//go:build !js

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/systems"
)

var (
	serverStarted  atomic.Bool
	providerMu     sync.RWMutex
	activeProvider GameStateProvider
)

func getProvider() GameStateProvider {
	providerMu.RLock()
	defer providerMu.RUnlock()
	return activeProvider
}

// StartServer launches the REST API on :8080 in a background goroutine.
// Subsequent calls update the provider without starting a second server.
func StartServer(provider GameStateProvider) {
	providerMu.Lock()
	activeProvider = provider
	providerMu.Unlock()

	if serverStarted.Swap(true) {
		fmt.Println("[API] Provider updated (server already running)")
		return
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/market", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetMarket(getProvider())})
	})

	mux.HandleFunc("/api/market/trade", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req TradeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.Resource == "" || req.Quantity <= 0 {
			writeErr(w, http.StatusBadRequest, "resource and positive quantity required")
			return
		}
		buy := strings.EqualFold(req.Action, "buy")

		// Execute trade on main goroutine via command channel
		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "trade",
			Data:   game.TradeCommandData{Resource: req.Resource, Quantity: req.Quantity, Buy: buy, PlanetID: req.PlanetID},
			Result: resultCh,
		}

		// Wait for result with timeout
		select {
		case result := <-resultCh:
			switch v := result.(type) {
			case error:
				writeErr(w, http.StatusBadRequest, v.Error())
			case economy.TradeRecord:
				writeJSON(w, APIResponse{OK: true, Data: TradeResult{
					Resource: v.Resource,
					Quantity: v.Quantity,
					Action:   v.Action,
					Total:    v.Total,
				}})
			default:
				writeErr(w, http.StatusInternalServerError, "unexpected result type")
			}
		case <-time.After(5 * time.Second):
			writeErr(w, http.StatusGatewayTimeout, "trade execution timed out")
		}
	})

	// Cargo load endpoint
	mux.HandleFunc("/api/cargo/load", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req CargoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.ShipID <= 0 || req.PlanetID <= 0 || req.Resource == "" || req.Quantity <= 0 {
			writeErr(w, http.StatusBadRequest, "ship_id, planet_id, resource, and positive quantity required")
			return
		}

		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type: "cargo_load",
			Data: game.CargoCommandData{
				ShipID:   req.ShipID,
				PlanetID: req.PlanetID,
				Resource: req.Resource,
				Quantity: req.Quantity,
				Load:     true,
			},
			Result: resultCh,
		}

		select {
		case result := <-resultCh:
			switch v := result.(type) {
			case error:
				writeErr(w, http.StatusBadRequest, v.Error())
			case int:
				writeJSON(w, APIResponse{OK: true, Data: CargoResult{
					ShipID:   req.ShipID,
					PlanetID: req.PlanetID,
					Resource: req.Resource,
					Quantity: v,
					Action:   "load",
				}})
			default:
				writeErr(w, http.StatusInternalServerError, "unexpected result type")
			}
		case <-time.After(5 * time.Second):
			writeErr(w, http.StatusGatewayTimeout, "cargo operation timed out")
		}
	})

	// Cargo unload endpoint
	mux.HandleFunc("/api/cargo/unload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req CargoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.ShipID <= 0 || req.PlanetID <= 0 || req.Resource == "" || req.Quantity <= 0 {
			writeErr(w, http.StatusBadRequest, "ship_id, planet_id, resource, and positive quantity required")
			return
		}

		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type: "cargo_unload",
			Data: game.CargoCommandData{
				ShipID:   req.ShipID,
				PlanetID: req.PlanetID,
				Resource: req.Resource,
				Quantity: req.Quantity,
				Load:     false,
			},
			Result: resultCh,
		}

		select {
		case result := <-resultCh:
			switch v := result.(type) {
			case error:
				writeErr(w, http.StatusBadRequest, v.Error())
			case int:
				writeJSON(w, APIResponse{OK: true, Data: CargoResult{
					ShipID:   req.ShipID,
					PlanetID: req.PlanetID,
					Resource: req.Resource,
					Quantity: v,
					Action:   "unload",
				}})
			default:
				writeErr(w, http.StatusInternalServerError, "unexpected result type")
			}
		case <-time.After(5 * time.Second):
			writeErr(w, http.StatusGatewayTimeout, "cargo operation timed out")
		}
	})

	mux.HandleFunc("/api/market/history", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				limit = n
			}
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetTradeHistory(getProvider(), limit)})
	})

	mux.HandleFunc("/api/galaxy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetGalaxy(getProvider())})
	})

	mux.HandleFunc("/api/systems/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/systems/")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid system ID")
			return
		}
		data, found := handleGetSystem(getProvider(), id)
		if !found {
			writeErr(w, http.StatusNotFound, "system not found")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: data})
	})

	mux.HandleFunc("/api/planets/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/planets/")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid planet ID")
			return
		}
		data, found := handleGetPlanet(getProvider(), id)
		if !found {
			writeErr(w, http.StatusNotFound, "planet not found")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: data})
	})

	mux.HandleFunc("/api/player/me", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		data := handleGetPlayerMe(getProvider())
		if data == nil {
			writeErr(w, http.StatusNotFound, "no human player")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: data})
	})

	mux.HandleFunc("/api/players", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetPlayers(getProvider())})
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetStatus(getProvider())})
	})

	mux.HandleFunc("/api/game", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetGame(getProvider())})
	})

	mux.HandleFunc("/api/build", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req BuildRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.PlanetID <= 0 || req.BuildingType == "" {
			writeErr(w, http.StatusBadRequest, "planet_id and building_type required")
			return
		}

		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type: "build",
			Data: game.BuildCommandData{
				PlanetID:     req.PlanetID,
				BuildingType: req.BuildingType,
				ResourceID:   req.ResourceID,
			},
			Result: resultCh,
		}

		select {
		case result := <-resultCh:
			switch v := result.(type) {
			case error:
				writeErr(w, http.StatusBadRequest, v.Error())
			default:
				writeJSON(w, APIResponse{OK: true, Data: v})
			}
		case <-time.After(5 * time.Second):
			writeErr(w, http.StatusGatewayTimeout, "build command timed out")
		}
	})

	mux.HandleFunc("/api/economy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetEconomy(getProvider())})
	})

	mux.HandleFunc("/api/ships", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetShips(getProvider())})
	})

	mux.HandleFunc("/api/fleets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetFleets(getProvider())})
	})

	mux.HandleFunc("/api/prices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetSystemPrices(getProvider())})
	})

	mux.HandleFunc("/api/planets/rates/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/planets/rates/")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid planet ID")
			return
		}
		data, found := handleGetPlanetRates(getProvider(), id)
		if !found {
			writeErr(w, http.StatusNotFound, "planet not found")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: data})
	})

	mux.HandleFunc("/api/planets/storage/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/planets/storage/")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid planet ID")
			return
		}
		data, found := handleGetPlanetStorage(getProvider(), id)
		if !found {
			writeErr(w, http.StatusNotFound, "planet not found")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: data})
	})

	mux.HandleFunc("/api/ships/build", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req ShipBuildRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.PlanetID <= 0 || req.ShipType == "" {
			writeErr(w, http.StatusBadRequest, "planet_id and ship_type required")
			return
		}
		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "build_ship",
			Data:   game.ShipBuildCommandData{PlanetID: req.PlanetID, ShipType: req.ShipType},
			Result: resultCh,
		}
		select {
		case result := <-resultCh:
			switch v := result.(type) {
			case error:
				writeErr(w, http.StatusBadRequest, v.Error())
			default:
				writeJSON(w, APIResponse{OK: true, Data: v})
			}
		case <-time.After(5 * time.Second):
			writeErr(w, http.StatusGatewayTimeout, "timed out")
		}
	})

	mux.HandleFunc("/api/ships/move", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req ShipMoveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.ShipID <= 0 || req.TargetSystemID < 0 {
			writeErr(w, http.StatusBadRequest, "ship_id and target_system_id required")
			return
		}
		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "move_ship",
			Data:   game.ShipMoveCommandData{ShipID: req.ShipID, TargetSystemID: req.TargetSystemID},
			Result: resultCh,
		}
		select {
		case result := <-resultCh:
			switch v := result.(type) {
			case error:
				writeErr(w, http.StatusBadRequest, v.Error())
			default:
				writeJSON(w, APIResponse{OK: true, Data: v})
			}
		case <-time.After(5 * time.Second):
			writeErr(w, http.StatusGatewayTimeout, "timed out")
		}
	})

	mux.HandleFunc("/api/ships/refuel", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req ShipRefuelRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "refuel",
			Data:   game.ShipRefuelCommandData{ShipID: req.ShipID, PlanetID: req.PlanetID, Amount: req.Amount},
			Result: resultCh,
		}
		select {
		case result := <-resultCh:
			switch v := result.(type) {
			case error:
				writeErr(w, http.StatusBadRequest, v.Error())
			default:
				writeJSON(w, APIResponse{OK: true, Data: v})
			}
		case <-time.After(5 * time.Second):
			writeErr(w, http.StatusGatewayTimeout, "timed out")
		}
	})

	mux.HandleFunc("/api/colonize", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req ColonizeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "colonize",
			Data:   game.ColonizeCommandData{ShipID: req.ShipID, PlanetID: req.PlanetID},
			Result: resultCh,
		}
		select {
		case result := <-resultCh:
			switch v := result.(type) {
			case error:
				writeErr(w, http.StatusBadRequest, v.Error())
			default:
				writeJSON(w, APIResponse{OK: true, Data: v})
			}
		case <-time.After(5 * time.Second):
			writeErr(w, http.StatusGatewayTimeout, "timed out")
		}
	})

	mux.HandleFunc("/api/upgrade", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req UpgradeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "upgrade",
			Data:   game.UpgradeCommandData{PlanetID: req.PlanetID, BuildingIndex: req.BuildingIndex},
			Result: resultCh,
		}
		select {
		case result := <-resultCh:
			switch v := result.(type) {
			case error:
				writeErr(w, http.StatusBadRequest, v.Error())
			default:
				writeJSON(w, APIResponse{OK: true, Data: v})
			}
		case <-time.After(5 * time.Second):
			writeErr(w, http.StatusGatewayTimeout, "timed out")
		}
	})

	mux.HandleFunc("/api/catalog", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetCatalog()})
	})

	mux.HandleFunc("/api/construction", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetConstructionQueue(getProvider())})
	})

	// Fleet management endpoints
	mux.HandleFunc("/api/fleets/move", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req FleetMoveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.FleetID <= 0 || req.TargetSystemID < 0 {
			writeErr(w, http.StatusBadRequest, "fleet_id and target_system_id required")
			return
		}
		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "fleet_move",
			Data:   game.FleetMoveCommandData{FleetID: req.FleetID, TargetSystemID: req.TargetSystemID},
			Result: resultCh,
		}
		select {
		case result := <-resultCh:
			switch v := result.(type) {
			case error:
				writeErr(w, http.StatusBadRequest, v.Error())
			default:
				writeJSON(w, APIResponse{OK: true, Data: v})
			}
		case <-time.After(5 * time.Second):
			writeErr(w, http.StatusGatewayTimeout, "timed out")
		}
	})

	mux.HandleFunc("/api/fleets/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req FleetCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.ShipID <= 0 {
			writeErr(w, http.StatusBadRequest, "ship_id required")
			return
		}
		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "fleet_create",
			Data:   game.FleetCreateCommandData{ShipID: req.ShipID},
			Result: resultCh,
		}
		select {
		case result := <-resultCh:
			switch v := result.(type) {
			case error:
				writeErr(w, http.StatusBadRequest, v.Error())
			default:
				writeJSON(w, APIResponse{OK: true, Data: v})
			}
		case <-time.After(5 * time.Second):
			writeErr(w, http.StatusGatewayTimeout, "timed out")
		}
	})

	mux.HandleFunc("/api/fleets/disband", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req FleetDisbandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.FleetID <= 0 {
			writeErr(w, http.StatusBadRequest, "fleet_id required")
			return
		}
		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "fleet_disband",
			Data:   game.FleetDisbandCommandData{FleetID: req.FleetID},
			Result: resultCh,
		}
		select {
		case result := <-resultCh:
			switch v := result.(type) {
			case error:
				writeErr(w, http.StatusBadRequest, v.Error())
			default:
				writeJSON(w, APIResponse{OK: true, Data: v})
			}
		case <-time.After(5 * time.Second):
			writeErr(w, http.StatusGatewayTimeout, "timed out")
		}
	})

	mux.HandleFunc("/api/fleets/add-ship", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req FleetAddShipRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.ShipID <= 0 || req.FleetID <= 0 {
			writeErr(w, http.StatusBadRequest, "ship_id and fleet_id required")
			return
		}
		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "fleet_add_ship",
			Data:   game.FleetAddShipCommandData{ShipID: req.ShipID, FleetID: req.FleetID},
			Result: resultCh,
		}
		select {
		case result := <-resultCh:
			switch v := result.(type) {
			case error:
				writeErr(w, http.StatusBadRequest, v.Error())
			default:
				writeJSON(w, APIResponse{OK: true, Data: v})
			}
		case <-time.After(5 * time.Second):
			writeErr(w, http.StatusGatewayTimeout, "timed out")
		}
	})

	mux.HandleFunc("/api/fleets/remove-ship", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req FleetRemoveShipRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.ShipID <= 0 || req.FleetID <= 0 {
			writeErr(w, http.StatusBadRequest, "ship_id and fleet_id required")
			return
		}
		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "fleet_remove_ship",
			Data:   game.FleetRemoveShipCommandData{ShipID: req.ShipID, FleetID: req.FleetID},
			Result: resultCh,
		}
		select {
		case result := <-resultCh:
			switch v := result.(type) {
			case error:
				writeErr(w, http.StatusBadRequest, v.Error())
			default:
				writeJSON(w, APIResponse{OK: true, Data: v})
			}
		case <-time.After(5 * time.Second):
			writeErr(w, http.StatusGatewayTimeout, "timed out")
		}
	})

	// Hyperlane/route info
	mux.HandleFunc("/api/routes/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/routes/")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid system ID")
			return
		}
		p := getProvider()
		connected := make([]int, 0)
		for _, hl := range p.GetHyperlanes() {
			if hl.From == id {
				connected = append(connected, hl.To)
			} else if hl.To == id {
				connected = append(connected, hl.From)
			}
		}
		writeJSON(w, APIResponse{OK: true, Data: map[string]interface{}{
			"system_id": id,
			"connected": connected,
		}})
	})

	mux.HandleFunc("/api/game/speed", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req SpeedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		speed, ok := parseSpeed(req.Speed)
		if !ok {
			writeErr(w, http.StatusBadRequest, "invalid speed; use: slow, normal, fast, very_fast")
			return
		}
		ch := getProvider().GetCommandChannel()
		ch <- game.GameCommand{Type: "set_speed", Data: speed}
		writeJSON(w, APIResponse{OK: true, Data: map[string]string{"speed": req.Speed}})
	})

	mux.HandleFunc("/api/game/pause", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		ch := getProvider().GetCommandChannel()
		ch <- game.GameCommand{Type: "toggle_pause"}
		writeJSON(w, APIResponse{OK: true, Data: map[string]string{"action": "toggled"}})
	})

	mux.HandleFunc("/api/game/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		p := getProvider()
		human := p.GetHumanPlayer()
		name := "Player"
		if human != nil {
			name = human.Name
		}
		ch := p.GetCommandChannel()
		ch <- game.GameCommand{Type: "save", Data: name}
		writeJSON(w, APIResponse{OK: true, Data: map[string]string{"action": "save_queued"}})
	})

	go func() {
		fmt.Println("[API] Starting REST server on :8080")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			fmt.Printf("[API] Server error: %v\n", err)
		}
	}()
}

func parseSpeed(s string) (systems.TickSpeed, bool) {
	switch strings.ToLower(s) {
	case "slow", "1x":
		return systems.TickSpeed1x, true
	case "normal", "2x":
		return systems.TickSpeed2x, true
	case "fast", "4x":
		return systems.TickSpeed4x, true
	case "very_fast", "8x":
		return systems.TickSpeed8x, true
	}
	return 0, false
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(APIResponse{OK: false, Error: msg})
}
