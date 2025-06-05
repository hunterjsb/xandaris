package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/hunterjsb/xandaris/internal/player" // Added import
	"github.com/hunterjsb/xandaris/internal/tick"
	"github.com/hunterjsb/xandaris/internal/websocket"

	_ "github.com/hunterjsb/xandaris/migrations"
	"github.com/hunterjsb/xandaris/pkg"
)

func main() {
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

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

	// Create default superuser from environment variables
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		email := os.Getenv("SUPERUSER_EMAIL")
		password := os.Getenv("SUPERUSER_PASSWORD")
		if email != "" && password != "" {
			// Check if admin already exists
			_, err := app.Dao().FindAdminByEmail(email)
			if err != nil {
				// Create new admin
				admin := &models.Admin{}
				admin.Email = email
				admin.SetPassword(password)

				if err := app.Dao().SaveAdmin(admin); err != nil {
					log.Printf("Failed to create superuser: %v", err)
				} else {
					log.Printf("Superuser created: %s", email)
				}
			}
		}
		return nil
	})

	// Set up user creation hook for starting resources
	app.OnModelAfterCreate().Add(func(e *core.ModelEvent) error {
		if e.Model.TableName() == "users" {
			user := e.Model.(*models.Record)
			log.Printf("New user created: %s, setting last resource update and spawning starting fleet", user.Id)

			// Set last resource update time on user record
			user.Set("last_resource_update", time.Now())
			if err := app.Dao().SaveRecord(user); err != nil {
				log.Printf("Error updating user resource timestamp for user %s: %v", user.Id, err)
				return err
			}

			// Create starting fleet with settler ship
			if err := createStartingFleet(app, user.Id); err != nil {
				log.Printf("Error creating starting fleet for user %s: %v", user.Id, err)
				return err
			}

			log.Printf("Initialized user %s with starting fleet", user.Id)
		}
		return nil
	})

	// Register unified API routes
	pkg.RegisterAPIRoutes(app)

	// Set up additional custom routes
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
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

// createStartingFleet creates a fleet with a settler ship for new users
func createStartingFleet(app *pocketbase.PocketBase, userID string) error {
	// Find a random starting system
	systems, err := app.Dao().FindRecordsByExpr("systems", nil, nil)
	if err != nil {
		log.Printf("Error finding systems: %v", err)
		return fmt.Errorf("error finding systems: %v", err)
	}
	if len(systems) == 0 {
		log.Printf("No systems found in database")
		return fmt.Errorf("no systems available for starting location")
	}
	log.Printf("Found %d systems for starting location", len(systems))

	// Pick a random system as starting location
	startingSystem := systems[rand.Intn(len(systems))]
	log.Printf("INFO: Selected starting system %s for user %s", startingSystem.Id, userID)

	// Call the centralized utility function
	fleet, ship, err := player.CreateUserStarterFleet(app, userID, startingSystem.Id)
	if err != nil {
		// The utility function already logs detailed errors
		return fmt.Errorf("failed to create user starter fleet for user %s in system %s: %w", userID, startingSystem.Id, err)
	}

	log.Printf("INFO: Successfully created starting fleet %s with ship %s for user %s at system %s", fleet.Id, ship.Id, userID, startingSystem.Id)
	return nil
}
