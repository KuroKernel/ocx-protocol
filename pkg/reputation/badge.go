package reputation

import (
	"fmt"
	"strings"
)

// BadgeStyle represents different badge visual styles
type BadgeStyle string

const (
	BadgeStyleFlat        BadgeStyle = "flat"
	BadgeStyleFlatSquare  BadgeStyle = "flat-square"
	BadgeStyleForTheBadge BadgeStyle = "for-the-badge"
)

// BadgeColor represents badge color based on trust score
type BadgeColor string

const (
	BadgeColorBrightGreen BadgeColor = "#4c1"
	BadgeColorGreen       BadgeColor = "#97ca00"
	BadgeColorYellow      BadgeColor = "#dfb317"
	BadgeColorOrange      BadgeColor = "#fe7d37"
	BadgeColorRed         BadgeColor = "#e05d44"
	BadgeColorGray        BadgeColor = "#9f9f9f"
)

// GetBadgeColor returns the appropriate color for a trust score
func GetBadgeColor(score float64) BadgeColor {
	switch {
	case score >= 90:
		return BadgeColorBrightGreen
	case score >= 75:
		return BadgeColorGreen
	case score >= 60:
		return BadgeColorYellow
	case score >= 40:
		return BadgeColorOrange
	case score > 0:
		return BadgeColorRed
	default:
		return BadgeColorGray
	}
}

// GenerateBadgeSVG creates an SVG badge for a trust score
func GenerateBadgeSVG(label string, score float64, style BadgeStyle) string {
	color := GetBadgeColor(score)
	scoreText := fmt.Sprintf("%.0f/100", score)

	switch style {
	case BadgeStyleFlatSquare:
		return generateFlatSquareBadge(label, scoreText, color)
	case BadgeStyleForTheBadge:
		return generateForTheBadgeBadge(label, scoreText, color)
	default:
		return generateFlatBadge(label, scoreText, color)
	}
}

// generateFlatBadge creates a flat-style badge (default)
func generateFlatBadge(label, value string, color BadgeColor) string {
	labelWidth := len(label)*7 + 20
	valueWidth := len(value)*7 + 20
	totalWidth := labelWidth + valueWidth

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s">
    <title>%s: %s</title>
    <linearGradient id="s" x2="0" y2="100%%">
        <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
        <stop offset="1" stop-opacity=".1"/>
    </linearGradient>
    <clipPath id="r">
        <rect width="%d" height="20" rx="3" fill="#fff"/>
    </clipPath>
    <g clip-path="url(#r)">
        <rect width="%d" height="20" fill="#555"/>
        <rect x="%d" width="%d" height="20" fill="%s"/>
        <rect width="%d" height="20" fill="url(#s)"/>
    </g>
    <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="110">
        <text aria-hidden="true" x="%d" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="%d">%s</text>
        <text x="%d" y="140" transform="scale(.1)" fill="#fff" textLength="%d">%s</text>
        <text aria-hidden="true" x="%d" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="%d">%s</text>
        <text x="%d" y="140" transform="scale(.1)" fill="#fff" textLength="%d">%s</text>
    </g>
</svg>`,
		totalWidth, label, value, label, value,
		totalWidth,
		labelWidth,
		labelWidth, valueWidth, color,
		totalWidth,
		labelWidth/2*10, (labelWidth-20)*10, escapeXML(label),
		labelWidth/2*10, (labelWidth-20)*10, escapeXML(label),
		(labelWidth+valueWidth/2)*10, (valueWidth-20)*10, escapeXML(value),
		(labelWidth+valueWidth/2)*10, (valueWidth-20)*10, escapeXML(value),
	)
}

// generateFlatSquareBadge creates a flat-square style badge
func generateFlatSquareBadge(label, value string, color BadgeColor) string {
	labelWidth := len(label)*7 + 20
	valueWidth := len(value)*7 + 20
	totalWidth := labelWidth + valueWidth

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s">
    <title>%s: %s</title>
    <g shape-rendering="crispEdges">
        <rect width="%d" height="20" fill="#555"/>
        <rect x="%d" width="%d" height="20" fill="%s"/>
    </g>
    <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="110">
        <text x="%d" y="140" transform="scale(.1)" fill="#fff" textLength="%d">%s</text>
        <text x="%d" y="140" transform="scale(.1)" fill="#fff" textLength="%d">%s</text>
    </g>
</svg>`,
		totalWidth, label, value, label, value,
		labelWidth,
		labelWidth, valueWidth, color,
		labelWidth/2*10, (labelWidth-20)*10, escapeXML(label),
		(labelWidth+valueWidth/2)*10, (valueWidth-20)*10, escapeXML(value),
	)
}

// generateForTheBadgeBadge creates a "for-the-badge" style badge
func generateForTheBadgeBadge(label, value string, color BadgeColor) string {
	labelWidth := len(label)*11 + 24
	valueWidth := len(value)*11 + 24
	totalWidth := labelWidth + valueWidth

	labelUpper := strings.ToUpper(label)
	valueUpper := strings.ToUpper(value)

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="28" role="img" aria-label="%s: %s">
    <title>%s: %s</title>
    <g shape-rendering="crispEdges">
        <rect width="%d" height="28" fill="#555"/>
        <rect x="%d" width="%d" height="28" fill="%s"/>
    </g>
    <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="100">
        <text x="%d" y="175" transform="scale(.1)" fill="#fff" textLength="%d">%s</text>
        <text x="%d" y="175" transform="scale(.1)" fill="#fff" textLength="%d">%s</text>
    </g>
</svg>`,
		totalWidth, label, value, label, value,
		labelWidth,
		labelWidth, valueWidth, color,
		labelWidth/2*10, (labelWidth-24)*10, escapeXML(labelUpper),
		(labelWidth+valueWidth/2)*10, (valueWidth-24)*10, escapeXML(valueUpper),
	)
}

// GenerateUnverifiedBadge creates a badge for unverified users
func GenerateUnverifiedBadge(style BadgeStyle) string {
	return GenerateBadgeSVG("TrustScore", 0, style)
}

// GenerateExpiredBadge creates a badge for expired verifications
func GenerateExpiredBadge(style BadgeStyle) string {
	label := "TrustScore"
	value := "Expired"
	color := BadgeColorRed

	switch style {
	case BadgeStyleFlatSquare:
		return generateFlatSquareBadge(label, value, color)
	case BadgeStyleForTheBadge:
		return generateForTheBadgeBadge(label, value, color)
	default:
		return generateFlatBadge(label, value, color)
	}
}

// escapeXML escapes special XML characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// BadgeMarkdown generates markdown for embedding a badge
func BadgeMarkdown(userID, verifyURL, badgeURL string) string {
	return fmt.Sprintf("[![TrustScore](%s)](%s)", badgeURL, verifyURL)
}

// BadgeHTML generates HTML for embedding a badge
func BadgeHTML(userID, verifyURL, badgeURL string) string {
	return fmt.Sprintf(`<a href="%s"><img src="%s" alt="TrustScore"></a>`, verifyURL, badgeURL)
}
