//go:build !js

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/systems"
)

const discordClientID = "584064302051229707"

type ctxKey string

const ctxPlayerName ctxKey = "playerName"
const ctxIsAdmin ctxKey = "isAdmin"

// getAuthPlayer returns the authenticated player name from the request context.
// Returns empty string for admin keys or unauthenticated requests.
func getAuthPlayer(r *http.Request) string {
	if v, ok := r.Context().Value(ctxPlayerName).(string); ok {
		return v
	}
	return ""
}

// newCommand creates a GameCommand with the authenticated player name attached.
func newCommand(r *http.Request, cmdType game.CommandType, data interface{}) game.GameCommand {
	return game.GameCommand{
		Type:       cmdType,
		Data:       data,
		Result:     make(chan interface{}, 1),
		PlayerName: getAuthPlayer(r),
	}
}

// isAdmin returns whether the request was authenticated with the admin key.
func isAdmin(r *http.Request) bool {
	if v, ok := r.Context().Value(ctxIsAdmin).(bool); ok {
		return v
	}
	return false
}

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
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
			Type:   "trade",
			Data:   game.TradeCommandData{Resource: req.Resource, Quantity: req.Quantity, Buy: buy, PlanetID: req.PlanetID},
			Result: resultCh,
		}
		p.GetCommandChannel() <- cmd

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
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
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
		p.GetCommandChannel() <- cmd

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
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
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
		p.GetCommandChannel() <- cmd

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
		data := handleGetPlayerMe(getProvider(), getAuthPlayer(r))
		if data == nil {
			writeErr(w, http.StatusNotFound, "no player found")
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

	mux.HandleFunc("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		p := getProvider()
		switch r.Method {
		case http.MethodGet:
			player := getAuthPlayer(r)
			orders := p.GetStandingOrders(player)
			if orders == nil {
				orders = []*game.StandingOrder{}
			}
			writeJSON(w, APIResponse{OK: true, Data: orders})
		case http.MethodPost:
			var req struct {
				PlanetID  int    `json:"planet_id"`
				Resource  string `json:"resource"`
				Action    string `json:"action"`
				Quantity  int    `json:"quantity"`
				Threshold int    `json:"threshold"`
				MaxPrice  int    `json:"max_price"`
				MinPrice  int    `json:"min_price"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeErr(w, http.StatusBadRequest, "invalid JSON")
				return
			}
			if req.Action != "buy" && req.Action != "sell" {
				writeErr(w, http.StatusBadRequest, "action must be 'buy' or 'sell'")
				return
			}
			if req.Quantity <= 0 {
				writeErr(w, http.StatusBadRequest, "quantity must be positive")
				return
			}
			resultCh := make(chan interface{}, 1)
			cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
				Type: "standing_order",
				Data: game.StandingOrderCommandData{
					PlanetID:  req.PlanetID,
					Resource:  req.Resource,
					Action:    req.Action,
					Quantity:  req.Quantity,
					Threshold: req.Threshold,
					MaxPrice:  req.MaxPrice,
					MinPrice:  req.MinPrice,
				},
				Result: resultCh,
			}
		p.GetCommandChannel() <- cmd
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
		case http.MethodDelete:
			var req struct {
				OrderID int `json:"order_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeErr(w, http.StatusBadRequest, "invalid JSON")
				return
			}
			resultCh := make(chan interface{}, 1)
			cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
				Type:   "cancel_order",
				Data:   game.CancelOrderCommandData{OrderID: req.OrderID},
				Result: resultCh,
			}
		p.GetCommandChannel() <- cmd
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
		default:
			writeErr(w, http.StatusMethodNotAllowed, "GET, POST, or DELETE")
		}
	})

	mux.HandleFunc("/api/shipping", func(w http.ResponseWriter, r *http.Request) {
		p := getProvider()
		switch r.Method {
		case http.MethodGet:
			sm := p.GetShippingManager()
			if sm == nil {
				writeJSON(w, APIResponse{OK: true, Data: []interface{}{}})
				return
			}
			writeJSON(w, APIResponse{OK: true, Data: sm.GetRoutes(getAuthPlayer(r))})
		case http.MethodPost:
			var req struct {
				SourcePlanet int    `json:"source_planet"`
				DestPlanet   int    `json:"dest_planet"`
				Resource     string `json:"resource"`
				Quantity     int    `json:"quantity"`
				ShipID       int    `json:"ship_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeErr(w, http.StatusBadRequest, "invalid JSON")
				return
			}
			sm := p.GetShippingManager()
			if sm == nil {
				writeErr(w, http.StatusInternalServerError, "shipping not available")
				return
			}
			player := getAuthPlayer(r)
			if player == "" {
				player = "Server"
			}
			id := sm.CreateRoute(player, req.SourcePlanet, req.DestPlanet, req.Resource, req.Quantity, req.ShipID)
			writeJSON(w, APIResponse{OK: true, Data: map[string]int{"route_id": id}})
		case http.MethodDelete:
			var req struct {
				RouteID int `json:"route_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeErr(w, http.StatusBadRequest, "invalid JSON")
				return
			}
			sm := p.GetShippingManager()
			if sm != nil && sm.CancelRoute(req.RouteID) {
				writeJSON(w, APIResponse{OK: true, Data: map[string]int{"cancelled": req.RouteID}})
			} else {
				writeErr(w, http.StatusNotFound, "route not found")
			}
		default:
			writeErr(w, http.StatusMethodNotAllowed, "GET, POST, or DELETE")
		}
	})

	mux.HandleFunc("/api/expansion", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetExpansionTargets(getProvider(), getAuthPlayer(r))})
	})

	mux.HandleFunc("/api/deliveries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		p := getProvider()
		dm := p.GetDeliveryManager()
		if dm == nil {
			writeJSON(w, APIResponse{OK: true, Data: []interface{}{}})
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: dm.GetActiveDeliveries()})
	})

	mux.HandleFunc("/api/power", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetPowerGrid(getProvider())})
	})

	mux.HandleFunc("/api/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetLeaderboard(getProvider())})
	})

	mux.HandleFunc("/api/defense", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetDefense(getProvider())})
	})

	mux.HandleFunc("/api/stations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetStations(getProvider())})
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetStatus(getProvider(), getAuthPlayer(r))})
	})

	mux.HandleFunc("/api/game", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetGame(getProvider())})
	})

	mux.HandleFunc("/api/diagnostics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		if !isAdmin(r) {
			writeErr(w, http.StatusForbidden, "admin only")
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: handleGetDiagnostics(getProvider())})
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
		cmd := newCommand(r, game.CmdBuild, game.BuildCommandData{
			PlanetID:     req.PlanetID,
			BuildingType: req.BuildingType,
			ResourceID:   req.ResourceID,
		})
		p.GetCommandChannel() <- cmd

		select {
		case result := <-cmd.Result:
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
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
			Type:   "build_ship",
			Data:   game.ShipBuildCommandData{PlanetID: req.PlanetID, ShipType: req.ShipType},
			Result: resultCh,
		}
		p.GetCommandChannel() <- cmd
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
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
			Type:   "move_ship",
			Data:   game.ShipMoveCommandData{ShipID: req.ShipID, TargetSystemID: req.TargetSystemID},
			Result: resultCh,
		}
		p.GetCommandChannel() <- cmd
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
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
			Type:   "refuel",
			Data:   game.ShipRefuelCommandData{ShipID: req.ShipID, PlanetID: req.PlanetID, Amount: req.Amount},
			Result: resultCh,
		}
		p.GetCommandChannel() <- cmd
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
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
			Type:   "colonize",
			Data:   game.ColonizeCommandData{ShipID: req.ShipID, PlanetID: req.PlanetID},
			Result: resultCh,
		}
		p.GetCommandChannel() <- cmd
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
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
			Type:   "upgrade",
			Data:   game.UpgradeCommandData{PlanetID: req.PlanetID, BuildingIndex: req.BuildingIndex},
			Result: resultCh,
		}
		p.GetCommandChannel() <- cmd
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

	mux.HandleFunc("/api/market/prices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		p := getProvider()
		market := p.GetMarket()
		if market == nil {
			writeJSON(w, APIResponse{OK: true, Data: map[string][]float64{}})
			return
		}
		snap := market.GetSnapshot()
		result := make(map[string][]float64)
		for name, rm := range snap.Resources {
			result[name] = rm.PriceHistory
		}
		writeJSON(w, APIResponse{OK: true, Data: result})
	})

	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		limit := 30
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				limit = n
			}
		}
		p := getProvider()
		el := p.GetEventLog()
		if el == nil {
			writeJSON(w, APIResponse{OK: true, Data: []game.GameEvent{}})
			return
		}
		writeJSON(w, APIResponse{OK: true, Data: el.Recent(limit)})
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

	mux.HandleFunc("/api/workforce/assign", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req WorkforceAssignRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		p := getProvider()
		resultCh := make(chan interface{}, 1)
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
			Type:   "workforce_assign",
			Data:   game.WorkforceAssignCommandData{PlanetID: req.PlanetID, BuildingIndex: req.BuildingIndex, Workers: req.Workers},
			Result: resultCh,
		}
		p.GetCommandChannel() <- cmd
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
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
			Type:   "cancel_construction",
			Data:   game.CancelConstructionCommandData{ConstructionID: req.ConstructionID},
			Result: resultCh,
		}
		p.GetCommandChannel() <- cmd
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
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
			Type:   "fleet_move",
			Data:   game.FleetMoveCommandData{FleetID: req.FleetID, TargetSystemID: req.TargetSystemID},
			Result: resultCh,
		}
		p.GetCommandChannel() <- cmd
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
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
			Type:   "fleet_create",
			Data:   game.FleetCreateCommandData{ShipID: req.ShipID},
			Result: resultCh,
		}
		p.GetCommandChannel() <- cmd
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
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
			Type:   "fleet_disband",
			Data:   game.FleetDisbandCommandData{FleetID: req.FleetID},
			Result: resultCh,
		}
		p.GetCommandChannel() <- cmd
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
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
			Type:   "fleet_add_ship",
			Data:   game.FleetAddShipCommandData{ShipID: req.ShipID, FleetID: req.FleetID},
			Result: resultCh,
		}
		p.GetCommandChannel() <- cmd
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
		cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
			Type:   "fleet_remove_ship",
			Data:   game.FleetRemoveShipCommandData{ShipID: req.ShipID, FleetID: req.FleetID},
			Result: resultCh,
		}
		p.GetCommandChannel() <- cmd
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

	// Discord OAuth2
	discordSecret := os.Getenv("DISCORD_CLIENT_SECRET")
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "https://hunterjsb.github.io/xandaris"
	}
	redirectURI := baseURL + "/api/auth/discord/callback"

	mux.HandleFunc("/api/auth/discord", func(w http.ResponseWriter, r *http.Request) {
		// Support local_callback for desktop OAuth flow
		state := r.URL.Query().Get("local_callback")
		authURL := fmt.Sprintf(
			"https://discord.com/api/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=identify&state=%s",
			discordClientID, url.QueryEscape(redirectURI), url.QueryEscape(state),
		)
		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	})

	mux.HandleFunc("/api/auth/discord/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			writeErr(w, http.StatusBadRequest, "missing code parameter")
			return
		}

		// Exchange code for access token
		tokenResp, err := http.PostForm("https://discord.com/api/oauth2/token", url.Values{
			"client_id":     {discordClientID},
			"client_secret": {discordSecret},
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {redirectURI},
		})
		if err != nil {
			writeErr(w, http.StatusBadGateway, "failed to contact Discord")
			return
		}
		defer tokenResp.Body.Close()

		tokenBody, _ := io.ReadAll(tokenResp.Body)
		fmt.Printf("[Auth] Discord token response (%d): %s\n", tokenResp.StatusCode, string(tokenBody))

		var tokenData struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			Error       string `json:"error"`
			ErrorDesc   string `json:"error_description"`
		}
		if err := json.Unmarshal(tokenBody, &tokenData); err != nil || tokenData.AccessToken == "" {
			errMsg := "failed to get Discord token"
			if tokenData.Error != "" {
				errMsg = tokenData.Error + ": " + tokenData.ErrorDesc
			}
			writeErr(w, http.StatusBadGateway, errMsg)
			return
		}

		// Fetch Discord user info
		userReq, _ := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
		userReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
		userResp, err := http.DefaultClient.Do(userReq)
		if err != nil {
			writeErr(w, http.StatusBadGateway, "failed to fetch Discord user")
			return
		}
		defer userResp.Body.Close()

		body, _ := io.ReadAll(userResp.Body)
		var discordUser struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		}
		if err := json.Unmarshal(body, &discordUser); err != nil || discordUser.ID == "" {
			writeErr(w, http.StatusBadGateway, "failed to parse Discord user")
			return
		}

		// Find or create account
		p := getProvider()
		registry := p.GetRegistry()
		if registry == nil {
			writeErr(w, http.StatusInternalServerError, "auth not available")
			return
		}

		account, isNew, err := registry.FindOrCreateByDiscord(discordUser.ID, discordUser.Username)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}

		// Create in-game faction for new players
		if isNew {
			resultCh := make(chan interface{}, 1)
			cmd := game.GameCommand{PlayerName: getAuthPlayer(r),
				Type:   "register_player",
				Data:   game.RegisterPlayerCommandData{Name: account.Name, AccountKey: account.APIKey},
				Result: resultCh,
			}
		p.GetCommandChannel() <- cmd
			select {
			case result := <-resultCh:
				if pid, ok := result.(int); ok {
					account.PlayerID = pid
					registry.Save() // persist player ID
				}
			case <-time.After(5 * time.Second):
				// faction creation timed out, but account exists
			}
		}

		// Redirect with credentials
		params := url.Values{
			"key":       {account.APIKey},
			"name":      {account.Name},
			"player_id": {strconv.Itoa(account.PlayerID)},
			"new":       {strconv.FormatBool(isNew)},
		}

		// Desktop OAuth: redirect to local callback with query params
		localCallback := r.URL.Query().Get("state")
		fmt.Printf("[Auth] Callback state=%q, account=%s, isNew=%v\n", localCallback, account.Name, isNew)
		if localCallback != "" {
			redirectTo := localCallback + "?" + params.Encode()
			fmt.Printf("[Auth] Redirecting to local callback: %s\n", redirectTo)
			http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		} else {
			// Web OAuth: redirect to frontend with fragment (never sent to server)
			http.Redirect(w, r, frontendURL+"/#"+params.Encode(), http.StatusTemporaryRedirect)
		}
	})

	// Admin-only player registration (no Discord OAuth needed)
	mux.HandleFunc("/api/admin/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		if !isAdmin(r) {
			writeErr(w, http.StatusForbidden, "admin only")
			return
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			writeErr(w, http.StatusBadRequest, "name required")
			return
		}
		p := getProvider()
		registry := p.GetRegistry()
		if registry == nil {
			writeErr(w, http.StatusInternalServerError, "registry not available")
			return
		}
		account, isNew, err := registry.FindOrCreateByName(req.Name)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if isNew {
			resultCh := make(chan interface{}, 1)
			cmd := game.GameCommand{
				Type:       game.CmdRegisterPlayer,
				Data:       game.RegisterPlayerCommandData{Name: account.Name, AccountKey: account.APIKey},
				Result:     resultCh,
				PlayerName: account.Name,
			}
			p.GetCommandChannel() <- cmd
			select {
			case result := <-resultCh:
				if pid, ok := result.(int); ok {
					account.PlayerID = pid
					registry.Save()
				}
			case <-time.After(5 * time.Second):
			}
		}
		writeJSON(w, APIResponse{OK: true, Data: map[string]interface{}{
			"name":      account.Name,
			"api_key":   account.APIKey,
			"player_id": account.PlayerID,
			"new":       isNew,
		}})
	})

	// Admin diagnostic: show mine-to-resource attachment details
	mux.HandleFunc("/api/admin/diagnose", func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			writeErr(w, http.StatusForbidden, "admin only")
			return
		}
		playerName := r.URL.Query().Get("player")
		p := getProvider()
		player := findPlayer(p, playerName)
		if player == nil {
			writeErr(w, http.StatusNotFound, "player not found")
			return
		}
		var result []map[string]interface{}
		for _, planet := range player.OwnedPlanets {
			if planet == nil {
				continue
			}
			pInfo := map[string]interface{}{
				"planet_id":   planet.GetID(),
				"planet_name": planet.Name,
				"owner":       planet.Owner,
				"resources":   []map[string]interface{}{},
				"mines":       []map[string]interface{}{},
			}
			for _, re := range planet.Resources {
				if res, ok := re.(*entities.Resource); ok {
					pInfo["resources"] = append(pInfo["resources"].([]map[string]interface{}), map[string]interface{}{
						"id": res.GetID(), "type": res.ResourceType, "owner": res.Owner,
						"abundance": res.Abundance, "rate": res.ExtractionRate,
					})
				}
			}
			for i, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == "Mine" {
					pInfo["mines"] = append(pInfo["mines"].([]map[string]interface{}), map[string]interface{}{
						"index": i, "level": b.Level, "attached_to": b.AttachedTo,
						"attachment_type": b.AttachmentType, "staffing": b.GetStaffingRatio(),
						"operational": b.IsOperational, "owner": b.Owner,
					})
				}
			}
			result = append(result, pInfo)
		}
		writeJSON(w, APIResponse{OK: true, Data: result})
	})

	// Admin-only player removal
	mux.HandleFunc("/api/admin/remove-player", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		if !isAdmin(r) {
			writeErr(w, http.StatusForbidden, "admin only")
			return
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			writeErr(w, http.StatusBadRequest, "name required")
			return
		}
		p := getProvider()

		result, err := handleRemovePlayer(p, req.Name)
		if err != nil {
			writeErr(w, http.StatusNotFound, err.Error())
			return
		}
		// Also remove from auth registry
		registry := p.GetRegistry()
		if registry != nil {
			registry.RemoveAccount(req.Name)
		}
		writeJSON(w, APIResponse{OK: true, Data: result})
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

	// LLM chat endpoint
	registerChatEndpoint(mux, getProvider)

	// Multiplayer chat
	mux.HandleFunc("/api/chat/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Message) == "" {
			writeErr(w, http.StatusBadRequest, "message required")
			return
		}
		player := getAuthPlayer(r)
		if player == "" {
			writeErr(w, http.StatusUnauthorized, "auth required")
			return
		}
		p := getProvider()
		tick, gameTime, _, _ := p.GetTickInfo()
		chatLog := p.GetChatLog()
		if chatLog == nil {
			writeErr(w, http.StatusInternalServerError, "chat not available")
			return
		}
		msg := strings.TrimSpace(req.Message)
		if len(msg) > 200 {
			msg = msg[:200]
		}
		chatLog.Send(tick, gameTime, player, msg)
		writeJSON(w, APIResponse{OK: true, Data: map[string]string{"status": "sent"}})
	})

	mux.HandleFunc("/api/chat/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		p := getProvider()
		chatLog := p.GetChatLog()
		if chatLog == nil {
			writeJSON(w, APIResponse{OK: true, Data: []interface{}{}})
			return
		}
		messages := chatLog.Recent(20)
		// Reverse to chronological order
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
		writeJSON(w, APIResponse{OK: true, Data: messages})
	})

	// Local market: what's available to trade in a specific system
	mux.HandleFunc("/api/local-market/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		// Parse system ID from /api/local-market/{system_id}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/local-market/"), "/")
		sysID, err := strconv.Atoi(parts[0])
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid system_id")
			return
		}

		p := getProvider()
		playerName := getAuthPlayer(r)
		_ = playerName
		systems := p.GetSystems()

		// Find all planets in this system with their stock
		type PlanetStock struct {
			PlanetID   int            `json:"planet_id"`
			PlanetName string         `json:"planet_name"`
			Owner      string         `json:"owner"`
			HasTP      bool           `json:"has_trading_post"`
			Stock      map[string]int `json:"stock"`
		}

		var result struct {
			SystemID     int           `json:"system_id"`
			Planets      []PlanetStock `json:"planets"`
			YourPlanets  []int         `json:"your_planets"`
			BuyableStock map[string]int `json:"buyable_stock"` // total stock on OTHER players' planets
		}
		result.SystemID = sysID
		result.BuyableStock = make(map[string]int)

		for _, sys := range systems {
			if sys.ID != sysID {
				continue
			}
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner == "" {
					continue
				}

				stock := make(map[string]int)
				for resType, s := range planet.StoredResources {
					if s != nil && s.Amount > 0 {
						stock[resType] = s.Amount
					}
				}

				hasTP := false
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
						hasTP = true
						break
					}
				}

				ps := PlanetStock{
					PlanetID: planet.GetID(), PlanetName: planet.Name,
					Owner: planet.Owner, HasTP: hasTP, Stock: stock,
				}
				result.Planets = append(result.Planets, ps)

				if planet.Owner == playerName {
					result.YourPlanets = append(result.YourPlanets, planet.GetID())
				} else {
					// Stock on other players' planets = what you can buy
					for resType, amt := range stock {
						result.BuyableStock[resType] += amt
					}
				}
			}
			break
		}

		// Add market prices for context
		market := p.GetMarket()
		type PricedStock struct {
			Resource  string  `json:"resource"`
			Available int     `json:"available"`
			BuyPrice  float64 `json:"buy_price"`
			SellPrice float64 `json:"sell_price"`
		}
		var priced []PricedStock
		if market != nil {
			for res, amt := range result.BuyableStock {
				priced = append(priced, PricedStock{
					Resource: res, Available: amt,
					BuyPrice: market.GetBuyPrice(res), SellPrice: market.GetSellPrice(res),
				})
			}
		}

		writeJSON(w, APIResponse{OK: true, Data: map[string]interface{}{
			"system_id":     result.SystemID,
			"planets":       result.Planets,
			"your_planets":  result.YourPlanets,
			"buyable_stock": priced,
		}})
	})

	// --- Logistics Endpoints ---

	// POST /api/ships/dock — dock a ship at a planet
	mux.HandleFunc("/api/ships/dock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req DockShipRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		cmd := newCommand(r, game.CmdDockShip, game.DockShipCommandData{
			ShipID: req.ShipID, PlanetID: req.PlanetID,
		})
		p := getProvider()
		p.GetCommandChannel() <- cmd
		select {
		case res := <-cmd.Result:
			if err, ok := res.(error); ok {
				writeErr(w, http.StatusBadRequest, err.Error())
			} else {
				writeJSON(w, APIResponse{OK: true, Data: res})
			}
		case <-r.Context().Done():
			writeErr(w, http.StatusGatewayTimeout, "timeout")
		}
	})

	// POST /api/ships/undock — undock a ship
	mux.HandleFunc("/api/ships/undock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req UndockShipRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		cmd := newCommand(r, game.CmdUndockShip, game.UndockShipCommandData{
			ShipID: req.ShipID,
		})
		p := getProvider()
		p.GetCommandChannel() <- cmd
		select {
		case res := <-cmd.Result:
			if err, ok := res.(error); ok {
				writeErr(w, http.StatusBadRequest, err.Error())
			} else {
				writeJSON(w, APIResponse{OK: true, Data: res})
			}
		case <-r.Context().Done():
			writeErr(w, http.StatusGatewayTimeout, "timeout")
		}
	})

	// POST /api/ships/sell-at-dock — sell cargo from a docked ship
	mux.HandleFunc("/api/ships/sell-at-dock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		var req SellAtDockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		cmd := newCommand(r, game.CmdSellAtDock, game.SellAtDockCommandData{
			ShipID: req.ShipID, Resource: req.Resource, Quantity: req.Quantity,
		})
		p := getProvider()
		p.GetCommandChannel() <- cmd
		select {
		case res := <-cmd.Result:
			if err, ok := res.(error); ok {
				writeErr(w, http.StatusBadRequest, err.Error())
			} else {
				writeJSON(w, APIResponse{OK: true, Data: res})
			}
		case <-r.Context().Done():
			writeErr(w, http.StatusGatewayTimeout, "timeout")
		}
	})

	// GET /api/shipping/routes — list shipping routes
	mux.HandleFunc("/api/shipping/routes", func(w http.ResponseWriter, r *http.Request) {
		p := getProvider()
		sm := p.GetShippingManager()
		if sm == nil {
			writeJSON(w, APIResponse{OK: true, Data: []ShippingRouteInfo{}})
			return
		}
		playerName := getAuthPlayer(r)
		routes := sm.GetRoutes(playerName)
		result := make([]ShippingRouteInfo, 0, len(routes))
		for _, rt := range routes {
			result = append(result, ShippingRouteInfo{
				ID: rt.ID, Owner: rt.Owner,
				SourcePlanet: rt.SourcePlanet, DestPlanet: rt.DestPlanet,
				Resource: rt.Resource, Quantity: rt.Quantity,
				ShipID: rt.ShipID, Active: rt.Active,
				TripsComplete: rt.TripsComplete,
			})
		}
		writeJSON(w, APIResponse{OK: true, Data: result})
	})

	// Rate limiter: 30 reads/sec, 10 writes/sec per key, burst of 60/20
	rateLimiter := NewRateLimiter(30, 10, 60, 20)

	// Wrap mux with auth + CORS + rate limiting middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, X-Player")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Rate limiting
		rlKey := r.Header.Get("X-API-Key")
		if rlKey == "" {
			rlKey = r.RemoteAddr
		}
		if !rateLimiter.Allow(rlKey, r.Method == http.MethodPost) {
			w.Header().Set("Retry-After", "1")
			writeErr(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		// Auth — inject player identity into context
		key := r.Header.Get("X-API-Key")
		if key != "" {
			p := getProvider()
			registry := p.GetRegistry()
			if registry != nil {
				playerName, admin, ok := registry.Authenticate(key)
				if ok {
					// Admin impersonation: X-Player header lets admin act as any faction
					if admin {
						if impersonate := r.Header.Get("X-Player"); impersonate != "" {
							playerName = impersonate
						}
					}
					ctx := context.WithValue(r.Context(), ctxPlayerName, playerName)
					ctx = context.WithValue(ctx, ctxIsAdmin, admin)
					r = r.WithContext(ctx)
				}
			}
		}
		// Require auth for POST (OAuth endpoints are GET, so no exemptions needed)
		if r.Method == http.MethodPost {
			player := getAuthPlayer(r)
			admin := isAdmin(r)
			if !admin && player == "" {
				// Fall back to legacy admin key check
				if apiKey != "" && key != apiKey {
					writeErr(w, http.StatusUnauthorized, "invalid or missing X-API-Key")
					return
				}
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
<div class="section">Power Grid</div>
<div id="power" style="font-size:10px"></div>
<hr>
<div class="section">Construction</div>
<div id="construction" style="font-size:10px;max-height:80px;overflow-y:auto;color:#889"></div>
<hr>
<div class="section">Events</div>
<div id="events" style="font-size:10px;max-height:120px;overflow-y:auto;color:#889"></div>
<hr>
<p style="font-size:10px;color:#334;margin-top:4px"><a href="/data">Data View</a> · <a href="/api/game" target="_blank">API</a> · <a href="https://github.com/hunterjsb/xandaris" target="_blank">GitHub</a></p>
</div>
<div id="tooltip"></div>
<div id="detail" style="display:none;position:fixed;bottom:10px;left:10px;background:rgba(8,12,24,0.95);border:1px solid #1a2040;border-radius:8px;padding:14px;color:#b0b8c8;font-size:12px;width:320px;max-height:50vh;overflow-y:auto;backdrop-filter:blur(8px)">
<div style="display:flex;justify-content:space-between"><span id="dtitle" style="color:#7fdbca;font-size:14px"></span><span onclick="selected=null;this.parentElement.parentElement.style.display='none'" style="cursor:pointer;color:#556">✕</span></div>
<div id="dbody" style="margin-top:8px"></div>
</div>
<div id="status">Connecting...</div>
<script>
const B=location.origin,C=document.getElementById('c'),X=C.getContext('2d');
let W,H,systems=[],ships=[],players=[],economy={},flows={},power=[],deliveries=[],construction=[],mx=0,my=0,hover=null,selected=null,detail=null,tracked=null,hoverShip=null,t=0;
const COLORS={Human:'#4caf50',Server:'#4caf50','Llama Logistics':'#ff9800','DeepSeek Ventures':'#e84040','Gemini Exchange':'#8bc34a','Grok Industries':'#ffca28','Opus Cartel':'#ab47bc'};
// Background stars
let stars=[];
function initStars(){stars=[];for(let i=0;i<200;i++)stars.push({x:Math.random(),y:Math.random(),s:Math.random()*1.5+0.5,b:Math.random()})}
initStars();
function resize(){W=C.width=innerWidth;H=C.height=innerHeight}
addEventListener('resize',()=>{resize();initStars()});resize();
C.addEventListener('mousemove',e=>{mx=e.clientX;my=e.clientY});
C.addEventListener('click',e=>{
if(hoverShip){tracked=tracked===hoverShip?null:hoverShip;selected=null;document.getElementById('detail').style.display='none'}
else if(hover){selected=hover;tracked=null;loadDetail(hover.id)}
else{selected=null;tracked=null;document.getElementById('detail').style.display='none'}});
// Zoom
let zoom=1,panX=0,panY=0,dragging=false,dragX=0,dragY=0;
C.addEventListener('wheel',e=>{
const oz=zoom;zoom=Math.max(0.3,Math.min(5,zoom*(e.deltaY>0?0.9:1.1)));
// Zoom toward mouse cursor (world-space pivot)
panX+=(mx-W/2-panX)*(1-zoom/oz);panY+=(my-H/2-panY)*(1-zoom/oz);
e.preventDefault()},{passive:false});
C.addEventListener('mousedown',e=>{if(e.button===0&&!hover){dragging=true;dragX=e.clientX-panX;dragY=e.clientY-panY}});
C.addEventListener('mouseup',()=>{dragging=false});
C.addEventListener('mousemove',e=>{if(dragging){panX=e.clientX-dragX;panY=e.clientY-dragY}});
function pc(name){return COLORS[name]||'#6688aa'}
function hexA(hex,a){const r=parseInt(hex.slice(1,3),16),g=parseInt(hex.slice(3,5),16),b=parseInt(hex.slice(5,7),16);return'rgba('+r+','+g+','+b+','+a+')'}
async function load(){
try{
const[g,s,p,e,f,ev,pw,dl,cx]=await Promise.all(['/api/galaxy','/api/ships','/api/players','/api/economy','/api/flows','/api/events?limit=15','/api/power','/api/deliveries','/api/construction'].map(u=>fetch(B+u).then(r=>r.json())));
systems=g.data;ships=s.data;players=p.data;economy=e.data;flows=f.data;power=pw.data||[];deliveries=dl.data||[];construction=cx.data||[];
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
'<div class="row"><span>GDP</span><span>'+(economy.gdp||0).toLocaleString(undefined,{maximumFractionDigits:0})+' cr/int</span></div>'+
'<div class="row"><span>Trade Volume</span><span>'+(economy.trade_volume||0).toFixed(0)+'</span></div>'+
'<div class="row"><span>Planets</span><span>'+(economy.total_planets||0)+'</span></div>'+
'<div class="row"><span>Routes</span><span>'+(economy.active_routes||0)+'</span></div>'+
'<div class="row"><span>Freight</span><span>'+(economy.active_deliveries||0)+' in transit</span></div>'+
Object.entries(nf).sort().map(([n,v])=>'<div class="row"><span>'+n+'</span><span class="'+(v>0?'g':v<-1?'r':'d')+'">'+((v>0?'+':'')+v.toFixed(0))+'/int</span></div>').join('');
// Power grid
document.getElementById('power').innerHTML=power.map(x=>{
const pct=x.consumed_mw>0?Math.min(1,x.generated_mw/x.consumed_mw):1;
const c=pct<0.5?'#c55':pct<0.8?'#ca4':'#5c5';
return'<div class="row"><span style="color:'+pc(x.owner)+'">'+x.owner+'</span><span style="color:'+c+'">'+x.generated_mw.toFixed(0)+'/'+x.consumed_mw.toFixed(0)+'MW</span></div>'}).join('')||'<span class="d">No data</span>';
// Construction queue
document.getElementById('construction').innerHTML=construction.map(x=>{
return'<div style="padding:1px 0;color:#889">'+x.name+' <span style="color:'+pc(x.owner)+'">'+x.owner+'</span> <span style="color:#5cf">'+x.progress+'%</span></div>'}).join('')||'<span class="d">Idle</span>';
// Events
const evts=ev.data||[];
document.getElementById('events').innerHTML=evts.map(x=>{
const c=x.type==='trade'?'#889':x.type==='colonize'?'#7fdbca':x.type==='build'?'#6c6':x.type==='alert'?'#c55':'#889';
return'<div style="color:'+c+';padding:1px 0">'+x.time+' '+x.message+'</div>'}).join('');
}catch(e){document.getElementById('status').textContent='Disconnected';document.getElementById('status').style.color='#a44'}}
async function loadDetail(sysId){
const dp=document.getElementById('detail');dp.style.display='block';
document.getElementById('dtitle').textContent=selected?.name||'';
document.getElementById('dbody').innerHTML='Loading...';
try{
const sys=await fetch(B+'/api/systems/'+sysId).then(r=>r.json());
const planets=sys.data.planets||[];
let h='<div style="color:#556">'+selected?.star_type+' · '+planets.length+' planets</div>';
if(selected?.resources?.length)h+='<div style="margin:6px 0;color:#889">Resources: '+selected.resources.join(', ')+'</div>';
planets.forEach(p=>{
h+='<div style="margin-top:8px;border-top:1px solid #1a2040;padding-top:6px">';
h+='<b style="color:#7fdbca">'+p.name+'</b> <span style="color:#556">('+p.planet_type+')</span>';
h+='<div style="color:#889">Pop: '+p.population.toLocaleString()+' / '+p.population_cap.toLocaleString()+'</div>';
if(p.owner)h+='<div>Owner: <span style="color:'+pc(p.owner)+'">'+p.owner+'</span></div>';
if(p.happiness!==undefined){const hc=p.happiness>0.7?'#5c5':p.happiness>0.4?'#ca4':'#c55';h+='<div style="color:'+hc+'">Happy: '+(p.happiness*100).toFixed(0)+'% · Prod: '+p.productivity_bonus.toFixed(1)+'x</div>'}
if(p.power_consumed>0){const pr=p.power_ratio||0;const pc2=pr>0.8?'#5c5':pr>0.5?'#ca4':'#c55';h+='<div style="color:'+pc2+'">Power: '+(pr*100).toFixed(0)+'% ('+p.power_generated.toFixed(0)+'/'+p.power_consumed.toFixed(0)+' MW)</div>'}
if(p.buildings?.length){h+='<div style="margin-top:4px">';
p.buildings.forEach(b=>{
const col=b.is_operational?'#6c6':'#c55';
h+='<span style="color:'+col+';margin-right:8px">'+b.type+(b.level>1?' L'+b.level:'')+'</span>'});
h+='</div>'}
if(p.stored_resources){h+='<div style="margin-top:4px;font-size:11px">';
Object.entries(p.stored_resources).sort().forEach(([k,v])=>{
const col=v===0?'#c55':v>800?'#6c6':'#889';
h+='<span style="color:'+col+';margin-right:8px">'+k+':'+v+'</span>'});
h+='</div>'}
h+='</div>'});
// Ships at this system
const sysShips=ships.filter(s=>s.system_id===sysId);
if(sysShips.length){h+='<div style="margin-top:8px;border-top:1px solid #1a2040;padding-top:6px;color:#889">Ships: ';
sysShips.forEach(s=>{h+='<span style="color:'+pc(s.owner)+'">'+s.name+'</span> '});h+='</div>'}
document.getElementById('dbody').innerHTML=h;
}catch(e){document.getElementById('dbody').innerHTML='<span style="color:#c55">Failed to load</span>'}}
function sp(s){const pad=80;
const bx=(s.x/1280)*(W-pad*2)+pad,by=(s.y/720)*(H-pad*2)+pad;
return[(bx-W/2)*zoom+W/2+panX,(by-H/2)*zoom+H/2+panY]}
function draw(){
t+=0.016;
// Background
X.fillStyle='#060810';X.fillRect(0,0,W,H);
// Stars (parallax at 30% of camera for depth)
stars.forEach(s=>{
const flicker=0.6+0.4*Math.sin(t*2+s.b*20);
X.fillStyle=hexA('#ffffff',flicker*0.5*s.s);
X.fillRect(s.x*W+panX*0.3,s.y*H+panY*0.3,s.s,s.s)});
if(!systems.length){requestAnimationFrame(draw);return}
// Hyperlanes
systems.forEach(s=>{(s.links||[]).forEach(lid=>{
const tgt=systems.find(x=>x.id===lid);if(!tgt)return;
const[x1,y1]=sp(s),[x2,y2]=sp(tgt);
X.strokeStyle='rgba(50,60,90,0.25)';X.lineWidth=1;
X.beginPath();X.moveTo(x1,y1);X.lineTo(x2,y2);X.stroke()})});
// Active delivery routes
deliveries.forEach(d=>{
const src=systems.find(x=>x.id===d.source_system),dst=systems.find(x=>x.id===d.dest_system);
if(!src||!dst)return;
const[x1,y1]=sp(src),[x2,y2]=sp(dst);
X.strokeStyle='rgba(127,219,202,0.08)';X.lineWidth=3;
X.beginPath();X.moveTo(x1,y1);X.lineTo(x2,y2);X.stroke()});
// Trade routes + moving ships
hoverShip=null;
const shipPositions=[];
ships.filter(s=>s.status==='Moving').forEach(s=>{
const src=systems.find(x=>x.id===s.system_id),tgt=systems.find(x=>x.id===s.target_system);
if(!src||!tgt)return;
const c=pc(s.owner);
// Build full route: current hop + remaining path
const routeIDs=[s.system_id,s.target_system,...(s.route_path||[])];
const routeSys=routeIDs.map(id=>systems.find(x=>x.id===id)).filter(Boolean);
// Draw route line along hyperlanes (only for laden ships)
if(s.cargo_used>0&&routeSys.length>=2){
X.strokeStyle=hexA(c,0.15);X.lineWidth=4;X.beginPath();
const[fx,fy]=sp(routeSys[0]);X.moveTo(fx,fy);
for(let i=1;i<routeSys.length;i++){const[nx,ny]=sp(routeSys[i]);X.lineTo(nx,ny)}
X.stroke();
X.strokeStyle=hexA(c,0.4);X.lineWidth=1;X.setLineDash([8,8]);
X.beginPath();X.moveTo(fx,fy);
for(let i=1;i<routeSys.length;i++){const[nx,ny]=sp(routeSys[i]);X.lineTo(nx,ny)}
X.stroke();X.setLineDash([])}
// Ship position: lerp along current hop (src→tgt)
const[x1,y1]=sp(src),[x2,y2]=sp(tgt);
const p=s.travel_progress||((t*0.3+s.id*0.1)%1);
const sx=x1+(x2-x1)*p,sy=y1+(y2-y1)*p;
shipPositions.push({ship:s,x:sx,y:sy});
// Track highlight
const isTracked=tracked&&tracked.id===s.id;
const r=isTracked?5:3;
X.fillStyle=c;X.shadowColor=c;X.shadowBlur=isTracked?15:8;
X.beginPath();X.arc(sx,sy,r,0,Math.PI*2);X.fill();
X.shadowBlur=0;
if(isTracked){X.strokeStyle='#7fdbca';X.lineWidth=1.5;X.beginPath();X.arc(sx,sy,10,0,Math.PI*2);X.stroke()}
// Label
X.fillStyle=hexA(c,0.7);X.font='8px monospace';X.textAlign='center';
if(s.cargo_used>0)X.fillText(s.cargo_used+'u',sx,sy-10);
if(isTracked)X.fillText(s.name,sx,sy+14);
// Hover detect
if((mx-sx)**2+(my-sy)**2<200)hoverShip=s});
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
const pop=s.population?s.population.toLocaleString():'';
X.fillStyle=hexA(c,0.8);X.font='9px monospace';
X.fillText((owner.length>10?owner.slice(0,8)+'..':owner)+(pop?' '+pop:''),sx,sy+nP*5+26)}
// Selection ring
if(selected&&selected.id===s.id){
X.strokeStyle='#7fdbca';X.lineWidth=2;X.setLineDash([4,4]);
X.beginPath();X.arc(sx,sy,25+nP*3,0,Math.PI*2);X.stroke();X.setLineDash([])}
// Hover
if((mx-sx)**2+(my-sy)**2<500*zoom)hover=s});
// Docked ships
ships.filter(s=>s.status!=='Moving').forEach(s=>{
const sys=systems.find(x=>x.id===s.system_id);if(!sys)return;
const[bx,by]=sp(sys);
const a=t+s.id*0.7;const r=18*zoom;
const sx=bx+r*Math.cos(a),sy=by+r*Math.sin(a);
const isTracked=tracked&&tracked.id===s.id;
X.fillStyle=hexA(pc(s.owner),isTracked?0.9:0.5);
X.beginPath();X.arc(sx,sy,isTracked?4:2,0,Math.PI*2);X.fill();
if(isTracked){X.strokeStyle='#7fdbca';X.lineWidth=1;X.beginPath();X.arc(sx,sy,8,0,Math.PI*2);X.stroke();
X.fillStyle='#889';X.font='8px monospace';X.textAlign='center';X.fillText(s.name,sx,sy+14)}
if((mx-sx)**2+(my-sy)**2<150)hoverShip=s});
// Tracked ship info panel
if(tracked){
const ts=ships.find(s=>s.id===tracked.id);
if(ts){tracked=ts; // update with fresh data
const dp=document.getElementById('detail');dp.style.display='block';
const c=pc(ts.owner);
document.getElementById('dtitle').innerHTML='<span style="color:'+c+'">'+ts.name+'</span>';
let h='<div style="color:#556">'+ts.type+' · '+ts.owner+'</div>';
h+='<div style="margin:6px 0"><span style="color:#889">Status:</span> <span style="color:'+(ts.status==='Moving'?'#5cf':'#6c6')+'">'+ts.status+'</span></div>';
if(ts.target_system>=0){const tgt=systems.find(x=>x.id===ts.target_system);h+='<div><span style="color:#889">Route:</span> SYS-'+ts.system_id+' → '+(tgt?tgt.name:'SYS-'+ts.target_system)+'</div>'}
// Fuel bar
const fp=ts.fuel_current/ts.fuel_max;
h+='<div style="margin:6px 0"><span style="color:#889">Fuel:</span> '+ts.fuel_current+'/'+ts.fuel_max+'</div>';
h+='<div style="background:#1a2040;border-radius:3px;height:6px;margin:2px 0"><div style="background:'+(fp>0.5?'#5c5':fp>0.25?'#ca4':'#c44')+';width:'+(fp*100)+'%;height:100%;border-radius:3px"></div></div>';
// Health bar
const hp=ts.health_current/ts.health_max;
h+='<div><span style="color:#889">Health:</span> '+ts.health_current+'/'+ts.health_max+'</div>';
h+='<div style="background:#1a2040;border-radius:3px;height:6px;margin:2px 0"><div style="background:'+(hp>0.5?'#5c5':'#c44')+';width:'+(hp*100)+'%;height:100%;border-radius:3px"></div></div>';
// Cargo
h+='<div style="margin:6px 0"><span style="color:#889">Cargo:</span> '+ts.cargo_used+'/'+ts.cargo_max+'</div>';
if(ts.cargo_hold&&Object.keys(ts.cargo_hold).length){h+='<div style="font-size:11px">';Object.entries(ts.cargo_hold).forEach(([k,v])=>{h+='<span style="color:#7fdbca;margin-right:8px">'+k+': '+v+'</span>'});h+='</div>'}
document.getElementById('dbody').innerHTML=h}}
// Tooltip
const tt=document.getElementById('tooltip');
if(hoverShip){
tt.style.display='block';tt.style.left=(mx+15)+'px';tt.style.top=Math.min(my-10,H-100)+'px';
const s=hoverShip,c=pc(s.owner);
let h='<b style="color:'+c+'">'+s.name+'</b><br><span style="color:#556">'+s.type+' · '+s.owner+'</span>';
h+='<br>'+s.status+(s.target_system>=0?' → SYS-'+s.target_system:'');
h+='<br>Fuel: '+s.fuel_current+'/'+s.fuel_max;
if(s.cargo_used>0)h+='<br>Cargo: '+s.cargo_used+'/'+s.cargo_max;
h+='<br><span style="color:#556">Click to track</span>';
tt.innerHTML=h;C.style.cursor='pointer';
}else if(hover){
tt.style.display='block';tt.style.left=(mx+15)+'px';tt.style.top=Math.min(my-10,H-120)+'px';
let h='<b style="color:#7fdbca">'+hover.name+'</b><br>Planets: '+hover.planets;
if(hover.owner)h+='<br>Owner: <span style="color:'+pc(hover.owner)+'">'+hover.owner+'</span>';
if(hover.population)h+='<br>Pop: '+hover.population.toLocaleString();
if(hover.resources?.length)h+='<br>Resources: '+hover.resources.join(', ');
const ls=ships.filter(x=>x.system_id===hover.id);
if(ls.length)h+='<br>Ships: '+ls.length;
h+='<br><span style="color:#556">Click for details</span>';
tt.innerHTML=h;C.style.cursor='pointer';
}else{tt.style.display='none';C.style.cursor=dragging?'grabbing':'default'}
requestAnimationFrame(draw)}
load();setInterval(load,3000);draw();
</script></body></html>`

const dashboardHTML = `<!DOCTYPE html>
<html><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Xandaris II — Live Economy Dashboard</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#0a0c14;color:#c0c8d8;font-family:'Courier New',monospace;padding:16px 24px}
h1{color:#7fdbca;margin-bottom:2px;font-size:1.6em;display:inline}
.sub{color:#445;font-size:12px}
.grid{display:grid;grid-template-columns:1fr 1fr;gap:10px;margin-top:12px}
.wide{grid-column:span 2}
.panel{background:#10142a;border:1px solid #1a2040;border-radius:5px;padding:10px 12px}
.panel h2{color:#7fdbca;font-size:11px;margin-bottom:6px;letter-spacing:0.5px;text-transform:uppercase;display:flex;justify-content:space-between;align-items:center}
.panel h2 .tag{font-size:9px;color:#334;font-weight:normal;text-transform:none}
table{width:100%;border-collapse:collapse;font-size:11px}
th{text-align:left;color:#445;padding:2px 4px;font-weight:normal;font-size:9px;text-transform:uppercase;letter-spacing:0.3px}
td{padding:3px 4px}
.g{color:#5cb85c}.r{color:#d9534f}.o{color:#c8a84e}.d{color:#334}.b{color:#5bc0de}.p{color:#b088df}.w{color:#c0c8d8}
.bar-bg{background:#0c1020;border-radius:2px;overflow:hidden;height:6px;margin:1px 0}
.bar-fill{height:100%;border-radius:2px;transition:width 0.5s}
.spark{font-size:10px;letter-spacing:-1px;color:#5cb85c}
.event{font-size:10px;padding:3px 0;border-bottom:1px solid #0c1020}
.rank{color:#7fdbca;font-weight:bold;margin-right:4px}
.st{position:fixed;bottom:6px;right:10px;font-size:9px;color:#3a5}
.stats-row{display:flex;gap:6px;margin:10px 0;flex-wrap:wrap;align-items:flex-end}
.stat{text-align:center;padding:2px 8px}
.stat .val{font-size:22px;color:#7fdbca;font-weight:bold}
.stat .lbl{font-size:8px;color:#334;text-transform:uppercase;letter-spacing:0.5px}
.stat-sep{width:1px;height:30px;background:#1a2040;align-self:center}
.sg{display:flex;gap:6px;background:#0c1020;border-radius:4px;padding:3px 8px}
.sg .stat .val{font-size:18px}
.pwr-grid{display:flex;flex-wrap:wrap;gap:3px}
.pwr-cell{width:30px;height:30px;border-radius:3px;display:flex;flex-direction:column;align-items:center;justify-content:center;font-size:7px;color:#fff;cursor:default;transition:transform 0.15s}
.pwr-cell:hover{transform:scale(1.2)}
.pwr-cell .pct{font-size:10px;font-weight:bold}
.row{display:flex;justify-content:space-between;align-items:center;padding:3px 0;border-bottom:1px solid #0c1020;font-size:11px}
.chain{font-size:11px;color:#667;line-height:1.8;padding:2px 0}
.chain b{color:#7fdbca}.chain .arr{color:#2a3a5a;margin:0 2px}
.chat{font-size:11px;padding:4px 0;border-bottom:1px solid #0c1020}
.chat .name{color:#5bc0de;font-weight:bold}
.res-bar{display:flex;align-items:center;gap:6px;padding:2px 0;font-size:11px}
.res-bar .lbl{width:80px;color:#889}
.res-bar .wrap{flex:1;position:relative;height:14px;background:#0c1020;border-radius:2px;overflow:hidden}
.res-bar .fill{height:100%;border-radius:2px;transition:width 0.5s}
.res-bar .txt{position:absolute;right:4px;top:0;font-size:9px;line-height:14px;color:#aab}
.lb-row{display:flex;align-items:center;gap:6px;padding:3px 0;font-size:11px}
.lb-bar{flex:1;height:10px;background:#0c1020;border-radius:2px;overflow:hidden}
.lb-fill{height:100%;background:#1a5a3a;border-radius:2px;transition:width 0.5s}
@media(max-width:800px){.grid{grid-template-columns:1fr}.wide{grid-column:span 1}}
.ticker-wrap{overflow:hidden;background:#080c18;border:1px solid #1a2040;border-radius:4px;margin:8px 0;height:28px;position:relative}
.ticker-wrap::before,.ticker-wrap::after{content:'';position:absolute;top:0;bottom:0;width:40px;z-index:1;pointer-events:none}
.ticker-wrap::before{left:0;background:linear-gradient(to right,#080c18,transparent)}
.ticker-wrap::after{right:0;background:linear-gradient(to left,#080c18,transparent)}
.ticker{display:flex;white-space:nowrap;align-items:center;height:100%}
.tick-item{display:inline-flex;align-items:center;gap:4px;padding:0 16px;font-size:11px;border-right:1px solid #151a30;height:100%;flex-shrink:0}
.tick-item .res{color:#7fdbca;font-weight:bold}
.tick-item .buy{color:#5cb85c}
.tick-item .sell{color:#d9534f}
.tick-item .who{color:#556}
.tick-item .price{color:#c8a84e}
</style></head><body>
<h1>XANDARIS II</h1> <span class="sub">&mdash; Live Economy</span>
<div class="stats-row" id="top"></div>
<div class="ticker-wrap"><div class="ticker" id="ticker"><span class="tick-item" style="color:#334">Loading trades...</span></div></div>
<div class="grid">
<div class="panel"><h2>Leaderboard <span class="tag">empire score</span></h2><div id="lb"></div></div>
<div class="panel"><h2>Faction Chat <span class="tag">live</span></h2><div id="ch" style="max-height:220px;overflow-y:auto"></div></div>
<div class="panel"><h2>Resource Balance <span class="tag">supply vs demand</span></h2><div id="rf"></div></div>
<div class="panel"><h2>Power Grid <span class="tag">MW</span></h2><div id="pw"></div></div>
<div class="panel" style="padding:6px 8px 0;overflow:hidden;position:relative"><div id="priceTickers" style="display:flex;gap:4px;margin-bottom:4px;flex-wrap:wrap"></div><canvas id="priceChart" style="width:100%;height:200px;display:block;cursor:crosshair"></canvas><div id="priceTooltip" style="display:none;position:absolute;background:rgba(8,12,24,0.95);border:1px solid #2a3060;border-radius:4px;padding:6px 8px;font-size:10px;color:#c0c8d8;pointer-events:none;z-index:10;white-space:nowrap"></div></div>
<div class="panel"><h2>Trading Hubs <span class="tag">top planets by stock</span></h2><div id="hubs"></div></div>
<div class="panel wide" style="background:#0d1125;border-color:#152040;padding:0;overflow:hidden"><canvas id="flowCanvas" style="width:100%;height:240px;display:block"></canvas></div>
<div class="panel"><h2>Events <span class="tag">activity feed</span></h2><div id="ev" style="max-height:220px;overflow-y:auto"></div></div>
<div class="panel"><h2>Fleet <span class="tag">ships by faction</span></h2><div id="sh"></div></div>
</div>
<div id="s" class="st">Loading...</div>
<script>
const B=location.origin,BL='\u2581\u2582\u2583\u2584\u2585\u2586\u2587\u2588';
function sp(h){if(!h||h.length<3)return'';const mn=Math.min(...h),mx=Math.max(...h),rng=mx-mn||1;return'<span class="spark">'+h.slice(-25).map(v=>BL[Math.min(7,Math.max(0,Math.round((v-mn)/rng*7)))]).join('')+'</span>'}
// Trade ticker — smooth scrolling tape
let tickerOffset=0,tickerItems=[],lastTradeId='';
function updateTicker(){
const el=document.getElementById('ticker');if(!el||!tickerItems.length)return;
// Build items HTML (duplicate for seamless loop)
const html=tickerItems.map(t=>{
const cls=t.action==='buy'||t.action==='bought'?'buy':'sell';
const verb=t.action==='buy'||t.action==='bought'?'\u25B2':'\u25BC';
return'<span class="tick-item"><span class="'+cls+'">'+verb+'</span><span class="res">'+t.resource+'</span><span>'+t.qty+'\u00d7</span><span class="price">'+t.price+'cr</span><span class="who">'+t.player+'</span></span>'}).join('');
el.innerHTML=html+html; // duplicate for seamless wrap
el.style.width=(el.scrollWidth/2)+'px';
}
function animateTicker(){
const el=document.getElementById('ticker');if(!el||el.children.length<2)return requestAnimationFrame(animateTicker);
tickerOffset-=0.5;
const halfW=el.scrollWidth/2;
if(Math.abs(tickerOffset)>=halfW)tickerOffset=0;
el.style.transform='translateX('+tickerOffset+'px)';
requestAnimationFrame(animateTicker)}
async function pollTrades(){try{
const r=await fetch(B+'/api/events?limit=30').then(r=>r.json());
const trades=(r.data||[]).filter(e=>e.type==='trade');
if(trades.length>0&&trades[0].message!==lastTradeId){
lastTradeId=trades[0].message;
tickerItems=trades.map(t=>{
const m=t.message;
const parts=m.match(/(\S+)\s+(bought|sold)\s+(\d+)\s+(\S+(?:\s+\S+)?)\s+@\s+(\d+)/);
if(!parts)return null;
return{player:parts[1],action:parts[2],qty:parts[3],resource:parts[4],price:parts[5]};
}).filter(Boolean);
updateTicker()}}catch(e){}}
pollTrades();setInterval(pollTrades,5000);requestAnimationFrame(animateTicker);
async function R(){try{
const[e,p,f,g,lb,pw,ev,sh,ch]=await Promise.all(['/api/economy','/api/players','/api/flows','/api/game','/api/leaderboard','/api/power','/api/events?limit=20','/api/ships','/api/chat/messages'].map(u=>fetch(B+u).then(r=>r.json())));
const d=g.data;
const st=(l,v)=>'<div class="stat"><div class="val">'+v+'</div><div class="lbl">'+l+'</div></div>';
const sep='<div class="stat-sep"></div>';
document.getElementById('top').innerHTML=
st('Time',d.game_time)+st('Speed',d.speed+(d.paused?' \u23F8':''))+sep+
'<div class="sg">'+st('Pop',e.data.total_population.toLocaleString())+st('Planets',e.data.total_planets||0)+st('Systems',d.systems)+'</div>'+sep+
'<div class="sg">'+st('GDP',(e.data.gdp||0).toLocaleString(undefined,{maximumFractionDigits:0}))+st('Credits',e.data.total_credits.toLocaleString())+st('Trade',e.data.trade_volume.toFixed(0))+'</div>'+sep+
'<div class="sg">'+st('Routes',e.data.active_routes||0)+st('Freight',e.data.active_deliveries||0)+'</div>';
// Leaderboard with bars
const maxScore=Math.max(...(lb.data||[]).map(x=>x.score),1);
document.getElementById('lb').innerHTML=(lb.data||[]).map(x=>{
const c=x.type=='human'?'b':'w';const pct=(x.score/maxScore*100).toFixed(0);
return'<div class="lb-row"><span class="rank">#'+x.rank+'</span><span class="'+c+'" style="width:90px">'+x.name+'</span><div class="lb-bar"><div class="lb-fill" style="width:'+pct+'%"></div></div><span class="d" style="width:60px;text-align:right;font-size:10px">'+x.score.toLocaleString()+'</span></div>'}).join('');
// Resource balance with visual bars
const fl=f.data,pr=fl.production,co=fl.consumption;
const allRes=[...new Set([...Object.keys(pr),...Object.keys(co)])].sort();
const maxFlow=Math.max(...allRes.map(r=>Math.max(pr[r]||0,co[r]||0)),1);
document.getElementById('rf').innerHTML=allRes.map(r=>{
const pv=pr[r]||0,cv=co[r]||0,nv=pv-cv;
const pw2=Math.round(pv/maxFlow*100),cw=Math.round(cv/maxFlow*100);
const nc=nv>1?'g':nv<-1?'r':'d';
return'<div class="res-bar"><span class="lbl">'+r+'</span><div class="wrap"><div class="fill" style="width:'+pw2+'%;background:#1a4a2a"></div><div class="fill" style="width:'+cw+'%;background:#4a1a1a;position:absolute;top:0;left:0;opacity:0.6"></div><div class="txt '+nc+'">'+(nv>0?'+':'')+nv.toFixed(0)+'/s</div></div></div>'}).join('');
// Power — compact rows with inline tiles + sparkline
const byOwner={};(pw.data||[]).forEach(x=>{if(!byOwner[x.owner])byOwner[x.owner]=[];byOwner[x.owner].push(x)});
document.getElementById('pw').innerHTML=Object.entries(byOwner).sort().map(([owner,planets])=>{
const totalGen=planets.reduce((s,p)=>s+p.generated_mw,0);
const totalCons=planets.reduce((s,p)=>s+p.consumed_mw,0);
const avgPct=totalCons>0?Math.min(1,totalGen/totalCons):1;
let hist=[];planets.forEach(p=>{if(p.history&&p.history.length>hist.length)hist=p.history});
const spark=hist.length>3?sp(hist):'';
const tiles=planets.map(p=>{
const pct=p.consumed_mw>0?Math.min(1,p.generated_mw/p.consumed_mw):1;
const bg2=pct<0.3?'#4a1515':pct<0.5?'#4a2a15':pct<0.8?'#3a3a15':'#153a15';
return'<span style="display:inline-block;width:16px;height:16px;border-radius:2px;background:'+bg2+'" title="'+p.planet_name+': '+(pct*100).toFixed(0)+'%"></span>'}).join('');
return'<div style="display:flex;align-items:center;gap:6px;padding:3px 0;border-bottom:1px solid #0c1020;font-size:11px"><span style="width:85px;color:#7fdbca;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">'+owner+'</span><span style="display:flex;gap:2px">'+tiles+'</span><span style="flex:1;text-align:right">'+spark+'</span><span class="d" style="width:55px;text-align:right;font-size:9px">'+(avgPct*100).toFixed(0)+'%</span></div>'}).join('');
// Market — store data for interactive chart
window._marketData=e.data.resources;
// Trading hubs — top planets by total stock value
const hubs=[];(p.data||[]).forEach(pl=>{pl.planets&&pl.planets.forEach&&0;/* players don't have planets inline */});
// Build from players data — sort by stock
const hubData=(p.data||[]).sort((a,b)=>b.stock-a.stock).slice(0,8);
document.getElementById('hubs').innerHTML=hubData.map(x=>{
const maxStock=Math.max(...hubData.map(h=>h.stock),1);
const bw2=Math.round(x.stock/maxStock*100);
const tc=x.type=='human'?'b':'w';
return'<div style="display:flex;align-items:center;gap:6px;padding:3px 0;border-bottom:1px solid #0c1020;font-size:11px"><span class="'+tc+'" style="width:85px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">'+x.name+'</span><div style="flex:1;height:8px;background:#0c1020;border-radius:2px;overflow:hidden"><div style="height:100%;width:'+bw2+'%;background:#1a3a5a;border-radius:2px"></div></div><span class="d" style="width:45px;text-align:right;font-size:10px">'+x.stock.toLocaleString()+'</span><span class="d" style="width:20px;text-align:right;font-size:9px">'+x.planets+'p</span></div>'}).join('');
// Production chains — update flow data for canvas animation
window._flowProd=pr;window._flowCons=co;
// Chat
document.getElementById('ch').innerHTML=(ch.data||[]).map(x=>{
return'<div class="chat"><span class="d">['+x.time+']</span> <span class="name">'+x.player+'</span> '+x.message+'</div>'}).join('')||'<div class="d" style="padding:20px;text-align:center;font-size:12px">Waiting for factions to chat...</div>';
// Events
document.getElementById('ev').innerHTML=(ev.data||[]).map(x=>{
const c=x.type=='trade'?'g':x.type=='build'?'b':x.type=='alert'?'r':x.type=='event'?'o':x.type=='join'||x.type=='colonize'?'p':'d';
return'<div class="event"><span class="d">['+x.time+']</span> <span class="'+c+'">'+x.message+'</span></div>'}).join('');
// Ships
const so={};(sh.data||[]).forEach(s=>{if(!so[s.owner])so[s.owner]=[];so[s.owner].push(s)});
document.getElementById('sh').innerHTML=Object.entries(so).sort().map(([o,ss])=>{
const t={};ss.forEach(s=>{t[s.type]=(t[s.type]||0)+1});const mv=ss.filter(s=>s.status==='Moving').length;
return'<div class="row"><span>'+o+'</span><span class="d">'+Object.entries(t).map(([k,v])=>v+'\u00d7'+k).join(' ')+(mv?' <span class="o">('+mv+' moving)</span>':'')+'</span></div>'}).join('');
document.getElementById('s').textContent='Live \u2022 '+new Date().toLocaleTimeString();
}catch(err){document.getElementById('s').textContent='Disconnected';document.getElementById('s').style.color='#a44'}}
R();setInterval(R,3000);
// === Candlestick Price Chart ===
(function(){
const pc=document.getElementById('priceChart');if(!pc)return;
const cx=pc.getContext('2d');
const tt=document.getElementById('priceTooltip');
const tickers=document.getElementById('priceTickers');
function resize(){pc.width=pc.offsetWidth*2;pc.height=pc.offsetHeight*2;cx.scale(2,2)}
resize();addEventListener('resize',resize);
const W=()=>pc.width/2,H=()=>pc.height/2;
const RC={Iron:'#b4784f',Water:'#508cc8',Oil:'#808080','Rare Metals':'#c8b464','Helium-3':'#b4dcff',Fuel:'#50a050',Electronics:'#5090d0'};
const UP='#26a69a',DN='#ef5350'; // green/red candle colors
let selected='Iron';
// Build ticker buttons
function buildTickers(){
const res=window._marketData;if(!res)return;
tickers.innerHTML=Object.keys(res).sort().map(n=>{
const r=res[n],c=RC[n]||'#888';
const cur=r.buy_price,base=r.base_price,diff=cur-base;
const dc=diff>0?DN:UP;
const sel=n===selected;
return'<span data-res="'+n+'" style="cursor:pointer;padding:3px 8px;border-radius:3px;font-size:10px;'
+'background:'+(sel?'#151a30':'transparent')+';border:1px solid '+(sel?c+'60':'#1a2040')
+';display:inline-flex;align-items:center;gap:4px">'
+'<span style="color:'+c+'">\u25CF</span>'
+'<span style="color:'+(sel?'#ccd':'#556')+'">'+n+'</span>'
+'<b style="color:'+(sel?'#ccd':'#556')+'">'+cur.toFixed(0)+'</b>'
+'<span style="color:'+dc+';font-size:9px">'+(diff>0?'+':'')+diff.toFixed(0)+'</span>'
+'</span>'}).join('');
tickers.querySelectorAll('span[data-res]').forEach(el=>{
el.onclick=()=>{selected=el.dataset.res;buildTickers()}})}
buildTickers();setInterval(buildTickers,3000);
// Mouse
let mouseX=-1,mouseIn=false;
pc.addEventListener('mousemove',e=>{const r=pc.getBoundingClientRect();mouseX=e.clientX-r.left;mouseIn=true});
pc.addEventListener('mouseleave',()=>{mouseIn=false;tt.style.display='none'});
// Build OHLC candles from price history (group every 3 ticks)
function buildCandles(hist){
const candles=[],gs=3;
for(let i=0;i<hist.length;i+=gs){
const slice=hist.slice(i,i+gs);
candles.push({o:slice[0],h:Math.max(...slice),l:Math.min(...slice),c:slice[slice.length-1]})}
return candles}
function drawChart(){
const w=W(),h=H();
cx.clearRect(0,0,w,h);
const res=window._marketData;
if(!res||!res[selected]){cx.fillStyle='#334';cx.font='11px monospace';cx.textAlign='center';cx.fillText('Waiting for data...',w/2,h/2);requestAnimationFrame(drawChart);return}
const r=res[selected],hist=r.price_history||[];
if(hist.length<3){requestAnimationFrame(drawChart);return}
const c=RC[selected]||'#888';
const candles=buildCandles(hist);
const pad={t:14,b:14,l:44,r:16};
const cw=w-pad.l-pad.r,ch=h-pad.t-pad.b;
// Y range from candle data
let yMin=Infinity,yMax=-Infinity;
candles.forEach(cd=>{if(cd.l<yMin)yMin=cd.l;if(cd.h>yMax)yMax=cd.h});
// Include base price in range
if(r.base_price<yMin)yMin=r.base_price;if(r.base_price>yMax)yMax=r.base_price;
const yPad=(yMax-yMin)*0.12||5;yMin-=yPad;yMax+=yPad;
if(yMin<0)yMin=0;
const toY=v=>pad.t+ch*(1-(v-yMin)/(yMax-yMin));
// Grid
cx.strokeStyle='#131828';cx.lineWidth=0.5;
const range=yMax-yMin;
let step=Math.pow(10,Math.floor(Math.log10(range)));
if(range/step<3)step/=2;if(range/step>8)step*=2;
for(let v=Math.ceil(yMin/step)*step;v<=yMax;v+=step){
const y=toY(v);
cx.beginPath();cx.moveTo(pad.l,y);cx.lineTo(pad.l+cw,y);cx.stroke();
cx.fillStyle='#2a2a3a';cx.font='8px monospace';cx.textAlign='right';
cx.fillText(v.toFixed(0),pad.l-4,y+3)}
// Base price line
const baseY=toY(r.base_price);
cx.strokeStyle=c+'25';cx.lineWidth=0.5;cx.setLineDash([4,4]);
cx.beginPath();cx.moveTo(pad.l,baseY);cx.lineTo(pad.l+cw,baseY);cx.stroke();
cx.setLineDash([]);
cx.fillStyle=c+'40';cx.font='7px monospace';cx.textAlign='left';
cx.fillText('base '+r.base_price.toFixed(0),pad.l+2,baseY-3);
// Candles
const candleW=Math.max(3,Math.min(12,(cw/candles.length)*0.7));
const gap=cw/candles.length;
let hoverCandle=-1;
candles.forEach((cd,i)=>{
const x=pad.l+gap*i+gap/2;
const bull=cd.c>=cd.o;
const col=bull?UP:DN;
// Wick (high-low line)
cx.strokeStyle=col+'80';cx.lineWidth=1;
cx.beginPath();cx.moveTo(x,toY(cd.h));cx.lineTo(x,toY(cd.l));cx.stroke();
// Body (open-close rect)
const bodyTop=toY(Math.max(cd.o,cd.c)),bodyBot=toY(Math.min(cd.o,cd.c));
const bodyH=Math.max(1,bodyBot-bodyTop);
if(bull){cx.fillStyle=col+'30';cx.strokeStyle=col;cx.lineWidth=1;
cx.fillRect(x-candleW/2,bodyTop,candleW,bodyH);cx.strokeRect(x-candleW/2,bodyTop,candleW,bodyH)}
else{cx.fillStyle=col;cx.fillRect(x-candleW/2,bodyTop,candleW,bodyH)}
// Hover detect
if(mouseIn){
const mx2=mouseX*2;
if(Math.abs(mx2-x)<gap/2)hoverCandle=i}});
// Current price label on right edge
const lastPrice=hist[hist.length-1];
const lastY=toY(lastPrice);
const lastDiff=lastPrice-r.base_price;
cx.fillStyle=lastDiff>=0?DN:UP;
cx.font='bold 10px monospace';cx.textAlign='left';
cx.fillText(lastPrice.toFixed(0),pad.l+cw+4,lastY+4);
// Hover tooltip
if(hoverCandle>=0&&hoverCandle<candles.length){
const cd=candles[hoverCandle];
const x=pad.l+gap*hoverCandle+gap/2;
// Highlight candle
cx.strokeStyle='#ffffff30';cx.lineWidth=0.5;
cx.beginPath();cx.moveTo(x,pad.t);cx.lineTo(x,pad.t+ch);cx.stroke();
const bull=cd.c>=cd.o;
tt.style.display='block';
tt.innerHTML='<div style="color:'+c+';font-weight:bold;margin-bottom:3px">'+selected+'</div>'
+'<div style="display:grid;grid-template-columns:auto auto;gap:1px 8px;font-size:10px">'
+'<span style="color:#556">O</span><span>'+cd.o.toFixed(1)+'</span>'
+'<span style="color:#556">H</span><span style="color:#5cb85c">'+cd.h.toFixed(1)+'</span>'
+'<span style="color:#556">L</span><span style="color:#d9534f">'+cd.l.toFixed(1)+'</span>'
+'<span style="color:#556">C</span><span style="color:'+(bull?UP:DN)+'">'+cd.c.toFixed(1)+'</span>'
+'</div>';
const tr=pc.getBoundingClientRect();
let tx=mouseX+12,ty=20;
if(mouseX>tr.width*0.7)tx=mouseX-tt.offsetWidth-12;
tt.style.left=tx+'px';tt.style.top=ty+'px'}
requestAnimationFrame(drawChart)}
drawChart()})();
// === Flow Diagram Canvas ===
(function(){
const fc=document.getElementById('flowCanvas');if(!fc)return;
const fx=fc.getContext('2d');
function resizeFlow(){fc.width=fc.offsetWidth*2;fc.height=fc.offsetHeight*2;fx.scale(2,2)}
resizeFlow();addEventListener('resize',resizeFlow);
const W=()=>fc.width/2,H=()=>fc.height/2;
// Layout: 3 rows
// Row 1 (y=0.22): Iron → Factory → Electronics → Tech Level
// Row 2 (y=0.50): Mines → Water/RM/He-3 → (middle) → Power → Happiness → Pop Growth
// Row 3 (y=0.78): Oil → Refinery → Fuel → Generator
const BW=52,BH=28; // box size
const nodes=[
// Col 1: extraction
{id:'mine',label:'\u26CF Mines',x:0.06,y:0.50,c:'#8a7050',kind:'building'},
// Col 2: raw resources
{id:'iron',label:'Iron',x:0.19,y:0.22,c:'#b4784f',res:'Iron'},
{id:'water',label:'Water',x:0.19,y:0.50,c:'#508cc8',res:'Water'},
{id:'oil',label:'Oil',x:0.19,y:0.78,c:'#808080',res:'Oil'},
{id:'rm',label:'Rare Metals',x:0.32,y:0.22,c:'#c8b464',res:'Rare Metals'},
{id:'he3',label:'Helium-3',x:0.32,y:0.50,c:'#b4dcff',res:'Helium-3'},
// Col 3: processing
{id:'factory',label:'\u2699 Factory',x:0.45,y:0.22,c:'#b482ff',kind:'building'},
{id:'refinery',label:'\u2699 Refinery',x:0.45,y:0.78,c:'#c88232',kind:'building'},
// Col 4: products
{id:'elec',label:'Electronics',x:0.58,y:0.22,c:'#5090d0',res:'Electronics'},
{id:'fuel',label:'Fuel',x:0.58,y:0.78,c:'#50a050',res:'Fuel'},
// Col 5: power
{id:'gen',label:'\u26A1 Generator',x:0.72,y:0.78,c:'#ffa030',kind:'building'},
{id:'fusion',label:'\u26A1 Fusion',x:0.72,y:0.50,c:'#64dcff',kind:'building'},
{id:'tech',label:'Tech Level',x:0.72,y:0.22,c:'#a0a0ff',kind:'outcome'},
// Col 6: outcomes
{id:'power',label:'\u26A1 Power',x:0.85,y:0.64,c:'#ffcc00',kind:'outcome'},
{id:'happy',label:'\u263A Happiness',x:0.85,y:0.36,c:'#50c878',kind:'outcome'},
{id:'growth',label:'\u2191 Growth',x:0.95,y:0.50,c:'#7fdbca',kind:'outcome'},
];
const edges=[
{from:'mine',to:'iron',c:'#b4784f'},{from:'mine',to:'water',c:'#508cc8'},{from:'mine',to:'oil',c:'#808080'},
{from:'mine',to:'rm',c:'#c8b464'},{from:'mine',to:'he3',c:'#b4dcff'},
{from:'oil',to:'refinery',c:'#808080',lbl:'2\u00d7 Oil'},
{from:'iron',to:'factory',c:'#b4784f',lbl:'1\u00d7 Iron'},
{from:'rm',to:'factory',c:'#c8b464',lbl:'2\u00d7 RM'},
{from:'refinery',to:'fuel',c:'#50a050',lbl:'\u2192 3\u00d7 Fuel'},
{from:'factory',to:'elec',c:'#5090d0',lbl:'\u2192 2\u00d7 Elec'},
{from:'fuel',to:'gen',c:'#ffa030',lbl:'burns Fuel'},
{from:'he3',to:'fusion',c:'#64dcff',lbl:'burns He-3'},
{from:'gen',to:'power',c:'#ffcc00',lbl:'50 MW'},
{from:'fusion',to:'power',c:'#ffcc00',lbl:'200 MW'},
{from:'power',to:'happy',c:'#50c878'},
{from:'elec',to:'tech',c:'#a0a0ff',lbl:'+3%/lvl'},
{from:'happy',to:'growth',c:'#7fdbca'},
];
const nMap={};nodes.forEach(n=>nMap[n.id]=n);
// Particles
let particles=[];
function spawnParticles(){
edges.forEach(e=>{
const pr=window._flowProd||{},co=window._flowCons||{};
const a=nMap[e.from],b=nMap[e.to];
// Flow rate determines particle density
let rate=3;
if(b&&b.res){rate=Math.max(1,(pr[b.res]||0))}
else if(a&&a.res){rate=Math.max(1,(pr[a.res]||0))}
const count=Math.min(4,Math.max(1,Math.round(rate/6)));
for(let i=0;i<count;i++){
particles.push({e,t:Math.random(),speed:0.0006+Math.random()*0.001})}})}
spawnParticles();setInterval(()=>{particles=[];spawnParticles()},15000);
function drawFlow(){
const w=W(),h=H();
fx.clearRect(0,0,w,h);
// Title
fx.fillStyle='#334';fx.font='9px monospace';fx.textAlign='left';
fx.fillText('PRODUCTION CHAINS',8,14);
// Edges (draw first, behind nodes)
edges.forEach(e=>{
const a=nMap[e.from],b=nMap[e.to];if(!a||!b)return;
const x1=a.x*w,y1=a.y*h,x2=b.x*w,y2=b.y*h;
fx.strokeStyle=e.c+'18';fx.lineWidth=1;
fx.beginPath();fx.moveTo(x1,y1);fx.lineTo(x2,y2);fx.stroke();
// Small arrow
const ang=Math.atan2(y2-y1,x2-x1),d=5;
const ax=x2-d*2*Math.cos(ang),ay=y2-d*2*Math.sin(ang);
fx.fillStyle=e.c+'30';fx.beginPath();
fx.moveTo(ax,ay);fx.lineTo(ax-d*Math.cos(ang-0.4),ay-d*Math.sin(ang-0.4));
fx.lineTo(ax-d*Math.cos(ang+0.4),ay-d*Math.sin(ang+0.4));fx.fill();
// Edge label
if(e.lbl){
const mx=(x1+x2)/2,my=(y1+y2)/2;
fx.fillStyle='#3a3a50';fx.font='7px monospace';fx.textAlign='center';
fx.fillText(e.lbl,mx,my-4)}});
// Particles — subtle glowing dots
particles.forEach(p=>{
p.t+=p.speed;if(p.t>1)p.t-=1;
const a=nMap[p.e.from],b=nMap[p.e.to];if(!a||!b)return;
const px=a.x*w+(b.x-a.x)*w*p.t,py=a.y*h+(b.y-a.y)*h*p.t;
const alpha=p.t<0.08?p.t/0.08:p.t>0.92?(1-p.t)/0.08:1;
// Soft glow
fx.globalAlpha=alpha*0.08;fx.fillStyle=p.e.c;
fx.beginPath();fx.arc(px,py,4,0,Math.PI*2);fx.fill();
// Core dot
fx.globalAlpha=alpha*0.35;
fx.beginPath();fx.arc(px,py,1.2,0,Math.PI*2);fx.fill();
fx.globalAlpha=1});
// Nodes — rectangular boxes
nodes.forEach(n=>{
const nx=n.x*w,ny=n.y*h;
const bw=n.kind?BW+8:BW,bh=BH;
// Box background — subtle rounded rect
fx.fillStyle=n.c+'10';
fx.beginPath();fx.roundRect(nx-bw/2,ny-bh/2,bw,bh,5);fx.fill();
fx.strokeStyle=n.c+'25';fx.lineWidth=0.5;
fx.beginPath();fx.roundRect(nx-bw/2,ny-bh/2,bw,bh,5);fx.stroke();
// Label
fx.fillStyle=n.kind?n.c+'aa':'#99a';fx.font='10px monospace';
fx.textAlign='center';fx.textBaseline='middle';
fx.fillText(n.label,nx,ny);
fx.textBaseline='alphabetic';
// Flow rate for resource nodes — small, below box
if(n.res){
const pr=window._flowProd||{},co=window._flowCons||{};
const v=(pr[n.res]||0)-(co[n.res]||0);
const fc2=v>0?'#4a9a4a':v<-1?'#9a4a4a':'#3a3a4a';
fx.fillStyle=fc2;fx.font='8px monospace';fx.textAlign='center';
fx.fillText((v>0?'+':'')+v.toFixed(0)+'/s',nx,ny+bh/2+10)}});
requestAnimationFrame(drawFlow)}
drawFlow()})();
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
