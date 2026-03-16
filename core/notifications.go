package core

import (
	"image/color"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/utils"
	"github.com/hunterjsb/xandaris/views"
)

const (
	toastDuration  = 4 * time.Second
	toastFadeStart = 3 * time.Second // start fading after this
	maxToasts      = 3
)

type toast struct {
	message   string
	color     color.RGBA
	createdAt time.Time
}

type notificationOverlay struct {
	mu     sync.Mutex
	toasts []toast
}

func newNotificationOverlay() *notificationOverlay {
	return &notificationOverlay{}
}

// subscribe hooks into the event log to show toasts for important events.
func (n *notificationOverlay) subscribe(el *game.EventLog, playerName string) {
	if el == nil {
		return
	}
	el.Subscribe(func(ev game.GameEvent) {
		// Only show events for the human player
		if ev.Player != playerName {
			return
		}
		// Filter to important event types
		c := utils.Theme.TextLight // default light
		switch ev.Type {
		case game.EventBuild:
			c = color.RGBA{100, 200, 140, 255} // green
		case game.EventUpgrade:
			c = color.RGBA{100, 200, 140, 255}
		case game.EventShipBuild:
			c = color.RGBA{100, 180, 255, 255} // blue
		case game.EventColonize:
			c = utils.Theme.Accent // accent
		case game.EventAlert:
			c = color.RGBA{255, 120, 100, 255} // red
		default:
			// Don't show trade/logistics/join as toasts — too noisy
			return
		}
		n.add(ev.Message, c)
	})
}

func (n *notificationOverlay) add(message string, c color.RGBA) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.toasts = append(n.toasts, toast{
		message:   message,
		color:     c,
		createdAt: time.Now(),
	})
	// Keep only the most recent
	if len(n.toasts) > maxToasts {
		n.toasts = n.toasts[len(n.toasts)-maxToasts:]
	}
}

func (n *notificationOverlay) draw(screen *ebiten.Image, screenWidth int) {
	n.mu.Lock()
	now := time.Now()
	// Remove expired toasts
	alive := n.toasts[:0]
	for _, t := range n.toasts {
		if now.Sub(t.createdAt) < toastDuration {
			alive = append(alive, t)
		}
	}
	n.toasts = alive
	// Copy for drawing outside lock
	snapshot := make([]toast, len(alive))
	copy(snapshot, alive)
	n.mu.Unlock()

	if len(snapshot) == 0 {
		return
	}

	// Draw toasts from bottom-right, stacking upward
	x := screenWidth - 310
	y := 720 - 90 // above status bar

	for i := len(snapshot) - 1; i >= 0; i-- {
		t := snapshot[i]
		age := now.Sub(t.createdAt)

		// Calculate alpha with fade (quantized to 10 steps to avoid cache churn)
		alpha := uint8(200)
		if age > toastFadeStart {
			fadeProgress := float64(age-toastFadeStart) / float64(toastDuration-toastFadeStart)
			alpha = uint8(200 * (1.0 - fadeProgress))
			alpha = (alpha / 20) * 20 // quantize to steps of 20
			if alpha < 20 {
				alpha = 20
			}
		}

		bgColor := color.RGBA{12, 16, 28, alpha}
		borderColor := color.RGBA{t.color.R / 2, t.color.G / 2, t.color.B / 2, alpha}
		textColor := color.RGBA{t.color.R, t.color.G, t.color.B, alpha}

		w := len(t.message)*6 + 20
		if w < 200 {
			w = 200
		}
		if w > 300 {
			w = 300
		}

		panel := &views.UIPanel{
			X: x, Y: y, Width: w, Height: 20,
			BgColor: bgColor, BorderColor: borderColor,
		}
		panel.Draw(screen)
		views.DrawText(screen, t.message, x+8, y+5, textColor)

		y -= 24
	}
}
