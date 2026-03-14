package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	flag.Parse()

	if *headless {
		runHeadless(*playerName, *loadPath)
		return
	}

	runGUI(*autoStart, *playerName, *startView)
}

// runGUI starts the game with the Ebiten graphical client.
func runGUI(autoStart bool, playerName string, startView string) {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Xandaris II - Space Trading Game")
	ebiten.SetFullscreen(true)

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

// runHeadless starts a headless server with no GUI.
// The game runs as a simulation with the REST API exposed on :8080.
func runHeadless(playerName string, loadPath string) {
	fmt.Println("=== Xandaris II — Headless Server ===")
	fmt.Println("API available at http://localhost:8080")

	gs := server.New(screenWidth, screenHeight)

	if loadPath != "" {
		fmt.Printf("Loading save: %s\n", loadPath)
		if err := gs.LoadGame(loadPath); err != nil {
			log.Fatalf("Failed to load game: %v", err)
		}
	} else {
		fmt.Printf("Starting new game for: %s\n", playerName)
		if err := gs.NewGame(playerName); err != nil {
			log.Fatalf("Failed to start new game: %v", err)
		}
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		// Auto-save on shutdown
		human := gs.GetHumanPlayer()
		if human != nil {
			gs.SaveGame(human.Name)
		}
		gs.Stop()
	}()

	// Run the simulation loop (blocks until Stop())
	gs.Run()
}
