package rendering

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/assets"
	"github.com/hunterjsb/xandaris/utils"
)

// SpriteRenderer handles rendering of sprites for entities
type SpriteRenderer struct {
	circleCache   *utils.CircleImageCache
	rectCache     *utils.RectImageCache
	triangleCache *utils.TriangleImageCache
	assetLoader   *assets.AssetLoader
	animationTick int
}

// NewSpriteRenderer creates a new sprite renderer
func NewSpriteRenderer() *SpriteRenderer {
	return &SpriteRenderer{
		circleCache:   utils.NewCircleImageCache(),
		rectCache:     utils.NewRectImageCache(),
		triangleCache: utils.NewTriangleImageCache(),
		assetLoader:   assets.GetLoader(),
		animationTick: 0,
	}
}

// Update increments the animation tick
func (sr *SpriteRenderer) Update() {
	sr.animationTick++
}

// RenderOptions contains options for rendering sprites
type RenderOptions struct {
	X, Y          int
	Scale         float64
	Rotation      float64
	CenterX       bool
	CenterY       bool
	Alpha         float32
	ColorScale    color.RGBA
	UseSprite     bool
	SpritePath    string
	FallbackColor color.RGBA
	FallbackSize  int
	FallbackShape string // "circle", "rect", "triangle"
}

// DefaultRenderOptions returns default render options
func DefaultRenderOptions() *RenderOptions {
	return &RenderOptions{
		X:             0,
		Y:             0,
		Scale:         1.0,
		Rotation:      0.0,
		CenterX:       true,
		CenterY:       true,
		Alpha:         1.0,
		ColorScale:    color.RGBA{255, 255, 255, 255},
		UseSprite:     false,
		FallbackColor: color.RGBA{100, 100, 100, 255},
		FallbackSize:  10,
		FallbackShape: "circle",
	}
}

// RenderSprite renders a sprite with the given options
func (sr *SpriteRenderer) RenderSprite(screen *ebiten.Image, opts *RenderOptions) error {
	if opts == nil {
		opts = DefaultRenderOptions()
	}

	var img *ebiten.Image
	var err error

	// Try to load sprite if requested
	if opts.UseSprite && opts.SpritePath != "" {
		// Check if it's an animated sprite (GIF)
		if sprite, err := sr.assetLoader.LoadAnimatedSprite(opts.SpritePath); err == nil {
			img = sprite.GetFrame(sr.animationTick)
		} else {
			// Try as static image
			img, err = sr.assetLoader.LoadStaticImage(opts.SpritePath)
			if err != nil {
				// Fall back to generated shape
				img = sr.generateFallbackSprite(opts)
			}
		}
	} else {
		// Use generated shape
		img = sr.generateFallbackSprite(opts)
	}

	if img == nil {
		return nil
	}

	// Create draw options
	drawOpts := &ebiten.DrawImageOptions{}

	// Get image bounds
	bounds := img.Bounds()
	width := float64(bounds.Dx())
	height := float64(bounds.Dy())

	// Center the image if requested
	translateX := 0.0
	translateY := 0.0
	if opts.CenterX {
		translateX = -width / 2
	}
	if opts.CenterY {
		translateY = -height / 2
	}
	if opts.CenterX || opts.CenterY {
		drawOpts.GeoM.Translate(translateX, translateY)
	}

	// Apply scale
	if opts.Scale != 1.0 {
		drawOpts.GeoM.Scale(opts.Scale, opts.Scale)
	}

	// Apply rotation
	if opts.Rotation != 0.0 {
		drawOpts.GeoM.Rotate(opts.Rotation)
	}

	// Apply translation
	drawOpts.GeoM.Translate(float64(opts.X), float64(opts.Y))

	// Apply color scale and alpha
	drawOpts.ColorScale.ScaleWithColor(opts.ColorScale)
	if opts.Alpha != 1.0 {
		drawOpts.ColorScale.ScaleAlpha(opts.Alpha)
	}

	screen.DrawImage(img, drawOpts)
	return err
}

// generateFallbackSprite creates a fallback sprite based on shape
func (sr *SpriteRenderer) generateFallbackSprite(opts *RenderOptions) *ebiten.Image {
	switch opts.FallbackShape {
	case "circle":
		return sr.circleCache.GetOrCreate(opts.FallbackSize, opts.FallbackColor)
	case "rect", "square":
		return sr.rectCache.GetOrCreate(opts.FallbackSize*2, opts.FallbackSize*2, opts.FallbackColor)
	case "triangle":
		return sr.triangleCache.GetOrCreate(opts.FallbackSize, opts.FallbackColor)
	default:
		return sr.circleCache.GetOrCreate(opts.FallbackSize, opts.FallbackColor)
	}
}

// RenderPlanet renders a planet with optional sprite
func (sr *SpriteRenderer) RenderPlanet(screen *ebiten.Image, x, y, radius int, planetType string, c color.RGBA) error {
	// Try to load planet sprite
	sprite, err := sr.assetLoader.LoadPlanetSprite(planetType)

	opts := &RenderOptions{
		X:             x,
		Y:             y,
		CenterX:       true,
		CenterY:       true,
		FallbackColor: c,
		FallbackSize:  radius,
		FallbackShape: "circle",
	}

	if err == nil && sprite != nil {
		// Calculate scale to match desired radius
		targetSize := float64(radius * 2)
		spriteSize := float64(sprite.Width)
		if spriteSize > 0 {
			opts.Scale = targetSize / spriteSize
		}
		opts.UseSprite = true
		opts.SpritePath = "" // Already loaded

		// Render the animated frame directly
		img := sprite.GetFrame(sr.animationTick)
		if img != nil {
			drawOpts := &ebiten.DrawImageOptions{}

			// Center and scale
			bounds := img.Bounds()
			width := float64(bounds.Dx())
			height := float64(bounds.Dy())
			drawOpts.GeoM.Translate(-width/2, -height/2)
			drawOpts.GeoM.Scale(opts.Scale, opts.Scale)
			drawOpts.GeoM.Translate(float64(x), float64(y))

			screen.DrawImage(img, drawOpts)
			return nil
		}
	}

	// Fallback to circle
	return sr.RenderSprite(screen, opts)
}

// RenderBuilding renders a building with optional sprite
func (sr *SpriteRenderer) RenderBuilding(screen *ebiten.Image, x, y, size int, buildingType string, c color.RGBA) error {
	opts := &RenderOptions{
		X:             x,
		Y:             y,
		CenterX:       true,
		CenterY:       true,
		FallbackColor: c,
		FallbackSize:  size,
		FallbackShape: "rect",
	}

	// Try to load building sprite
	img, err := sr.assetLoader.LoadBuildingSprite(buildingType)
	if err == nil && img != nil {
		// Calculate scale to match desired size
		bounds := img.Bounds()
		targetSize := float64(size * 2)
		spriteSize := float64(bounds.Dx())
		if spriteSize > 0 {
			opts.Scale = targetSize / spriteSize
		}

		drawOpts := &ebiten.DrawImageOptions{}
		width := float64(bounds.Dx())
		height := float64(bounds.Dy())
		drawOpts.GeoM.Translate(-width/2, -height/2)
		drawOpts.GeoM.Scale(opts.Scale, opts.Scale)
		drawOpts.GeoM.Translate(float64(x), float64(y))

		screen.DrawImage(img, drawOpts)
		return nil
	}

	// Fallback to rectangle
	return sr.RenderSprite(screen, opts)
}

// RenderResource renders a resource node
func (sr *SpriteRenderer) RenderResource(screen *ebiten.Image, x, y, radius int, resourceType string, c color.RGBA) error {
	opts := &RenderOptions{
		X:             x,
		Y:             y,
		CenterX:       true,
		CenterY:       true,
		FallbackColor: c,
		FallbackSize:  radius,
		FallbackShape: "circle",
	}

	// Resources use circle fallback for now
	// Can be extended to load resource-specific sprites later
	return sr.RenderSprite(screen, opts)
}

// RenderFleet renders a fleet/ship
func (sr *SpriteRenderer) RenderFleet(screen *ebiten.Image, x, y, size int, c color.RGBA) error {
	opts := &RenderOptions{
		X:             x,
		Y:             y,
		CenterX:       true,
		CenterY:       true,
		FallbackColor: c,
		FallbackSize:  size,
		FallbackShape: "triangle",
	}

	// Fleets use triangle fallback
	return sr.RenderSprite(screen, opts)
}

// RenderStar renders a star
func (sr *SpriteRenderer) RenderStar(screen *ebiten.Image, x, y, radius int, starType string, c color.RGBA) error {
	opts := &RenderOptions{
		X:             x,
		Y:             y,
		CenterX:       true,
		CenterY:       true,
		FallbackColor: c,
		FallbackSize:  radius,
		FallbackShape: "circle",
	}

	// Stars use circle fallback for now
	// Can be extended to load star-specific sprites later
	return sr.RenderSprite(screen, opts)
}

// GetAnimationTick returns the current animation tick
func (sr *SpriteRenderer) GetAnimationTick() int {
	return sr.animationTick
}

// SetAnimationTick sets the animation tick (useful for synchronization)
func (sr *SpriteRenderer) SetAnimationTick(tick int) {
	sr.animationTick = tick
}

// GetAssetLoader returns the asset loader
func (sr *SpriteRenderer) GetAssetLoader() *assets.AssetLoader {
	return sr.assetLoader
}

// Release deallocates cached resources
func (sr *SpriteRenderer) Release() {
	sr.circleCache.ReleaseAll()
	sr.rectCache.ReleaseAll()
	sr.triangleCache.ReleaseAll()
}
