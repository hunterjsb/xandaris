//go:build !js

package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hunterjsb/xandaris/entities"
)

// RemoteSync periodically fetches state from a remote server
// and updates the local GameServer to mirror it.
type RemoteSync struct {
	serverURL string
	apiKey    string
	gs        *GameServer
	stopCh    chan struct{}
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
		// Initial sync
		rs.syncOnce()

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-rs.stopCh:
				return
			case <-ticker.C:
				rs.syncOnce()
			}
		}
	}()
}

func (rs *RemoteSync) Stop() {
	close(rs.stopCh)
}

func (rs *RemoteSync) syncOnce() {
	// Sync player data
	rs.syncPlayerMe()
}

func (rs *RemoteSync) syncPlayerMe() {
	data, err := rs.apiGet("/api/player/me")
	if err != nil {
		return
	}

	var resp struct {
		OK   bool `json:"ok"`
		Data struct {
			Name    string `json:"name"`
			Credits int    `json:"credits"`
			Ships   []struct {
				ID           int            `json:"id"`
				Name         string         `json:"name"`
				Type         string         `json:"type"`
				Status       string         `json:"status"`
				SystemID     int            `json:"system_id"`
				FuelCurrent  int            `json:"fuel_current"`
				FuelMax      int            `json:"fuel_max"`
				CargoUsed    int            `json:"cargo_used"`
				CargoMax     int            `json:"cargo_max"`
				CargoHold    map[string]int `json:"cargo_hold"`
			} `json:"ships"`
			Planets []struct {
				ID              int            `json:"id"`
				Name            string         `json:"name"`
				Population      int64          `json:"population"`
				StoredResources map[string]int `json:"stored_resources"`
				Buildings       []struct {
					Type          string `json:"type"`
					Level         int    `json:"level"`
					IsOperational bool   `json:"is_operational"`
				} `json:"buildings"`
			} `json:"planets"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return
	}

	// Update local player state
	if rs.gs.State.HumanPlayer != nil {
		rs.gs.State.HumanPlayer.Credits = resp.Data.Credits
	}
}

// SendCommand sends a POST command to the remote server.
func (rs *RemoteSync) SendCommand(endpoint string, body string) ([]byte, error) {
	return rs.apiPost(endpoint, body)
}

// Register creates an account on the remote server.
func (rs *RemoteSync) Register(name, password string) (string, error) {
	body := fmt.Sprintf(`{"name":"%s","password":"%s"}`, name, password)
	data, err := rs.apiPost("/api/register", body)
	if err != nil {
		return "", err
	}
	var resp struct {
		OK   bool   `json:"ok"`
		Error string `json:"error"`
		Data struct {
			APIKey string `json:"api_key"`
			Name   string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}
	if !resp.OK {
		return "", fmt.Errorf(resp.Error)
	}
	rs.apiKey = resp.Data.APIKey
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
		OK   bool   `json:"ok"`
		Error string `json:"error"`
		Data struct {
			APIKey string `json:"api_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}
	if !resp.OK {
		return "", fmt.Errorf(resp.Error)
	}
	rs.apiKey = resp.Data.APIKey
	return resp.Data.APIKey, nil
}

// FetchGalaxy loads the full galaxy state from remote.
func (rs *RemoteSync) FetchGalaxy() error {
	// Fetch galaxy layout
	data, err := rs.apiGet("/api/galaxy")
	if err != nil {
		return err
	}
	var resp struct {
		OK   bool `json:"ok"`
		Data []struct {
			ID       int     `json:"id"`
			Name     string  `json:"name"`
			X        float64 `json:"x"`
			Y        float64 `json:"y"`
			Planets  int     `json:"planets"`
			Owner    string  `json:"owner"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return fmt.Errorf("failed to fetch galaxy")
	}

	fmt.Printf("[Remote] Loaded galaxy: %d systems\n", len(resp.Data))
	_ = entities.System{} // ensure import used
	return nil
}

func (rs *RemoteSync) apiGet(endpoint string) ([]byte, error) {
	resp, err := http.Get(rs.serverURL + endpoint)
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
