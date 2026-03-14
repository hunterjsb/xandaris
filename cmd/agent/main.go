package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const systemPrompt = `You are an AI agent playing Xandaris, a space trading game. You control a faction and must build infrastructure, trade resources, and grow your economy.

IMPORTANT RULES:
- ALWAYS call get_status first each turn to see your credits and planet IDs
- ALWAYS call get_planet before trading to check your actual stock levels
- Only sell resources you HAVE (check stored_resources in get_planet response)
- Only build_ship if your Shipyard construction is COMPLETE (check get_construction)
- Mine costs 500cr, Refinery 1500cr, Shipyard 2000cr, Cargo ship 1000cr + 60 Iron + 15 Fuel

STRATEGY:
1. Build mines on ALL unmined resource deposits (check resource_deposits, build on ones where has_mine=false)
2. Build Refinery when mines are done (converts Oil→Fuel)
3. Build Shipyard when you have 2000cr
4. Build Cargo ship for trade routes (needs operational Shipyard + Iron + Fuel on planet)
5. Sell surplus resources (storage > 200) at market for credits
6. Upgrade mines (750cr) when affordable to boost production
7. Move Cargo ships to adjacent systems using get_routes then move_ship

Think step by step. Be precise — check your actual resources before trading.`

var tools = []openai.Tool{
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_status",
			Description: "Get your current game status including credits, planets, ships, and hints",
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_economy",
			Description: "Get galaxy-wide economy data: resource prices, supply, demand, scarcity",
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_planet",
			Description: "Get detailed info about your planet including resource deposits and buildings",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"planet_id":{"type":"integer","description":"Planet ID"}},"required":["planet_id"]}`),
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_flows",
			Description: "Get galaxy-wide production vs consumption rates for all resources",
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "build",
			Description: "Build a structure on your planet. Types: Mine, Trading Post, Refinery, Habitat, Shipyard. Mines require resource_id.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"planet_id":{"type":"integer"},"building_type":{"type":"string","enum":["Mine","Trading Post","Refinery","Habitat","Shipyard"]},"resource_id":{"type":"integer","description":"Required for mines: the resource deposit ID to attach to"}},"required":["planet_id","building_type"]}`),
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "trade",
			Description: "Buy or sell resources at the market",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"resource":{"type":"string"},"quantity":{"type":"integer"},"action":{"type":"string","enum":["buy","sell"]}},"required":["resource","quantity","action"]}`),
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "build_ship",
			Description: "Build a ship at your shipyard. Types: Scout, Cargo, Colony, Frigate",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"planet_id":{"type":"integer"},"ship_type":{"type":"string","enum":["Scout","Cargo","Colony","Frigate"]}},"required":["planet_id","ship_type"]}`),
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "upgrade",
			Description: "Upgrade a building on your planet by its index",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"planet_id":{"type":"integer"},"building_index":{"type":"integer"}},"required":["planet_id","building_index"]}`),
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_construction",
			Description: "Check construction queue — see what's being built and progress",
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "move_ship",
			Description: "Move a ship to an adjacent system via hyperlane",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"ship_id":{"type":"integer"},"target_system_id":{"type":"integer"}},"required":["ship_id","target_system_id"]}`),
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_routes",
			Description: "Get connected systems (hyperlanes) from a system — for planning ship routes",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"system_id":{"type":"integer"}},"required":["system_id"]}`),
		},
	},
}

var serverURL string

func callAPI(method, endpoint string, body string) (string, error) {
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	// Truncate very long responses
	result := string(data)
	if len(result) > 3000 {
		result = result[:3000] + "...(truncated)"
	}
	return result, nil
}

func executeTool(name string, args string) string {
	var params map[string]interface{}
	json.Unmarshal([]byte(args), &params)

	switch name {
	case "get_status":
		result, err := callAPI("GET", "/api/status", "")
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result

	case "get_economy":
		result, err := callAPI("GET", "/api/economy", "")
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result

	case "get_flows":
		result, err := callAPI("GET", "/api/flows", "")
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result

	case "get_planet":
		pid := int(params["planet_id"].(float64))
		result, err := callAPI("GET", fmt.Sprintf("/api/planets/%d", pid), "")
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result

	case "build":
		body, _ := json.Marshal(params)
		result, err := callAPI("POST", "/api/build", string(body))
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result

	case "trade":
		body, _ := json.Marshal(params)
		result, err := callAPI("POST", "/api/market/trade", string(body))
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result

	case "build_ship":
		body, _ := json.Marshal(params)
		result, err := callAPI("POST", "/api/ships/build", string(body))
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result

	case "upgrade":
		body, _ := json.Marshal(params)
		result, err := callAPI("POST", "/api/upgrade", string(body))
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result

	case "get_construction":
		result, err := callAPI("GET", "/api/construction", "")
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result

	case "move_ship":
		body, _ := json.Marshal(params)
		result, err := callAPI("POST", "/api/ships/move", string(body))
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result

	case "get_routes":
		sysID := int(params["system_id"].(float64))
		result, err := callAPI("GET", fmt.Sprintf("/api/routes/%d", sysID), "")
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result

	default:
		return fmt.Sprintf("Unknown tool: %s", name)
	}
}

func main() {
	apiKey := flag.String("key", os.Getenv("OPENROUTER_API_KEY"), "OpenRouter API key")
	model := flag.String("model", "z-ai/glm-4.7-flash", "Model to use")
	server := flag.String("server", "http://localhost:8080", "Game server URL")
	turns := flag.Int("turns", 10, "Number of decision turns")
	interval := flag.Duration("interval", 30*time.Second, "Time between turns")
	flag.Parse()

	serverURL = *server

	if *apiKey == "" {
		log.Fatal("Set OPENROUTER_API_KEY or use -key flag")
	}

	// Configure OpenAI client for OpenRouter
	config := openai.DefaultConfig(*apiKey)
	config.BaseURL = "https://openrouter.ai/api/v1"
	client := openai.NewClientWithConfig(config)

	fmt.Printf("🤖 Xandaris AI Agent\n")
	fmt.Printf("   Model: %s\n", *model)
	fmt.Printf("   Turns: %d (every %s)\n", *turns, *interval)
	fmt.Printf("   Server: %s\n\n", serverURL)

	// Verify server is running
	if _, err := callAPI("GET", "/api/game", ""); err != nil {
		log.Fatalf("Cannot reach game server at %s: %v", serverURL, err)
	}

	for turn := 1; turn <= *turns; turn++ {
		fmt.Printf("━━━ Turn %d/%d ━━━\n", turn, *turns)

		messages := []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: fmt.Sprintf("Turn %d. Check your status and decide what to do next. Take 1-3 actions.", turn)},
		}

		// Agent loop: keep calling until no more tool calls
		for step := 0; step < 8; step++ {
			resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
				Model:    *model,
				Messages: messages,
				Tools:    tools,
			})
			if err != nil {
				fmt.Printf("  ❌ LLM error: %v\n", err)
				break
			}

			choice := resp.Choices[0]

			// If the model wants to call tools
			if len(choice.Message.ToolCalls) > 0 {
				messages = append(messages, choice.Message)

				for _, tc := range choice.Message.ToolCalls {
					fmt.Printf("  🔧 %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
					result := executeTool(tc.Function.Name, tc.Function.Arguments)

					// Print a summary of the result
					if len(result) > 200 {
						fmt.Printf("     → %s...\n", result[:200])
					} else {
						fmt.Printf("     → %s\n", result)
					}

					messages = append(messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    result,
						ToolCallID: tc.ID,
					})
				}
				continue
			}

			// Model is done — print its reasoning
			if choice.Message.Content != "" {
				fmt.Printf("  💭 %s\n", choice.Message.Content)
			}
			break
		}

		if turn < *turns {
			fmt.Printf("  ⏳ Waiting %s...\n\n", *interval)
			time.Sleep(*interval)
		}
	}

	fmt.Println("\n🏁 Agent session complete")
}
