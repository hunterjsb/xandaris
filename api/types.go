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
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	X         float64  `json:"x"`
	Y         float64  `json:"y"`
	StarType  string   `json:"star_type"`
	Planets   int      `json:"planets"`
	Links     []int    `json:"links,omitempty"`
	Owner     string   `json:"owner,omitempty"`    // faction that owns a planet here
	Resources []string `json:"resources,omitempty"` // resource types available
}

// PlanetDetail is the detailed planet endpoint representation.
type PlanetDetail struct {
	ID                int                `json:"id"`
	Name              string             `json:"name"`
	PlanetType        string             `json:"planet_type"`
	Population        int64              `json:"population"`
	PopulationCap     int64              `json:"population_cap"`
	Habitability      int                `json:"habitability"`
	Happiness         float64            `json:"happiness"`          // 0.0-1.0
	ProductivityBonus float64            `json:"productivity_bonus"` // 0.5-1.5
	TechLevel         float64            `json:"tech_level"`         // 0.0-5.0
	Owner             string             `json:"owner,omitempty"`
	StoredResources   map[string]int     `json:"stored_resources"`
	ResourceDeposits  []ResourceDeposit  `json:"resource_deposits"`
	Buildings         []BuildingInfo     `json:"buildings"`
	SystemID          int                `json:"system_id"`
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
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Credits    int    `json:"credits"`
	Planets    int    `json:"planets"`
	Ships      int    `json:"ships"`
	Fleets     int    `json:"fleets"`
	Mines      int    `json:"mines"`
	Buildings  int    `json:"buildings"`
	Population int64  `json:"population"`
	Stock      int    `json:"stock"`
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
	CargoUsed      int            `json:"cargo_used"`
	CargoMax       int            `json:"cargo_max"`
	CargoHold      map[string]int `json:"cargo_hold"`
	TravelProgress float64        `json:"travel_progress"` // 0.0-1.0 for moving ships
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
	BuildingType string `json:"building_type"` // "Mine", "Trading Post", "Refinery", "Factory", "Habitat", "Shipyard"
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

// ColonizeRequest is the body for POST /api/colonize.
type ColonizeRequest struct {
	ShipID   int `json:"ship_id"`
	PlanetID int `json:"planet_id"`
}

// UpgradeRequest is the body for POST /api/upgrade.
type UpgradeRequest struct {
	PlanetID      int `json:"planet_id"`
	BuildingIndex int `json:"building_index"` // index in the buildings array
}

// CancelConstructionRequest is the body for POST /api/construction/cancel.
type CancelConstructionRequest struct {
	ConstructionID string `json:"construction_id"`
}

// FleetMoveRequest is the body for POST /api/fleets/move.
type FleetMoveRequest struct {
	FleetID        int `json:"fleet_id"`
	TargetSystemID int `json:"target_system_id"`
}

// FleetCreateRequest is the body for POST /api/fleets/create.
type FleetCreateRequest struct {
	ShipID int `json:"ship_id"` // ship to promote to a fleet
}

// FleetDisbandRequest is the body for POST /api/fleets/disband.
type FleetDisbandRequest struct {
	FleetID int `json:"fleet_id"`
}

// FleetAddShipRequest is the body for POST /api/fleets/add-ship.
type FleetAddShipRequest struct {
	ShipID  int `json:"ship_id"`
	FleetID int `json:"fleet_id"`
}

// FleetRemoveShipRequest is the body for POST /api/fleets/remove-ship.
type FleetRemoveShipRequest struct {
	ShipID  int `json:"ship_id"`
	FleetID int `json:"fleet_id"`
}

// CatalogBuilding describes an available building type.
type CatalogBuilding struct {
	Type           string         `json:"type"`
	Description    string         `json:"description"`
	Cost           int            `json:"cost"`
	MaxLevel       int            `json:"max_level"`
	Workers        int            `json:"workers"`
	CreditUpkeep   int            `json:"credit_upkeep"`
	ResourceUpkeep map[string]int `json:"resource_upkeep"`
	Produces       map[string]int `json:"produces,omitempty"`  // resources produced per interval
	Consumes       map[string]int `json:"consumes,omitempty"`  // resources consumed for production (not upkeep)
}

// CatalogShip describes an available ship type.
type CatalogShip struct {
	Type         string         `json:"type"`
	Cost         int            `json:"cost"`
	BuildTime    int            `json:"build_time"`
	Resources    map[string]int `json:"resources"`
	MaxFuel      int            `json:"max_fuel"`
	MaxCargo     int            `json:"max_cargo"`
	MaxHealth    int            `json:"max_health"`
}

// PopConsumptionRate describes per-population resource consumption.
type PopConsumptionRate struct {
	Resource      string  `json:"resource"`
	PerPopulation float64 `json:"per_population"` // units consumed
	PopDivisor    float64 `json:"pop_divisor"`     // per this many population
}

// LeaderboardEntry ranks a player by empire score.
type LeaderboardEntry struct {
	Rank       int    `json:"rank"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Score      int    `json:"score"`
	Credits    int    `json:"credits"`
	Population int64  `json:"population"`
	Planets    int    `json:"planets"`
	Ships      int    `json:"ships"`
	Buildings  int    `json:"buildings"`
	StockValue int    `json:"stock_value"`
}

// CatalogResource describes a tradeable resource.
type CatalogResource struct {
	Name      string  `json:"name"`
	BasePrice float64 `json:"base_price"`
	Source    string  `json:"source"` // "mining", "refining", "manufacturing"
}

// Catalog lists all available buildings, ships, and resources with costs.
type Catalog struct {
	Buildings             []CatalogBuilding    `json:"buildings"`
	Ships                 []CatalogShip        `json:"ships"`
	Resources             []CatalogResource    `json:"resources"`
	PopulationConsumption []PopConsumptionRate  `json:"population_consumption"`
}

// GalaxyFlows shows galaxy-wide aggregate production and consumption rates.
type GalaxyFlows struct {
	Production  map[string]float64 `json:"production"`  // total production per interval
	Consumption map[string]float64 `json:"consumption"` // total consumption per interval
	NetFlow     map[string]float64 `json:"net_flow"`     // production - consumption
	Population  int64              `json:"population"`
}

// WorkforceInfo shows workforce allocation for a planet.
type WorkforceInfo struct {
	PlanetID       int              `json:"planet_id"`
	PlanetName     string           `json:"planet_name"`
	Population     int64            `json:"population"`
	PopulationCap  int64            `json:"population_cap"`
	WorkforceTotal int64            `json:"workforce_total"`
	WorkforceUsed  int64            `json:"workforce_used"`
	WorkforceFree  int64            `json:"workforce_free"`
	Buildings      []WorkforceEntry `json:"buildings"`
}

// WorkforceEntry shows workforce allocation for a single building.
type WorkforceEntry struct {
	Index    int     `json:"index"`
	Type     string  `json:"type"`
	Level    int     `json:"level"`
	Assigned int64   `json:"assigned"`
	Required int64   `json:"required"`
	Staffing float64 `json:"staffing"` // 0.0-1.0
	Online   bool    `json:"online"`
}

// WorkforceAssignRequest is the body for POST /api/workforce/assign.
type WorkforceAssignRequest struct {
	PlanetID      int `json:"planet_id"`
	BuildingIndex int `json:"building_index"`
	Workers       int `json:"workers"` // -1 = auto, 0 = disable, N = set target
}

// SystemPrices shows local prices for a system (for trade route planning).
type SystemPrices struct {
	SystemID   int                `json:"system_id"`
	SystemName string             `json:"system_name"`
	Prices     map[string]float64 `json:"prices"` // resource -> local buy price
}

// PlanetRates shows production and consumption for a planet.
type PlanetRates struct {
	PlanetID          int                `json:"planet_id"`
	PlanetName        string             `json:"planet_name"`
	Population        int64              `json:"population"`
	Happiness         float64            `json:"happiness"`
	ProductivityBonus float64            `json:"productivity_bonus"`
	TechLevel         float64            `json:"tech_level"`
	Production        map[string]float64 `json:"production"`
	Consumption       map[string]float64 `json:"consumption"`
	NetFlow           map[string]float64 `json:"net_flow"`
}

// GameStatus is a comprehensive snapshot for agents — everything needed in one call.
type GameStatus struct {
	Tick        int64           `json:"tick"`
	GameTime    string          `json:"game_time"`
	Speed       string          `json:"speed"`
	Paused      bool            `json:"paused"`
	Player      PlayerStatus    `json:"player"`
	Economy     EconomyOverview `json:"economy"`
	Hints       []string        `json:"hints"` // actionable suggestions
}

// PlayerStatus is the human player's state in the GameStatus response.
type PlayerStatus struct {
	Name    string         `json:"name"`
	Credits int            `json:"credits"`
	Planets []PlanetBrief  `json:"planets"`
	Ships   int            `json:"ships"`
}

// PlanetBrief is a compact planet summary for GameStatus.
type PlanetBrief struct {
	ID         int            `json:"id"`
	Name       string         `json:"name"`
	SystemID   int            `json:"system_id"`
	Population int64          `json:"population"`
	Storage    map[string]int `json:"storage"`
	Buildings  int            `json:"buildings"`
	Mines      int            `json:"mines"`
}

// EconomyOverview is a galaxy-wide economic summary.
type EconomyOverview struct {
	TotalPopulation int64                      `json:"total_population"`
	TotalCredits    int                        `json:"total_credits"`
	TradeVolume     float64                    `json:"trade_volume"` // recent trade activity
	Resources       map[string]ResourceSummary `json:"resources"`
}

// ResourceSummary aggregates supply/demand data for one resource.
type ResourceSummary struct {
	TotalSupply  int       `json:"total_supply"`
	BuyPrice     float64   `json:"buy_price"`
	SellPrice    float64   `json:"sell_price"`
	BasePrice    float64   `json:"base_price"`
	Demand       float64   `json:"demand"`
	Trend        float64   `json:"trend"`
	ImportFee    float64   `json:"import_fee"`
	Scarcity     string    `json:"scarcity"`
	PriceRatio   float64   `json:"price_ratio"`
	PriceHistory []float64 `json:"price_history,omitempty"` // last 100 mid prices
}
