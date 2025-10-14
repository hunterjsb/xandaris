package views

import "math"

// ContextMenuProvider interface for objects that can provide context menu data
type ContextMenuProvider interface {
	GetContextMenuTitle() string
	GetContextMenuItems() []string
}

// Clickable interface for objects that can be clicked and have a position
type Clickable interface {
	ContextMenuProvider
	GetPosition() (float64, float64) // Returns X, Y coordinates
	GetClickRadius() float64         // Returns click detection radius
}

// ClickHandler manages clickable objects and context menus
type ClickHandler struct {
	clickables     []Clickable
	activeMenu     *ContextMenu
	selectedObject Clickable
}

// NewClickHandler creates a new click handler
func NewClickHandler() *ClickHandler {
	return &ClickHandler{
		clickables: make([]Clickable, 0),
	}
}

// AddClickable adds an object to the click handler
func (c *ClickHandler) AddClickable(clickable Clickable) {
	c.clickables = append(c.clickables, clickable)
}

// ClearClickables removes all clickable objects
func (c *ClickHandler) ClearClickables() {
	c.clickables = make([]Clickable, 0)
	c.activeMenu = nil
	c.selectedObject = nil
}

// HandleClick processes a click at the given coordinates
func (c *ClickHandler) HandleClick(x, y int) bool {
	// Check clickables in reverse order (last added = first checked)
	// This gives priority to smaller objects added later
	for i := len(c.clickables) - 1; i >= 0; i-- {
		clickable := c.clickables[i]
		objX, objY := clickable.GetPosition()
		dx := float64(x) - objX
		dy := float64(y) - objY
		distance := math.Sqrt(dx*dx + dy*dy)

		if distance <= clickable.GetClickRadius() {
			// Toggle selection - if already selected, deselect
			if c.selectedObject == clickable {
				c.selectedObject = nil
				c.activeMenu = nil
			} else {
				c.selectedObject = clickable
				c.createContextMenu(clickable, int(objX), int(objY))
			}
			return true // Click was handled
		}
	}

	// No object was clicked, deselect
	c.selectedObject = nil
	c.activeMenu = nil
	return false
}

// Select programmatically selects a clickable object
func (c *ClickHandler) Select(clickable Clickable) {
	if clickable == nil {
		c.selectedObject = nil
		c.activeMenu = nil
		return
	}

	objX, objY := clickable.GetPosition()
	c.selectedObject = clickable
	c.createContextMenu(clickable, int(objX), int(objY))
}

// createContextMenu creates and positions a context menu for the given object
func (c *ClickHandler) createContextMenu(clickable Clickable, x, y int) {
	c.activeMenu = NewContextMenuFromProvider(clickable)
	c.activeMenu.PositionNear(x, y, int(clickable.GetClickRadius())+20)
}

// GetSelectedObject returns the currently selected object
func (c *ClickHandler) GetSelectedObject() Clickable {
	return c.selectedObject
}

// GetActiveMenu returns the active context menu
func (c *ClickHandler) GetActiveMenu() *ContextMenu {
	return c.activeMenu
}

// HasActiveMenu returns whether there is an active menu
func (c *ClickHandler) HasActiveMenu() bool {
	return c.activeMenu != nil
}
