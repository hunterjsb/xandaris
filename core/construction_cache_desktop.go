//go:build !js

package core

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/hunterjsb/xandaris/api"
)

func (a *App) getRemoteConstructionItems(playerName string) []CachedConstructionItem {
	if a.constructionCacheMu == nil {
		a.constructionCacheMu = &constructionCacheMu{}
	}
	cc := a.constructionCacheMu

	cc.mu.RLock()
	stale := time.Since(cc.lastFetch) > 2*time.Second
	items := cc.items
	fetching := cc.fetching
	cc.mu.RUnlock()

	if stale && !fetching {
		cc.mu.Lock()
		cc.fetching = true
		cc.mu.Unlock()

		go func() {
			defer func() {
				cc.mu.Lock()
				cc.fetching = false
				cc.mu.Unlock()
			}()

			data, err := a.remoteGet("/api/construction")
			if err != nil {
				return
			}

			var resp struct {
				OK   bool                        `json:"ok"`
				Data []api.ConstructionQueueItem `json:"data"`
			}
			if err := json.Unmarshal(data, &resp); err != nil || !resp.OK {
				return
			}

			result := make([]CachedConstructionItem, 0, len(resp.Data))
			for _, item := range resp.Data {
				result = append(result, CachedConstructionItem{
					ID:             item.ID,
					Name:           item.Name,
					Location:       item.Location,
					Progress:       item.Progress,
					RemainingTicks: item.RemainingTicks,
				})
			}

			cc.mu.Lock()
			cc.items = result
			cc.lastFetch = time.Now()
			cc.mu.Unlock()
		}()
	}

	return items
}

func (a *App) remoteGet(endpoint string) ([]byte, error) {
	req, err := http.NewRequest("GET", a.remoteServerURL+endpoint, nil)
	if err != nil {
		return nil, err
	}
	if a.remoteAPIKey != "" {
		req.Header.Set("X-API-Key", a.remoteAPIKey)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
