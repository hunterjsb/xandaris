//go:build !js

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
func runRemote(serverURL, playerName, password string) {
	if password == "" {
		password = "default123"
	}

	fmt.Printf("Connecting to %s as %s...\n", serverURL, playerName)

	gs := server.New(screenWidth, screenHeight)
	remote := server.NewRemoteSync(gs, serverURL, "")

	// Try login first, then register
	apiKey, err := remote.Login(playerName, password)
	if err != nil {
		fmt.Printf("Login failed (%v), registering...\n", err)
		apiKey, err = remote.Register(playerName, password)
		if err != nil {
			log.Fatalf("Failed to register: %v", err)
		}
		fmt.Printf("Registered! Key: %s...\n", apiKey[:20])
	} else {
		fmt.Printf("Logged in! Key: %s...\n", apiKey[:20])
	}

	if err := gs.NewGame(playerName); err != nil {
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
	if err := app.InitializeForMenu(); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}
	if err := app.InitializeNewGame(playerName); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}

	fmt.Printf("Connected to %s! Playing as %s\n", serverURL, playerName)

	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}
