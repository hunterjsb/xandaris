package utils

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// ImageCache manages reusable ebiten.Image instances to avoid frequent allocations
type ImageCache struct {
	mu     sync.RWMutex
	images map[string]*ebiten.Image
}

// NewImageCache creates a new image cache
func NewImageCache() *ImageCache {
	return &ImageCache{
		images: make(map[string]*ebiten.Image),
	}
}

// GetOrCreate retrieves a cached image or creates a new one if it doesn't exist
// The key should uniquely identify the image's properties (e.g., "rect_100x50")
func (ic *ImageCache) GetOrCreate(key string, width, height int) *ebiten.Image {
	ic.mu.RLock()
	img, exists := ic.images[key]
	ic.mu.RUnlock()

	if exists && img != nil {
		return img
	}

	ic.mu.Lock()
	defer ic.mu.Unlock()

	// Double-check after acquiring write lock
	if img, exists := ic.images[key]; exists && img != nil {
		return img
	}

	// Create new image
	img = ebiten.NewImage(width, height)
	ic.images[key] = img
	return img
}

// Clear clears an image in the cache (useful for reuse with different content)
func (ic *ImageCache) Clear(key string) {
	ic.mu.RLock()
	img, exists := ic.images[key]
	ic.mu.RUnlock()

	if exists && img != nil {
		img.Clear()
	}
}

// ClearAll clears all images in the cache
func (ic *ImageCache) ClearAll() {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	for _, img := range ic.images {
		if img != nil {
			img.Clear()
		}
	}
}

// Release deallocates a specific cached image
func (ic *ImageCache) Release(key string) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	if img, exists := ic.images[key]; exists && img != nil {
		img.Deallocate()
		delete(ic.images, key)
	}
}

// ReleaseAll deallocates all cached images
func (ic *ImageCache) ReleaseAll() {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	for key, img := range ic.images {
		if img != nil {
			img.Deallocate()
		}
		delete(ic.images, key)
	}
}

// CircleImageCache manages cached circle images of various sizes and colors
type CircleImageCache struct {
	mu     sync.RWMutex
	images map[circleKey]*ebiten.Image
}

type circleKey struct {
	radius     int
	r, g, b, a uint8
}

// NewCircleImageCache creates a new circle image cache
func NewCircleImageCache() *CircleImageCache {
	return &CircleImageCache{
		images: make(map[circleKey]*ebiten.Image),
	}
}

// GetOrCreate retrieves or creates a circle image with the given radius and color
func (cc *CircleImageCache) GetOrCreate(radius int, c color.RGBA) *ebiten.Image {
	key := circleKey{radius: radius, r: c.R, g: c.G, b: c.B, a: c.A}

	cc.mu.RLock()
	img, exists := cc.images[key]
	cc.mu.RUnlock()

	if exists && img != nil {
		return img
	}

	cc.mu.Lock()
	defer cc.mu.Unlock()

	// Double-check after acquiring write lock
	if img, exists := cc.images[key]; exists && img != nil {
		return img
	}

	// Create new circle image
	img = createCircleImage(radius, c)
	cc.images[key] = img
	return img
}

// ReleaseAll deallocates all cached circle images
func (cc *CircleImageCache) ReleaseAll() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	for key, img := range cc.images {
		if img != nil {
			img.Deallocate()
		}
		delete(cc.images, key)
	}
}

// createCircleImage creates a filled circle image
func createCircleImage(radius int, c color.RGBA) *ebiten.Image {
	img := ebiten.NewImage(radius*2, radius*2)
	for py := 0; py < radius*2; py++ {
		for px := 0; px < radius*2; px++ {
			dx := px - radius
			dy := py - radius
			if dx*dx+dy*dy <= radius*radius {
				img.Set(px, py, c)
			}
		}
	}
	return img
}

// RectImageCache manages cached rectangle images
type RectImageCache struct {
	mu     sync.RWMutex
	images map[rectKey]*ebiten.Image
}

type rectKey struct {
	width, height int
	r, g, b, a    uint8
}

// NewRectImageCache creates a new rectangle image cache
func NewRectImageCache() *RectImageCache {
	return &RectImageCache{
		images: make(map[rectKey]*ebiten.Image),
	}
}

// GetOrCreate retrieves or creates a filled rectangle image
func (rc *RectImageCache) GetOrCreate(width, height int, c color.RGBA) *ebiten.Image {
	key := rectKey{width: width, height: height, r: c.R, g: c.G, b: c.B, a: c.A}

	rc.mu.RLock()
	img, exists := rc.images[key]
	rc.mu.RUnlock()

	if exists && img != nil {
		return img
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Double-check after acquiring write lock
	if img, exists := rc.images[key]; exists && img != nil {
		return img
	}

	// Create new rectangle image
	img = ebiten.NewImage(width, height)
	img.Fill(c)
	rc.images[key] = img
	return img
}

// ReleaseAll deallocates all cached rectangle images
func (rc *RectImageCache) ReleaseAll() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	for key, img := range rc.images {
		if img != nil {
			img.Deallocate()
		}
		delete(rc.images, key)
	}
}

// TriangleImageCache manages cached triangle images for ships
type TriangleImageCache struct {
	mu     sync.RWMutex
	images map[triangleKey]*ebiten.Image
}

type triangleKey struct {
	size       int
	r, g, b, a uint8
}

// NewTriangleImageCache creates a new triangle image cache
func NewTriangleImageCache() *TriangleImageCache {
	return &TriangleImageCache{
		images: make(map[triangleKey]*ebiten.Image),
	}
}

// GetOrCreate retrieves or creates a triangle image (pointing upward)
func (tc *TriangleImageCache) GetOrCreate(size int, c color.RGBA) *ebiten.Image {
	key := triangleKey{size: size, r: c.R, g: c.G, b: c.B, a: c.A}

	tc.mu.RLock()
	img, exists := tc.images[key]
	tc.mu.RUnlock()

	if exists && img != nil {
		return img
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Double-check after acquiring write lock
	if img, exists := tc.images[key]; exists && img != nil {
		return img
	}

	// Create new triangle image
	img = createTriangleImage(size, c)
	tc.images[key] = img
	return img
}

// ReleaseAll deallocates all cached triangle images
func (tc *TriangleImageCache) ReleaseAll() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	for key, img := range tc.images {
		if img != nil {
			img.Deallocate()
		}
		delete(tc.images, key)
	}
}

// createTriangleImage creates a triangle image (pointing upward)
func createTriangleImage(size int, c color.RGBA) *ebiten.Image {
	img := ebiten.NewImage(size*2, size*2)
	for py := 0; py < size*2; py++ {
		for px := 0; px < size*2; px++ {
			dx := float64(px - size)
			dy := float64(py - size)
			// Triangle pointing upward: y > 0 and within horizontal bounds
			if dy > 0 && math.Abs(dx) < float64(size)-dy/2 {
				img.Set(px, py, c)
			}
		}
	}
	return img
}
