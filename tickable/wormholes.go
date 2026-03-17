package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&WormholeSystem{
		BaseSystem: NewBaseSystem("Wormholes", 38),
	})
}

// WormholeSystem manages temporary wormholes that connect distant systems.
// Wormholes appear randomly, last for a limited time, and create shortcut
// trade routes across the galaxy.
//
// A wormhole acts like a temporary hyperlane — ships can jump through it.
// This creates exciting trading opportunities: a wormhole to a distant
// system with cheap Oil suddenly makes cross-galaxy arbitrage possible.
//
// Wormholes spawn every ~10,000 ticks (~17 min) and last ~5,000 ticks (~8 min).
type WormholeSystem struct {
	*BaseSystem
	wormholes []*Wormhole
	nextSpawn int64
}

// Wormhole is a temporary connection between two systems.
type Wormhole struct {
	ID        int
	SystemA   int
	SystemB   int
	TicksLeft int
	Active    bool
}

func (ws *WormholeSystem) OnTick(tick int64) {
	if tick%100 != 0 {
		return
	}

	ctx := ws.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	systems := game.GetSystems()

	// Initialize spawn timer
	if ws.nextSpawn == 0 {
		ws.nextSpawn = tick + 5000 + int64(rand.Intn(5000))
	}

	// Decay existing wormholes
	for _, wh := range ws.wormholes {
		if !wh.Active {
			continue
		}
		wh.TicksLeft -= 100
		if wh.TicksLeft <= 0 {
			wh.Active = false
			game.LogEvent("event", "",
				fmt.Sprintf("🌀 Wormhole between %s and %s has collapsed!",
					systems[wh.SystemA%len(systems)].Name,
					systems[wh.SystemB%len(systems)].Name))
		}
	}

	// Spawn new wormhole
	if tick >= ws.nextSpawn && len(systems) > 5 {
		ws.nextSpawn = tick + 8000 + int64(rand.Intn(5000))

		// Pick two distant systems
		a := rand.Intn(len(systems))
		b := rand.Intn(len(systems))
		for b == a || abs(a-b) < 5 {
			b = rand.Intn(len(systems))
		}

		wh := &Wormhole{
			ID:        len(ws.wormholes) + 1,
			SystemA:   systems[a].ID,
			SystemB:   systems[b].ID,
			TicksLeft: 4000 + rand.Intn(3000), // 4000-7000 ticks (~7-12 min)
			Active:    true,
		}
		ws.wormholes = append(ws.wormholes, wh)

		game.LogEvent("event", "",
			fmt.Sprintf("🌀 WORMHOLE OPENED between %s (SYS-%d) and %s (SYS-%d)! Lasts ~%d minutes. Ships can jump through!",
				systems[a].Name, systems[a].ID+1,
				systems[b].Name, systems[b].ID+1,
				wh.TicksLeft/600))

		// Auto-move ships through if they're at one end
		// (Handled by the ship movement system checking wormholes as valid routes)
	}
}

// GetActiveWormholes returns currently open wormholes.
func (ws *WormholeSystem) GetActiveWormholes() []*Wormhole {
	var result []*Wormhole
	for _, wh := range ws.wormholes {
		if wh.Active {
			result = append(result, wh)
		}
	}
	return result
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
