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

// EventLog is a thread-safe ring buffer of recent events with subscriber support.
type EventLog struct {
	mu        sync.RWMutex
	events    []GameEvent
	max       int
	listeners []func(GameEvent)
}

// NewEventLog creates a new event log.
func NewEventLog(maxEvents int) *EventLog {
	return &EventLog{
		events: make([]GameEvent, 0, maxEvents),
		max:    maxEvents,
	}
}

// Subscribe registers a callback that fires on every new event.
func (el *EventLog) Subscribe(fn func(GameEvent)) {
	el.mu.Lock()
	defer el.mu.Unlock()
	// Cap listeners to prevent memory leak from repeated UI subscriptions
	if len(el.listeners) >= 50 {
		el.listeners = el.listeners[len(el.listeners)-25:] // keep newest 25
	}
	el.listeners = append(el.listeners, fn)
}

// Add records a new event and notifies subscribers.
func (el *EventLog) Add(tick int64, gameTime string, eventType EventType, player string, msg string) {
	el.mu.Lock()
	ev := GameEvent{
		Tick:      tick,
		Time:      gameTime,
		Type:      eventType,
		Player:    player,
		Message:   msg,
		Timestamp: time.Now(),
	}
	el.events = append(el.events, ev)
	if len(el.events) > el.max {
		el.events = el.events[len(el.events)-el.max:]
	}
	// Copy listeners to call outside lock
	listeners := make([]func(GameEvent), len(el.listeners))
	copy(listeners, el.listeners)
	el.mu.Unlock()

	for _, fn := range listeners {
		fn(ev)
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
