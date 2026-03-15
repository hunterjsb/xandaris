//go:build !js

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
	apiKey         string // if set, POST endpoints require X-API-Key header
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

	// Load API key from environment
	apiKey = os.Getenv("XANDARIS_API_KEY")
	if apiKey != "" {
		fmt.Println("[API] API key auth enabled for POST endpoints")
	}

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
		filterResource := r.URL.Query().Get("resource")
		filterPlayer := r.URL.Query().Get("player")
		result := handleGetTradeHistory(getProvider(), limit)
		// Apply filters if provided
		if filterResource != "" || filterPlayer != "" {
			if entries, ok := result.([]TradeHistoryEntry); ok {
				filtered := make([]TradeHistoryEntry, 0)
				for _, e := range entries {
					if filterResource != "" && !strings.EqualFold(e.Resource, filterResource) {
						continue
					}
					if filterPlayer != "" && !strings.EqualFold(e.Player, filterPlayer) {
						continue
					}
					filtered = append(filtered, e)
				}
				result = filtered
			}
		}
		writeJSON(w, APIResponse{OK: true, Data: result})
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
		result := handleGetShips(getProvider())
		filterOwner := r.URL.Query().Get("owner")
		if filterOwner != "" {
			if ships, ok := result.([]ShipInfo); ok {
				filtered := make([]ShipInfo, 0)
				for _, s := range ships {
					if strings.EqualFold(s.Owner, filterOwner) {
						filtered = append(filtered, s)
					}
				}
				result = filtered
			}
		}
		writeJSON(w, APIResponse{OK: true, Data: result})
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

	mux.HandleFunc("/api/deposits", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		filterResource := r.URL.Query().Get("resource")
		filterUnmined := r.URL.Query().Get("unmined") == "true"
		filterOwner := r.URL.Query().Get("owner")
		writeJSON(w, APIResponse{OK: true, Data: handleGetDeposits(getProvider(), filterResource, filterUnmined, filterOwner)})
	})

	mux.HandleFunc("/api/planets/workforce/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/planets/workforce/")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid planet ID")
			return
		}
		data, found := handleGetWorkforce(getProvider(), id)
		if !found {
			writeErr(w, http.StatusNotFound, "planet not found")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: data})
	})

	mux.HandleFunc("/api/flows", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetGalaxyFlows(getProvider())})
	})

	mux.HandleFunc("/api/catalog", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetCatalog()})
	})

	mux.HandleFunc("/api/construction/cancel", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req CancelConstructionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.ConstructionID == "" {
			writeErr(w, http.StatusBadRequest, "construction_id required")
			return
		}
		p := getProvider()
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "cancel_construction",
			Data:   game.CancelConstructionCommandData{ConstructionID: req.ConstructionID},
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

	// Serve pages
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			fmt.Fprint(w, spectatorHTML)
		case "/data":
			fmt.Fprint(w, dashboardHTML)
		default:
			http.NotFound(w, r)
		}
	})

	// Wrap mux with auth + CORS middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Auth for POST
		if apiKey != "" && r.Method == http.MethodPost {
			key := r.Header.Get("X-API-Key")
			if key != apiKey {
				writeErr(w, http.StatusUnauthorized, "invalid or missing X-API-Key")
				return
			}
		}
		mux.ServeHTTP(w, r)
	})

	go func() {
		fmt.Println("[API] Starting REST server on :8080")
		if err := http.ListenAndServe(":8080", handler); err != nil {
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

const spectatorHTML = `<!DOCTYPE html>
<html><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Xandaris II — Live Galaxy</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#060810;overflow:hidden;font-family:'Courier New',monospace}
canvas{display:block}
#hud{position:fixed;top:10px;right:10px;background:rgba(8,12,24,0.92);border:1px solid #1a2040;border-radius:8px;padding:14px;color:#b0b8c8;font-size:12px;width:260px;max-height:90vh;overflow-y:auto;backdrop-filter:blur(8px)}
#hud h2{color:#7fdbca;font-size:14px;margin-bottom:10px;letter-spacing:1px}
#hud .row{display:flex;justify-content:space-between;padding:3px 0}
#hud .g{color:#5cdb5c}#hud .r{color:#db5555}#hud .o{color:#dba855}#hud .d{color:#556}
#hud hr{border:none;border-top:1px solid #1a2040;margin:10px 0}
#hud .section{font-size:11px;color:#7fdbca;margin:6px 0 4px;text-transform:uppercase;letter-spacing:1px}
#status{position:fixed;bottom:10px;left:12px;color:#4a4;font-size:11px}
#title{position:fixed;top:14px;left:16px;color:#7fdbca;font-size:20px;letter-spacing:2px;text-shadow:0 0 20px rgba(127,219,202,0.3)}
#subtitle{position:fixed;top:38px;left:16px;color:#445;font-size:11px}
#tooltip{position:fixed;display:none;background:rgba(8,12,24,0.95);border:1px solid #2a3060;border-radius:6px;padding:10px 12px;color:#c0c8d8;font-size:11px;pointer-events:none;z-index:100;backdrop-filter:blur(4px);max-width:220px}
a{color:#446}
</style></head><body>
<canvas id="c"></canvas>
<div id="title">XANDARIS II</div>
<div id="subtitle">Live Galaxy — Spectator Mode</div>
<div id="hud">
<div class="section">Factions</div>
<div id="players"></div>
<hr>
<div class="section">Market</div>
<div id="market"></div>
<hr>
<div class="section">Economy</div>
<div id="info"></div>
<hr>
<p style="font-size:10px;color:#334;margin-top:4px"><a href="/data">Data View</a> · <a href="/api/game" target="_blank">API</a> · <a href="https://github.com/hunterjsb/xandaris" target="_blank">GitHub</a></p>
</div>
<div id="tooltip"></div>
<div id="status">Connecting...</div>
<script>
const B=location.origin,C=document.getElementById('c'),X=C.getContext('2d');
let W,H,systems=[],ships=[],players=[],economy={},flows={},mx=0,my=0,hover=null,t=0;
const COLORS={Human:'#4caf50','Orion Exchange':'#ff9800','Lyra Cartel':'#e84040','Helios Commodities':'#8bc34a','Ceres Brokers':'#ffca28','Nova Frontier Co.':'#ab47bc',Server:'#4caf50'};
// Background stars
let stars=[];
function initStars(){stars=[];for(let i=0;i<200;i++)stars.push({x:Math.random(),y:Math.random(),s:Math.random()*1.5+0.5,b:Math.random()})}
initStars();
function resize(){W=C.width=innerWidth;H=C.height=innerHeight}
addEventListener('resize',()=>{resize();initStars()});resize();
C.addEventListener('mousemove',e=>{mx=e.clientX;my=e.clientY});
function pc(name){return COLORS[name]||'#6688aa'}
function hexA(hex,a){const r=parseInt(hex.slice(1,3),16),g=parseInt(hex.slice(3,5),16),b=parseInt(hex.slice(5,7),16);return'rgba('+r+','+g+','+b+','+a+')'}
async function load(){
try{
const[g,s,p,e,f]=await Promise.all(['/api/galaxy','/api/ships','/api/players','/api/economy','/api/flows'].map(u=>fetch(B+u).then(r=>r.json())));
systems=g.data;ships=s.data;players=p.data;economy=e.data;flows=f.data;
document.getElementById('status').textContent='Live · '+new Date().toLocaleTimeString();
document.getElementById('players').innerHTML=players.sort((a,b)=>b.credits-a.credits).map(p=>{
const c=pc(p.name);return'<div class="row"><span style="color:'+c+'">'+p.name+'</span><span>'+p.credits.toLocaleString()+'cr</span></div>'}).join('');
const res=economy.resources||{};
document.getElementById('market').innerHTML=Object.entries(res).sort().map(([n,r])=>{
const c=r.price_ratio>1.5?'r':r.price_ratio<0.5?'g':'d';
return'<div class="row"><span>'+n+'</span><span class="'+c+'">'+r.buy_price.toFixed(0)+' ('+r.price_ratio.toFixed(1)+'x)</span></div>'}).join('');
const nf=flows.net_flow||{};
document.getElementById('info').innerHTML=
'<div class="row"><span>Population</span><span>'+(economy.total_population||0).toLocaleString()+'</span></div>'+
'<div class="row"><span>Trade Volume</span><span>'+(economy.trade_volume||0).toFixed(0)+'</span></div>'+
Object.entries(nf).sort().map(([n,v])=>'<div class="row"><span>'+n+'</span><span class="'+(v>0?'g':v<-1?'r':'d')+'">'+((v>0?'+':'')+v.toFixed(0))+'/int</span></div>').join('');
}catch(e){document.getElementById('status').textContent='Disconnected';document.getElementById('status').style.color='#a44'}}
function sp(s){const pad=80;return[(s.x/1280)*(W-pad*2)+pad,(s.y/720)*(H-pad*2)+pad]}
function draw(){
t+=0.016;
// Background
X.fillStyle='#060810';X.fillRect(0,0,W,H);
// Stars
stars.forEach(s=>{
const flicker=0.6+0.4*Math.sin(t*2+s.b*20);
X.fillStyle=hexA('#ffffff',flicker*0.5*s.s);
X.fillRect(s.x*W,s.y*H,s.s,s.s)});
if(!systems.length){requestAnimationFrame(draw);return}
// Hyperlanes
systems.forEach(s=>{(s.links||[]).forEach(lid=>{
const tgt=systems.find(x=>x.id===lid);if(!tgt)return;
const[x1,y1]=sp(s),[x2,y2]=sp(tgt);
X.strokeStyle='rgba(50,60,90,0.25)';X.lineWidth=1;
X.beginPath();X.moveTo(x1,y1);X.lineTo(x2,y2);X.stroke()})});
// Trade routes
ships.filter(s=>s.status==='Moving'&&s.cargo_used>0).forEach(s=>{
const src=systems.find(x=>x.id===s.system_id),tgt=systems.find(x=>x.id===s.target_system);
if(!src||!tgt)return;
const[x1,y1]=sp(src),[x2,y2]=sp(tgt),c=pc(s.owner);
// Glowing route line
X.strokeStyle=hexA(c,0.15);X.lineWidth=4;X.beginPath();X.moveTo(x1,y1);X.lineTo(x2,y2);X.stroke();
X.strokeStyle=hexA(c,0.4);X.lineWidth=1;X.setLineDash([8,8]);
X.beginPath();X.moveTo(x1,y1);X.lineTo(x2,y2);X.stroke();X.setLineDash([]);
// Animated ship
const p=((t*0.3+s.system_id)%1);
const sx=x1+(x2-x1)*p,sy=y1+(y2-y1)*p;
X.fillStyle=c;X.shadowColor=c;X.shadowBlur=8;
X.beginPath();X.arc(sx,sy,3,0,Math.PI*2);X.fill();
X.shadowBlur=0;
// Cargo label
X.fillStyle=hexA(c,0.7);X.font='8px monospace';X.textAlign='center';
X.fillText(s.cargo_used+'u',sx,sy-8)});
// Systems
hover=null;
systems.forEach(s=>{
const[sx,sy]=sp(s);
const owner=s.owner||'';const c=owner?pc(owner):'#556';
const nP=s.planets||1;
// Glow for owned systems
if(owner){X.fillStyle=hexA(c,0.06);X.beginPath();X.arc(sx,sy,20+nP*3,0,Math.PI*2);X.fill()}
// Orbit rings with animated planets
for(let i=0;i<nP;i++){
const r=8+i*5;
X.strokeStyle=hexA(c,owner?0.2:0.1);X.lineWidth=0.5;
X.beginPath();X.arc(sx,sy,r,0,Math.PI*2);X.stroke();
// Planet dot orbiting
const pa=t*0.5/(i+1)+i*2.1;
const px=sx+r*Math.cos(pa),py=sy+r*Math.sin(pa);
X.fillStyle=hexA(c,0.6);X.beginPath();X.arc(px,py,1.5,0,Math.PI*2);X.fill()}
// Star with glow
X.fillStyle=c;X.shadowColor=c;X.shadowBlur=owner?12:4;
X.beginPath();X.arc(sx,sy,owner?3.5:2.5,0,Math.PI*2);X.fill();
X.shadowBlur=0;
// System name
X.fillStyle='#667';X.font='10px monospace';X.textAlign='center';
X.fillText(s.name,sx,sy+nP*5+16);
// Owner label
if(owner){
const pl=players.find(p=>p.name===owner);
X.fillStyle=hexA(c,0.8);X.font='9px monospace';
X.fillText((owner.length>10?owner.slice(0,8)+'..':owner)+' '+(pl?pl.stock:''),sx,sy+nP*5+26)}
// Hover
if((mx-sx)**2+(my-sy)**2<500)hover=s});
// Docked ships
ships.filter(s=>s.status!=='Moving').forEach(s=>{
const sys=systems.find(x=>x.id===s.system_id);if(!sys)return;
const[sx,sy]=sp(sys);
X.fillStyle=hexA(pc(s.owner),0.5);
const a=t+s.fuel_current;
X.beginPath();X.arc(sx+15*Math.cos(a),sy+15*Math.sin(a),2,0,Math.PI*2);X.fill()});
// Tooltip
const tt=document.getElementById('tooltip');
if(hover){
tt.style.display='block';tt.style.left=(mx+15)+'px';tt.style.top=Math.min(my-10,H-120)+'px';
let h='<b style="color:#7fdbca">'+hover.name+'</b><br>Planets: '+hover.planets;
if(hover.owner)h+='<br>Owner: <span style="color:'+pc(hover.owner)+'">'+hover.owner+'</span>';
if(hover.resources?.length)h+='<br>Resources: '+hover.resources.join(', ');
const ls=ships.filter(x=>x.system_id===hover.id);
if(ls.length)h+='<br>Ships: '+ls.length;
tt.innerHTML=h}else{tt.style.display='none'}
requestAnimationFrame(draw)}
load();setInterval(load,3000);draw();
</script></body></html>`

const dashboardHTML = `<!DOCTYPE html>
<html><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Xandaris II — Live Economy</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#0a0c14;color:#c0c8d8;font-family:'Courier New',monospace;padding:20px}
h1{color:#7fdbca;margin-bottom:4px}
.sub{color:#556;margin-bottom:20px;font-size:14px}
.grid{display:grid;grid-template-columns:1fr 1fr;gap:16px;max-width:1000px}
.panel{background:#12162a;border:1px solid #1e2844;border-radius:6px;padding:16px}
.panel h2{color:#7fdbca;font-size:14px;margin-bottom:12px;border-bottom:1px solid #1e2844;padding-bottom:6px}
table{width:100%;border-collapse:collapse;font-size:13px}
th{text-align:left;color:#889;padding:4px 8px}
td{padding:4px 8px}
.g{color:#6dcc6d}.r{color:#c55}.o{color:#cca444}.d{color:#556}
.st{position:fixed;bottom:8px;right:12px;font-size:11px;color:#4a4}
@media(max-width:700px){.grid{grid-template-columns:1fr}}
</style></head><body>
<h1>Xandaris II — Live Economy</h1>
<p class="sub">Real-time data from game server • Auto-refreshes every 3s</p>
<div class="grid">
<div class="panel"><h2>Market Prices</h2>
<table><thead><tr><th>Resource</th><th>Buy</th><th>Base</th><th>Ratio</th><th>Scarcity</th></tr></thead><tbody id="m"></tbody></table></div>
<div class="panel"><h2>Players</h2>
<table><thead><tr><th>Name</th><th>Credits</th><th>Pop</th><th>Mines</th><th>Stock</th></tr></thead><tbody id="p"></tbody></table></div>
<div class="panel"><h2>Galaxy Flows</h2>
<table><thead><tr><th>Resource</th><th>Prod</th><th>Cons</th><th>Net</th></tr></thead><tbody id="f"></tbody></table></div>
<div class="panel"><h2>Game Info</h2><div id="g"></div></div>
</div>
<div id="s" class="st">Loading...</div>
<script>
const B=location.origin;
async function R(){try{
const[e,p,f,g]=await Promise.all([B+'/api/economy',B+'/api/players',B+'/api/flows',B+'/api/game'].map(u=>fetch(u).then(r=>r.json())));
document.getElementById('m').innerHTML=Object.entries(e.data.resources).sort().map(([n,r])=>{
const c=r.price_ratio>1.5?'r':r.price_ratio>0.8?'':'g';
const s=r.scarcity=='Scarce'||r.scarcity=='Critical'?'o':r.scarcity=='Depleted'?'r':'d';
return'<tr><td>'+n+'</td><td class="'+c+'">'+r.buy_price.toFixed(0)+'</td><td class="d">'+r.base_price+'</td><td class="'+c+'">'+r.price_ratio.toFixed(1)+'x</td><td class="'+s+'">'+r.scarcity+'</td></tr>'}).join('');
document.getElementById('p').innerHTML=p.data.sort((a,b)=>b.credits-a.credits).map(x=>{
const c=x.credits<100?'r':x.credits<500?'o':'';
return'<tr><td>'+x.name+'</td><td class="'+c+'">'+x.credits+'</td><td>'+x.population+'</td><td>'+x.mines+'</td><td>'+x.stock+'</td></tr>'}).join('');
const a=new Set([...Object.keys(f.data.production),...Object.keys(f.data.consumption)]);
document.getElementById('f').innerHTML=[...a].sort().map(r=>{
const pr=(f.data.production[r]||0).toFixed(1),co=(f.data.consumption[r]||0).toFixed(1),n=(f.data.net_flow[r]||0).toFixed(1);
return'<tr><td>'+r+'</td><td class="g">+'+pr+'</td><td class="r">-'+co+'</td><td class="'+(n>0?'g':n<-1?'r':'')+'">'+((n>0?'+':'')+n)+'</td></tr>'}).join('');
const d=g.data;
document.getElementById('g').innerHTML='<p>Tick: '+d.tick+' • Time: '+d.game_time+' • Speed: '+d.speed+'</p><p>Systems: '+d.systems+' • Players: '+d.players+'</p><p>Population: '+e.data.total_population.toLocaleString()+'</p><p>Credits: '+e.data.total_credits.toLocaleString()+'</p><p>Trade Volume: '+e.data.trade_volume.toFixed(0)+'</p>';
document.getElementById('s').textContent='Live • '+new Date().toLocaleTimeString();
}catch(e){document.getElementById('s').textContent='Disconnected';document.getElementById('s').style.color='#a44'}}
R();setInterval(R,3000);
</script></body></html>`

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(APIResponse{OK: false, Error: msg})
}
