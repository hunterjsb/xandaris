package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tokens"
	// "github.com/labstack/echo/v5" // Not directly needed for client logic test
)

// Helper function to create a test application instance (similar to tick_test.go)
func setupWebSocketTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	// Use a different path for migrations if manager_test.go is in a different depth
	testApp, err := tests.NewTestApp("../../../migrations")
	if err != nil {
		t.Fatalf("Failed to initialize test app: %v", err)
	}
	t.Cleanup(func() {
		if errDb := os.Remove(testApp.DataDir() + "/storage.db"); errDb != nil && !os.IsNotExist(errDb) {
			t.Logf("Failed to remove test db: %v", errDb)
		}
		if errDir := os.RemoveAll(testApp.DataDir()); errDir != nil {
			t.Logf("Failed to remove test data directory: %v", errDir)
		}
	})
	return testApp
}

// Helper to create a user and get an auth token
func createTestUserWithToken(t *testing.T, app *tests.TestApp, email, password string) (*models.Record, string) {
	t.Helper()
	user := &models.Record{}
	user.CollectionID = "_pb_users_auth_"
	user.Email = email
	user.SetPassword(password)
	user.VerificationToken = "test_verified_token_ws" // Mark as verified

	if err := app.Dao().SaveRecord(user); err != nil {
		t.Fatalf("Failed to create test user %s: %v", email, err)
	}

	token, err := tokens.NewRecordAuthToken(app.App, user)
	if err != nil {
		t.Fatalf("Failed to create auth token for user %s: %v", email, err)
	}
	return user, token
}

// Mock a WebSocket connection for testing client logic
// This is a simplified mock. For more complex scenarios, a library might be needed.
type mockConn struct {
	*websocket.Conn // Embed for some passthrough if needed, but mostly override
	SentMessages    [][]byte
	ReceivedMessages chan []byte
	Closed          bool
}

func (m *mockConn) WriteMessage(messageType int, data []byte) error {
	if m.Closed {
		return websocket.ErrCloseSent
	}
	m.SentMessages = append(m.SentMessages, data)
	return nil
}

func (m *mockConn) ReadMessage() (messageType int, p []byte, err error) {
	if m.Closed {
		return 0, nil, websocket.ErrCloseSent
	}
	if m.ReceivedMessages == nil { // Ensure channel is initialized
		return 0, nil, websocket.ErrCloseSent // Or some other error indicating not ready
	}
	data, ok := <-m.ReceivedMessages
	if !ok {
		return websocket.CloseMessage, []byte{}, nil // Simulate connection closed by peer
	}
	return websocket.TextMessage, data, nil
}

func (m *mockConn) Close() error {
	if !m.Closed {
		m.Closed = true
		if m.ReceivedMessages != nil {
			close(m.ReceivedMessages)
		}
	}
	return nil
}

func NewMockConn() *mockConn {
	return &mockConn{
		SentMessages:    make([][]byte, 0),
		ReceivedMessages: make(chan []byte, 10), // Buffered channel
	}
}


func TestWebSocketAuth_ValidToken(t *testing.T) {
	app := setupWebSocketTestApp(t)
	defer app.Cleanup()

	user, validToken := createTestUserWithToken(t, app, "wsuser1@example.com", "password123")

	mockWsConn := NewMockConn()
	client := &Client{
		conn:   mockWsConn,
		send:   make(chan []byte, 10), // Buffered channel for client's send queue
		userId: "",                     // Initially no user ID
		app:    app.App,                // Pass the PocketBase app instance
	}

	// Simulate running writePump in a goroutine to consume messages from client.send
	go func() {
		for msg := range client.send {
			mockWsConn.WriteMessage(websocket.TextMessage, msg)
		}
	}()

	// Prepare the auth message
	authMsg := Message{
		Type:    "auth",
		Payload: validToken,
	}
	authMsgBytes, _ := json.Marshal(authMsg)

	// Simulate the client receiving this message in readPump
	// We are not running the full readPump loop here, but testing its auth handling logic.
	// To do this more directly, one might refactor readPump to make the message handling part separately testable.
	// For now, we'll send the message to the mock connection's receive channel,
	// then call readPump and immediately close the connection to stop the loop.

	go func() {
		mockWsConn.ReceivedMessages <- authMsgBytes
		time.Sleep(100 * time.Millisecond) // Give readPump a moment to process
		mockWsConn.Close() // This will cause readPump to exit its loop
	}()

	client.readPump() // This will block until the mockConn is closed

	if client.userId != user.Id {
		t.Errorf("Expected client.userId to be '%s', got '%s'", user.Id, client.userId)
	}

	// Check for auth_success message
	foundSuccess := false
	for _, sentBytes := range mockWsConn.SentMessages {
		var respMsg Message
		if err := json.Unmarshal(sentBytes, &respMsg); err == nil {
			if respMsg.Type == "auth_success" {
				foundSuccess = true
				payload, ok := respMsg.Payload.(map[string]interface{})
				if !ok || payload["userId"] != user.Id {
					t.Errorf("Auth success message payload incorrect: %+v", respMsg.Payload)
				}
				break
			}
		}
	}
	if !foundSuccess {
		t.Error("Expected 'auth_success' message to be sent to client, none found.")
		t.Logf("Sent messages: %v", mockWsConn.SentMessages)
	}
	// Ensure client.send is closed by the unregister mechanism if readPump exits normally
    // This part is tricky to test without the full Hub running.
    // For now, we focus on userId and sent message.
}


func TestWebSocketAuth_InvalidToken(t *testing.T) {
	app := setupWebSocketTestApp(t)
	defer app.Cleanup()

	_, _ = createTestUserWithToken(t, app, "wsuser2@example.com", "password123") // User exists but we'll use bad token
	invalidToken := "this.is.not.a.valid.jwt.token"

	mockWsConn := NewMockConn()
	client := &Client{
		conn:   mockWsConn,
		send:   make(chan []byte, 10),
		userId: "",
		app:    app.App,
	}
	go func() {
		for msg := range client.send {
			mockWsConn.WriteMessage(websocket.TextMessage, msg)
		}
	}()

	authMsg := Message{Type: "auth", Payload: invalidToken}
	authMsgBytes, _ := json.Marshal(authMsg)

	go func() {
		mockWsConn.ReceivedMessages <- authMsgBytes
		time.Sleep(100 * time.Millisecond)
		mockWsConn.Close()
	}()
	client.readPump()

	if client.userId != "" {
		t.Errorf("Expected client.userId to be empty, got '%s'", client.userId)
	}

	foundFailure := false
	for _, sentBytes := range mockWsConn.SentMessages {
		var respMsg Message
		if err := json.Unmarshal(sentBytes, &respMsg); err == nil {
			if respMsg.Type == "auth_failed" {
				foundFailure = true
				// Optionally check payload message: e.g. "Invalid or expired token"
				break
			}
		}
	}
	if !foundFailure {
		t.Error("Expected 'auth_failed' message to be sent to client for invalid token.")
	}
}

func TestWebSocketAuth_MalformedMessage(t *testing.T) {
	app := setupWebSocketTestApp(t)
	defer app.Cleanup()

	mockWsConn := NewMockConn()
	client := &Client{
		conn:   mockWsConn,
		send:   make(chan []byte, 10),
		userId: "",
		app:    app.App,
	}
	go func() {
		for msg := range client.send {
			mockWsConn.WriteMessage(websocket.TextMessage, msg)
		}
	}()

	// Malformed payload: not a string
	authMsg := Message{Type: "auth", Payload: 12345}
	authMsgBytes, _ := json.Marshal(authMsg)

	go func() {
		mockWsConn.ReceivedMessages <- authMsgBytes
		time.Sleep(100 * time.Millisecond)
		mockWsConn.Close()
	}()
	client.readPump()

	if client.userId != "" {
		t.Errorf("Expected client.userId to be empty after malformed message, got '%s'", client.userId)
	}

	foundFailure := false
	for _, sentBytes := range mockWsConn.SentMessages {
		var respMsg Message
		if err := json.Unmarshal(sentBytes, &respMsg); err == nil {
			if respMsg.Type == "auth_failed" {
				foundFailure = true
				// Optionally check payload message: e.g. "Invalid token payload format"
				break
			}
		}
	}
	if !foundFailure {
		t.Error("Expected 'auth_failed' message for malformed payload.")
	}
}

// Note: Testing the full WebSocket handshake and server-side Hub logic
// would require a more involved setup, potentially using httptest.NewServer
// and gorilla/websocket.Dialer to connect as a real client.
// The tests above focus on the auth logic within readPump given a Client object.
// The GlobalHub isn't directly tested here, nor is the HandleWebSocket handler itself.
// To test HandleWebSocket, you'd typically:
// 1. Create an echo instance with the handler.
// 2. Start an httptest.Server with echo.
// 3. Use websocket.Dialer to connect to the test server.
// 4. Send and receive messages over the real WebSocket connection.
// This is a more complex integration test.
// The current tests are unit/component tests for the auth processing part of readPump.
// They rely on the Client struct being correctly populated with `app` instance.
// The mockConn helps isolate the readPump logic from actual network I/O.

// Helper for full WebSocket integration test (example structure, not used by above tests)
/*
func TestWebSocketIntegration_HandleWebSocketAuth(t *testing.T) {
	app := setupWebSocketTestApp(t)
	defer app.Cleanup()

	user, validToken := createTestUserWithToken(t, app, "wsintuser@example.com", "password")

	// Setup Echo, PocketBase app, and the WebSocket handler
	e := echo.New()
	e.GET("/ws", HandleWebSocket(app.App)) // Assuming HandleWebSocket is accessible

	server := httptest.NewServer(e)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Connect to the server
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer ws.Close()

	// Test valid auth
	authMsg := Message{Type: "auth", Payload: validToken}
	if err := ws.WriteJSON(authMsg); err != nil {
		t.Fatalf("Failed to send auth message: %v", err)
	}

	// Expect auth_success response
	var responseMsg Message
	err = ws.ReadJSON(&responseMsg)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if responseMsg.Type != "auth_success" {
		t.Errorf("Expected auth_success, got %s", responseMsg.Type)
	}
	payload, _ := responseMsg.Payload.(map[string]interface{})
	if payload["userId"] != user.Id {
		t.Errorf("Expected userId %s, got %s", user.Id, payload["userId"])
	}

	// TODO: Test invalid token, malformed message etc. using the same integration setup
}
*/
