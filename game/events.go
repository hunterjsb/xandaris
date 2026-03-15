package game

import (
	"fmt"
	"sync"
	"time"
)

// EventType categorizes game events.
type EventType string

const (
	EventTrade      EventType = "trade"
	EventBuild      EventType = "build"
	EventColonize   EventType = "colonize"
	EventUpgrade    EventType = "upgrade"
	EventShipBuild  EventType = "ship_build"
	EventLogistics  EventType = "logistics"
	EventAlert      EventType = "alert"
	EventJoin       EventType = "join"
)

// GameEvent represents something that happened in the game.
type GameEvent struct {
	Tick      int64     `json:"tick"`
	Time      string    `json:"time"`
	Type      EventType `json:"type"`
	Player    string    `json:"player"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// EventLog is a thread-safe ring buffer of recent events.
type EventLog struct {
	mu     sync.RWMutex
	events []GameEvent
	max    int
}

// NewEventLog creates a new event log.
func NewEventLog(maxEvents int) *EventLog {
	return &EventLog{
		events: make([]GameEvent, 0, maxEvents),
		max:    maxEvents,
	}
}

// Add records a new event.
func (el *EventLog) Add(tick int64, gameTime string, eventType EventType, player string, msg string) {
	el.mu.Lock()
	defer el.mu.Unlock()
	el.events = append(el.events, GameEvent{
		Tick:      tick,
		Time:      gameTime,
		Type:      eventType,
		Player:    player,
		Message:   msg,
		Timestamp: time.Now(),
	})
	if len(el.events) > el.max {
		el.events = el.events[len(el.events)-el.max:]
	}
}

// Addf is a convenience method with fmt.Sprintf.
func (el *EventLog) Addf(tick int64, gameTime string, eventType EventType, player string, format string, args ...interface{}) {
	el.Add(tick, gameTime, eventType, player, fmt.Sprintf(format, args...))
}

// Recent returns the last N events (newest first).
func (el *EventLog) Recent(n int) []GameEvent {
	el.mu.RLock()
	defer el.mu.RUnlock()
	if n > len(el.events) {
		n = len(el.events)
	}
	result := make([]GameEvent, n)
	for i := 0; i < n; i++ {
		result[i] = el.events[len(el.events)-1-i]
	}
	return result
}
