package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/hunterjsb/xandaris/internal/tick"
	"github.com/hunterjsb/xandaris/internal/websocket"
	_ "github.com/hunterjsb/xandaris/migrations"
	"github.com/hunterjsb/xandaris/pkg"
)

func main() {
	app := pocketbase.New()

	// loosely check if it was executed using "go run"
	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())

	// register migrate CLI commands (create, up, down, etc.)
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// enable auto creation of migration files when making collection changes in the Dashboard
		// (the isGoRun check is to enable it only during development)
		Automigrate: isGoRun,
	})

	// Set up continuous game tick system
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// Start the game tick processor in a goroutine
		go tick.StartContinuousProcessor(app)
		return nil
	})

	// Set up user creation hook for starting resources
	app.OnModelAfterCreate().Add(func(e *core.ModelEvent) error {
		if e.Model.TableName() == "users" {
			user := e.Model.(*models.Record)
			log.Printf("New user created: %s, setting starting resources", user.Id)

			// Set starting resources - stored in a separate table or user fields
			// For now, we'll use user custom fields
			user.Set("credits", 1000) // Starting credits
			user.Set("last_resource_update", time.Now())

			if err := app.Dao().SaveRecord(user); err != nil {
				log.Printf("Error setting starting resources for user %s: %v", user.Id, err)
				return err
			}
			log.Printf("Set starting resources for user %s: 1000 credits", user.Id)
		}
		return nil
	})

	// Set up custom API routes
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// Map endpoint
		e.Router.GET("/api/map", pkg.MapHandler(app))

		// Game status endpoint
		e.Router.GET("/api/status", func(c echo.Context) error {
			currentTick := tick.GetCurrentTick(app)
			ticksPerMinute := tick.GetTickRate()

			return c.JSON(200, map[string]interface{}{
				"current_tick":     currentTick,
				"ticks_per_minute": ticksPerMinute,
				"server_time":      time.Now().Format(time.RFC3339),
			})
		})

		// Game action endpoints (require authentication)
		e.Router.POST("/api/orders/fleet", pkg.FleetOrderHandler(app), apis.RequireRecordAuth())
		e.Router.POST("/api/orders/build", pkg.BuildOrderHandler(app), apis.RequireRecordAuth())
		e.Router.POST("/api/orders/trade", pkg.TradeOrderHandler(app), apis.RequireRecordAuth())
		e.Router.POST("/api/diplomacy", pkg.DiplomacyHandler(app), apis.RequireRecordAuth())

		// WebSocket endpoint for real-time updates
		e.Router.GET("/api/stream", websocket.HandleWebSocket(app))

		// Debug endpoint to manually trigger a tick
		e.Router.POST("/api/debug/tick", func(c echo.Context) error {
			if err := tick.ProcessTick(app); err != nil {
				return c.JSON(500, map[string]string{"error": err.Error()})
			}
			return c.JSON(200, map[string]string{"status": "tick processed"})
		})

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
