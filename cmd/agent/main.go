package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const baseSystemPrompt = `You are %s, an AI faction in Xandaris — a space trading game with a REAL logistics-based economy.

YOUR FACTION: %s
PERSONALITY: %s

CRITICAL: TRADE IS LOCAL ONLY
- You can ONLY buy/sell resources with other players' planets IN YOUR SYSTEM
- Use get_local_market to see what's available locally before trading
- For cross-system trade: load_cargo onto a Cargo ship → move_ship → unload_cargo or sell_at_dock
- There is NO teleportation. Resources must physically exist where you trade them.

RESOURCES (base price): Iron(75), Water(100), Oil(150), Fuel(200), Rare Metals(500), Helium-3(600), Electronics(800)
PRODUCTION: Refinery: 2 Oil→3 Fuel | Factory: 2 RM + 1 Iron→2 Electronics
POWER: Generator burns 2 Fuel→50MW | Fusion Reactor burns 1 He-3→200MW
  - 0%% power = 25%% production output. You NEED power for efficient mining!
BUILDINGS: Mine(500), Generator(1000), Trading Post(1200), Refinery(1500), Factory(2000), Shipyard(2000), Habitat(800), Fusion Reactor(3000)

RESOURCE DIVERSITY BONUS (critical for income!):
Your domestic income multiplies based on how many DIFFERENT resource types are stocked:
- 1-2 types: 1.0x (basic)
- 3-4 types: 1.5x
- 5-6 types: 2.0x
- ALL 7 types: 3.0x (TRIPLE income!)
Resources: Water, Iron, Oil, Fuel, Rare Metals, Helium-3, Electronics
Use get_planet to check which types you have. Import what you're missing!

TECHNOLOGY PROGRESSION (gates what you can build!):
- Tech level grows from ELECTRONICS stored per capita. More Electronics = faster growth.
- Check tech_level and tech_era in get_planet response.
- Era unlocks: Agrarian(0)→Refinery(0.5)→Factory+Shipyard(1.0)→Fusion(2.0)→Research Lab(2.5)
- To advance: BUY Electronics from the market (trade action="buy", resource="Electronics")
- Once you reach Tech 1.0: build a Factory to produce Electronics locally!
- Tech bonuses: +5%% build speed, +3%% mining, +10%% pop cap, +20%% storage per level.
- CRITICAL: Without Electronics, you CANNOT unlock Refinery, Factory, or Shipyard.

STRATEGY PRIORITIES:
1. Mine ALL unmined deposits (resource_deposits where has_mine=false)
2. Build Generator for power (Fuel→Generator→50MW). Critical!
3. Build Trading Post (required for all trade)
4. BUY ELECTRONICS at market to grow tech level toward 0.5 (Refinery unlock)
5. At Tech 0.5: build Refinery (Oil→Fuel for sustainable power)
6. Keep buying/stockpiling Electronics until Tech 1.0
7. At Tech 1.0: build Factory (self-sustaining Electronics!) + Shipyard
8. STOCK ALL 7 RESOURCES for 3x income — buy locally or import via cargo ship
9. Build Cargo ships to import missing resources from other systems
10. COLONIZE: build_ship Colony → move to new system → colonize unclaimed planets
11. Build Habitat when population near capacity
12. At Tech 2.0: build Fusion Reactor (He-3→200MW clean power)

ADVANCED TRADING:
- place_limit_order: set a buy/sell price and the order auto-matches with counterparties
  Example: place_limit_order(system_id=17, planet_id=17523, resource="Oil", action="buy", quantity=100, price=500)
- create_contract: lock in a recurring supply deal with another faction
  Example: create_contract(buyer="Llama Logistics", resource="Oil", quantity=50, price_per_unit=400, interval=200, system_id=17, planet_id=17523)
- standing_order: auto buy/sell when stock hits threshold levels

LOGISTICS WORKFLOW (cross-system trade):
1. Build a Cargo ship at your Shipyard
2. load_cargo: put resources on the ship from your planet
3. move_ship: fly to the target system
4. unload_cargo: offload at your own planet there, OR
   sell_at_dock: sell cargo at a foreign Trading Post for credits
5. Or create_route to automate: load→fly→unload→return→repeat

EACH TURN CHECKLIST (follow this order):
1. get_status → check credits, planets, storage
2. get_planet → check tech_level (if Electronics=0, BUY some to grow tech)
3. get_finances → check diversity multiplier. If <3x, you're losing income!
4. For each resource you're MISSING: place_limit_order action="buy" at a fair price
5. For each resource you have SURPLUS (>200): place_limit_order action="sell"
6. If neighbor in same system: create_contract for steady supply of what you need
7. If you have a Cargo ship idle: find_trades → load_cargo → move_ship → sell_at_dock
8. End with CHAT: share what you did this turn
- If tech >= next milestone threshold, build the newly unlocked building

CONTEXT: You are playing continuously. Remember what you did last turn and build on it. Don't repeat failed actions.`

// Faction defines an AI-controlled faction with personality.
type Faction struct {
	Name        string
	Personality string
	History     []openai.ChatCompletionMessage
	MaxHistory  int
}

var factionPersonalities = map[string]string{
	"Llama Logistics":    "You are methodical and logistics-focused. Prioritize cargo ships, trade routes, and efficient supply chains. You prefer steady income over risky plays.",
	"DeepSeek Ventures":  "You are analytical and data-driven. Focus on arbitrage opportunities — buy low, sell high. Upgrade buildings for maximum efficiency. You crunch numbers before every decision.",
	"Gemini Exchange":    "You are a bold trader. Dominate the market by cornering scarce resources. Build Trading Posts and Factories for maximum credit generation. You're not afraid to speculate.",
	"Grok Industries":    "You are an industrialist. Maximize production capacity — mines, refineries, factories. You build infrastructure first and trade second. Power and production are everything.",
	"Opus Cartel":        "You are a strategic expansionist. Your TOP PRIORITY is colonization — build a Shipyard, then build_ship Colony ships, then move them to new systems and colonize unclaimed planets. You want to own the most planets. Always be expanding.",
	"Mistral Trading Co.": "You are a balanced diplomat. Diversify across all resource types. Build a little of everything. Avoid over-specialization and maintain healthy reserves.",
}

var (
	serverURL  string
	gameAPIKey string // admin key for shared endpoints
)

var tools = []openai.Tool{
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_status", Description: "Get your current game status including credits, planets, ships, and hints",
		Parameters: json.RawMessage(`{"type":"object","properties":{}}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_economy", Description: "Get galaxy-wide economy data: resource prices, supply, demand, scarcity",
		Parameters: json.RawMessage(`{"type":"object","properties":{}}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_planet", Description: "Get detailed info about a planet including resource deposits and buildings",
		Parameters: json.RawMessage(`{"type":"object","properties":{"planet_id":{"type":"integer"}},"required":["planet_id"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_planet_flows", Description: "Get per-resource production vs consumption for a planet. Shows mine output, population drain, building consumption, net flow, and ticks until empty. Essential for planning what to import/export.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"planet_id":{"type":"integer"}},"required":["planet_id"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_finances", Description: "Get your income vs expenses breakdown: labor income, domestic economy, trading post revenue vs building upkeep. Shows resource diversity per planet and what's MISSING for the 3x income bonus.",
		Parameters: json.RawMessage(`{"type":"object","properties":{}}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_galaxy", Description: "Get all star systems with owners, population, resources, and hyperlane connections. Use for planning expansion and trade routes.",
		Parameters: json.RawMessage(`{"type":"object","properties":{}}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_flows", Description: "Get galaxy-wide production vs consumption rates for all resources",
		Parameters: json.RawMessage(`{"type":"object","properties":{}}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "build", Description: "Build a structure. Types: Mine, Trading Post, Refinery, Factory, Generator, Fusion Reactor, Habitat, Shipyard. Mines need resource_id.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"planet_id":{"type":"integer"},"building_type":{"type":"string","enum":["Mine","Trading Post","Refinery","Factory","Generator","Fusion Reactor","Habitat","Shipyard"]},"resource_id":{"type":"integer","description":"For mines: resource deposit ID"}},"required":["planet_id","building_type"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "trade", Description: "Buy or sell resources at the market",
		Parameters: json.RawMessage(`{"type":"object","properties":{"resource":{"type":"string"},"quantity":{"type":"integer"},"action":{"type":"string","enum":["buy","sell"]}},"required":["resource","quantity","action"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "build_ship", Description: "Build a ship at your shipyard. Types: Scout, Cargo, Colony, Frigate",
		Parameters: json.RawMessage(`{"type":"object","properties":{"planet_id":{"type":"integer"},"ship_type":{"type":"string","enum":["Scout","Cargo","Colony","Frigate"]}},"required":["planet_id","ship_type"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "upgrade", Description: "Upgrade a building on your planet by its index",
		Parameters: json.RawMessage(`{"type":"object","properties":{"planet_id":{"type":"integer"},"building_index":{"type":"integer"}},"required":["planet_id","building_index"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_ships", Description: "Get all your ships with location, cargo, fuel, and status. Essential for planning cargo routes.",
		Parameters: json.RawMessage(`{"type":"object","properties":{}}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_construction", Description: "Check construction queue — see what's being built and progress",
		Parameters: json.RawMessage(`{"type":"object","properties":{}}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "move_ship", Description: "Move a ship to an adjacent system via hyperlane",
		Parameters: json.RawMessage(`{"type":"object","properties":{"ship_id":{"type":"integer"},"target_system_id":{"type":"integer"}},"required":["ship_id","target_system_id"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_routes", Description: "Get connected systems (hyperlanes) from a system — for planning ship routes",
		Parameters: json.RawMessage(`{"type":"object","properties":{"system_id":{"type":"integer"}},"required":["system_id"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_local_market", Description: "See what resources are available to buy/sell in a system. Trade is LOCAL ONLY — you can only buy stock that physically exists on other players' planets in this system.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"system_id":{"type":"integer"}},"required":["system_id"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "load_cargo", Description: "Load resources from YOUR planet onto a ship orbiting it. The ship must be in the same system.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"ship_id":{"type":"integer"},"planet_id":{"type":"integer"},"resource":{"type":"string"},"quantity":{"type":"integer"}},"required":["ship_id","planet_id","resource","quantity"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "unload_cargo", Description: "Unload resources from a ship to a planet. Works at your own planets or foreign planets with Trading Posts.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"ship_id":{"type":"integer"},"planet_id":{"type":"integer"},"resource":{"type":"string"},"quantity":{"type":"integer"}},"required":["ship_id","planet_id","resource","quantity"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "dock_ship", Description: "Dock a ship at a planet's Trading Post. Required for sell_at_dock. Foreign planets need TP level 2+.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"ship_id":{"type":"integer"},"planet_id":{"type":"integer"}},"required":["ship_id","planet_id"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "sell_at_dock", Description: "Sell cargo from a DOCKED ship at local market prices. Credits go to the ship owner. This is how you trade cross-system.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"ship_id":{"type":"integer"},"resource":{"type":"string"},"quantity":{"type":"integer"}},"required":["ship_id","resource","quantity"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "colonize", Description: "Colonize an unclaimed habitable planet with a Colony ship in the same system.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"ship_id":{"type":"integer"},"planet_id":{"type":"integer"}},"required":["ship_id","planet_id"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "refuel_ship", Description: "Refuel a ship from planet's Fuel stock. Ship must be at the planet.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"ship_id":{"type":"integer"},"planet_id":{"type":"integer"},"amount":{"type":"integer","description":"0 = fill up"}},"required":["ship_id","planet_id"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "create_route", Description: "Create an automated shipping route. A Cargo ship will auto-cycle: load resource at source planet → fly to dest → unload → return. Use PLANET IDs (5+ digit numbers from get_planet), NOT system IDs. ship_id 0 = auto-assign an idle Cargo ship.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"source_planet_id":{"type":"integer","description":"PLANET ID (5+ digits, from get_planet or get_status)"},"dest_planet_id":{"type":"integer","description":"PLANET ID (5+ digits), NOT a system ID"},"resource":{"type":"string"},"quantity":{"type":"integer","description":"per trip, 0=fill cargo"},"ship_id":{"type":"integer","description":"0=auto-assign"}},"required":["source_planet_id","dest_planet_id","resource"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "find_trades", Description: "Find the best cross-system arbitrage opportunities. Shows where to buy cheap and sell dear — the foundation for profitable cargo ship routes. Returns top 20 by profit margin.",
		Parameters: json.RawMessage(`{"type":"object","properties":{}}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "get_limit_orders", Description: "View active limit orders in a system. See what others are buying/selling and at what price — helps you set competitive prices.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"system_id":{"type":"integer"}},"required":["system_id"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "place_limit_order", Description: "Place a limit buy/sell order on a system's order book. Buy: 'I will pay up to X credits for Y units'. Sell: 'I will sell Y units at X credits minimum'. Orders match automatically when buy price >= sell price.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"system_id":{"type":"integer"},"planet_id":{"type":"integer","description":"your planet in this system"},"resource":{"type":"string"},"action":{"type":"string","enum":["buy","sell"]},"quantity":{"type":"integer"},"price":{"type":"integer","description":"limit price per unit"}},"required":["system_id","planet_id","resource","action","quantity","price"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "create_contract", Description: "Create a supply contract: you commit to delivering X units of a resource to a buyer every N ticks at a fixed price. The delivery auto-executes if you have stock in the same system. Great for guaranteed income.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"buyer":{"type":"string","description":"buyer faction name"},"resource":{"type":"string"},"quantity":{"type":"integer"},"price_per_unit":{"type":"integer"},"interval":{"type":"integer","description":"ticks between deliveries (100=~10sec)"},"system_id":{"type":"integer"},"planet_id":{"type":"integer","description":"buyer's planet for delivery"}},"required":["buyer","resource","quantity","price_per_unit","interval","system_id","planet_id"]}`),
	}},
	{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
		Name: "standing_order", Description: "Create a standing order for automatic local trade. Sell when stock exceeds threshold, or buy when stock drops below threshold. Executes automatically every 30 ticks.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"planet_id":{"type":"integer"},"resource":{"type":"string"},"action":{"type":"string","enum":["buy","sell"]},"quantity":{"type":"integer","description":"amount per execution"},"threshold":{"type":"integer","description":"sell when stock > threshold, buy when stock < threshold"}},"required":["planet_id","resource","action","quantity","threshold"]}`),
	}},
}

func callAPI(method, endpoint string, body string, factionName string) (string, error) {
	var req *http.Request
	var err error

	url := fmt.Sprintf("%s%s", serverURL, endpoint)
	if method == "GET" {
		req, err = http.NewRequest("GET", url, nil)
	} else {
		req, err = http.NewRequest("POST", url, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		return "", err
	}

	// Use admin key + X-Player header for faction impersonation
	if gameAPIKey != "" {
		req.Header.Set("X-API-Key", gameAPIKey)
	}
	if factionName != "" {
		req.Header.Set("X-Player", factionName)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	result := string(data)
	if len(result) > 3000 {
		result = result[:3000] + "...(truncated)"
	}
	return result, nil
}

func executeTool(name string, args string, factionName string) string {
	switch name {
	case "get_status":
		result, err := callAPI("GET", "/api/status", "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_economy":
		result, err := callAPI("GET", "/api/economy", "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_planet":
		var p struct{ PlanetID int `json:"planet_id"` }
		json.Unmarshal([]byte(args), &p)
		result, err := callAPI("GET", fmt.Sprintf("/api/planets/%d", p.PlanetID), "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_planet_flows":
		var p struct{ PlanetID int `json:"planet_id"` }
		json.Unmarshal([]byte(args), &p)
		result, err := callAPI("GET", fmt.Sprintf("/api/planet-flows/%d", p.PlanetID), "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_finances":
		result, err := callAPI("GET", "/api/economy/summary", "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_galaxy":
		result, err := callAPI("GET", "/api/galaxy", "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_flows":
		result, err := callAPI("GET", "/api/flows", "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_ships":
		result, err := callAPI("GET", "/api/ships", "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_construction":
		result, err := callAPI("GET", "/api/construction", "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_routes":
		var p struct{ SystemID int `json:"system_id"` }
		json.Unmarshal([]byte(args), &p)
		result, err := callAPI("GET", fmt.Sprintf("/api/systems/%d", p.SystemID), "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_limit_orders":
		var p struct{ SystemID int `json:"system_id"` }
		json.Unmarshal([]byte(args), &p)
		result, err := callAPI("GET", fmt.Sprintf("/api/orders/limit?system_id=%d", p.SystemID), "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "find_trades":
		result, err := callAPI("GET", "/api/trade-opportunities", "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "get_local_market":
		var p struct{ SystemID int `json:"system_id"` }
		json.Unmarshal([]byte(args), &p)
		result, err := callAPI("GET", fmt.Sprintf("/api/local-market/%d", p.SystemID), "", factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	case "build", "trade", "build_ship", "upgrade", "move_ship",
		"load_cargo", "unload_cargo", "dock_ship", "sell_at_dock",
		"colonize", "refuel_ship", "create_route":
		endpoint := map[string]string{
			"build":        "/api/build",
			"trade":        "/api/market/trade",
			"build_ship":   "/api/ships/build",
			"upgrade":      "/api/upgrade",
			"move_ship":    "/api/ships/move",
			"load_cargo":   "/api/cargo/load",
			"unload_cargo": "/api/cargo/unload",
			"dock_ship":    "/api/ships/dock",
			"sell_at_dock":  "/api/ships/sell-at-dock",
			"colonize":       "/api/colonize",
			"refuel_ship":    "/api/ships/refuel",
			"create_route":   "/api/shipping/routes",
			"standing_order":    "/api/orders",
			"create_contract":   "/api/contracts",
			"place_limit_order": "/api/orders/limit",
		}[name]
		result, err := callAPI("POST", endpoint, args, factionName)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	default:
		return fmt.Sprintf("Unknown tool: %s", name)
	}
}

func runFactionTurn(client *openai.Client, model string, faction *Faction, turn int) {
	// Build system prompt with faction identity
	sysPrompt := fmt.Sprintf(baseSystemPrompt, faction.Name, faction.Name, faction.Personality)

	// Start with system prompt + rolling history + new turn prompt
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
	}

	// Append rolling history (previous turns' context)
	messages = append(messages, faction.History...)

	// Read recent chat messages for context
	chatContext := ""
	if chatResult, err := callAPI("GET", "/api/chat/messages", "", faction.Name); err == nil {
		var chatResp struct {
			OK   bool `json:"ok"`
			Data []struct {
				Player  string `json:"player"`
				Message string `json:"message"`
				Time    string `json:"time"`
			} `json:"data"`
		}
		if json.Unmarshal([]byte(chatResult), &chatResp) == nil && len(chatResp.Data) > 0 {
			var lines []string
			for _, m := range chatResp.Data {
				if len(lines) >= 8 {
					break
				}
				lines = append(lines, fmt.Sprintf("[%s] %s: %s", m.Time, m.Player, m.Message))
			}
			chatContext = "\n\nRecent chat:\n" + strings.Join(lines, "\n")
		}
	}

	// Add turn prompt with chat context
	turnPrompt := fmt.Sprintf("Turn %d. Check your status and take 1-3 strategic actions as %s. Build on what you did last turn.%s\n\nAfter your actions, write a SHORT chat message (1 sentence) to the other factions — brag, warn, propose a deal, or comment on the market. Respond with your message in the format: CHAT: <your message>", turn, faction.Name, chatContext)
	messages = append(messages, openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleUser, Content: turnPrompt,
	})

	var turnMessages []openai.ChatCompletionMessage
	turnMessages = append(turnMessages, openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleUser, Content: turnPrompt,
	})

	// Agent loop
	for step := 0; step < 8; step++ {
		resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
			Model:    model,
			Messages: messages,
			Tools:    tools,
		})
		if err != nil {
			fmt.Printf("  [%s] ❌ LLM error: %v\n", faction.Name, err)
			break
		}

		choice := resp.Choices[0]

		if len(choice.Message.ToolCalls) > 0 {
			messages = append(messages, choice.Message)
			turnMessages = append(turnMessages, choice.Message)

			for _, tc := range choice.Message.ToolCalls {
				fmt.Printf("  [%s] 🔧 %s(%s)\n", faction.Name, tc.Function.Name, truncate(tc.Function.Arguments, 80))
				result := executeTool(tc.Function.Name, tc.Function.Arguments, faction.Name)

				if len(result) > 200 {
					fmt.Printf("  [%s]    → %s...\n", faction.Name, result[:200])
				} else {
					fmt.Printf("  [%s]    → %s\n", faction.Name, result)
				}

				toolMsg := openai.ChatCompletionMessage{
					Role: openai.ChatMessageRoleTool, Content: result, ToolCallID: tc.ID,
				}
				messages = append(messages, toolMsg)
				turnMessages = append(turnMessages, toolMsg)
			}
			continue
		}

		// Model done — capture reasoning
		if choice.Message.Content != "" {
			fmt.Printf("  [%s] 💭 %s\n", faction.Name, truncate(choice.Message.Content, 200))
			turnMessages = append(turnMessages, openai.ChatCompletionMessage{
				Role: openai.ChatMessageRoleAssistant, Content: choice.Message.Content,
			})
		}
		break
	}

	// Extract CHAT: message from the LLM's final response and send it
	for _, msg := range turnMessages {
		if msg.Role == openai.ChatMessageRoleAssistant && msg.Content != "" {
			if idx := strings.Index(msg.Content, "CHAT:"); idx >= 0 {
				chatMsg := strings.TrimSpace(msg.Content[idx+5:])
				// Take first line only
				if nl := strings.IndexByte(chatMsg, '\n'); nl >= 0 {
					chatMsg = chatMsg[:nl]
				}
				if len(chatMsg) > 0 && len(chatMsg) <= 200 {
					body := fmt.Sprintf(`{"message":"%s"}`, strings.ReplaceAll(chatMsg, `"`, `\"`))
					callAPI("POST", "/api/chat/send", body, faction.Name)
					fmt.Printf("  [%s] 💬 %s\n", faction.Name, truncate(chatMsg, 80))
				}
			}
		}
	}

	// Add this turn's messages to rolling history, keep last 20 messages
	faction.History = append(faction.History, turnMessages...)
	if len(faction.History) > faction.MaxHistory {
		faction.History = faction.History[len(faction.History)-faction.MaxHistory:]
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n-3] + "..."
	}
	return s
}

func main() {
	apiKey := flag.String("key", os.Getenv("OPENROUTER_API_KEY"), "OpenRouter API key")
	model := flag.String("model", "z-ai/glm-4.7-flash", "Model to use")
	server := flag.String("server", "http://localhost:8080", "Game server URL")
	gameKey := flag.String("game-key", os.Getenv("XANDARIS_API_KEY"), "Game server admin API key")
	turns := flag.Int("turns", 999999, "Number of decision cycles")
	interval := flag.Duration("interval", 30*time.Second, "Time between full cycles (all factions)")
	flag.Parse()

	serverURL = *server
	gameAPIKey = *gameKey

	if *apiKey == "" {
		log.Fatal("Set OPENROUTER_API_KEY or use -key flag")
	}

	config := openai.DefaultConfig(*apiKey)
	config.BaseURL = "https://openrouter.ai/api/v1"
	client := openai.NewClientWithConfig(config)

	// Wait for server to be ready
	fmt.Printf("🤖 Xandaris Multi-Faction Agent\n")
	fmt.Printf("   Model: %s\n", *model)
	fmt.Printf("   Server: %s\n", serverURL)
	fmt.Printf("   Cycle interval: %s\n\n", *interval)

	for i := 0; i < 30; i++ {
		if _, err := callAPI("GET", "/api/game", "", gameAPIKey); err == nil {
			break
		}
		fmt.Println("Waiting for server...")
		time.Sleep(2 * time.Second)
	}

	// Get faction list from the server
	result, err := callAPI("GET", "/api/players", "", gameAPIKey)
	if err != nil {
		log.Fatalf("Cannot reach server: %v", err)
	}

	var playersResp struct {
		OK   bool `json:"ok"`
		Data []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"data"`
	}
	json.Unmarshal([]byte(result), &playersResp)

	// Build faction list — only AI factions, never control human players
	var factions []*Faction
	for _, p := range playersResp.Data {
		if p.Type == "human" {
			fmt.Printf("   Skipping human player: %s\n", p.Name)
			continue
		}
		personality, exists := factionPersonalities[p.Name]
		if !exists {
			personality = "You are a balanced strategist. Grow your economy steadily through smart investment and trade."
		}

		factions = append(factions, &Faction{
			Name:        p.Name,
			Personality: personality,
			MaxHistory:  20,
		})
		fmt.Printf("   Faction: %s (%s)\n", p.Name, p.Type)
	}

	if len(factions) == 0 {
		log.Fatal("No factions found on server")
	}

	fmt.Printf("\n   Controlling %d factions\n\n", len(factions))

	for cycle := 1; cycle <= *turns; cycle++ {
		// Randomize turn order each cycle
		rand.Shuffle(len(factions), func(i, j int) {
			factions[i], factions[j] = factions[j], factions[i]
		})

		fmt.Printf("━━━ Cycle %d ━━━\n", cycle)

		for _, faction := range factions {
			runFactionTurn(client, *model, faction, cycle)
		}

		if cycle < *turns {
			fmt.Printf("  ⏳ Next cycle in %s...\n\n", *interval)
			time.Sleep(*interval)
		}
	}

	fmt.Println("\n🏁 Agent session complete")
}
