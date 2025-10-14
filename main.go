package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	_ "github.com/hunterjsb/xandaris/entities/building"
	_ "github.com/hunterjsb/xandaris/entities/planet"
	_ "github.com/hunterjsb/xandaris/entities/resource"
	_ "github.com/hunterjsb/xandaris/entities/star"
	_ "github.com/hunterjsb/xandaris/entities/station"
	"github.com/hunterjsb/xandaris/core"
	_ "github.com/hunterjsb/xandaris/tickable" // Import tickable systems for auto-registration
)

const (
	screenWidth  = 1280
	screenHeight = 720
)

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Xandaris II - Space Trading Game")

	// Create and initialize app
	app := core.New(screenWidth, screenHeight)

	// Initialize for main menu
	if err := app.InitializeForMenu(); err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Run game loop
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}
