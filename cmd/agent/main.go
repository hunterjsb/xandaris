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

AVAILABLE TOOLS:
You have tools to interact with the game API. Use them to:
1. Check your status (get_status)
2. View the market (get_economy)
3. Check your planet details (get_planet)
4. Build mines on resource deposits (build)
5. Trade resources at market (trade)
6. Build ships at shipyard (build_ship)
7. Check galaxy flows (get_flows)

STRATEGY:
- Build mines on all available resource deposits first
- Build a Refinery when you have Oil production
- Build a Shipyard when you can afford it (2000cr)
- Trade surplus resources for profit
- Upgrade mines when prices are high
- Build Cargo ships for trade routes

Think step by step. Check your status first, then decide what to do.`

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

func callAPI(method, endpoint string, body string) (string, error) {
	var req *http.Request
	var err error

	url := fmt.Sprintf("http://localhost:8080%s", endpoint)
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
	turns := flag.Int("turns", 10, "Number of decision turns")
	interval := flag.Duration("interval", 30*time.Second, "Time between turns")
	flag.Parse()

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
	fmt.Printf("   Server: http://localhost:8080\n\n")

	// Verify server is running
	if _, err := callAPI("GET", "/api/game", ""); err != nil {
		log.Fatalf("Cannot reach game server: %v\nStart the game first with: ./xandaris-bin --headless --auto --player Agent", err)
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
