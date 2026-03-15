package server

import (
	"fmt"
	"testing"

	"github.com/hunterjsb/xandaris/tickable"

	// Side-effect imports: register entity generators and tickable systems
	_ "github.com/hunterjsb/xandaris/entities/building"
	_ "github.com/hunterjsb/xandaris/entities/planet"
	_ "github.com/hunterjsb/xandaris/entities/resource"
	_ "github.com/hunterjsb/xandaris/entities/star"
	_ "github.com/hunterjsb/xandaris/entities/station"
)

// newTestServer creates a fully initialized game server for benchmarking.
func newTestServer(playerName string) *GameServer {
	gs := New(1280, 720)
	if err := gs.NewGame(playerName); err != nil {
		panic(fmt.Sprintf("failed to create test game: %v", err))
	}
	return gs
}

// BenchmarkTickLoop measures the cost of processing game ticks.
// Run with: go test -bench=BenchmarkTickLoop -cpuprofile=cpu.prof -memprofile=mem.prof ./server/
func BenchmarkTickLoop(b *testing.B) {
	gs := newTestServer("BenchPlayer")

	// Fast-forward to tick 500 so AI factions are active and economy is running
	for i := 0; i < 500; i++ {
		gs.DrainCommands()
		tickable.UpdateAllSystemsSequential(int64(i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		gs.DrainCommands()
		tickable.UpdateAllSystemsSequential(int64(500 + i))
	}
}

// BenchmarkGetSystemsMap measures the cached vs uncached map lookup.
func BenchmarkGetSystemsMap(b *testing.B) {
	gs := newTestServer("BenchPlayer")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = gs.State.GetSystemsMap()
	}
}

// BenchmarkNewGame measures full game initialization time.
func BenchmarkNewGame(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		gs := New(1280, 720)
		gs.NewGame(fmt.Sprintf("Bench%d", i))
	}
}
