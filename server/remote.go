//go:build !js

package server

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/utils"
)

// RemoteSync periodically fetches state from a remote server
// and updates the local GameServer to mirror it.
type RemoteSync struct {
	serverURL  string
	apiKey     string
	playerName string
	gs         *GameServer
	stopCh     chan struct{}
}

// NewRemoteSync creates a sync client for the remote server.
func NewRemoteSync(gs *GameServer, serverURL, apiKey string) *RemoteSync {
	return &RemoteSync{
		serverURL: strings.TrimSuffix(serverURL, "/"),
		apiKey:    apiKey,
		gs:        gs,
		stopCh:    make(chan struct{}),
	}
}

// Start begins periodic syncing in a goroutine.
func (rs *RemoteSync) Start() {
	go func() {
		rs.syncAll()
		fastTicker := time.NewTicker(2 * time.Second)
		slowTicker := time.NewTicker(10 * time.Second)
		defer fastTicker.Stop()
		defer slowTicker.Stop()
		for {
			select {
			case <-rs.stopCh:
				return
			case <-fastTicker.C:
				rs.syncPlayer()
			case <-slowTicker.C:
				rs.syncEconomy()
				rs.SyncOwnership()
				rs.syncFactions()
				rs.syncShips()
			}
		}
	}()
}

func (rs *RemoteSync) Stop() {
	close(rs.stopCh)
}

// syncAll fetches everything from the remote server.
func (rs *RemoteSync) syncAll() {
	rs.syncFactions()
	rs.syncPlayer()
	rs.syncEconomy()
	rs.SyncOwnership()
	rs.syncShips()
}

// syncEconomy updates market prices from the remote server.
func (rs *RemoteSync) syncEconomy() {
	data, err := rs.apiGet("/api/economy")
	if err != nil {
		return
	}
	var resp struct {
		OK   bool `json:"ok"`
		Data struct {
			Resources map[string]struct {
				BuyPrice  float64 `json:"buy_price"`
				SellPrice float64 `json:"sell_price"`
			} `json:"resources"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return
	}
	// Update local market with remote prices
	if rs.gs.State.Market != nil {
		for name, r := range resp.Data.Resources {
			rs.gs.State.Market.SetPrice(name, r.BuyPrice, r.SellPrice)
		}
	}
}

// syncPlayer updates the human player's credits and storage from remote.
func (rs *RemoteSync) syncPlayer() {
	data, err := rs.apiGet("/api/player/me")
	if err != nil {
		return
	}
	var resp struct {
		OK   bool `json:"ok"`
		Data struct {
			Credits int `json:"credits"`
			Planets []struct {
				ID              int            `json:"id"`
				StoredResources map[string]int `json:"stored_resources"`
				Population      int64          `json:"population"`
			} `json:"planets"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return
	}

	hp := rs.gs.State.HumanPlayer
	if hp == nil {
		return
	}

	hp.Credits = resp.Data.Credits

	// Sync each planet's storage by matching IDs
	for _, rp := range resp.Data.Planets {
		for _, lp := range hp.OwnedPlanets {
			if lp == nil || lp.GetID() != rp.ID {
				continue
			}
			lp.Population = rp.Population
			for resType, amount := range rp.StoredResources {
				if s, ok := lp.StoredResources[resType]; ok && s != nil {
					s.Amount = amount
				}
			}
			break
		}
	}
}

// ForwardTrade sends a trade to the remote server instead of local.
func (rs *RemoteSync) ForwardTrade(resource string, quantity int, buy bool) ([]byte, error) {
	action := "sell"
	if buy {
		action = "buy"
	}
	body := fmt.Sprintf(`{"resource":"%s","quantity":%d,"action":"%s"}`, resource, quantity, action)
	return rs.apiPost("/api/market/trade", body)
}

// ForwardBuild sends a build command to the remote server.
func (rs *RemoteSync) ForwardBuild(planetID int, buildingType string, resourceID int) ([]byte, error) {
	body := fmt.Sprintf(`{"planet_id":%d,"building_type":"%s","resource_id":%d}`, planetID, buildingType, resourceID)
	return rs.apiPost("/api/build", body)
}

// Register creates an account on the remote server.
func (rs *RemoteSync) Register(name, password string) (string, error) {
	body := fmt.Sprintf(`{"name":"%s","password":"%s"}`, name, password)
	data, err := rs.apiPost("/api/register", body)
	if err != nil {
		return "", err
	}
	var resp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		Data  struct {
			APIKey string `json:"api_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}
	if !resp.OK {
		return "", fmt.Errorf("%s", resp.Error)
	}
	rs.apiKey = resp.Data.APIKey
	rs.playerName = name
	return resp.Data.APIKey, nil
}

// Login authenticates with the remote server.
func (rs *RemoteSync) Login(name, password string) (string, error) {
	body := fmt.Sprintf(`{"name":"%s","password":"%s"}`, name, password)
	data, err := rs.apiPost("/api/login", body)
	if err != nil {
		return "", err
	}
	var resp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		Data  struct {
			APIKey string `json:"api_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}
	if !resp.OK {
		return "", fmt.Errorf("%s", resp.Error)
	}
	rs.apiKey = resp.Data.APIKey
	rs.playerName = name
	return resp.Data.APIKey, nil
}

// FetchSeed gets the galaxy seed from the remote server.
func (rs *RemoteSync) FetchSeed() (int64, error) {
	data, err := rs.apiGet("/api/game")
	if err != nil {
		return 0, err
	}
	var resp struct {
		OK   bool `json:"ok"`
		Data struct {
			Seed int64 `json:"seed"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return 0, fmt.Errorf("failed to fetch game info")
	}
	return resp.Data.Seed, nil
}

// SyncOwnership updates planet ownership from the remote galaxy and links
// planets to their local Player objects.
func (rs *RemoteSync) SyncOwnership() {
	data, err := rs.apiGet("/api/galaxy")
	if err != nil {
		return
	}
	var resp struct {
		OK   bool `json:"ok"`
		Data []struct {
			ID    int    `json:"id"`
			Owner string `json:"owner"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return
	}

	// Build owner map from remote
	owners := make(map[int]string)
	for _, sys := range resp.Data {
		if sys.Owner != "" {
			owners[sys.ID] = sys.Owner
		}
	}

	// Build player name → Player lookup
	playerByName := make(map[string]*entities.Player)
	for _, p := range rs.gs.State.Players {
		playerByName[p.Name] = p
	}

	// Update local planet ownership and link to Player objects
	for _, sys := range rs.gs.State.Systems {
		remoteOwner := owners[sys.ID]
		for _, e := range sys.Entities {
			p, ok := e.(*entities.Planet)
			if !ok {
				continue
			}
			if remoteOwner == "" {
				continue
			}
			oldOwner := p.Owner
			p.Owner = remoteOwner

			// Link planet to player if ownership changed or is new
			if oldOwner != remoteOwner {
				if player, ok := playerByName[remoteOwner]; ok {
					// Check if planet is already in player's list
					alreadyOwned := false
					for _, owned := range player.OwnedPlanets {
						if owned == p {
							alreadyOwned = true
							break
						}
					}
					if !alreadyOwned {
						player.AddOwnedPlanet(p)
					}
				}
			}
		}
	}
}

// syncFactions fetches the player list from the remote server and creates local
// Player objects for AI factions so the UI can display their names and colors.
func (rs *RemoteSync) syncFactions() {
	data, err := rs.apiGet("/api/players")
	if err != nil {
		return
	}
	var resp struct {
		OK   bool `json:"ok"`
		Data []struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Type    string `json:"type"`
			Credits int    `json:"credits"`
			Planets int    `json:"planets"`
			Ships   int    `json:"ships"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return
	}

	aiColors := utils.GetAIPlayerColors()
	colorIdx := 0

	for _, rp := range resp.Data {
		// Skip self
		if rp.Name == rs.playerName {
			continue
		}

		// Check if we already have this player locally
		found := false
		for _, lp := range rs.gs.State.Players {
			if lp.Name == rp.Name {
				lp.Credits = rp.Credits
				found = true
				break
			}
		}
		if found {
			continue
		}

		// Create a new local player for this remote faction
		pType := entities.PlayerTypeAI
		if rp.Type == "human" {
			pType = entities.PlayerTypeHuman
		}
		var pColor color.RGBA
		if pType == entities.PlayerTypeAI {
			pColor = aiColors[colorIdx%len(aiColors)]
			colorIdx++
		} else {
			pColor = utils.PlayerBlue
		}
		newPlayer := entities.NewPlayer(rp.ID, rp.Name, pColor, pType)
		newPlayer.Credits = rp.Credits
		rs.gs.State.Players = append(rs.gs.State.Players, newPlayer)
		fmt.Printf("[Sync] Discovered faction: %s (%s)\n", rp.Name, rp.Type)
	}
}

// remoteShipInfo holds ship data from the remote API.
type remoteShipInfo struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	Owner          string  `json:"owner"`
	Status         string  `json:"status"`
	SystemID       int     `json:"system_id"`
	TargetSystem   int     `json:"target_system"`
	FuelCurrent    int     `json:"fuel_current"`
	FuelMax        int     `json:"fuel_max"`
	TravelProgress float64 `json:"travel_progress"`
}

// syncShips fetches ship data from the remote server and updates local state.
// This lets the remote client see other players' ships moving around the galaxy.
func (rs *RemoteSync) syncShips() {
	data, err := rs.apiGet("/api/ships")
	if err != nil {
		return
	}
	var resp struct {
		OK   bool             `json:"ok"`
		Data []remoteShipInfo `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return
	}

	// Build remote ship lookup by ID
	remoteByID := make(map[int]remoteShipInfo, len(resp.Data))
	for _, s := range resp.Data {
		remoteByID[s.ID] = s
	}

	// Update existing local ships (owned by self)
	if rs.gs.State.HumanPlayer != nil {
		for _, ship := range rs.gs.State.HumanPlayer.OwnedShips {
			if rship, ok := remoteByID[ship.GetID()]; ok {
				ship.CurrentSystem = rship.SystemID
				ship.TargetSystem = rship.TargetSystem
				ship.CurrentFuel = rship.FuelCurrent
				ship.TravelProgress = rship.TravelProgress
				ship.Status = entities.ShipStatus(rship.Status)
			}
		}
	}

	// For other players, update their ships from remote data
	for _, player := range rs.gs.State.Players {
		if player == rs.gs.State.HumanPlayer {
			continue
		}

		// Create missing ships for this player
		existingIDs := make(map[int]bool)
		for _, ship := range player.OwnedShips {
			existingIDs[ship.GetID()] = true
		}

		for _, rship := range resp.Data {
			if rship.Owner != player.Name {
				continue
			}
			if existingIDs[rship.ID] {
				// Update existing ship
				for _, ship := range player.OwnedShips {
					if ship.GetID() == rship.ID {
						ship.CurrentSystem = rship.SystemID
						ship.TargetSystem = rship.TargetSystem
						ship.CurrentFuel = rship.FuelCurrent
						ship.TravelProgress = rship.TravelProgress
						ship.Status = entities.ShipStatus(rship.Status)
						break
					}
				}
			} else {
				// Create new local ship for this remote ship
				ship := entities.NewShip(rship.ID, rship.Name, entities.ShipType(rship.Type), rship.SystemID, player.Name, player.Color)
				ship.CurrentFuel = rship.FuelCurrent
				ship.MaxFuel = rship.FuelMax
				ship.TravelProgress = rship.TravelProgress
				ship.Status = entities.ShipStatus(rship.Status)
				ship.TargetSystem = rship.TargetSystem
				player.AddOwnedShip(ship)
			}
		}
	}
}

func (rs *RemoteSync) apiGet(endpoint string) ([]byte, error) {
	req, err := http.NewRequest("GET", rs.serverURL+endpoint, nil)
	if err != nil {
		return nil, err
	}
	if rs.apiKey != "" {
		req.Header.Set("X-API-Key", rs.apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (rs *RemoteSync) apiPost(endpoint string, body string) ([]byte, error) {
	req, err := http.NewRequest("POST", rs.serverURL+endpoint, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if rs.apiKey != "" {
		req.Header.Set("X-API-Key", rs.apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
