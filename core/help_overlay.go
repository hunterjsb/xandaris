package core

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/ui/widgets"
	"github.com/hunterjsb/xandaris/utils"
	"github.com/hunterjsb/xandaris/views"
)

type helpOverlay struct {
	visible bool
}

func (h *helpOverlay) Toggle() {
	h.visible = !h.visible
}

func (h *helpOverlay) IsVisible() bool {
	return h.visible
}

type helpEntry struct {
	key  string
	desc string
}

func (h *helpOverlay) Draw(screen *ebiten.Image, kb views.KeyBindingsInterface, isRemote bool) {
	if !h.visible {
		return
	}

	entries := []helpEntry{
		{"H", "Toggle this help"},
		{"T", "Open/close chat"},
		{"Esc", "Back / close"},
		{"B", "Build menu (planet view)"},
		{"W", "Workforce overlay (planet view)"},
		{"M", "Market view"},
		{"P", "Player directory"},
		{"Tab", "Focus home system"},
	}

	if !isRemote {
		entries = append(entries,
			helpEntry{"Space", "Pause / resume"},
			helpEntry{"1-4", "Set game speed"},
			helpEntry{"F5", "Quick save"},
		)
	}

	entries = append(entries,
		helpEntry{"", ""},
		helpEntry{"Mouse", ""},
		helpEntry{"Click", "Select object"},
		helpEntry{"Dbl-click", "Enter system/planet"},
		helpEntry{"Shift+Click", "Build mine on resource"},
		helpEntry{"Right-click", "Cancel construction"},
		helpEntry{"Scroll", "Scroll lists"},
	)

	p := widgets.NewPanel(widgets.AnchorCenter, 30)
	p.BgColor = utils.Theme.PanelBgSolid

	p.LineCenter("KEYBOARD SHORTCUTS", utils.Theme.Accent)
	p.Sep()

	for _, e := range entries {
		if e.key == "" {
			p.Sep()
			continue
		}
		if e.desc == "" {
			p.LineCenter(e.key, utils.Theme.Accent)
			continue
		}
		p.LinePair(
			fmt.Sprintf("[%s]", e.key), utils.Theme.TextLight,
			e.desc, utils.Theme.TextDim,
		)
	}

	p.Sep()
	p.LineCenter("Press H to close", utils.Theme.TextDim)

	p.Draw(screen)
}
