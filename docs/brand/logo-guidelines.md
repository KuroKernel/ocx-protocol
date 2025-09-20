# OCX Protocol Logo Guidelines

## Logo System Overview

The OCX Protocol logo system consists of three primary variants designed for different use cases and contexts.

### Primary Lockup (A)
- **Use**: Main brand identity, headers, business cards
- **File**: `ocx-logo-primary.svg`
- **Components**: Hexagon mark + "OCX" + "PROTOCOL" + tagline
- **Clearspace**: Minimum 1× hex diameter on all sides

### Aperture Variant (B)
- **Use**: Technical documentation, computational contexts
- **File**: `ocx-aperture-variant.svg`
- **Components**: Concentric hexagons with core dot
- **Aesthetic**: Camera/aperture for computational focus

### Symbol-Only (C)
- **Use**: Favicons, app icons, social media, watermarks
- **File**: `ocx-symbol-only.svg`
- **Components**: Hexagon mark with crosshair core
- **Scaling**: Optimized for 16-32px sizes

## Technical Specifications

### Proportions
- **Outer hex diameter**: 0.90 × "OCX" cap height
- **Inner ring**: 0.62 × outer radius
- **Core circle**: 0.18 × outer radius
- **Stroke weight**: Outer hex diameter ÷ 24

### Colors
- **Primary**: Pure black (#000000)
- **Secondary**: Pure white (#FFFFFF)
- **No color variations**: Maintains mathematical precision

### Typography
- **Primary text**: Times New Roman, serif, 300 weight
- **Secondary text**: Helvetica Neue, sans-serif, 400 weight
- **Tagline**: Times New Roman, serif, 300 weight, italic
- **Letter spacing**: Calculated for optimal readability

## Usage Guidelines

### Do's
- ✅ Use primary lockup for main brand applications
- ✅ Maintain clearspace requirements
- ✅ Use symbol-only for small applications
- ✅ Preserve proportional relationships
- ✅ Use aperture variant for technical contexts

### Don'ts
- ❌ Don't modify proportions or spacing
- ❌ Don't add colors or effects
- ❌ Don't use on busy backgrounds
- ❌ Don't distort or stretch
- ❌ Don't use below minimum size (16px)

## File Organization

```
public/assets/logos/
├── ocx-logo-primary.svg      # Main horizontal lockup
├── ocx-logo-stacked.svg      # Vertical stacked version
├── ocx-symbol-only.svg       # Symbol for small uses
└── ocx-aperture-variant.svg  # Technical variant

public/assets/icons/
├── favicon.ico               # Browser favicon
└── apple-touch-icon.png      # iOS home screen icon
```

## Implementation Notes

- All logos are SVG for scalability
- Optimized for web performance
- Cross-browser compatible
- Accessible with proper alt text
- Responsive design ready
