package ui

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/utils"
)

const (
	maxConsoleLines = 10
	consoleHeight   = 200
	consolePadding  = 10
	consoleFontSize = 12
)

var (
	consoleColor = color.NRGBA{R: 20, G: 20, B: 20, A: 200}
)

// NewConsole creates a new Console instance
func NewConsole() *Console {
	return &Console{
		history:     []string{},
		currentLine: "",
	}
}

type Console struct {
	history     []string
	currentLine string
	active      bool
}

func (c *Console) IsActive() bool {
	return c.active
}

func (c *Console) Toggle() {
	c.active = !c.active
}

func (c *Console) Update() {
	if !c.active {
		return
	}

	c.currentLine += string(ebiten.InputChars())
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		c.processCommand()
		c.currentLine = ""
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(c.currentLine) > 0 {
			c.currentLine = c.currentLine[:len(c.currentLine)-1]
		}
	}
}

func (c *Console) Draw(screen *ebiten.Image) {
	if !c.active {
		return
	}

	// Draw console background
	screenWidth, _ := ebiten.WindowSize()
	panel := utils.NewUIPanel(0, 0, screenWidth, consoleHeight)
	panel.Draw(screen)

	// Draw history
	y := consoleHeight - consolePadding - consoleFontSize
	for i := len(c.history) - 1; i >= 0; i-- {
		utils.DrawText(screen, c.history[i], consolePadding, y, utils.TextPrimary)
		y -= consoleFontSize
		if y < 0 {
			break
		}
	}

	// Draw current line
	utils.DrawText(screen, "> "+c.currentLine, consolePadding, consoleHeight-consolePadding, utils.TextPrimary)
}

func (c *Console) processCommand() {
	c.addLine(c.currentLine)
	parts := strings.Split(c.currentLine, " ")
	command := parts[0]
	args := parts[1:]

	switch command {
	case "echo":
		c.addLine(strings.Join(args, " "))
	default:
		c.addLine("Unknown command: " + command)
	}
}

func (c *Console) addLine(line string) {
	c.history = append(c.history, line)
	if len(c.history) > maxConsoleLines {
		c.history = c.history[1:]
	}
}
