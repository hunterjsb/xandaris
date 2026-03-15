//go:build !js

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hunterjsb/xandaris/game"
)

const chatSystemPrompt = `You are an AI assistant for Xandaris, a space trading game. A player is asking you to do something in their game. You have tools to check their status and execute actions.

RULES:
- Be concise — 1-2 sentences max in your response
- ALWAYS call get_status first to understand the player's current state
- Execute the player's request using the available tools
- If you can't do something, explain briefly why
- Report what you did in plain language
- Use the navigate tool to show the player relevant locations (e.g. navigate to their planet, a system, the market, etc.)
- When the player asks to "show me" or "go to" something, use navigate`

var chatTools = []map[string]interface{}{
	{"type": "function", "function": map[string]interface{}{
		"name": "get_status", "description": "Get player's current status",
		"parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
	}},
	{"type": "function", "function": map[string]interface{}{
		"name": "get_planet", "description": "Get planet details",
		"parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
			"planet_id": map[string]interface{}{"type": "integer"},
		}, "required": []string{"planet_id"}},
	}},
	{"type": "function", "function": map[string]interface{}{
		"name": "build", "description": "Build: Mine, Trading Post, Refinery, Factory, Generator (power from Fuel), Fusion Reactor (power from He-3), Habitat, Shipyard",
		"parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
			"planet_id":     map[string]interface{}{"type": "integer"},
			"building_type": map[string]interface{}{"type": "string", "enum": []string{"Mine", "Trading Post", "Refinery", "Factory", "Generator", "Fusion Reactor", "Habitat", "Shipyard"}},
			"resource_id":   map[string]interface{}{"type": "integer", "description": "For mines: resource deposit ID"},
		}, "required": []string{"planet_id", "building_type"}},
	}},
	{"type": "function", "function": map[string]interface{}{
		"name": "trade", "description": "Buy or sell resources",
		"parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
			"resource": map[string]interface{}{"type": "string"},
			"quantity": map[string]interface{}{"type": "integer"},
			"action":   map[string]interface{}{"type": "string", "enum": []string{"buy", "sell"}},
		}, "required": []string{"resource", "quantity", "action"}},
	}},
	{"type": "function", "function": map[string]interface{}{
		"name": "build_ship", "description": "Build a ship: Scout, Cargo, Colony, Frigate",
		"parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
			"planet_id": map[string]interface{}{"type": "integer"},
			"ship_type": map[string]interface{}{"type": "string", "enum": []string{"Scout", "Cargo", "Colony", "Frigate"}},
		}, "required": []string{"planet_id", "ship_type"}},
	}},
	{"type": "function", "function": map[string]interface{}{
		"name": "get_economy", "description": "Get market prices and supply/demand",
		"parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
	}},
	{"type": "function", "function": map[string]interface{}{
		"name": "navigate", "description": "Navigate the player's UI to a location: galaxy map, a system, a planet, the market, or player directory",
		"parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
			"target": map[string]interface{}{"type": "string", "enum": []string{"galaxy", "system", "planet", "market", "players"}},
			"id":     map[string]interface{}{"type": "integer", "description": "System or planet ID (required for system/planet targets)"},
		}, "required": []string{"target"}},
	}},
}

// handleChat processes a natural language message from the player via LLM.
// If conversationHistory is provided, it includes prior turns for context.
func handleChat(p GameStateProvider, playerName string, message string, conversationHistory []map[string]interface{}) (string, []string, error) {
	openrouterKey := os.Getenv("OPENROUTER_API_KEY")
	if openrouterKey == "" {
		return "", nil, fmt.Errorf("AI agent not configured on this server")
	}

	// Build conversation with system prompt + history + new message
	messages := []map[string]interface{}{
		{"role": "system", "content": chatSystemPrompt},
	}
	// Add conversation history (prior user/assistant turns)
	for _, msg := range conversationHistory {
		messages = append(messages, msg)
	}
	messages = append(messages, map[string]interface{}{
		"role": "user", "content": message,
	})

	var actions []string

	// LLM loop: call model, execute tools, repeat until text response
	for i := 0; i < 5; i++ { // max 5 tool rounds
		respMsg, err := callOpenRouter(openrouterKey, messages)
		if err != nil {
			return "", actions, err
		}

		// Check for tool calls
		toolCalls, _ := respMsg["tool_calls"].([]interface{})
		if len(toolCalls) == 0 {
			// Final text response
			content, _ := respMsg["content"].(string)
			return content, actions, nil
		}

		// Add assistant message with tool calls
		messages = append(messages, respMsg)

		// Execute each tool call
		for _, tc := range toolCalls {
			tcMap, ok := tc.(map[string]interface{})
			if !ok {
				continue
			}
			fn, _ := tcMap["function"].(map[string]interface{})
			fnName, _ := fn["name"].(string)
			fnArgs, _ := fn["arguments"].(string)
			tcID, _ := tcMap["id"].(string)

			result := executeToolCall(p, playerName, fnName, fnArgs)

			// Navigate actions get a special prefix for client-side handling
			if fnName == "navigate" {
				var navResult struct {
					Navigate string `json:"navigate"`
					ID       int    `json:"id"`
				}
				json.Unmarshal([]byte(result), &navResult)
				actions = append(actions, fmt.Sprintf("navigate:%s:%d", navResult.Navigate, navResult.ID))
			} else {
				actions = append(actions, fmt.Sprintf("%s(%s)", fnName, truncate(fnArgs, 50)))
			}

			messages = append(messages, map[string]interface{}{
				"role":         "tool",
				"tool_call_id": tcID,
				"content":      truncate(result, 2000),
			})
		}
	}

	return "I took too many steps — please try a simpler request.", actions, nil
}

func callOpenRouter(apiKey string, messages []map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"model":    "x-ai/grok-4.1-fast",
		"messages": messages,
		"tools":    chatTools,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenRouter request failed: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message map[string]interface{} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("LLM error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	return result.Choices[0].Message, nil
}

func executeToolCall(p GameStateProvider, playerName, fnName, fnArgs string) string {
	var args map[string]interface{}
	json.Unmarshal([]byte(fnArgs), &args)

	switch fnName {
	case "get_status":
		data := handleGetStatus(p, playerName)
		result, _ := json.Marshal(data)
		return string(result)

	case "get_planet":
		planetID := int(getFloat(args, "planet_id"))
		data, found := handleGetPlanet(p, planetID)
		if !found {
			return `{"error":"planet not found"}`
		}
		result, _ := json.Marshal(data)
		return string(result)

	case "get_economy":
		data := handleGetEconomy(p)
		result, _ := json.Marshal(data)
		return string(result)

	case "build":
		planetID := int(getFloat(args, "planet_id"))
		buildingType, _ := args["building_type"].(string)
		resourceID := int(getFloat(args, "resource_id"))
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "build",
			Data:   game.BuildCommandData{PlanetID: planetID, BuildingType: buildingType, ResourceID: resourceID},
			Result: resultCh,
		}
		return waitForResult(resultCh)

	case "trade":
		resource, _ := args["resource"].(string)
		quantity := int(getFloat(args, "quantity"))
		action, _ := args["action"].(string)
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "trade",
			Data:   game.TradeCommandData{Resource: resource, Quantity: quantity, Buy: action == "buy"},
			Result: resultCh,
		}
		return waitForResult(resultCh)

	case "build_ship":
		planetID := int(getFloat(args, "planet_id"))
		shipType, _ := args["ship_type"].(string)
		resultCh := make(chan interface{}, 1)
		p.GetCommandChannel() <- game.GameCommand{
			Type:   "build_ship",
			Data:   game.ShipBuildCommandData{PlanetID: planetID, ShipType: shipType},
			Result: resultCh,
		}
		return waitForResult(resultCh)

	case "navigate":
		target, _ := args["target"].(string)
		id := int(getFloat(args, "id"))
		return fmt.Sprintf(`{"ok":true,"navigate":"%s","id":%d}`, target, id)

	default:
		return fmt.Sprintf(`{"error":"unknown tool: %s"}`, fnName)
	}
}

func waitForResult(ch chan interface{}) string {
	select {
	case result := <-ch:
		switch v := result.(type) {
		case error:
			return fmt.Sprintf(`{"error":%q}`, v.Error())
		default:
			data, _ := json.Marshal(v)
			return string(data)
		}
	case <-time.After(5 * time.Second):
		return `{"error":"timeout"}`
	}
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

// registerChatEndpoint registers the POST /api/chat handler.
func registerChatEndpoint(mux *http.ServeMux, getProvider func() GameStateProvider) {
	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}

		var req struct {
			Message string                   `json:"message"`
			History []map[string]interface{} `json:"history"` // prior conversation turns
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &req); err != nil || strings.TrimSpace(req.Message) == "" {
			writeErr(w, http.StatusBadRequest, "message required")
			return
		}

		player := getAuthPlayer(r)
		if player == "" {
			player = "Player" // fallback for unauthenticated
		}

		response, actions, err := handleChat(getProvider(), player, req.Message, req.History)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, APIResponse{OK: true, Data: map[string]interface{}{
			"response": response,
			"actions":  actions,
		}})
	})
}
