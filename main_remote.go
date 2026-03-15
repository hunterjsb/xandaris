package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/core"
	"github.com/hunterjsb/xandaris/server"
)

// runRemote connects to a remote server and runs the GUI client.
func runRemote(serverURL, playerName, apiKey string) {
	if apiKey == "" {
		log.Fatal("API key required for remote play. Sign in at the web portal to get one.")
	}

	fmt.Printf("Connecting to %s as %s...\n", serverURL, playerName)

	gs := server.New(screenWidth, screenHeight)
	remote := server.NewRemoteSync(gs, serverURL, apiKey)

	// Fetch the remote galaxy seed so we generate the same universe
	seed, err := remote.FetchSeed()
	if err != nil {
		log.Fatalf("Failed to fetch galaxy seed: %v", err)
	}
	fmt.Printf("Galaxy seed: %d\n", seed)

	// Generate the same galaxy as the remote server using the seed
	if err := gs.NewGameWithSeed(playerName, seed); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	gs.SetRemoteSync(remote)
	remote.Start()
	defer remote.Stop()

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle(fmt.Sprintf("Xandaris II - %s @ %s", playerName, serverURL))
	if runtime.GOARCH != "wasm" {
		ebiten.SetFullscreen(true)
	}

	app := core.New(screenWidth, screenHeight)
	app.Server = gs // Use the remote-synced GameServer instead of the default
	if err := app.InitializeForMenu(); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}
	// Skip InitializeNewGame (which calls NewGame with random seed) —
	// the GameServer already has the remote seed and state.
	app.InitializeClientViews()
	app.ConfigureCommandBar(serverURL, apiKey)
	app.SwitchToGalaxyView()

	fmt.Printf("Connected to %s! Playing as %s\n", serverURL, playerName)

	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}
