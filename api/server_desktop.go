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
			p.GetCommandChannel() <- game.GameCommand{
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
			p.GetCommandChannel() <- game.GameCommand{
				Type:   "cancel_order",
				Data:   game.CancelOrderCommandData{OrderID: req.OrderID},
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
		default:
			writeErr(w, http.StatusMethodNotAllowed, "GET, POST, or DELETE")
		}
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
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "workforce_assign",
			Data:   game.WorkforceAssignCommandData{PlanetID: req.PlanetID, BuildingIndex: req.BuildingIndex, Workers: req.Workers},
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
			p.GetCommandChannel() <- game.GameCommand{
				Type:   "register_player",
				Data:   game.RegisterPlayerCommandData{Name: account.Name, AccountKey: account.APIKey},
				Result: resultCh,
			}
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
let W,H,systems=[],ships=[],players=[],economy={},flows={},mx=0,my=0,hover=null,selected=null,detail=null,tracked=null,hoverShip=null,t=0;
const COLORS={Human:'#4caf50','Orion Exchange':'#ff9800','Lyra Cartel':'#e84040','Helios Commodities':'#8bc34a','Ceres Brokers':'#ffca28','Nova Frontier Co.':'#ab47bc',Server:'#4caf50'};
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
const oz=zoom;zoom=Math.max(0.5,Math.min(3,zoom*(e.deltaY>0?0.9:1.1)));
// Zoom toward cursor
panX+=(mx-W/2)*(1-zoom/oz);panY+=(my-H/2)*(1-zoom/oz);
e.preventDefault()},{passive:false});
C.addEventListener('mousedown',e=>{if(e.button===0&&!hover){dragging=true;dragX=e.clientX-panX;dragY=e.clientY-panY}});
C.addEventListener('mouseup',()=>{dragging=false});
C.addEventListener('mousemove',e=>{if(dragging){panX=e.clientX-dragX;panY=e.clientY-dragY}});
function pc(name){return COLORS[name]||'#6688aa'}
function hexA(hex,a){const r=parseInt(hex.slice(1,3),16),g=parseInt(hex.slice(3,5),16),b=parseInt(hex.slice(5,7),16);return'rgba('+r+','+g+','+b+','+a+')'}
async function load(){
try{
const[g,s,p,e,f,ev]=await Promise.all(['/api/galaxy','/api/ships','/api/players','/api/economy','/api/flows','/api/events?limit=15'].map(u=>fetch(B+u).then(r=>r.json())));
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
// Events
const evts=ev.data||[];
document.getElementById('events').innerHTML=evts.map(x=>{
const c=x.type==='trade'?'#889':x.type==='colonize'?'#7fdbca':x.type==='build'?'#6c6':'#889';
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
// Trade routes + moving ships
hoverShip=null;
const shipPositions=[];
ships.filter(s=>s.status==='Moving').forEach(s=>{
const src=systems.find(x=>x.id===s.system_id),tgt=systems.find(x=>x.id===s.target_system);
if(!src||!tgt)return;
const[x1,y1]=sp(src),[x2,y2]=sp(tgt),c=pc(s.owner);
// Route line (only for laden ships)
if(s.cargo_used>0){
X.strokeStyle=hexA(c,0.15);X.lineWidth=4;X.beginPath();X.moveTo(x1,y1);X.lineTo(x2,y2);X.stroke();
X.strokeStyle=hexA(c,0.4);X.lineWidth=1;X.setLineDash([8,8]);
X.beginPath();X.moveTo(x1,y1);X.lineTo(x2,y2);X.stroke();X.setLineDash([])}
// Ship position from actual travel progress
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
const pl=players.find(p=>p.name===owner);
X.fillStyle=hexA(c,0.8);X.font='9px monospace';
X.fillText((owner.length>10?owner.slice(0,8)+'..':owner)+' '+(pl?pl.stock:''),sx,sy+nP*5+26)}
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
body{background:#0a0c14;color:#c0c8d8;font-family:'Courier New',monospace;padding:20px}
h1{color:#7fdbca;margin-bottom:4px}
.sub{color:#556;margin-bottom:20px;font-size:14px}
.grid{display:grid;grid-template-columns:1fr 1fr 1fr;gap:12px;max-width:1400px}
.wide{grid-column:span 2}
.panel{background:#12162a;border:1px solid #1e2844;border-radius:6px;padding:14px}
.panel h2{color:#7fdbca;font-size:13px;margin-bottom:10px;border-bottom:1px solid #1e2844;padding-bottom:5px}
table{width:100%;border-collapse:collapse;font-size:12px}
th{text-align:left;color:#667;padding:3px 6px;font-weight:normal}
td{padding:3px 6px}
.g{color:#6dcc6d}.r{color:#c55}.o{color:#cca444}.d{color:#556}.b{color:#6688cc}.p{color:#b88fdf}
.bar-bg{background:#1a1e30;border-radius:3px;overflow:hidden;height:8px;margin:2px 0}
.bar-fill{height:100%;border-radius:3px;transition:width 0.5s}
.spark{font-size:11px;letter-spacing:-1px}
.event{font-size:11px;padding:2px 0;border-bottom:1px solid #111525}
.rank{color:#7fdbca;font-weight:bold}
.st{position:fixed;bottom:8px;right:12px;font-size:11px;color:#4a4}
.stats-row{display:flex;gap:20px;margin-bottom:16px;flex-wrap:wrap}
.stat{text-align:center}
.stat .val{font-size:22px;color:#7fdbca;font-weight:bold}
.stat .lbl{font-size:10px;color:#556}
.row{display:flex;justify-content:space-between;padding:3px 0;border-bottom:1px solid #111525;font-size:12px}
@media(max-width:900px){.grid{grid-template-columns:1fr}.wide{grid-column:span 1}}
</style></head><body>
<h1>XANDARIS II</h1>
<p class="sub">Live Economy Dashboard &bull; Auto-refreshes every 3s</p>
<div class="stats-row" id="top"></div>
<div class="grid">
<div class="panel"><h2>Leaderboard</h2><div id="lb"></div></div>
<div class="panel"><h2>Power Grid</h2><div id="pw"></div></div>
<div class="panel"><h2>Events</h2><div id="ev" style="max-height:220px;overflow-y:auto"></div></div>
<div class="panel wide"><h2>Market Prices + Trends</h2>
<table><thead><tr><th>Resource</th><th>Buy</th><th>Sell</th><th>Base</th><th>Ratio</th><th>Supply</th><th>Scarcity</th><th>Trend</th></tr></thead><tbody id="m"></tbody></table></div>
<div class="panel"><h2>Galaxy Flows</h2>
<table><thead><tr><th>Resource</th><th>Prod</th><th>Cons</th><th>Net</th></tr></thead><tbody id="f"></tbody></table></div>
</div>
<div id="s" class="st">Loading...</div>
<script>
const B=location.origin,blocks='\u2581\u2582\u2583\u2584\u2585\u2586\u2587\u2588';
function spark(h){if(!h||h.length<3)return'';const mn=Math.min(...h),mx=Math.max(...h),rng=mx-mn||1;return'<span class="spark">'+h.slice(-20).map(v=>blocks[Math.min(7,Math.max(0,Math.round((v-mn)/rng*7)))]).join('')+'</span>'}
async function R(){try{
const[e,p,f,g,lb,pw,ev]=await Promise.all(['/api/economy','/api/players','/api/flows','/api/game','/api/leaderboard','/api/power','/api/events?limit=12'].map(u=>fetch(B+u).then(r=>r.json())));
const d=g.data;
document.getElementById('top').innerHTML=[
['Tick',d.tick],['Time',d.game_time],['Speed',d.speed+(d.paused?' \u23F8':'')],
['Population',e.data.total_population.toLocaleString()],['Credits',e.data.total_credits.toLocaleString()],
['Trade Vol',e.data.trade_volume.toFixed(0)],['Players',d.players],['Systems',d.systems]
].map(([l,v])=>'<div class="stat"><div class="val">'+v+'</div><div class="lbl">'+l+'</div></div>').join('');
document.getElementById('m').innerHTML=Object.entries(e.data.resources).sort().map(([n,r])=>{
const c=r.price_ratio>1.5?'r':r.price_ratio>0.8?'':'g';
const s=r.scarcity=='Scarce'||r.scarcity=='Critical'?'o':r.scarcity=='Depleted'?'r':'d';
return'<tr><td>'+n+'</td><td class="'+c+'">'+r.buy_price.toFixed(0)+'</td><td class="'+c+'">'+r.sell_price.toFixed(0)+'</td><td class="d">'+r.base_price+'</td><td class="'+c+'">'+r.price_ratio.toFixed(2)+'x</td><td>'+r.total_supply+'</td><td class="'+s+'">'+r.scarcity+'</td><td>'+spark(r.price_history)+'</td></tr>'}).join('');
document.getElementById('lb').innerHTML=(lb.data||[]).map(x=>{
const c=x.type=='human'?'b':'';
return'<div class="row"><span><span class="rank">#'+x.rank+'</span> <span class="'+c+'">'+x.name+'</span></span><span class="d">'+x.score.toLocaleString()+' pts</span></div>'}).join('');
document.getElementById('pw').innerHTML=(pw.data||[]).map(x=>{
const pct=x.consumed_mw>0?Math.min(1,x.generated_mw/x.consumed_mw):1;
const c=pct<0.5?'#c55':pct<0.8?'#cca444':'#6dcc6d';
return'<div style="margin-bottom:6px"><div style="display:flex;justify-content:space-between;font-size:11px"><span>'+x.owner+'</span><span class="d">'+x.generated_mw.toFixed(0)+'/'+x.consumed_mw.toFixed(0)+' MW</span></div><div class="bar-bg"><div class="bar-fill" style="width:'+(pct*100).toFixed(0)+'%;background:'+c+'"></div></div></div>'}).join('');
document.getElementById('ev').innerHTML=(ev.data||[]).map(x=>{
const c=x.type=='trade'?'g':x.type=='build'?'b':x.type=='alert'?'r':x.type=='event'?'o':x.type=='join'?'p':'d';
return'<div class="event"><span class="d">['+x.time+']</span> <span class="'+c+'">'+x.message+'</span></div>'}).join('');
const a=new Set([...Object.keys(f.data.production),...Object.keys(f.data.consumption)]);
document.getElementById('f').innerHTML=[...a].sort().map(r=>{
const pr=(f.data.production[r]||0).toFixed(1),co=(f.data.consumption[r]||0).toFixed(1),n=(f.data.net_flow[r]||0).toFixed(1);
return'<tr><td>'+r+'</td><td class="g">+'+pr+'</td><td class="r">-'+co+'</td><td class="'+(n>0?'g':n<-1?'r':'')+'">'+((n>0?'+':'')+n)+'</td></tr>'}).join('');
document.getElementById('s').textContent='Live \u2022 '+new Date().toLocaleTimeString();
}catch(err){document.getElementById('s').textContent='Disconnected';document.getElementById('s').style.color='#a44'}}
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
