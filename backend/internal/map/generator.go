package mapgen

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// GenerateMap creates a new galaxy with systems
func GenerateMap(app *pocketbase.PocketBase, systemCount int) error {
	rand.Seed(time.Now().UnixNano())
	fmt.Printf("Starting map generation with %d systems...\n", systemCount)

	// Clear existing systems
	if err := clearExistingSystems(app); err != nil {
		return fmt.Errorf("failed to clear existing systems: %w", err)
	}
	fmt.Println("Cleared existing systems")

	// Generate systems
	systems := generateSystems(systemCount)
	fmt.Printf("Generated %d system positions\n", len(systems))

	// Save systems to database
	sysCollection, err := app.Dao().FindCollectionByNameOrId("systems")
	if err != nil {
		return fmt.Errorf("systems collection not found: %w", err)
	}
	fmt.Println("Found systems collection")

	// Check if planets collection exists (optional)
	planetCollection, err := app.Dao().FindCollectionByNameOrId("planets")
	planetsEnabled := err == nil
	fmt.Printf("Planets collection enabled: %v\n", planetsEnabled)

	for i, sys := range systems {
		systemRecord := models.NewRecord(sysCollection)
		systemRecord.Set("x", sys.X)
		systemRecord.Set("y", sys.Y)
		systemRecord.Set("richness", sys.Richness)

		fmt.Printf("Saving system %d: x=%d, y=%d, richness=%d\n", i+1, sys.X, sys.Y, sys.Richness)
		if err := app.Dao().SaveRecord(systemRecord); err != nil {
			return fmt.Errorf("failed to save system %d: %w", i+1, err)
		}
		fmt.Printf("Saved system %d with ID: %s\n", i+1, systemRecord.Id)

		// Create 1-4 planets for each system (if planets collection exists)
		if planetsEnabled {
			// Get planet types once
			var allPlanetTypes []*models.Record
			allPlanetTypes, errPT := app.Dao().FindRecordsByExpr("planet_types", nil, nil)
			if errPT != nil || len(allPlanetTypes) == 0 {
				fmt.Printf("Warning: No planet types found in database for map generation for system %s.\n", systemRecord.Id)
			}

			// Each system has 1-4 planets
			planetCount := rand.Intn(4) + 1
			for j := 0; j < planetCount; j++ {
				planet := models.NewRecord(planetCollection)
				planet.Set("name", fmt.Sprintf("Planet-%d-%d", i+1, j+1))
				planet.Set("system_id", systemRecord.Id)
				planet.Set("size", rand.Intn(5)+1) // Size 1-5

				// Assign planet type
				if len(allPlanetTypes) > 0 {
					randomType := allPlanetTypes[rand.Intn(len(allPlanetTypes))]
					planet.Set("planet_type", randomType.Id)
				}

				if err := app.Dao().SaveRecord(planet); err != nil {
					return fmt.Errorf("failed to save planet %d for system %s: %w", j+1, systemRecord.Id, err)
				}
			}
			fmt.Printf("Saved %d planets for system %d\n", planetCount, i+1)
		}
	}

	fmt.Printf("Successfully saved all %d systems and their initial planets to database\n", len(systems))
	return nil
}

type System struct {
	X        int
	Y        int
	Richness int
}

func generateSystems(count int) []System {
	systems := make([]System, 0, count)

	// Galaxy dimensions optimized for 200+ systems (larger scale)
	galaxyWidth := 6000
	galaxyHeight := 4500
	centerX := galaxyWidth / 2
	centerY := galaxyHeight / 2

	// Create a spiral galaxy with multiple arms
	numArms := 4
	armSeparation := (2 * math.Pi) / float64(numArms)
	
	// Generate systems along spiral arms with some random scatter
	systemsPerArm := count / numArms
	remainingSystems := count % numArms

	for arm := 0; arm < numArms; arm++ {
		systemsInThisArm := systemsPerArm
		if arm < remainingSystems {
			systemsInThisArm++
		}

		baseAngle := float64(arm) * armSeparation
		
		for i := 0; i < systemsInThisArm; i++ {
			// Spiral parameters
			t := float64(i) / float64(systemsInThisArm) // 0 to 1 along arm
			radius := 200 + t*1800 // Start 200 units from center, extend to 2000 units
			
			// Spiral equation: angle increases with radius for natural spiral shape
			spiralTightness := 3.0 // Controls how tightly wound the spiral is
			angle := baseAngle + t*spiralTightness
			
			// Add more randomness to spread systems out
			radiusNoise := (rand.Float64() - 0.5) * 400 // ±200 units
			angleNoise := (rand.Float64() - 0.5) * 1.0  // ±0.5 radians
			
			finalRadius := radius + radiusNoise
			finalAngle := angle + angleNoise
			
			// Convert polar to cartesian coordinates
			x := centerX + int(finalRadius*math.Cos(finalAngle))
			y := centerY + int(finalRadius*math.Sin(finalAngle))
			
			// Ensure systems stay within galaxy bounds
			if x < 100 {
				x = 100
			} else if x > galaxyWidth-100 {
				x = galaxyWidth - 100
			}
			
			if y < 100 {
				y = 100
			} else if y > galaxyHeight-100 {
				y = galaxyHeight - 100
			}

			// Generate richness with galactic center bias (richer toward center)
			distanceFromCenter := math.Sqrt(float64((x-centerX)*(x-centerX) + (y-centerY)*(y-centerY)))
			maxDistance := math.Sqrt(float64(galaxyWidth*galaxyWidth + galaxyHeight*galaxyHeight)) / 2
			centerBias := 1.0 - (distanceFromCenter / maxDistance) // 1.0 at center, 0.0 at edge
			
			richness := 3 + rand.Intn(5) // Base 3-7
			if rand.Float64() < centerBias*0.3 {
				richness += 2 // Bonus richness near center
			}
			if rand.Float32() < 0.05 {
				richness = 1 + rand.Intn(2) // 5% chance of poor systems
			} else if rand.Float32() < 0.05 {
				richness = 9 + rand.Intn(2) // 5% chance of very rich systems
			}
			
			if richness > 10 {
				richness = 10
			}

			systems = append(systems, System{
				X:        x,
				Y:        y,
				Richness: richness,
			})
		}
	}

	// Add scattered systems in the galactic halo for variety
	haloSystems := count / 8 // ~12% of systems in sparse outer regions
	for i := 0; i < haloSystems && len(systems) < count; i++ {
		// Random position in outer galaxy
		angle := rand.Float64() * 2 * math.Pi
		radius := 1200 + rand.Float64()*800 // Between 1200-2000 units from center
		
		x := centerX + int(radius*math.Cos(angle))
		y := centerY + int(radius*math.Sin(angle))
		
		// Bounds check
		if x >= 100 && x <= galaxyWidth-100 && y >= 100 && y <= galaxyHeight-100 {
			systems = append(systems, System{
				X:        x,
				Y:        y,
				Richness: 1 + rand.Intn(4), // Halo systems are generally poorer
			})
		}
	}

	return systems
}

func clearExistingSystems(app *pocketbase.PocketBase) error {
	// delete planets first (if collection exists)
	if planets, err := app.Dao().FindRecordsByFilter("planets", "", "", 100, 0); err == nil {
		for _, p := range planets {
			_ = app.Dao().DeleteRecord(p)
		}
	}

	// delete systems
	systems, err := app.Dao().FindRecordsByFilter("systems", "", "", 50, 0)
	if err != nil {
		return nil
	}
	for _, system := range systems {
		if err := app.Dao().DeleteRecord(system); err != nil {
			return fmt.Errorf("failed to delete system %s: %w", system.Id, err)
		}
	}

	return nil
}

// GetMapData returns the current map state for the frontend
func GetMapData(app *pocketbase.PocketBase) (map[string]interface{}, error) {
	// Use a simple query to get all systems
	query := app.Dao().RecordQuery("systems")
	systems := []*models.Record{}

	if err := query.All(&systems); err != nil {
		return nil, fmt.Errorf("failed to fetch systems: %w", err)
	}

	systemsData := make([]map[string]interface{}, len(systems))
	for i, system := range systems {
		systemsData[i] = map[string]interface{}{
			"id":       system.Id,
			"x":        system.GetInt("x"),
			"y":        system.GetInt("y"),
			"richness": system.GetInt("richness"),
			"owner_id": system.GetString("owner_id"),
		}
	}

	// fetch planets (if collection exists)
	planetsData := make([]map[string]interface{}, 0)
	if planetRecords, err := app.Dao().FindRecordsByExpr("planets", nil, nil); err == nil {
		planetsData = make([]map[string]interface{}, len(planetRecords))
		for i, p := range planetRecords {
			planetsData[i] = map[string]interface{}{
				"id":        p.Id,
				"name":      p.GetString("name"),
				"system_id": p.GetString("system_id"),
				"type_id":   p.GetString("type_id"),
			}
		}
	}

	// Generate lanes (connections between nearby systems)
	lanes := generateLanes(systemsData)

	return map[string]interface{}{
		"systems": systemsData,
		"planets": planetsData,
		"lanes":   lanes,
	}, nil
}

func generateLanes(systems []map[string]interface{}) []map[string]interface{} {
	lanes := make([]map[string]interface{}, 0)
	minDistance := 200.0 // Minimum distance to avoid too many close connections
	maxDistance := 650.0 // Maximum distance for lane connections (scaled for larger galaxy)

	systemConnections := make(map[string]int) // Track connections per system
	connected := make(map[string]bool)        // Track which systems are in main component
	
	// Initialize connection tracking
	for _, sys := range systems {
		systemConnections[sys["id"].(string)] = 0
		connected[sys["id"].(string)] = false
	}
	

	// Phase 1: Build minimum spanning tree to ensure connectivity
	// Start with the center-most system
	centerX := 0.0
	centerY := 0.0
	for _, sys := range systems {
		centerX += float64(sys["x"].(int))
		centerY += float64(sys["y"].(int))
	}
	centerX /= float64(len(systems))
	centerY /= float64(len(systems))
	
	// Find system closest to center as starting point
	var startSystem map[string]interface{}
	minDistFromCenter := math.Inf(1)
	for _, sys := range systems {
		x := float64(sys["x"].(int))
		y := float64(sys["y"].(int))
		dist := math.Sqrt((x-centerX)*(x-centerX) + (y-centerY)*(y-centerY))
		if dist < minDistFromCenter {
			minDistFromCenter = dist
			startSystem = sys
		}
	}
	
	// Mark start system as connected
	connected[startSystem["id"].(string)] = true
	connectedSystems := []map[string]interface{}{startSystem}
	
	// Build MST: repeatedly connect closest unconnected system to connected component
	for len(connectedSystems) < len(systems) {
		var bestConnection struct {
			from     map[string]interface{}
			to       map[string]interface{}
			distance float64
		}
		bestConnection.distance = math.Inf(1)
		
		// Find shortest edge from connected to unconnected systems
		for _, connectedSys := range connectedSystems {
			x1 := float64(connectedSys["x"].(int))
			y1 := float64(connectedSys["y"].(int))
			
			for _, sys := range systems {
				if connected[sys["id"].(string)] {
					continue // Skip already connected systems
				}
				
				x2 := float64(sys["x"].(int))
				y2 := float64(sys["y"].(int))
				distance := math.Sqrt((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1))
				
				if distance < bestConnection.distance {
					bestConnection.from = connectedSys
					bestConnection.to = sys
					bestConnection.distance = distance
				}
			}
		}
		
		// Add the best connection found
		if bestConnection.distance != math.Inf(1) {
			fromId := bestConnection.from["id"].(string)
			toId := bestConnection.to["id"].(string)
			
			lanes = append(lanes, map[string]interface{}{
				"from":     fromId,
				"to":       toId,
				"fromX":    bestConnection.from["x"],
				"fromY":    bestConnection.from["y"],
				"toX":      bestConnection.to["x"],
				"toY":      bestConnection.to["y"],
				"distance": int(bestConnection.distance),
			})
			
			systemConnections[fromId]++
			systemConnections[toId]++
			connected[toId] = true
			connectedSystems = append(connectedSystems, bestConnection.to)
		}
	}
	


	// Phase 2: Add strategic additional connections (avoiding crossings)
	// Allow up to 60 additional strategic connections for redundant paths and loops (scaled for 200 systems)
	maxAdditionalConnections := 60
	additionalAdded := 0
	
	type potentialConnection struct {
		from     map[string]interface{}
		to       map[string]interface{}
		distance float64
		angle    float64
	}
	
	var additionalConnections []potentialConnection
	
	// Find additional connections that don't cross existing lanes
	for i, sys1 := range systems {
		// Allow up to 5 connections per system for strategic redundancy
		if systemConnections[sys1["id"].(string)] >= 5 {
			continue
		}
		
		x1 := float64(sys1["x"].(int))
		y1 := float64(sys1["y"].(int))
		
		for j := i + 1; j < len(systems); j++ {
			sys2 := systems[j]
			if systemConnections[sys2["id"].(string)] >= 5 {
				continue
			}
			
			x2 := float64(sys2["x"].(int))
			y2 := float64(sys2["y"].(int))
			distance := math.Sqrt((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1))
			
			if distance >= minDistance && distance <= maxDistance {
				// Calculate angle for crossing detection
				angle := math.Atan2(y2-y1, x2-x1)
				
				// Check if this connection would cross existing lanes
				// Allow some crossings for strategic routes if both systems have low connectivity
				wouldCross := false
				crossingCount := 0
				for _, lane := range lanes {
					lx1 := float64(lane["fromX"].(int))
					ly1 := float64(lane["fromY"].(int))
					lx2 := float64(lane["toX"].(int))
					ly2 := float64(lane["toY"].(int))
					
					if linesIntersect(x1, y1, x2, y2, lx1, ly1, lx2, ly2) {
						crossingCount++
					}
				}
				
				// Allow crossings for strategic routes: up to 2 crossings for low connectivity systems
				lowConnectivity := systemConnections[sys1["id"].(string)] <= 2 && systemConnections[sys2["id"].(string)] <= 2
				mediumConnectivity := systemConnections[sys1["id"].(string)] <= 3 && systemConnections[sys2["id"].(string)] <= 3
				wouldCross = crossingCount > 0 && (!lowConnectivity || (crossingCount > 2)) && (!mediumConnectivity || crossingCount > 1)
				
				if !wouldCross {
					additionalConnections = append(additionalConnections, potentialConnection{
						from:     sys1,
						to:       sys2,
						distance: distance,
						angle:    angle,
					})
				}
			}
		}
	}
	
	// Sort additional connections by distance (prefer shorter)
	for i := 0; i < len(additionalConnections)-1; i++ {
		for j := 0; j < len(additionalConnections)-i-1; j++ {
			if additionalConnections[j].distance > additionalConnections[j+1].distance {
				additionalConnections[j], additionalConnections[j+1] = additionalConnections[j+1], additionalConnections[j]
			}
		}
	}
	
	// Add non-crossing connections up to limit
	for _, conn := range additionalConnections {
		if additionalAdded >= maxAdditionalConnections {
			break
		}
		
		fromId := conn.from["id"].(string)
		toId := conn.to["id"].(string)
		
		if systemConnections[fromId] < 5 && systemConnections[toId] < 5 {
			lanes = append(lanes, map[string]interface{}{
				"from":     fromId,
				"to":       toId,
				"fromX":    conn.from["x"],
				"fromY":    conn.from["y"],
				"toX":      conn.to["x"],
				"toY":      conn.to["y"],
				"distance": int(conn.distance),
			})
			
			systemConnections[fromId]++
			systemConnections[toId]++
			additionalAdded++
		}
	}
	


	return lanes
}

// linesIntersect checks if two line segments intersect
func linesIntersect(x1, y1, x2, y2, x3, y3, x4, y4 float64) bool {
	// Calculate the direction of the lines
	denom := (x1-x2)*(y3-y4) - (y1-y2)*(x3-x4)
	if math.Abs(denom) < 1e-10 {
		return false // Lines are parallel
	}
	
	t := ((x1-x3)*(y3-y4) - (y1-y3)*(x3-x4)) / denom
	u := -((x1-x2)*(y1-y3) - (y1-y2)*(x1-x3)) / denom
	
	// Check if intersection point is within both line segments
	return t >= 0 && t <= 1 && u >= 0 && u <= 1
}
