package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	_ "github.com/hunterjsb/xandaris/entities/building"
	_ "github.com/hunterjsb/xandaris/entities/planet"
	_ "github.com/hunterjsb/xandaris/entities/resource"
	_ "github.com/hunterjsb/xandaris/entities/star"
	_ "github.com/hunterjsb/xandaris/entities/station"
	"github.com/hunterjsb/xandaris/tickable"
	_ "github.com/hunterjsb/xandaris/tickable" // Import tickable systems for auto-registration
)

// StartMode represents how the game was started
type StartMode int

const (
	StartModeMenu StartMode = iota
	StartModeNewGame
)

const (
	screenWidth      = 1280
	screenHeight     = 720
	systemCount      = 40
	circleRadius     = 8
	maxHyperlanes    = 3
	minDistance      = 60.0
	maxDistance      = 180.0
	minSystemSpacing = 45.0
)

// GameSystemContext implements tickable.SystemContext
type GameSystemContext struct {
	game *Game
}

func (gsc *GameSystemContext) GetGame() interface{} {
	return gsc.game
}

func (gsc *GameSystemContext) GetPlayers() interface{} {
	return gsc.game.players
}

func (gsc *GameSystemContext) GetTick() int64 {
	return gsc.game.tickManager.GetCurrentTick()
}

// Game implements ebiten.Game interface
type Game struct {
	systems     []*entities.System
	hyperlanes  []entities.Hyperlane
	viewManager *ViewManager
	seed        int64
	players     []*entities.Player
	humanPlayer *entities.Player
	tickManager *TickManager
}

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		systems:    make([]*entities.System, 0),
		hyperlanes: make([]entities.Hyperlane, 0),
		seed:       time.Now().UnixNano(),
		players:    make([]*entities.Player, 0),
	}

	// Initialize tick system (10 ticks per second at 1x speed)
	g.tickManager = NewTickManager(10.0)

	// Generate galaxy data
	g.generateSystems()
	g.generateHyperlanes()

	// Create human player
	playerColor := color.RGBA{100, 200, 100, 255} // Green for player
	g.humanPlayer = entities.NewPlayer(0, "Player", playerColor, entities.PlayerTypeHuman)
	g.players = append(g.players, g.humanPlayer)

	// Initialize player with starting planet
	entities.InitializePlayer(g.humanPlayer, g.systems)

	// Initialize tickable systems
	context := &GameSystemContext{game: g}
	tickable.InitializeAllSystems(context)

	// Register construction completion handler
	g.registerConstructionHandler()

	// Initialize view system
	g.viewManager = NewViewManager(g)

	// Create and register views
	galaxyView := NewGalaxyView(g)
	systemView := NewSystemView(g)
	planetView := NewPlanetView(g)
	mainMenuView := NewMainMenuView(g)

	g.viewManager.RegisterView(galaxyView)
	g.viewManager.RegisterView(systemView)
	g.viewManager.RegisterView(planetView)
	g.viewManager.RegisterView(mainMenuView)

	// Start with galaxy view
	g.viewManager.SwitchTo(ViewTypeGalaxy)

	return g
}

// NewGameForMenu creates a minimal game instance for the main menu
func NewGameForMenu() *Game {
	g := &Game{
		systems:    make([]*entities.System, 0),
		hyperlanes: make([]entities.Hyperlane, 0),
		players:    make([]*entities.Player, 0),
	}

	// Initialize tick manager for menu (though it won't really be used)
	g.tickManager = NewTickManager(10.0)

	// Initialize view system
	g.viewManager = NewViewManager(g)

	// Create and register all views
	mainMenuView := NewMainMenuView(g)
	galaxyView := NewGalaxyView(g)
	systemView := NewSystemView(g)
	planetView := NewPlanetView(g)

	g.viewManager.RegisterView(mainMenuView)
	g.viewManager.RegisterView(galaxyView)
	g.viewManager.RegisterView(systemView)
	g.viewManager.RegisterView(planetView)

	// Start with main menu
	g.viewManager.SwitchTo(ViewTypeMainMenu)

	return g
}

// generateSystems creates systems at random coordinates
func (g *Game) generateSystems() {
	colors := GetSystemColors()

	// Generate systems with random positions
	for i := 0; i < systemCount; i++ {
		var x, y float64
		var validPosition bool
		attempts := 0

		// Keep trying until we find a position that's not too close to existing systems
		for !validPosition && attempts < 200 {
			x = 80 + rand.Float64()*(screenWidth-160)
			y = 80 + rand.Float64()*(screenHeight-160)
			validPosition = true

			// Check distance to all existing systems
			for _, existing := range g.systems {
				distance := math.Sqrt(math.Pow(x-existing.X, 2) + math.Pow(y-existing.Y, 2))
				if distance < minSystemSpacing {
					validPosition = false
					break
				}
			}
			attempts++
		}

		system := &entities.System{
			ID:          i,
			X:           x,
			Y:           y,
			Name:        fmt.Sprintf("SYS-%d", i+1),
			Color:       colors[rand.Intn(len(colors))],
			Connections: make([]int, 0),
		}

		g.systems = append(g.systems, system)

		// Generate entities for this system using the new entity generator system
		seed := int64(i) + g.seed
		generatedEntities := entities.GenerateEntitiesForSystem(i, seed)
		for _, entity := range generatedEntities {
			system.AddEntity(entity)
		}
	}
}

// generateHyperlanes creates connections between systems
func (g *Game) generateHyperlanes() {
	for _, system := range g.systems {
		// Find nearby systems for potential connections
		var nearbySystemsWithDistance []struct {
			system   *entities.System
			distance float64
		}

		for _, other := range g.systems {
			if other.ID == system.ID {
				continue
			}

			distance := math.Sqrt(math.Pow(system.X-other.X, 2) + math.Pow(system.Y-other.Y, 2))
			if distance >= minDistance && distance <= maxDistance {
				nearbySystemsWithDistance = append(nearbySystemsWithDistance, struct {
					system   *entities.System
					distance float64
				}{other, distance})
			}
		}

		// Sort by distance (closest first)
		for i := 0; i < len(nearbySystemsWithDistance)-1; i++ {
			for j := i + 1; j < len(nearbySystemsWithDistance); j++ {
				if nearbySystemsWithDistance[i].distance > nearbySystemsWithDistance[j].distance {
					nearbySystemsWithDistance[i], nearbySystemsWithDistance[j] = nearbySystemsWithDistance[j], nearbySystemsWithDistance[i]
				}
			}
		}

		// Connect to closest systems (max connections per system)
		connectionsToMake := maxHyperlanes
		if len(nearbySystemsWithDistance) < maxHyperlanes {
			connectionsToMake = len(nearbySystemsWithDistance)
		}

		for i := 0; i < connectionsToMake; i++ {
			other := nearbySystemsWithDistance[i].system

			// Check if connection already exists
			connectionExists := false
			for _, hyperlane := range g.hyperlanes {
				if (hyperlane.From == system.ID && hyperlane.To == other.ID) ||
					(hyperlane.From == other.ID && hyperlane.To == system.ID) {
					connectionExists = true
					break
				}
			}

			if !connectionExists {
				// Add hyperlane
				g.hyperlanes = append(g.hyperlanes, entities.Hyperlane{
					From: system.ID,
					To:   other.ID,
				})

				// Add to both systems' connection lists
				system.Connections = append(system.Connections, other.ID)
				other.Connections = append(other.Connections, system.ID)
			}
		}
	}
}

// GetPlayers returns the game's players
func (g *Game) GetPlayers() []*entities.Player {
	return g.players
}

// GetSystems returns a map of systems indexed by ID
func (g *Game) GetSystems() map[int]*entities.System {
	systemsMap := make(map[int]*entities.System)
	for _, system := range g.systems {
		systemsMap[system.ID] = system
	}
	return systemsMap
}

// GetHyperlanes returns all hyperlanes
func (g *Game) GetHyperlanes() []entities.Hyperlane {
	return g.hyperlanes
}

// Update updates the game state
func (g *Game) Update() error {
	// Handle global keyboard shortcuts
	g.handleGlobalInput()

	// Update tick system (this will also update tickable systems)
	g.tickManager.Update()

	// Update current view
	return g.viewManager.Update()
}

// handleGlobalInput handles keyboard input for game-wide controls
func (g *Game) handleGlobalInput() {
	// Don't handle game controls in main menu
	if g.viewManager.GetCurrentView().GetType() == ViewTypeMainMenu {
		return
	}

	// Space to toggle pause
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.tickManager.TogglePause()
	}

	// Number keys for speed control
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.tickManager.SetSpeed(TickSpeed1x)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.tickManager.SetSpeed(TickSpeed2x)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		g.tickManager.SetSpeed(TickSpeed4x)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		g.tickManager.SetSpeed(TickSpeed8x)
	}

	// Plus/Minus to cycle speed
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) {
		g.tickManager.CycleSpeed()
	}

	// F5 to quick save
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		if g.humanPlayer != nil {
			err := g.SaveGameToFile(g.humanPlayer.Name)
			if err != nil {
				fmt.Printf("Failed to save game: %v\n", err)
			} else {
				fmt.Println("Game saved successfully!")
			}
		}
	}
}

// Draw draws the game screen
func (g *Game) Draw(screen *ebiten.Image) {
	g.viewManager.Draw(screen)

	// Draw tick info overlay
	g.drawTickInfo(screen)
}

// drawTickInfo draws tick information overlay
func (g *Game) drawTickInfo(screen *ebiten.Image) {
	// Don't draw in main menu
	if g.viewManager.GetCurrentView().GetType() == ViewTypeMainMenu {
		return
	}

	// Draw in bottom-left corner
	x := 10
	y := screenHeight - 60

	// Create small panel
	panel := NewUIPanel(x, y, 200, 50)
	panel.Draw(screen)

	// Draw tick info
	textX := x + 10
	textY := y + 15

	speedStr := g.tickManager.GetSpeedString()
	DrawText(screen, "Speed: "+speedStr, textX, textY, UITextPrimary)
	DrawText(screen, g.tickManager.GetGameTimeFormatted(), textX, textY+15, UITextSecondary)
	DrawText(screen, "[Space] Pause  [F5] Save", textX, textY+30, UITextSecondary)
}

// Layout returns the game's screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Xandaris II - Space Trading Game")

	// Start with main menu
	game := NewGameForMenu()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// registerConstructionHandler sets up handler for completed constructions
func (g *Game) registerConstructionHandler() {
	constructionSystem := tickable.GetSystemByName("Construction")
	if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
		cs.RegisterCompletionHandler(func(completion tickable.ConstructionCompletion) {
			g.handleConstructionComplete(completion)
		})
	}
}

// handleConstructionComplete adds completed buildings/ships to the game
func (g *Game) handleConstructionComplete(completion tickable.ConstructionCompletion) {
	// Handle ship construction
	if completion.Item.Type == "Ship" {
		g.handleShipConstruction(completion)
		return
	}

	// Handle building construction
	// Find the planet or resource by ID
	locationID := completion.Location

	// Search all systems for the entity
	for _, system := range g.systems {
		for _, entity := range system.Entities {
			// Check planets
			if planet, ok := entity.(*entities.Planet); ok {
				if fmt.Sprintf("%d", planet.GetID()) == locationID {
					// Found the planet, add building
					building := g.createBuildingFromCompletion(completion, planet)
					if building != nil {
						planet.Buildings = append(planet.Buildings, building)

						// Initialize Fuel storage when a refinery is built
						if b, ok := building.(*entities.Building); ok && b.BuildingType == "Refinery" {
							// Ensure the planet has Fuel storage initialized
							if _, exists := planet.StoredResources["Fuel"]; !exists {
								planet.AddStoredResource("Fuel", 0) // Initialize with 0 fuel
							}
						}
					}
					// Refresh planet view if it's currently viewing this planet
					g.refreshPlanetViewIfActive(planet)
					return
				}

				// Check resources on this planet
				for _, resource := range planet.Resources {
					if fmt.Sprintf("%d", resource.GetID()) == locationID {
						// Found the resource, add building
						building := g.createBuildingFromCompletion(completion, resource)
						if building != nil {
							// Buildings on resources need to be tracked somewhere
							// For now, we'll add to the parent planet
							planet.Buildings = append(planet.Buildings, building)
						}
						// Refresh planet view if it's currently viewing this planet
						g.refreshPlanetViewIfActive(planet)
						return
					}
				}
			}
		}
	}
}

// handleShipConstruction spawns a completed ship
func (g *Game) handleShipConstruction(completion tickable.ConstructionCompletion) {
	// Parse location to find planet (format: "planet_<ID>")
	var planetID int
	fmt.Sscanf(completion.Location, "planet_%d", &planetID)

	// Find the planet and system
	var targetPlanet *entities.Planet
	var targetSystem *entities.System

	for _, system := range g.systems {
		for _, entity := range system.Entities {
			if planet, ok := entity.(*entities.Planet); ok {
				if planet.GetID() == planetID {
					targetPlanet = planet
					targetSystem = system
					break
				}
			}
		}
		if targetPlanet != nil {
			break
		}
	}

	if targetPlanet == nil || targetSystem == nil {
		fmt.Printf("[Game] ERROR: Could not find planet %d for ship construction\n", planetID)
		return
	}

	// Parse ship type from completion name
	shipType := entities.ShipType(completion.Item.Name)

	// Generate unique ship ID
	shipID := int(g.tickManager.GetCurrentTick())*1000 + rand.Intn(1000)

	// Find owner player
	var owner *entities.Player
	for _, player := range g.players {
		if player.Name == completion.Owner {
			owner = player
			break
		}
	}

	if owner == nil {
		fmt.Printf("[Game] ERROR: Could not find owner %s for ship\n", completion.Owner)
		return
	}

	// Create the ship
	shipName := fmt.Sprintf("%s %s-%d", owner.Name, shipType, len(owner.OwnedShips)+1)
	ship := entities.NewShip(shipID, shipName, shipType, targetSystem.ID, owner.Name, owner.Color)

	// Set ship position to orbit the PLANET, not the star
	// OrbitDistance = 0 means it orbits the planet at the planet's location
	// This makes it only visible in PlanetView, not SystemView
	ship.OrbitDistance = targetPlanet.OrbitDistance                                      // Store which planet's orbit
	ship.OrbitAngle = targetPlanet.OrbitAngle + 1.0 + float64(len(owner.OwnedShips))*0.3 // Spread ships around planet

	// Add ship to system BEFORE adding to player (important for save/load)
	targetSystem.AddEntity(ship)

	// Add ship to player's owned ships
	owner.AddOwnedShip(ship)

	fmt.Printf("[Game] Ship constructed: %s (%s) for %s at %s\n",
		shipName, shipType, owner.Name, targetPlanet.Name)
	fmt.Printf("[Game] Ship orbit: distance=%.2f, angle=%.2f (planet: dist=%.2f, angle=%.2f)\n",
		ship.OrbitDistance, ship.OrbitAngle, targetPlanet.OrbitDistance, targetPlanet.OrbitAngle)
}

// refreshPlanetViewIfActive refreshes planet view if the given planet is currently displayed
func (g *Game) refreshPlanetViewIfActive(planet *entities.Planet) {
	if g.viewManager.GetCurrentView().GetType() == ViewTypePlanet {
		if planetView, ok := g.viewManager.GetCurrentView().(*PlanetView); ok {
			if planetView.planet == planet {
				planetView.RefreshPlanet()
			}
		}
	}
}

// createBuildingFromCompletion creates a building entity from a completion
func (g *Game) createBuildingFromCompletion(completion tickable.ConstructionCompletion, attachedTo entities.Entity) entities.Entity {
	// Generate parameters for building
	params := entities.GenerationParams{
		SystemID:      0,
		OrbitDistance: 20.0 + float64(len(g.systems))*5.0, // Position around planet
		OrbitAngle:    float64(completion.Tick%628) / 100.0,
		SystemSeed:    completion.Tick,
	}

	// Get the appropriate building generator based on item name
	generators := entities.GetGeneratorsByType(entities.EntityTypeBuilding)
	for _, gen := range generators {
		if gen.GetSubType() == completion.Item.Type ||
			gen.GetSubType()+" Complex" == completion.Item.Name ||
			gen.GetSubType()+" Module" == completion.Item.Name ||
			"Orbital "+gen.GetSubType() == completion.Item.Name ||
			"Oil "+gen.GetSubType() == completion.Item.Name ||
			"Mining Complex" == completion.Item.Name && gen.GetSubType() == "Mine" {
			building := gen.Generate(params)
			if b, ok := building.(*entities.Building); ok {
				b.Owner = completion.Owner
				b.AttachedTo = completion.Location
				return b
			}
		}
	}

	return nil
}
