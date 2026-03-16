package utils

import "image/color"

// UIScale controls the global text and UI element scaling factor.
// 1.0 = original size (designed for 1280x720). 1.25 = 25% larger (good for 1920x1080).
var UIScale = 1.25

// CharWidth returns the pixel width of a single character at current scale.
func CharWidth() int {
	return int(6.0 * UIScale)
}

// Theme defines the centralized UI color palette.
// Change these values to restyle all panels, text, and controls in one place.
var Theme = struct {
	// Panel backgrounds (varying opacity for layering)
	PanelBg       color.RGBA // primary panel background
	PanelBgSolid  color.RGBA // fully opaque variant (for full-screen views)
	PanelBgLight  color.RGBA // slightly lighter (for sub-items within panels)
	PanelBorder   color.RGBA // panel border / separator lines
	PanelHover    color.RGBA // hovered row / item highlight

	// Text hierarchy
	Accent    color.RGBA // titles, interactive labels, important values
	TextLight color.RGBA // primary body text, important stats
	TextDim   color.RGBA // secondary text, labels, hints

	// Bar backgrounds (inside progress/capacity bars)
	BarBg color.RGBA

	// Button states
	ButtonActive   color.RGBA // active/selected button bg
	ButtonAccentBg color.RGBA // accent-highlighted button bg
}{
	PanelBg:      color.RGBA{12, 16, 28, 220},
	PanelBgSolid: color.RGBA{12, 16, 28, 245},
	PanelBgLight: color.RGBA{18, 22, 38, 230},
	PanelBorder:  color.RGBA{30, 40, 68, 255},
	PanelHover:   color.RGBA{25, 32, 55, 200},

	Accent:    color.RGBA{127, 219, 202, 255},
	TextLight: color.RGBA{192, 200, 216, 255},
	TextDim:   color.RGBA{80, 95, 115, 255},

	BarBg: color.RGBA{20, 25, 40, 255},

	ButtonActive:   color.RGBA{25, 35, 60, 230},
	ButtonAccentBg: color.RGBA{30, 35, 65, 230},
}
