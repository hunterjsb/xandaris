# Planet Animation GIFs

This directory contains animated GIF files for different planet types in XANDARIS.

## File Structure

Place your planet GIF files here with the following naming convention:

```
/planets/
├── abundant.gif     - Lush, resource-rich planets
├── fertile.gif      - Agricultural/biological planets
├── mountain.gif     - Rocky, mineral-rich planets
├── desert.gif       - Arid, sandy planets
├── volcanic.gif     - Volcanic/molten planets
├── highlands.gif    - Elevated terrain planets
├── swamp.gif        - Wetland/marsh planets
├── barren.gif       - Desolate, empty planets
├── radiant.gif      - Energy-rich, glowing planets
├── barred.gif       - Special planets with unique resources
└── null.gif         - Hidden/empty planets (optional)
```

## Usage

The system will automatically load these GIFs when displaying planets in:
- Planet lists in system view
- Planet detail modals
- Any other planet displays

## Requirements

- **Format**: GIF (animated)
- **Size**: Recommended 64x64 to 128x128 pixels
- **Quality**: Optimized for web (keep file size reasonable)
- **Style**: Should match the space theme of the game

## Fallback

If a GIF file is missing, the system will fall back to the static planet type icon.

## Adding New Planet Types

When adding new planet types:
1. Add the GIF file with the planet type name (lowercase)
2. Update the `gifMap` in `uiController.js` if needed
3. The system will automatically pick up the new GIF