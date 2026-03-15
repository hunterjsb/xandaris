package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/core"
	"github.com/hunterjsb/xandaris/server"
	"github.com/hunterjsb/xandaris/views"

	// Register entity generators and tickable systems (side-effect imports)
	_ "github.com/hunterjsb/xandaris/entities/building"
	_ "github.com/hunterjsb/xandaris/entities/planet"
	_ "github.com/hunterjsb/xandaris/entities/resource"
	_ "github.com/hunterjsb/xandaris/entities/star"
	_ "github.com/hunterjsb/xandaris/entities/station"
	_ "github.com/hunterjsb/xandaris/tickable"
)

const (
	screenWidth  = 1280
	screenHeight = 720
)

func main() {
	headless := flag.Bool("headless", false, "Run as headless server (no GUI)")
	autoStart := flag.Bool("auto", false, "Skip menu and start new game immediately (GUI mode)")
	startView := flag.String("view", "", "Start in specific view: market, players, galaxy (requires --auto)")
	playerName := flag.String("player", "Player", "Player name")
	loadPath := flag.String("load", "", "Path to save file to load")
	connectURL := flag.String("connect", "", "Connect to remote server (e.g. https://api.xandaris.space)")
	apiKeyFlag := flag.String("key", "", "API key for remote server authentication")
	flag.Parse()

	if *headless {
		runHeadless(*playerName, *loadPath)
		return
	}

	if *connectURL != "" {
		runRemote(*connectURL, *playerName, *apiKeyFlag)
		return
	}

	// In WASM, check URL params for remote connection
	if runtime.GOARCH == "wasm" {
		serverURL, player, key := getWASMConnectParams()
		if serverURL != "" && key != "" {
			runRemote(serverURL, player, key)
			return
		}
	}

	runGUI(*autoStart, *playerName, *startView)
}

// runGUI starts the game with the Ebiten graphical client.
func runGUI(autoStart bool, playerName string, startView string) {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Xandaris II - Space Trading Game")
	if runtime.GOARCH != "wasm" {
		ebiten.SetFullscreen(true)
	}

	app := core.New(screenWidth, screenHeight)

	if err := app.InitializeForMenu(); err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Auto-start: skip menu and start a new game immediately
	if autoStart {
		if err := app.InitializeNewGame(playerName); err != nil {
			log.Fatalf("Failed to auto-start game: %v", err)
		}
		// Optionally switch to a specific view
		switch startView {
		case "market":
			app.GetViewManager().SwitchTo(views.ViewTypeMarket)
		case "players":
			app.GetViewManager().SwitchTo(views.ViewTypePlayers)
		}
	}

	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}

const autosavePath = "saves/autosave.xsave"

// runHeadless starts a headless server with no GUI.
// The game runs as a simulation with the REST API exposed on :8080.
func runHeadless(playerName string, loadPath string) {
	fmt.Println("=== Xandaris II — Headless Server ===")
	fmt.Println("API available at http://localhost:8080")

	gs := server.New(screenWidth, screenHeight)

	// Priority: explicit --load path > autosave > new game
	loaded := false
	if loadPath != "" {
		fmt.Printf("Loading save: %s\n", loadPath)
		if err := gs.LoadGame(loadPath); err != nil {
			log.Fatalf("Failed to load game: %v", err)
		}
		loaded = true
	} else if _, err := os.Stat(autosavePath); err == nil {
		fmt.Printf("Loading autosave: %s\n", autosavePath)
		if err := gs.LoadGame(autosavePath); err != nil {
			fmt.Printf("Autosave corrupted, starting new game: %v\n", err)
		} else {
			loaded = true
			fmt.Printf("[Server] Resumed from autosave (tick %d)\n", gs.TickManager.GetCurrentTick())
		}
	}

	if !loaded {
		fmt.Printf("Starting new game for: %s\n", playerName)
		if err := gs.NewGame(playerName); err != nil {
			log.Fatalf("Failed to start new game: %v", err)
		}
	}

	// Periodic autosave every 2 minutes
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := gs.AutoSave(autosavePath); err != nil {
				fmt.Printf("[Autosave] Error: %v\n", err)
			}
		}
	}()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		if err := gs.AutoSave(autosavePath); err != nil {
			fmt.Printf("[Autosave] Shutdown save failed: %v\n", err)
		}
		gs.Stop()
	}()

	// Run the simulation loop (blocks until Stop())
	gs.Run()
}
