package assets

import (
	"embed"
	"fmt"
	"image"
	"image/gif"
	"io"
	"io/fs"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed gifs/**
var assetsFS embed.FS

// AssetLoader manages loading and caching of game assets
type AssetLoader struct {
	mu              sync.RWMutex
	staticImages    map[string]*ebiten.Image
	animatedSprites map[string]*AnimatedSprite
}

var (
	globalLoader     *AssetLoader
	globalLoaderOnce sync.Once
)

// GetLoader returns the global asset loader instance
func GetLoader() *AssetLoader {
	globalLoaderOnce.Do(func() {
		globalLoader = &AssetLoader{
			staticImages:    make(map[string]*ebiten.Image),
			animatedSprites: make(map[string]*AnimatedSprite),
		}
	})
	return globalLoader
}

// AnimatedSprite represents an animated GIF sprite
type AnimatedSprite struct {
	Frames   []*ebiten.Image
	Delays   []int // Delay in 100ths of a second for each frame
	Width    int
	Height   int
	LoopMode int // 0 = loop forever, -1 = no loop
}

// GetFrame returns the appropriate frame for the given time
func (as *AnimatedSprite) GetFrame(tick int) *ebiten.Image {
	if len(as.Frames) == 0 {
		return nil
	}
	if len(as.Frames) == 1 {
		return as.Frames[0]
	}

	// Calculate total animation duration
	totalDelay := 0
	for _, delay := range as.Delays {
		totalDelay += delay
	}
	if totalDelay == 0 {
		return as.Frames[0]
	}

	// Loop the tick based on total delay
	tickMod := tick % totalDelay

	// Find which frame to show
	currentDelay := 0
	for i, delay := range as.Delays {
		currentDelay += delay
		if tickMod < currentDelay {
			return as.Frames[i]
		}
	}

	return as.Frames[len(as.Frames)-1]
}

// FrameCount returns the number of frames in the animation
func (as *AnimatedSprite) FrameCount() int {
	return len(as.Frames)
}

// LoadStaticImage loads a static image from the embedded assets
func (al *AssetLoader) LoadStaticImage(path string) (*ebiten.Image, error) {
	al.mu.RLock()
	if img, exists := al.staticImages[path]; exists {
		al.mu.RUnlock()
		return img, nil
	}
	al.mu.RUnlock()

	al.mu.Lock()
	defer al.mu.Unlock()

	// Double-check after acquiring write lock
	if img, exists := al.staticImages[path]; exists {
		return img, nil
	}

	// Load from embedded filesystem
	data, err := assetsFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset %s: %w", path, err)
	}

	// Decode image
	img, _, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %s: %w", path, err)
	}

	// Convert to ebiten image
	ebitenImg := ebiten.NewImageFromImage(img)
	al.staticImages[path] = ebitenImg

	return ebitenImg, nil
}

// LoadAnimatedSprite loads an animated GIF from the embedded assets
func (al *AssetLoader) LoadAnimatedSprite(path string) (*AnimatedSprite, error) {
	al.mu.RLock()
	if sprite, exists := al.animatedSprites[path]; exists {
		al.mu.RUnlock()
		return sprite, nil
	}
	al.mu.RUnlock()

	al.mu.Lock()
	defer al.mu.Unlock()

	// Double-check after acquiring write lock
	if sprite, exists := al.animatedSprites[path]; exists {
		return sprite, nil
	}

	// Load from embedded filesystem
	data, err := assetsFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset %s: %w", path, err)
	}

	// Decode GIF
	gifImg, err := gif.DecodeAll(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode GIF %s: %w", path, err)
	}

	// Convert frames to ebiten images
	sprite := &AnimatedSprite{
		Frames:   make([]*ebiten.Image, len(gifImg.Image)),
		Delays:   gifImg.Delay,
		LoopMode: gifImg.LoopCount,
	}

	for i, frame := range gifImg.Image {
		sprite.Frames[i] = ebiten.NewImageFromImage(frame)
		if i == 0 {
			sprite.Width = frame.Bounds().Dx()
			sprite.Height = frame.Bounds().Dy()
		}
	}

	al.animatedSprites[path] = sprite

	return sprite, nil
}

// LoadPlanetSprite loads a planet sprite by planet type
func (al *AssetLoader) LoadPlanetSprite(planetType string) (*AnimatedSprite, error) {
	// Normalize planet type to lowercase for filename
	normalizedType := strings.ToLower(planetType)
	path := fmt.Sprintf("gifs/planets/%s.gif", normalizedType)

	sprite, err := al.LoadAnimatedSprite(path)
	if err != nil {
		// Return a default/fallback if specific type not found
		return nil, fmt.Errorf("planet sprite not found for type %s: %w", planetType, err)
	}

	return sprite, nil
}

// ListAvailablePlanetTypes returns all available planet sprite types
func (al *AssetLoader) ListAvailablePlanetTypes() ([]string, error) {
	entries, err := fs.ReadDir(assetsFS, "gifs/planets")
	if err != nil {
		return nil, fmt.Errorf("failed to read planets directory: %w", err)
	}

	types := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".gif") {
			// Remove .gif extension and capitalize
			typeName := strings.TrimSuffix(entry.Name(), ".gif")
			types = append(types, typeName)
		}
	}

	return types, nil
}

// PreloadPlanetSprites preloads all planet sprites into cache
func (al *AssetLoader) PreloadPlanetSprites() error {
	types, err := al.ListAvailablePlanetTypes()
	if err != nil {
		return err
	}

	for _, planetType := range types {
		_, err := al.LoadPlanetSprite(planetType)
		if err != nil {
			return fmt.Errorf("failed to preload %s: %w", planetType, err)
		}
	}

	return nil
}

// LoadBuildingSprite loads a building sprite by building type
func (al *AssetLoader) LoadBuildingSprite(buildingType string) (*ebiten.Image, error) {
	// Normalize building type to lowercase for filename
	normalizedType := strings.ToLower(strings.ReplaceAll(buildingType, " ", "_"))
	path := fmt.Sprintf("gifs/buildings/%s.gif", normalizedType)

	// Try to load as animated first
	sprite, err := al.LoadAnimatedSprite(path)
	if err == nil && len(sprite.Frames) > 0 {
		// Return first frame for now (we can enhance this later)
		return sprite.Frames[0], nil
	}

	// Try as static image
	pngPath := fmt.Sprintf("images/buildings/%s.png", normalizedType)
	img, err := al.LoadStaticImage(pngPath)
	if err != nil {
		return nil, fmt.Errorf("building sprite not found for type %s: %w", buildingType, err)
	}

	return img, nil
}

// ClearCache clears all cached assets and releases memory
func (al *AssetLoader) ClearCache() {
	al.mu.Lock()
	defer al.mu.Unlock()

	// Deallocate static images
	for _, img := range al.staticImages {
		if img != nil {
			img.Deallocate()
		}
	}
	al.staticImages = make(map[string]*ebiten.Image)

	// Deallocate animated sprite frames
	for _, sprite := range al.animatedSprites {
		if sprite != nil {
			for _, frame := range sprite.Frames {
				if frame != nil {
					frame.Deallocate()
				}
			}
		}
	}
	al.animatedSprites = make(map[string]*AnimatedSprite)
}

// GetCacheStats returns statistics about cached assets
func (al *AssetLoader) GetCacheStats() map[string]int {
	al.mu.RLock()
	defer al.mu.RUnlock()

	totalFrames := 0
	for _, sprite := range al.animatedSprites {
		totalFrames += len(sprite.Frames)
	}

	return map[string]int{
		"static_images":    len(al.staticImages),
		"animated_sprites": len(al.animatedSprites),
		"total_gif_frames": totalFrames,
	}
}

// ReadAssetFile reads a raw asset file (for non-image assets)
func (al *AssetLoader) ReadAssetFile(path string) ([]byte, error) {
	return assetsFS.ReadFile(path)
}

// OpenAssetFile opens an asset file for streaming reads
func (al *AssetLoader) OpenAssetFile(path string) (fs.File, error) {
	return assetsFS.Open(path)
}

// AssetExists checks if an asset exists at the given path
func (al *AssetLoader) AssetExists(path string) bool {
	_, err := fs.Stat(assetsFS, path)
	return err == nil
}

// WalkAssets walks through all embedded assets
func (al *AssetLoader) WalkAssets(root string, fn func(path string, info fs.DirEntry) error) error {
	return fs.WalkDir(assetsFS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return fn(path, d)
	})
}

// Helper function to load image from io.Reader
func decodeImage(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	return img, err
}

// GetPlanetSpriteForType is a convenience function for getting planet sprites
func GetPlanetSpriteForType(planetType string) (*AnimatedSprite, error) {
	return GetLoader().LoadPlanetSprite(planetType)
}

// GetBuildingSpriteForType is a convenience function for getting building sprites
func GetBuildingSpriteForType(buildingType string) (*ebiten.Image, error) {
	return GetLoader().LoadBuildingSprite(buildingType)
}

// PreloadCommonAssets preloads commonly used assets
func PreloadCommonAssets() error {
	loader := GetLoader()

	// Preload all planet sprites
	if err := loader.PreloadPlanetSprites(); err != nil {
		return fmt.Errorf("failed to preload planet sprites: %w", err)
	}

	return nil
}
