package core

import (
	"sync"
	"time"

	"github.com/hunterjsb/xandaris/tickable"
)

// CachedConstructionItem is a UI-friendly snapshot of a construction queue entry.
type CachedConstructionItem struct {
	ID             string
	Name           string
	Location       string
	Owner          string
	Progress       int
	RemainingTicks int
}

// constructionCacheMu provides thread-safe cached construction data for remote mode.
type constructionCacheMu struct {
	mu        sync.RWMutex
	items     []CachedConstructionItem
	lastFetch time.Time
	fetching  bool
}

// getConstructionItems returns cached construction items for a player.
// Works in both local and remote mode.
func (a *App) getConstructionItems(playerName string) []CachedConstructionItem {
	if !a.IsRemote() {
		return a.getLocalConstructionItems(playerName)
	}
	return a.getRemoteConstructionItems(playerName)
}

func (a *App) getLocalConstructionItems(playerName string) []CachedConstructionItem {
	cs := tickable.GetConstructionSystem()
	if cs == nil {
		return nil
	}
	items := cs.GetConstructionsByOwner(playerName)
	result := make([]CachedConstructionItem, 0, len(items))
	for _, item := range items {
		item.Mutex.RLock()
		result = append(result, CachedConstructionItem{
			ID:             item.ID,
			Name:           item.Name,
			Location:       item.Location,
			Owner:          item.Owner,
			Progress:       item.Progress,
			RemainingTicks: item.RemainingTicks,
		})
		item.Mutex.RUnlock()
	}
	return result
}
