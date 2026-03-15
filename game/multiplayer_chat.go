package game

import (
	"sync"
	"time"
)

// ChatMsg represents a player-to-player chat message.
type ChatMsg struct {
	Tick      int64     `json:"tick"`
	Time      string    `json:"time"`
	Player    string    `json:"player"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatLog is a thread-safe ring buffer of chat messages.
type ChatLog struct {
	mu        sync.RWMutex
	messages  []ChatMsg
	max       int
	listeners []func(ChatMsg)
}

// NewChatLog creates a new chat log.
func NewChatLog(maxMessages int) *ChatLog {
	return &ChatLog{
		messages: make([]ChatMsg, 0, maxMessages),
		max:      maxMessages,
	}
}

// Send adds a message and notifies subscribers.
func (cl *ChatLog) Send(tick int64, gameTime, player, message string) {
	cl.mu.Lock()
	msg := ChatMsg{
		Tick:      tick,
		Time:      gameTime,
		Player:    player,
		Message:   message,
		Timestamp: time.Now(),
	}
	cl.messages = append(cl.messages, msg)
	if len(cl.messages) > cl.max {
		cl.messages = cl.messages[len(cl.messages)-cl.max:]
	}
	listeners := make([]func(ChatMsg), len(cl.listeners))
	copy(listeners, cl.listeners)
	cl.mu.Unlock()

	for _, fn := range listeners {
		fn(msg)
	}
}

// Subscribe registers a callback for new messages.
func (cl *ChatLog) Subscribe(fn func(ChatMsg)) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.listeners = append(cl.listeners, fn)
}

// Recent returns the last N messages (newest first).
func (cl *ChatLog) Recent(n int) []ChatMsg {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	if n > len(cl.messages) {
		n = len(cl.messages)
	}
	result := make([]ChatMsg, n)
	for i := 0; i < n; i++ {
		result[i] = cl.messages[len(cl.messages)-1-i]
	}
	return result
}
