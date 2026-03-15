//go:build !js

package api

import (
	"math"
	"sort"

	"github.com/hunterjsb/xandaris/entities"
)

// ExpansionTarget represents a colonization candidate.
type ExpansionTarget struct {
	SystemID     int      `json:"system_id"`
	SystemName   string   `json:"system_name"`
	PlanetID     int      `json:"planet_id"`
	PlanetName   string   `json:"planet_name"`
	PlanetType   string   `json:"planet_type"`
	Habitability int      `json:"habitability"`
	Resources    []string `json:"resources"`
	Distance     int      `json:"distance"`    // hyperlane jumps from nearest owned system
	Score        int      `json:"score"`        // higher = better target
}

// handleGetExpansionTargets finds the best colonization candidates for a player.
func handleGetExpansionTargets(p GameStateProvider, playerName string) interface{} {
	player := findPlayer(p, playerName)
	if player == nil {
		return []ExpansionTarget{}
	}

	systems := p.GetSystems()
	hyperlanes := p.GetHyperlanes()

	// Build adjacency map
	adj := make(map[int][]int)
	for _, hl := range hyperlanes {
		adj[hl.From] = append(adj[hl.From], hl.To)
		adj[hl.To] = append(adj[hl.To], hl.From)
	}

	// Find owned system IDs
	ownedSystems := make(map[int]bool)
	for _, planet := range player.OwnedPlanets {
		if planet == nil {
			continue
		}
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if pl, ok := e.(*entities.Planet); ok && pl.GetID() == planet.GetID() {
					ownedSystems[sys.ID] = true
				}
			}
		}
	}

	// BFS from owned systems to find distances
	dist := make(map[int]int)
	queue := make([]int, 0)
	for sysID := range ownedSystems {
		dist[sysID] = 0
		queue = append(queue, sysID)
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, next := range adj[cur] {
			if _, visited := dist[next]; !visited {
				dist[next] = dist[cur] + 1
				queue = append(queue, next)
			}
		}
	}

	// Find unclaimed habitable planets
	var targets []ExpansionTarget
	for _, sys := range systems {
		d, ok := dist[sys.ID]
		if !ok {
			continue
		}
		if d == 0 {
			continue // skip owned systems
		}

		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner != "" || !planet.IsHabitable() {
				continue
			}

			// Collect resource types
			var resources []string
			seen := make(map[string]bool)
			for _, re := range planet.Resources {
				if r, ok := re.(*entities.Resource); ok && !seen[r.ResourceType] {
					resources = append(resources, r.ResourceType)
					seen[r.ResourceType] = true
				}
			}

			// Score: habitability + resource diversity - distance penalty
			score := planet.Habitability + len(resources)*15 - d*10
			if score < 0 {
				score = 0
			}

			targets = append(targets, ExpansionTarget{
				SystemID:     sys.ID,
				SystemName:   sys.Name,
				PlanetID:     planet.GetID(),
				PlanetName:   planet.Name,
				PlanetType:   planet.PlanetType,
				Habitability: planet.Habitability,
				Resources:    resources,
				Distance:     d,
				Score:        score,
			})
		}
	}

	// Sort by score descending
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Score > targets[j].Score
	})

	// Return top 10
	if len(targets) > 10 {
		targets = targets[:10]
	}

	return targets
}

// distBetweenSystems computes euclidean distance between two systems.
func distBetweenSystems(a, b *entities.System) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Sqrt(dx*dx + dy*dy)
}
