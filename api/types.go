package api

// APIResponse wraps all API responses.
type APIResponse struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// MarketCommodity represents a single commodity in the market.
type MarketCommodity struct {
	Resource      string  `json:"resource"`
	BasePrice     float64 `json:"base_price"`
	CurrentPrice  float64 `json:"current_price"`
	BuyPrice      float64 `json:"buy_price"`
	SellPrice     float64 `json:"sell_price"`
	TotalSupply   float64 `json:"total_supply"`
	TotalDemand   float64 `json:"total_demand"`
	PriceVelocity float64 `json:"price_velocity"`
}

// TradeRequest is the body for POST /api/market/trade.
type TradeRequest struct {
	Resource string `json:"resource"`
	Quantity int    `json:"quantity"`
	Action   string `json:"action"`    // "buy" or "sell"
	PlanetID int    `json:"planet_id"` // optional: specific planet for the trade
}

// CargoRequest is the body for POST /api/cargo/load and /api/cargo/unload.
type CargoRequest struct {
	ShipID   int    `json:"ship_id"`
	PlanetID int    `json:"planet_id"`
	Resource string `json:"resource"`
	Quantity int    `json:"quantity"`
}

// CargoResult is the response for a successful cargo operation.
type CargoResult struct {
	ShipID   int    `json:"ship_id"`
	PlanetID int    `json:"planet_id"`
	Resource string `json:"resource"`
	Quantity int    `json:"quantity"`
	Action   string `json:"action"` // "load" or "unload"
}

// TradeResult is the response for a successful trade.
type TradeResult struct {
	Resource string `json:"resource"`
	Quantity int    `json:"quantity"`
	Action   string `json:"action"`
	Total    int    `json:"total"`
}

// SystemSummary is a compact system representation for the galaxy endpoint.
type SystemSummary struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	X        float64   `json:"x"`
	Y        float64   `json:"y"`
	StarType string    `json:"star_type"`
	Planets  int       `json:"planets"`
	Links    []int     `json:"links,omitempty"`
}

// PlanetDetail is the detailed planet endpoint representation.
type PlanetDetail struct {
	ID              int                `json:"id"`
	Name            string             `json:"name"`
	PlanetType      string             `json:"planet_type"`
	Population      int64              `json:"population"`
	PopulationCap   int64              `json:"population_cap"`
	Habitability    int                `json:"habitability"`
	Owner           string             `json:"owner,omitempty"`
	StoredResources map[string]int     `json:"stored_resources"`
	ResourceDeposits []ResourceDeposit `json:"resource_deposits"`
	Buildings       []BuildingInfo     `json:"buildings"`
	SystemID        int                `json:"system_id"`
}

// ResourceDeposit is a minable resource node on a planet.
type ResourceDeposit struct {
	ID             int     `json:"id"`
	ResourceType   string  `json:"resource_type"`
	Abundance      int     `json:"abundance"`
	ExtractionRate float64 `json:"extraction_rate"`
	HasMine        bool    `json:"has_mine"`
}

// BuildingInfo represents a building with its operational details.
type BuildingInfo struct {
	Index         int     `json:"index"` // position in buildings array (for upgrade API)
	Type          string  `json:"type"`
	Level         int     `json:"level"`
	MaxLevel      int     `json:"max_level"`
	IsOperational bool    `json:"is_operational"`
	Staffing      float64 `json:"staffing"`     // 0.0-1.0
	UpgradeCost   int     `json:"upgrade_cost"` // 0 if max level
}

// ConstructionQueueItem represents an item being constructed.
type ConstructionQueueItem struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Location       string `json:"location"`
	Progress       int    `json:"progress"` // 0-100
	RemainingTicks int    `json:"remaining_ticks"`
	TotalTicks     int    `json:"total_ticks"`
}

// PlayerInfo represents a player in the directory endpoint.
type PlayerInfo struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Credits  int      `json:"credits"`
	Planets  int      `json:"planets"`
	Ships    int      `json:"ships"`
	Fleets   int      `json:"fleets"`
}

// GameInfo represents the game state endpoint.
type GameInfo struct {
	Tick      int64   `json:"tick"`
	GameTime  string  `json:"game_time"`
	Speed     string  `json:"speed"`
	Paused    bool    `json:"paused"`
	Systems   int     `json:"systems"`
	Players   int     `json:"players"`
	Seed      int64   `json:"seed"`
}

// SpeedRequest is the body for POST /api/game/speed.
type SpeedRequest struct {
	Speed string `json:"speed"` // "slow", "normal", "fast", "very_fast"
}

// TradeHistoryEntry is a trade record for the API.
type TradeHistoryEntry struct {
	Tick      int64   `json:"tick"`
	Player    string  `json:"player"`
	Resource  string  `json:"resource"`
	Quantity  int     `json:"quantity"`
	Action    string  `json:"action"`
	UnitPrice float64 `json:"unit_price"`
	Total     int     `json:"total"`
}

// ShipInfo represents a ship for the API.
type ShipInfo struct {
	ID            int            `json:"id"`
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	Owner         string         `json:"owner"`
	Status        string         `json:"status"`
	SystemID      int            `json:"system_id"`
	TargetSystem  int            `json:"target_system"`
	FuelCurrent   int            `json:"fuel_current"`
	FuelMax       int            `json:"fuel_max"`
	HealthCurrent int            `json:"health_current"`
	HealthMax     int            `json:"health_max"`
	CargoUsed     int            `json:"cargo_used"`
	CargoMax      int            `json:"cargo_max"`
	CargoHold     map[string]int `json:"cargo_hold"`
}

// FleetInfo represents a fleet for the API.
type FleetInfo struct {
	ID    int        `json:"id"`
	Owner string     `json:"owner"`
	Size  int        `json:"size"`
	Ships []ShipInfo `json:"ships"`
}

// PlanetStorageInfo gives detailed storage for a planet.
type PlanetStorageInfo struct {
	Resource string `json:"resource"`
	Amount   int    `json:"amount"`
	Capacity int    `json:"capacity"`
}

// BuildRequest is the body for POST /api/build.
type BuildRequest struct {
	PlanetID     int    `json:"planet_id"`
	BuildingType string `json:"building_type"` // "Mine", "Trading Post", "Refinery", "Habitat", "Shipyard"
	ResourceID   int    `json:"resource_id"`   // for mines: which resource node
}

// ShipBuildRequest is the body for POST /api/ships/build.
type ShipBuildRequest struct {
	PlanetID int    `json:"planet_id"`
	ShipType string `json:"ship_type"` // "Scout", "Cargo", "Colony", etc.
}

// ShipMoveRequest is the body for POST /api/ships/move.
type ShipMoveRequest struct {
	ShipID         int `json:"ship_id"`
	TargetSystemID int `json:"target_system_id"`
}

// ShipRefuelRequest is the body for POST /api/ships/refuel.
type ShipRefuelRequest struct {
	ShipID   int `json:"ship_id"`
	PlanetID int `json:"planet_id"`
	Amount   int `json:"amount"` // 0 = fill up
}

// UpgradeRequest is the body for POST /api/upgrade.
type UpgradeRequest struct {
	PlanetID      int `json:"planet_id"`
	BuildingIndex int `json:"building_index"` // index in the buildings array
}

// PlanetRates shows production and consumption for a planet.
type PlanetRates struct {
	PlanetID    int                    `json:"planet_id"`
	PlanetName  string                 `json:"planet_name"`
	Population  int64                  `json:"population"`
	Production  map[string]float64     `json:"production"`  // resource -> units/interval
	Consumption map[string]float64     `json:"consumption"` // resource -> units/interval
	NetFlow     map[string]float64     `json:"net_flow"`    // production - consumption
}

// EconomyOverview is a galaxy-wide economic summary.
type EconomyOverview struct {
	TotalPopulation int64                      `json:"total_population"`
	TotalCredits    int                        `json:"total_credits"`
	Resources       map[string]ResourceSummary `json:"resources"`
}

// ResourceSummary aggregates supply/demand data for one resource.
type ResourceSummary struct {
	TotalSupply  int     `json:"total_supply"`
	BuyPrice     float64 `json:"buy_price"`
	SellPrice    float64 `json:"sell_price"`
	BasePrice    float64 `json:"base_price"`
	Demand       float64 `json:"demand"`
	Trend        float64 `json:"trend"`
	ImportFee    float64 `json:"import_fee"` // dynamic fee rate (0.05-0.20)
}
