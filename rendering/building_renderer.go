package rendering

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/utils"
)

// BuildingRenderer handles rendering of buildings and their attachments
type BuildingRenderer struct {
	spriteRenderer *SpriteRenderer
}

// NewBuildingRenderer creates a new building renderer
func NewBuildingRenderer(spriteRenderer *SpriteRenderer) *BuildingRenderer {
	return &BuildingRenderer{
		spriteRenderer: spriteRenderer,
	}
}

// RenderBuilding renders a building entity with all its visual elements
func (br *BuildingRenderer) RenderBuilding(screen *ebiten.Image, building *entities.Building, centerX, centerY int) error {
	// Get building position
	x, y := building.GetAbsolutePosition()
	if x == 0 && y == 0 {
		// Use provided center if no absolute position set
		x = float64(centerX)
		y = float64(centerY)
	}

	buildingX := int(x)
	buildingY := int(y)
	size := building.Size

	// Draw ownership ring if owned
	if building.Owner != "" {
		if ownerColor, ok := br.getOwnerColorFromContext(building.Owner); ok {
			br.drawOwnershipRing(screen, buildingX, buildingY, float64(size+2), ownerColor)
		}
	}

	// Render the building sprite or fallback
	err := br.spriteRenderer.RenderBuilding(
		screen,
		buildingX,
		buildingY,
		size,
		building.BuildingType,
		building.Color,
	)
	if err != nil {
		return fmt.Errorf("failed to render building: %w", err)
	}

	// Draw building label
	if building.Name != "" {
		br.drawBuildingLabel(screen, buildingX, buildingY+size+12, building.BuildingType)
	}

	// Draw operational status indicator if not operational
	if !building.IsOperational {
		br.drawOfflineIndicator(screen, buildingX, buildingY-size-8)
	}

	// Draw attachment indicator if building has attachments
	if building.HasAttachments() {
		br.drawAttachmentIndicator(screen, buildingX+size, buildingY-size)
	}

	return nil
}

// RenderBuildingWithAttachments renders a building and all its attached entities
func (br *BuildingRenderer) RenderBuildingWithAttachments(screen *ebiten.Image, building *entities.Building, centerX, centerY int) error {
	// Render the building itself
	if err := br.RenderBuilding(screen, building, centerX, centerY); err != nil {
		return err
	}

	// Render attachments
	attachments := building.GetAttachments()
	for i, attachment := range attachments {
		br.renderAttachment(screen, building, attachment, i, len(attachments))
	}

	return nil
}

// renderAttachment renders an attached entity relative to its parent
func (br *BuildingRenderer) renderAttachment(screen *ebiten.Image, parent *entities.Building, attachment entities.Entity, index, total int) {
	x, y := parent.GetAbsolutePosition()
	if x == 0 && y == 0 {
		return // Can't render without parent position
	}

	// Get attachment position
	attachPos := attachment.(*entities.BaseEntity).GetAttachmentPosition()

	// Calculate position relative to parent
	attachX := x + attachPos.OffsetX
	attachY := y + attachPos.OffsetY

	// If no explicit position is set, arrange in a circle around parent
	if attachPos.OffsetX == 0 && attachPos.OffsetY == 0 && total > 0 {
		angle := (float64(index) / float64(total)) * 2 * math.Pi
		radius := float64(parent.Size + 10)
		attachX = x + radius*math.Cos(angle)
		attachY = y + radius*math.Sin(angle)
	}

	// Render based on entity type
	switch attachment.GetType() {
	case entities.EntityTypeBuilding:
		if building, ok := attachment.(*entities.Building); ok {
			br.RenderBuilding(screen, building, int(attachX), int(attachY))
		}
	case entities.EntityTypeResource:
		if resource, ok := attachment.(*entities.Resource); ok {
			br.renderResourceAttachment(screen, resource, int(attachX), int(attachY))
		}
	default:
		// Render generic attachment
		br.renderGenericAttachment(screen, attachment, int(attachX), int(attachY))
	}

	// Draw connection line from parent to attachment
	br.drawAttachmentLine(screen, int(x), int(y), int(attachX), int(attachY), parent.Color)
}

// renderResourceAttachment renders a resource as an attachment
func (br *BuildingRenderer) renderResourceAttachment(screen *ebiten.Image, resource *entities.Resource, x, y int) {
	br.spriteRenderer.RenderResource(
		screen,
		x,
		y,
		resource.Size,
		resource.ResourceType,
		resource.Color,
	)
}

// renderGenericAttachment renders a generic entity attachment
func (br *BuildingRenderer) renderGenericAttachment(screen *ebiten.Image, entity entities.Entity, x, y int) {
	// Draw a small circle for generic attachments
	opts := &RenderOptions{
		X:             x,
		Y:             y,
		CenterX:       true,
		CenterY:       true,
		FallbackColor: entity.GetColor(),
		FallbackSize:  3,
		FallbackShape: "circle",
	}
	br.spriteRenderer.RenderSprite(screen, opts)
}

// drawAttachmentLine draws a connection line between parent and attached entity
func (br *BuildingRenderer) drawAttachmentLine(screen *ebiten.Image, x1, y1, x2, y2 int, c color.RGBA) {
	// Make line semi-transparent
	lineColor := c
	lineColor.A = 100

	dx := x2 - x1
	dy := y2 - y1
	steps := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy))))

	if steps == 0 {
		return
	}

	xStep := float64(dx) / float64(steps)
	yStep := float64(dy) / float64(steps)

	for i := 0; i <= steps; i++ {
		x := x1 + int(float64(i)*xStep)
		y := y1 + int(float64(i)*yStep)
		screen.Set(x, y, lineColor)
	}
}

// drawBuildingLabel draws the building type label
func (br *BuildingRenderer) drawBuildingLabel(screen *ebiten.Image, x, y int, label string) {
	// This will be replaced with proper text rendering
	// For now, we'll leave it as a placeholder that can be called from the view
}

// drawOwnershipRing draws a colored ring around a building to indicate ownership
func (br *BuildingRenderer) drawOwnershipRing(screen *ebiten.Image, centerX, centerY int, radius float64, ownerColor color.RGBA) {
	ringColor := ownerColor
	ringColor.A = 180

	segments := 32
	for i := 0; i < segments; i++ {
		angle1 := float64(i) * 2 * math.Pi / float64(segments)
		angle2 := float64(i+1) * 2 * math.Pi / float64(segments)

		x1 := centerX + int(radius*math.Cos(angle1))
		y1 := centerY + int(radius*math.Sin(angle1))
		x2 := centerX + int(radius*math.Cos(angle2))
		y2 := centerY + int(radius*math.Sin(angle2))

		br.drawLine(screen, x1, y1, x2, y2, ringColor)
	}
}

// drawLine draws a simple line between two points
func (br *BuildingRenderer) drawLine(screen *ebiten.Image, x1, y1, x2, y2 int, c color.RGBA) {
	dx := x2 - x1
	dy := y2 - y1
	steps := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy))))

	if steps == 0 {
		return
	}

	xStep := float64(dx) / float64(steps)
	yStep := float64(dy) / float64(steps)

	for i := 0; i <= steps; i++ {
		x := x1 + int(float64(i)*xStep)
		y := y1 + int(float64(i)*yStep)
		screen.Set(x, y, c)
	}
}

// drawOfflineIndicator draws an indicator showing the building is offline
func (br *BuildingRenderer) drawOfflineIndicator(screen *ebiten.Image, x, y int) {
	offlineColor := color.RGBA{200, 50, 50, 255}
	// Draw a small X or circle to indicate offline status
	for dx := -2; dx <= 2; dx++ {
		screen.Set(x+dx, y+dx, offlineColor)
		screen.Set(x+dx, y-dx, offlineColor)
	}
}

// drawAttachmentIndicator draws a small indicator showing the building has attachments
func (br *BuildingRenderer) drawAttachmentIndicator(screen *ebiten.Image, x, y int) {
	indicatorColor := color.RGBA{100, 200, 255, 255}
	// Draw a small dot
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			screen.Set(x+dx, y+dy, indicatorColor)
		}
	}
}

// getOwnerColorFromContext gets the owner color (placeholder for actual context access)
func (br *BuildingRenderer) getOwnerColorFromContext(owner string) (color.RGBA, bool) {
	// This will need to be implemented with actual context access
	// For now, return a default color
	if owner != "" {
		return utils.PlayerGreen, true
	}
	return color.RGBA{}, false
}

// AttachBuildingToResource attaches a building to a resource node
func AttachBuildingToResource(building *entities.Building, resource *entities.Resource, offsetX, offsetY float64) {
	// Set attachment metadata on building
	building.AttachedTo = fmt.Sprintf("%d", resource.GetID())
	building.AttachmentType = "Resource"
	building.ResourceNodeID = resource.GetID()

	// Set attachment position
	baseBuilding := &building.BaseEntity
	baseBuilding.SetAttachmentPosition(entities.AttachmentPosition{
		OffsetX:       offsetX,
		OffsetY:       offsetY,
		RelativeAngle: 0,
		RelativeScale: 1.0,
	})

	// Attach to resource's base entity
	baseResource := &resource.BaseEntity
	baseResource.AttachEntity(building)
}

// DetachBuildingFromResource detaches a building from a resource node
func DetachBuildingFromResource(building *entities.Building, resource *entities.Resource) bool {
	baseResource := &resource.BaseEntity
	success := baseResource.DetachEntity(building.GetID())

	if success {
		building.AttachedTo = ""
		building.AttachmentType = ""
		building.ResourceNodeID = 0
	}

	return success
}

// GetAttachedBuildings returns all buildings attached to an entity
func GetAttachedBuildings(entity entities.Entity) []*entities.Building {
	baseEntity, ok := entity.(*entities.BaseEntity)
	if !ok {
		return nil
	}

	attachments := baseEntity.GetAttachmentsByType(entities.EntityTypeBuilding)
	buildings := make([]*entities.Building, 0, len(attachments))

	for _, attachment := range attachments {
		if building, ok := attachment.(*entities.Building); ok {
			buildings = append(buildings, building)
		}
	}

	return buildings
}
