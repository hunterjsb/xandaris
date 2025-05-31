package main

import (
	"log"

	"github.com/pocketbase/pocketbase"

	mapgen "github.com/hunterjsb/xandaris/internal/map"
	_ "github.com/hunterjsb/xandaris/migrations"
)

func main() {
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: "pb_data",
	})

	// Bootstrap the application to initialize the database
	if err := app.Bootstrap(); err != nil {
		log.Fatal("Failed to bootstrap app:", err)
	}

	// Generate initial map with 50 planets
	if err := mapgen.GenerateMap(app, 50); err != nil {
		log.Fatal("Failed to generate map:", err)
	}

	log.Println("Map generated successfully with 50 planets!")
}