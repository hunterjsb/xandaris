package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin (for development)
		return true
	},
}

type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	userId string
	app    *pocketbase.PocketBase // Added app instance
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

var GlobalHub *Hub

func init() {
	GlobalHub = &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	go GlobalHub.run()
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected. Total: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected. Total: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		GlobalHub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle auth message
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse WebSocket message: %v", err)
			continue
		}

		if msg.Type == "auth" {
			// 'c' is the *Client receiver for readPump
			token, ok := msg.Payload.(string)
			if !ok {
				log.Printf("WebSocket auth error: Invalid token format in payload for client.")
				errorMsg := Message{Type: "auth_failed", Payload: "Invalid token payload format"}
				errorData, _ := json.Marshal(errorMsg)
				c.send <- errorData
				continue
			}

			// Validate token using c.app
			// Assuming users are in "_pb_users_auth_" collection.
			authRecord, err := c.app.Dao().FindAuthRecordByToken(token, "_pb_users_auth_")
			if err != nil {
				log.Printf("WebSocket auth failed for token '%s': %v", token, err)
				errorMsg := Message{Type: "auth_failed", Payload: "Invalid or expired token"}
				errorData, _ := json.Marshal(errorMsg)
				c.send <- errorData
				// Optional: consider closing the connection after a failed auth attempt
				// GlobalHub.unregister <- c
				// c.conn.Close()
				// return // This would exit the readPump goroutine
				continue
			}

			c.userId = authRecord.Id
			log.Printf("WebSocket client authenticated: UserID %s, connection %p", c.userId, c.conn)

			// Send auth success message
			authSuccessMsg := Message{Type: "auth_success", Payload: map[string]string{"userId": c.userId, "message": "Authentication successful"}}
			successData, _ := json.Marshal(authSuccessMsg)
			c.send <- successData
		}
	}
}

// HandleWebSocket upgrades HTTP connection to WebSocket
func HandleWebSocket(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			return err
		}

		client := &Client{
			conn: conn,
			send: make(chan []byte, 256),
			app:  app, // Pass app instance to client
		}

		GlobalHub.register <- client

		// Start goroutines for reading and writing
		go client.writePump()
		go client.readPump()

		return nil
	}
}

// BroadcastTickUpdate sends tick update to all connected clients
func BroadcastTickUpdate(tick int, _ string) {
	message := Message{
		Type: "tick",
		Payload: map[string]interface{}{
			"tick": tick,
		},
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal tick update: %v", err)
		return
	}

	GlobalHub.broadcast <- data
}

// BroadcastSystemUpdate sends system update to all connected clients
func BroadcastSystemUpdate(systemData interface{}) {
	message := Message{
		Type:    "system_update",
		Payload: systemData,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal system update: %v", err)
		return
	}

	GlobalHub.broadcast <- data
}

// BroadcastFleetUpdate sends fleet update to all connected clients
func BroadcastFleetUpdate(fleetData interface{}) {
	message := Message{
		Type:    "fleet_update",
		Payload: fleetData,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal fleet update: %v", err)
		return
	}

	GlobalHub.broadcast <- data
}