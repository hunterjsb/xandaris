# Rendering System Documentation

## Overview

The rendering system has been refactored to provide a clean, flexible architecture for rendering game entities with support for sprites, animations, and entity attachments.

## Architecture

### Core Components

1. **Asset Loader** (`assets/loader.go`)
   - Manages loading and caching of embedded assets
   - Supports static images and animated GIFs
   - Thread-safe with automatic caching

2. **Sprite Renderer** (`rendering/sprite_renderer.go`)
   - Handles rendering of sprites with various options
   - Provides fallback shapes (circles, rectangles, triangles)
   - Manages animation timing

3. **Building Renderer** (`rendering/building_renderer.go`)
   - Specialized renderer for buildings and attachments
   - Handles ownership rings, status indicators
   - Renders attachment relationships

4. **Entity Attachment System** (`entities/types.go`)
   - Generic parent-child relationship system
   - Thread-safe attachment management
   - Supports any entity type attaching to any other

## Asset Loading

### Loading Planet Sprites

```go
import "github.com/hunterjsb/xandaris/assets"

// Get the global loader
loader := assets.GetLoader()

// Load a specific planet type
sprite, err := loader.LoadPlanetSprite("desert")
if err != nil {
    // Handle error
}

// Get current frame for rendering
frame := sprite.GetFrame(animationTick)
```

### Preloading Assets

```go
// Preload all planet sprites at startup
err := assets.PreloadCommonAssets()
if err != nil {
    log.Fatal(err)
}
```

### Available Planet Types

The system automatically loads GIF files from `assets/gifs/planets/`:
- abandoned
- abundant
- barred
- barren
- desert
- fertile
- highlands
- jungle
- mountain
- radiant
- swamp
- volcanic

## Sprite Rendering

### Basic Usage

```go
import "github.com/hunterjsb/xandaris/rendering"

// Create a sprite renderer
renderer := rendering.NewSpriteRenderer()

// Update animation tick each frame
renderer.Update()

// Render a planet with sprite
err := renderer.RenderPlanet(
    screen,
    x, y,           // Center position
    radius,         // Desired radius
    "desert",       // Planet type
    fallbackColor,  // Fallback color if sprite fails to load
)
```

### Render Options

```go
opts := &rendering.RenderOptions{
    X:             100,
    Y:             100,
    Scale:         2.0,              // 2x size
    Rotation:      math.Pi / 4,     // 45 degrees
    CenterX:       true,             // Center horizontally
    CenterY:       true,             // Center vertically
    Alpha:         0.8,              // 80% opacity
    UseSprite:     true,
    SpritePath:    "gifs/planets/desert.gif",
    FallbackColor: color.RGBA{200, 180, 100, 255},
    FallbackSize:  20,
    FallbackShape: "circle",
}

renderer.RenderSprite(screen, opts)
```

## Entity Attachment System

### Overview

Any entity can now attach other entities as children. This is useful for:
- Buildings attached to resources (mines on ore deposits)
- Stations attached to planets
- Modules attached to ships
- Any custom parent-child relationship

### Attaching Entities

```go
// Attach a building to a resource
resource.AttachBuilding(mine)

// Using the generic attachment system
parentEntity.AttachEntity(childEntity)

// Set attachment position relative to parent
childEntity.SetAttachmentPosition(entities.AttachmentPosition{
    OffsetX:       10.0,
    OffsetY:       5.0,
    RelativeAngle: math.Pi / 4,
    RelativeScale: 0.5,  // Half the size of parent
})
```

### Detaching Entities

```go
// Detach a specific building
success := resource.DetachBuilding(buildingID)

// Using generic system
success := parentEntity.DetachEntity(childID)

// Clear all attachments
parentEntity.ClearAttachments()
```

### Querying Attachments

```go
// Get all attachments
attachments := entity.GetAttachments()

// Get attachments of a specific type
buildings := entity.GetAttachmentsByType(entities.EntityTypeBuilding)

// Check if entity has attachments
if entity.HasAttachments() {
    // Handle attached entities
}

// Get parent ID
parentID := entity.GetParentID()
```

### Resource-Specific Methods

Resources have convenience methods for managing buildings:

```go
// Attach a building to a resource
resource.AttachBuilding(mine)

// Get all attached buildings
buildings := resource.GetAttachedBuildings()

// Check if resource has buildings
if resource.HasAttachedBuildings() {
    // Process buildings
}
```

## Building Rendering

### Basic Building Rendering

```go
import "github.com/hunterjsb/xandaris/rendering"

// Create building renderer with sprite renderer
spriteRenderer := rendering.NewSpriteRenderer()
buildingRenderer := rendering.NewBuildingRenderer(spriteRenderer)

// Render a building
err := buildingRenderer.RenderBuilding(screen, building, centerX, centerY)
```

### Rendering with Attachments

```go
// Render building and all attached entities
err := buildingRenderer.RenderBuildingWithAttachments(
    screen,
    building,
    centerX,
    centerY,
)
```

This will:
1. Render the building sprite or fallback shape
2. Draw ownership ring if building is owned
3. Render all attached entities in a circular layout
4. Draw connection lines between parent and children
5. Show status indicators (offline, attachments, etc.)

### Attachment Helper Functions

```go
import "github.com/hunterjsb/xandaris/rendering"

// Attach a building to a resource with offset
rendering.AttachBuildingToResource(building, resource, offsetX, offsetY)

// Detach building from resource
success := rendering.DetachBuildingFromResource(building, resource)

// Get all buildings attached to any entity
buildings := rendering.GetAttachedBuildings(entity)
```

## Migration Guide

### Updating Existing Code

**Before:**
```go
// Old hardcoded circle rendering
planetImg := ebiten.NewImage(radius*2, radius*2)
for py := 0; py < radius*2; py++ {
    for px := 0; px < radius*2; px++ {
        // Draw pixel by pixel...
    }
}
screen.DrawImage(planetImg, opts)
```

**After:**
```go
// New sprite-based rendering with fallback
renderer.RenderPlanet(screen, x, y, radius, planet.PlanetType, planet.Color)
```

### Building Rendering

**Before:**
```go
// Old direct rendering
buildingImg := ebiten.NewImage(size*2, size*2)
buildingImg.Fill(building.Color)
screen.DrawImage(buildingImg, opts)
```

**After:**
```go
// New sprite-based rendering
buildingRenderer.RenderBuilding(screen, building, x, y)
```

## Performance Considerations

### Image Caching

All rendering systems use image caching internally:
- Static images are loaded once and cached
- Animated sprite frames are cached
- Fallback shapes (circles, rectangles, triangles) are cached by size and color

### Animation Performance

- Animation ticks are managed globally by the sprite renderer
- GIF frame delays are respected automatically
- No per-frame image creation

### Memory Management

```go
// Clear cache when switching scenes or levels
loader.ClearCache()

// Release renderer resources
renderer.Release()
```

## Adding New Assets

### Adding Planet Sprites

1. Create an animated GIF of the planet
2. Name it after the planet type (lowercase): `desert.gif`
3. Place in `assets/gifs/planets/`
4. The system will automatically load it

### Adding Building Sprites

1. Create a sprite (PNG or GIF) for the building
2. Name it after the building type (lowercase, underscores for spaces): `mining_facility.gif`
3. Place in `assets/gifs/buildings/` or `assets/images/buildings/`
4. The system will automatically try to load it, falling back to colored rectangles if not found

## Example: Complete Integration

```go
package main

import (
    "github.com/hunterjsb/xandaris/assets"
    "github.com/hunterjsb/xandaris/entities"
    "github.com/hunterjsb/xandaris/rendering"
)

func main() {
    // Preload assets at startup
    if err := assets.PreloadCommonAssets(); err != nil {
        panic(err)
    }

    // Create renderers
    spriteRenderer := rendering.NewSpriteRenderer()
    buildingRenderer := rendering.NewBuildingRenderer(spriteRenderer)

    // Create entities
    planet := entities.NewPlanet(1, "Earth", "Terrestrial", 0, 0, planetColor)
    planet.PlanetType = "fertile"
    
    resource := entities.NewResource(2, "Iron Ore", "Iron", 100, 0, resourceColor)
    mine := entities.NewBuilding(3, "Iron Mine", "Mine", 110, 0.5, buildingColor)
    
    // Attach mine to resource
    resource.AttachBuilding(mine)

    // In your render loop
    game.Update = func() error {
        spriteRenderer.Update() // Advance animation
        return nil
    }

    game.Draw = func(screen *ebiten.Image) {
        // Render planet with animated sprite
        spriteRenderer.RenderPlanet(screen, 400, 300, 50, planet.PlanetType, planet.Color)
        
        // Render resource with attached mine
        spriteRenderer.RenderResource(screen, 450, 300, 10, resource.ResourceType, resource.Color)
        buildingRenderer.RenderBuildingWithAttachments(screen, mine, 460, 310)
    }
}
```

## Future Enhancements

1. **Shader-based Effects**: Add glow, shimmer, or other effects to sprites
2. **Particle Systems**: Attach particle emitters to entities
3. **Layered Rendering**: Support multiple sprite layers (base, overlay, effects)
4. **Custom Animations**: Define animation sequences beyond GIF loops
5. **LOD System**: Render different detail levels based on zoom
6. **Sprite Sheets**: Support for sprite sheet atlases for better performance
7. **Dynamic Lighting**: Apply lighting effects to sprites

## Troubleshooting

### Sprites Not Loading

1. Check file path matches planet/building type (lowercase)
2. Ensure GIF is in `assets/gifs/` directory
3. Verify GIF is valid and not corrupted
4. Check console for error messages

### Animation Not Playing

1. Ensure `renderer.Update()` is called each frame
2. Check GIF has multiple frames and delays
3. Verify animation tick is incrementing

### Attachments Not Showing

1. Ensure `RenderBuildingWithAttachments` is used instead of `RenderBuilding`
2. Check attachment was properly added with `AttachEntity`
3. Verify attached entity has valid position data

## API Reference Summary

### Asset Loader
- `GetLoader()` - Get global loader instance
- `LoadPlanetSprite(type)` - Load planet sprite
- `LoadBuildingSprite(type)` - Load building sprite
- `LoadAnimatedSprite(path)` - Load any animated GIF
- `LoadStaticImage(path)` - Load static image
- `PreloadCommonAssets()` - Preload all common assets
- `ClearCache()` - Clear all cached assets

### Sprite Renderer
- `NewSpriteRenderer()` - Create renderer
- `Update()` - Advance animations
- `RenderPlanet(screen, x, y, radius, type, color)` - Render planet
- `RenderBuilding(screen, x, y, size, type, color)` - Render building
- `RenderResource(screen, x, y, radius, type, color)` - Render resource
- `RenderFleet(screen, x, y, size, color)` - Render fleet
- `RenderSprite(screen, opts)` - Generic sprite rendering
- `Release()` - Free resources

### Building Renderer
- `NewBuildingRenderer(spriteRenderer)` - Create renderer
- `RenderBuilding(screen, building, x, y)` - Render building
- `RenderBuildingWithAttachments(screen, building, x, y)` - Render with children

### Entity Attachment
- `AttachEntity(child)` - Attach child entity
- `DetachEntity(childID)` - Detach child entity
- `GetAttachments()` - Get all attachments
- `GetAttachmentsByType(type)` - Get typed attachments
- `HasAttachments()` - Check for attachments
- `ClearAttachments()` - Remove all attachments
- `SetAttachmentPosition(pos)` - Set relative position
- `GetAttachmentPosition()` - Get relative position