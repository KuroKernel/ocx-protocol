# OCX Protocol Logo Usage Guide

## Quick Reference

### Logo Files
- **Primary Logo**: `/assets/logos/ocx-logo-primary.svg` - Full horizontal lockup
- **Symbol Only**: `/assets/logos/ocx-symbol-only.svg` - For headers, favicons, small spaces
- **Stacked Logo**: `/assets/logos/ocx-logo-stacked.svg` - Vertical layout
- **Aperture Variant**: `/assets/logos/ocx-aperture-variant.svg` - Technical contexts

### Implementation Examples

#### React Component
```jsx
// Header logo
<img 
  src="/assets/logos/ocx-symbol-only.svg" 
  alt="OCX Protocol" 
  className="w-8 h-8"
/>

// Hero section logo
<img 
  src="/assets/logos/ocx-logo-primary.svg" 
  alt="OCX Protocol - Mathematical proof for computational integrity" 
  className="h-16 w-auto"
/>
```

#### HTML
```html
<!-- Favicon -->
<link rel="icon" type="image/svg+xml" href="/assets/logos/ocx-symbol-only.svg" />

<!-- Apple Touch Icon -->
<link rel="apple-touch-icon" href="/assets/logos/ocx-symbol-only.svg" />
```

### Size Guidelines

| Context | Size | File |
|---------|------|------|
| Favicon | 16x16px | `ocx-symbol-only.svg` |
| Header | 32x32px | `ocx-symbol-only.svg` |
| Hero | 64px height | `ocx-logo-primary.svg` |
| Footer | 24x24px | `ocx-symbol-only.svg` |
| Business Cards | 1.5" width | `ocx-logo-primary.svg` |
| Presentations | 2" width | `ocx-logo-primary.svg` |

### Responsive Behavior

#### Mobile (< 768px)
- Use symbol-only in header
- Reduce hero logo to 48px height
- Maintain clearspace requirements

#### Tablet (768px - 1024px)
- Use symbol-only in header
- Hero logo at 56px height
- Full logo in footer

#### Desktop (> 1024px)
- Symbol-only in header
- Full primary logo in hero
- Symbol-only in footer

### Accessibility

#### Alt Text Guidelines
- **Primary Logo**: "OCX Protocol - Mathematical proof for computational integrity"
- **Symbol Only**: "OCX Protocol"
- **Stacked Logo**: "OCX Protocol"
- **Aperture Variant**: "OCX Protocol - Technical variant"

#### Color Contrast
- Logos are black on white backgrounds
- Ensure sufficient contrast (4.5:1 minimum)
- Test with color blindness simulators

### Technical Specifications

#### SVG Optimization
- All logos are SVG for scalability
- Optimized for web performance
- Cross-browser compatible
- No external dependencies

#### File Sizes
- `ocx-symbol-only.svg`: ~1.2KB
- `ocx-logo-primary.svg`: ~2.8KB
- `ocx-logo-stacked.svg`: ~1.5KB
- `ocx-aperture-variant.svg`: ~1.1KB

### Common Mistakes to Avoid

❌ **Don't:**
- Stretch or distort logos
- Add colors or effects
- Use on busy backgrounds
- Modify proportions
- Use below 16px size

✅ **Do:**
- Maintain aspect ratios
- Use on clean backgrounds
- Preserve clearspace
- Test at all sizes
- Follow brand guidelines

### Integration Checklist

- [ ] Logo files placed in `/public/assets/logos/`
- [ ] Favicon updated in `index.html`
- [ ] Header logo implemented
- [ ] Hero section logo added
- [ ] Footer logo updated
- [ ] Alt text provided
- [ ] Responsive behavior tested
- [ ] Accessibility verified
- [ ] Cross-browser compatibility checked
