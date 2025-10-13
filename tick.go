package main

import (
	"time"

	"github.com/hunterjsb/xandaris/tickable"
)

// TickSpeed represents the speed multiplier for game ticks
type TickSpeed float64

const (
	TickSpeedPaused TickSpeed = 0.0
	TickSpeed1x     TickSpeed = 1.0
	TickSpeed2x     TickSpeed = 2.0
	TickSpeed4x     TickSpeed = 4.0
	TickSpeed8x     TickSpeed = 8.0
)

// TickListener interface for objects that respond to game ticks
type TickListener interface {
	OnTick(tick int64)
}

// TickManager manages game time and ticks
type TickManager struct {
	currentTick    int64
	isPaused       bool
	speed          TickSpeed
	ticksPerSecond float64 // Base ticks per second at 1x speed
	accumulator    float64 // Accumulated time for tick calculation
	lastUpdateTime time.Time
	listeners      []TickListener
	tickCallbacks  []func(tick int64)
}

// NewTickManager creates a new tick manager
func NewTickManager(ticksPerSecond float64) *TickManager {
	return &TickManager{
		currentTick:    0,
		isPaused:       false,
		speed:          TickSpeed1x,
		ticksPerSecond: ticksPerSecond,
		accumulator:    0.0,
		lastUpdateTime: time.Now(),
		listeners:      make([]TickListener, 0),
		tickCallbacks:  make([]func(tick int64), 0),
	}
}

// Update updates the tick system based on elapsed time
func (tm *TickManager) Update() {
	if tm.isPaused {
		tm.lastUpdateTime = time.Now()
		return
	}

	// Calculate delta time
	now := time.Now()
	deltaTime := now.Sub(tm.lastUpdateTime).Seconds()
	tm.lastUpdateTime = now

	// Accumulate time scaled by speed
	tm.accumulator += deltaTime * float64(tm.speed) * tm.ticksPerSecond

	// Process ticks
	ticksToProcess := int(tm.accumulator)
	if ticksToProcess > 0 {
		tm.accumulator -= float64(ticksToProcess)

		// Limit max ticks per update to prevent death spiral
		if ticksToProcess > 100 {
			ticksToProcess = 100
		}

		for i := 0; i < ticksToProcess; i++ {
			tm.currentTick++
			tm.processTick()
		}
	}
}

// processTick notifies all listeners about a new tick
func (tm *TickManager) processTick() {
	// Update all tickable systems concurrently
	tickable.UpdateAllSystems(tm.currentTick)

	// Notify all listeners
	for _, listener := range tm.listeners {
		listener.OnTick(tm.currentTick)
	}

	// Execute all callbacks
	for _, callback := range tm.tickCallbacks {
		callback(tm.currentTick)
	}
}

// GetCurrentTick returns the current tick number
func (tm *TickManager) GetCurrentTick() int64 {
	return tm.currentTick
}

// IsPaused returns whether the game is paused
func (tm *TickManager) IsPaused() bool {
	return tm.isPaused
}

// GetSpeed returns the current game speed
func (tm *TickManager) GetSpeed() interface{} {
	return tm.speed
}

// GetSpeedFloat returns the current game speed as float64 for views
func (tm *TickManager) GetSpeedFloat() float64 {
	return float64(tm.speed)
}

// Pause pauses the game
func (tm *TickManager) Pause() {
	tm.isPaused = true
}

// Resume resumes the game
func (tm *TickManager) Resume() {
	tm.isPaused = false
	tm.lastUpdateTime = time.Now()
}

// TogglePause toggles pause state
func (tm *TickManager) TogglePause() {
	if tm.isPaused {
		tm.Resume()
	} else {
		tm.Pause()
	}
}

// SetSpeed sets the game speed
func (tm *TickManager) SetSpeed(speed interface{}) {
	if ts, ok := speed.(TickSpeed); ok {
		tm.speed = ts
		if ts == TickSpeedPaused {
			tm.isPaused = true
		} else if tm.isPaused {
			tm.Resume()
		}
	}
}

// CycleSpeed cycles through available speeds
func (tm *TickManager) CycleSpeed() {
	switch tm.speed {
	case TickSpeed1x:
		tm.SetSpeed(TickSpeed2x)
	case TickSpeed2x:
		tm.SetSpeed(TickSpeed4x)
	case TickSpeed4x:
		tm.SetSpeed(TickSpeed8x)
	case TickSpeed8x:
		tm.SetSpeed(TickSpeed1x)
	default:
		tm.SetSpeed(TickSpeed1x)
	}
}

// AddListener adds a tick listener
func (tm *TickManager) AddListener(listener TickListener) {
	tm.listeners = append(tm.listeners, listener)
}

// RemoveListener removes a tick listener
func (tm *TickManager) RemoveListener(listener TickListener) {
	for i, l := range tm.listeners {
		if l == listener {
			tm.listeners = append(tm.listeners[:i], tm.listeners[i+1:]...)
			break
		}
	}
}

// AddTickCallback adds a callback function that executes on each tick
func (tm *TickManager) AddTickCallback(callback func(tick int64)) {
	tm.tickCallbacks = append(tm.tickCallbacks, callback)
}

// GetTicksPerSecond returns the base ticks per second
func (tm *TickManager) GetTicksPerSecond() float64 {
	return tm.ticksPerSecond
}

// GetEffectiveTicksPerSecond returns the current effective ticks per second (base * speed)
func (tm *TickManager) GetEffectiveTicksPerSecond() float64 {
	if tm.isPaused {
		return 0.0
	}
	return tm.ticksPerSecond * float64(tm.speed)
}

// GetSpeedString returns a string representation of the current speed
func (tm *TickManager) GetSpeedString() string {
	if tm.isPaused {
		return "PAUSED"
	}
	switch tm.speed {
	case TickSpeed1x:
		return "1x"
	case TickSpeed2x:
		return "2x"
	case TickSpeed4x:
		return "4x"
	case TickSpeed8x:
		return "8x"
	default:
		return "?x"
	}
}

// Reset resets the tick manager to initial state
func (tm *TickManager) Reset() {
	tm.currentTick = 0
	tm.isPaused = false
	tm.speed = TickSpeed1x
	tm.accumulator = 0.0
	tm.lastUpdateTime = time.Now()
}

// GetGameTime returns the total game time in seconds (based on ticks)
func (tm *TickManager) GetGameTime() float64 {
	return float64(tm.currentTick) / tm.ticksPerSecond
}

// GetGameTimeFormatted returns formatted game time (days, hours, minutes, seconds)
func (tm *TickManager) GetGameTimeFormatted() string {
	totalSeconds := tm.GetGameTime()
	days := int(totalSeconds / 86400)
	hours := int((totalSeconds - float64(days*86400)) / 3600)
	minutes := int((totalSeconds - float64(days*86400) - float64(hours*3600)) / 60)
	seconds := int(totalSeconds - float64(days*86400) - float64(hours*3600) - float64(minutes*60))

	if days > 0 {
		return formatTime(days, "d", hours, "h", minutes, "m")
	} else if hours > 0 {
		return formatTime(hours, "h", minutes, "m")
	} else if minutes > 0 {
		return formatTime(minutes, "m", seconds, "s")
	} else {
		return formatTime(seconds, "s")
	}
}

// formatTime helper to format time strings
func formatTime(values ...interface{}) string {
	result := ""
	for i := 0; i < len(values); i += 2 {
		if i > 0 {
			result += " "
		}
		value := values[i].(int)
		unit := values[i+1].(string)
		result += formatInt(value) + unit
	}
	return result
}

// formatInt formats an integer to string
func formatInt(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return itoa(n)
}

// itoa converts int to string (simple implementation)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+(n%10)))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}
