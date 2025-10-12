package main

import (
	"fmt"
	"image/color"
	"math/rand"
)

// SpaceStation represents a space station entity
type SpaceStation struct {
	ID            int
	Name          string
	Color         color.RGBA
	OrbitDistance float64
	OrbitAngle    float64
	StationType   string
	Capacity      int
	CurrentPop    int
	Services      []string
	Owner         string
	TradeGoods    []string
	DefenseLevel  int
}

func (s *SpaceStation) GetID() int                { return s.ID }
func (s *SpaceStation) GetName() string           { return s.Name }
func (s *SpaceStation) GetType() EntityType       { return "Station" }
func (s *SpaceStation) GetOrbitDistance() float64 { return s.OrbitDistance }
func (s *SpaceStation) GetOrbitAngle() float64    { return s.OrbitAngle }
func (s *SpaceStation) GetColor() color.RGBA      { return s.Color }

func (s *SpaceStation) GetDescription() string {
	return fmt.Sprintf("%s Station", s.StationType)
}

// GetDetailedInfo returns detailed information about the station
func (s *SpaceStation) GetDetailedInfo() map[string]string {
	return map[string]string{
		"Type":       s.StationType,
		"Capacity":   fmt.Sprintf("%d", s.Capacity),
		"Population": fmt.Sprintf("%d/%d", s.CurrentPop, s.Capacity),
		"Owner":      s.Owner,
		"Defense":    fmt.Sprintf("Level %d", s.DefenseLevel),
		"Occupancy":  fmt.Sprintf("%.1f%%", float64(s.CurrentPop)/float64(s.Capacity)*100),
	}
}

// Station type constants
const (
	StationTypeTrading  = "Trading"
	StationTypeMilitary = "Military"
	StationTypeResearch = "Research"
	StationTypeMining   = "Mining"
	StationTypeRefinery = "Refinery"
	StationTypeShipyard = "Shipyard"
)

// GetStationTypes returns all available station types
func GetStationTypes() []string {
	return []string{
		StationTypeTrading,
		StationTypeMilitary,
		StationTypeResearch,
		StationTypeMining,
		StationTypeRefinery,
		StationTypeShipyard,
	}
}

// GenerateSpaceStation creates a random space station
func GenerateSpaceStation(systemID int, orbitDistance float64) *SpaceStation {
	stationTypes := GetStationTypes()
	owners := []string{"Independent", "Trade Union", "Military Corp", "Research Guild", "Mining Consortium"}

	stationType := stationTypes[rand.Intn(len(stationTypes))]
	owner := owners[rand.Intn(len(owners))]

	// Generate color based on station type
	stationColor := getColorForStationType(stationType)

	// Generate capacity based on station type
	var baseCapacity int
	switch stationType {
	case StationTypeTrading:
		baseCapacity = 2000
	case StationTypeMilitary:
		baseCapacity = 1500
	case StationTypeResearch:
		baseCapacity = 800
	case StationTypeMining:
		baseCapacity = 1200
	case StationTypeRefinery:
		baseCapacity = 1000
	case StationTypeShipyard:
		baseCapacity = 3000
	default:
		baseCapacity = 1000
	}

	capacity := baseCapacity + rand.Intn(baseCapacity/2)
	currentPop := rand.Intn(capacity)

	station := &SpaceStation{
		ID:            systemID*10000 + 999,
		Name:          generateStationName(stationType),
		Color:         stationColor,
		OrbitDistance: orbitDistance,
		OrbitAngle:    rand.Float64() * 6.28,
		StationType:   stationType,
		Capacity:      capacity,
		CurrentPop:    currentPop,
		Services:      generateStationServices(stationType),
		Owner:         owner,
		TradeGoods:    generateTradeGoods(stationType),
		DefenseLevel:  generateDefenseLevel(stationType),
	}

	return station
}

// getColorForStationType returns the appropriate color for a station type
func getColorForStationType(stationType string) color.RGBA {
	switch stationType {
	case StationTypeTrading:
		return ColorStationTrading
	case StationTypeMilitary:
		return ColorStationMilitary
	case StationTypeResearch:
		return ColorStationResearch
	case StationTypeMining:
		return ColorStationMining
	case StationTypeRefinery:
		return ColorStationRefinery
	case StationTypeShipyard:
		return ColorStationShipyard
	default:
		return ColorStationTrading
	}
}

// generateStationName creates a name based on station type
func generateStationName(stationType string) string {
	prefixes := map[string][]string{
		StationTypeTrading:  {"Commerce", "Trade", "Market", "Exchange"},
		StationTypeMilitary: {"Fortress", "Guardian", "Sentinel", "Bastion"},
		StationTypeResearch: {"Discovery", "Insight", "Laboratory", "Observatory"},
		StationTypeMining:   {"Excavator", "Harvester", "Extractor", "Drill"},
		StationTypeRefinery: {"Processor", "Refinery", "Converter", "Smelter"},
		StationTypeShipyard: {"Forge", "Constructor", "Shipwright", "Assembly"},
	}

	suffixes := []string{"Alpha", "Beta", "Prime", "One", "Central", "Hub", "Station", "Complex"}

	typePrefix := prefixes[stationType]
	if len(typePrefix) == 0 {
		typePrefix = []string{"Station"}
	}

	prefix := typePrefix[rand.Intn(len(typePrefix))]
	suffix := suffixes[rand.Intn(len(suffixes))]

	return fmt.Sprintf("%s %s", prefix, suffix)
}

// generateStationServices creates services based on station type
func generateStationServices(stationType string) []string {
	baseServices := []string{"Docking", "Fuel", "Repairs"}

	typeServices := map[string][]string{
		StationTypeTrading:  {"Trading Post", "Cargo Storage", "Market Access", "Banking"},
		StationTypeMilitary: {"Weapon Systems", "Fleet Command", "Intelligence", "Training"},
		StationTypeResearch: {"Laboratory", "Data Analysis", "Prototype Testing", "Academic Library"},
		StationTypeMining:   {"Ore Processing", "Equipment Rental", "Surveying", "Cargo Transport"},
		StationTypeRefinery: {"Material Processing", "Quality Control", "Chemical Analysis", "Waste Management"},
		StationTypeShipyard: {"Ship Construction", "Upgrades", "Design Bureau", "Component Manufacturing"},
	}

	services := make([]string, len(baseServices))
	copy(services, baseServices)

	if typeSpecific, exists := typeServices[stationType]; exists {
		// Add 2-4 type-specific services
		count := 2 + rand.Intn(3)
		if count > len(typeSpecific) {
			count = len(typeSpecific)
		}

		for i := 0; i < count; i++ {
			services = append(services, typeSpecific[i])
		}
	}

	return services
}

// generateTradeGoods creates trade goods based on station type
func generateTradeGoods(stationType string) []string {
	commonGoods := []string{"Food", "Water", "Basic Components"}

	typeGoods := map[string][]string{
		StationTypeTrading:  {"Luxury Items", "Electronics", "Textiles", "Spices", "Art"},
		StationTypeMilitary: {"Weapons", "Armor", "Military Supplies", "Ammunition"},
		StationTypeResearch: {"Scientific Equipment", "Data Cores", "Rare Elements", "Prototypes"},
		StationTypeMining:   {"Raw Ore", "Precious Metals", "Industrial Minerals", "Crystals"},
		StationTypeRefinery: {"Refined Metals", "Chemicals", "Fuel", "Alloys", "Plastics"},
		StationTypeShipyard: {"Ship Components", "Engines", "Hull Plates", "Navigation Systems"},
	}

	goods := make([]string, 0)

	// Add 1-2 common goods
	commonCount := 1 + rand.Intn(2)
	for i := 0; i < commonCount && i < len(commonGoods); i++ {
		goods = append(goods, commonGoods[i])
	}

	// Add type-specific goods
	if typeSpecific, exists := typeGoods[stationType]; exists {
		count := 2 + rand.Intn(3)
		if count > len(typeSpecific) {
			count = len(typeSpecific)
		}

		used := make(map[int]bool)
		for len(goods)-commonCount < count {
			idx := rand.Intn(len(typeSpecific))
			if !used[idx] {
				goods = append(goods, typeSpecific[idx])
				used[idx] = true
			}
		}
	}

	return goods
}

// generateDefenseLevel creates defense level based on station type
func generateDefenseLevel(stationType string) int {
	switch stationType {
	case StationTypeMilitary:
		return 8 + rand.Intn(3) // 8-10
	case StationTypeShipyard:
		return 6 + rand.Intn(3) // 6-8
	case StationTypeRefinery:
		return 4 + rand.Intn(3) // 4-6
	case StationTypeTrading:
		return 3 + rand.Intn(3) // 3-5
	case StationTypeMining:
		return 2 + rand.Intn(3) // 2-4
	case StationTypeResearch:
		return 1 + rand.Intn(3) // 1-3
	default:
		return 3 + rand.Intn(3) // 3-5
	}
}

// IsHostile returns whether the station is hostile to players
func (s *SpaceStation) IsHostile() bool {
	return s.StationType == StationTypeMilitary && s.Owner == "Military Corp"
}

// CanDock returns whether a player can dock at this station
func (s *SpaceStation) CanDock() bool {
	return s.CurrentPop < s.Capacity && !s.IsHostile()
}

// GetDockingFee returns the fee for docking at this station
func (s *SpaceStation) GetDockingFee() int {
	baseFee := 100

	switch s.StationType {
	case StationTypeTrading:
		return baseFee + rand.Intn(50)
	case StationTypeMilitary:
		return baseFee * 2 // Military stations charge more
	case StationTypeResearch:
		return baseFee / 2 // Research stations are cheaper
	case StationTypeShipyard:
		return baseFee + rand.Intn(100)
	default:
		return baseFee + rand.Intn(25)
	}
}
