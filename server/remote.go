//go:build !js

package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-rs.stopCh:
				return
			case <-ticker.C:
				rs.syncPlayer()
			}
		}
	}()
}

func (rs *RemoteSync) Stop() {
	close(rs.stopCh)
}

// syncAll fetches everything from the remote server.
func (rs *RemoteSync) syncAll() {
	rs.syncPlayer()
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
				StoredResources map[string]int `json:"stored_resources"`
				Population      int64          `json:"population"`
			} `json:"planets"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return
	}

	if rs.gs.State.HumanPlayer != nil {
		rs.gs.State.HumanPlayer.Credits = resp.Data.Credits
		// Sync planet storage
		if len(resp.Data.Planets) > 0 && len(rs.gs.State.HumanPlayer.OwnedPlanets) > 0 {
			rp := resp.Data.Planets[0]
			lp := rs.gs.State.HumanPlayer.OwnedPlanets[0]
			if lp != nil {
				lp.Population = rp.Population
				for resType, amount := range rp.StoredResources {
					if s, ok := lp.StoredResources[resType]; ok && s != nil {
						s.Amount = amount
					}
				}
			}
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
