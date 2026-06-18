package badge

import (
	"fmt"
	"strings"
)

type Style string

const (
	StyleFlat       Style = "flat"
	StyleFlatSquare Style = "flat-square"
	StylePlastic    Style = "plastic"
)

type Options struct {
	Label      string
	Message    string
	Color      string // message background
	LabelColor string // label background
	Style      Style
	Height     int
	Radius     int // corner radius (-1 means use style default)
}

func DefaultOptions() Options {
	return Options{
		Color:      "#007ec6",
		LabelColor: "#555",
		Style:      StyleFlat,
		Height:     0,  // 0 means use style default
		Radius:     -1, // -1 means use style default
	}
}

func Render(opts Options) string {
	if opts.Style == "" {
		opts.Style = StyleFlat
	}
	if opts.Color == "" {
		opts.Color = "#007ec6"
	}
	if opts.LabelColor == "" {
		opts.LabelColor = "#555"
	}

	height := opts.Height
	radius := opts.Radius

	switch opts.Style {
	case StylePlastic:
		if height == 0 {
			height = 18
		}
		if radius < 0 {
			radius = 4
		}
	case StyleFlatSquare:
		if height == 0 {
			height = 20
		}
		if radius < 0 {
			radius = 0
		}
	default: // flat
		if height == 0 {
			height = 20
		}
		if radius < 0 {
			radius = 3
		}
	}

	horizPadding := 5.0
	labelWidth := float64(roundUpToOdd(int(measureText(opts.Label)))) + 2*horizPadding
	msgWidth := float64(roundUpToOdd(int(measureText(opts.Message)))) + 2*horizPadding

	if opts.Label == "" {
		labelWidth = 0
	}

	totalWidth := labelWidth + msgWidth

	labelTextX := labelWidth / 2.0
	msgTextX := labelWidth + msgWidth/2.0

	fontSize := 110 // 11px * 10 (rendered at scale 0.1 for sub-pixel precision)
	textY := int(float64(height)*10*0.7) + 5
	shadowY := textY + 10

	labelTextW := int(measureText(opts.Label) * 10)
	msgTextW := int(measureText(opts.Message) * 10)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%d" role="img" aria-label="%s: %s">`,
		totalWidth, height, escXML(opts.Label), escXML(opts.Message)))
	sb.WriteString(fmt.Sprintf(`<title>%s: %s</title>`, escXML(opts.Label), escXML(opts.Message)))

	// Gradient definitions
	switch opts.Style {
	case StyleFlat:
		sb.WriteString(`<linearGradient id="s" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient>`)
	case StylePlastic:
		sb.WriteString(`<linearGradient id="s" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient>`)
	}

	// Clip path for rounded corners
	if radius > 0 {
		sb.WriteString(fmt.Sprintf(`<clipPath id="r"><rect width="%.0f" height="%d" rx="%d" fill="#fff"/></clipPath>`, totalWidth, height, radius))
		sb.WriteString(`<g clip-path="url(#r)">`)
	} else {
		sb.WriteString(`<g shape-rendering="crispEdges">`)
	}

	// Background rects
	if labelWidth > 0 {
		sb.WriteString(fmt.Sprintf(`<rect width="%.0f" height="%d" fill="%s"/>`, labelWidth, height, opts.LabelColor))
	}
	sb.WriteString(fmt.Sprintf(`<rect x="%.0f" width="%.0f" height="%d" fill="%s"/>`, labelWidth, msgWidth, height, opts.Color))

	// Gradient overlay
	if opts.Style == StyleFlat || opts.Style == StylePlastic {
		sb.WriteString(fmt.Sprintf(`<rect width="%.0f" height="%d" fill="url(#s)"/>`, totalWidth, height))
	}
	sb.WriteString(`</g>`)

	// Text
	sb.WriteString(fmt.Sprintf(`<g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="%d">`, fontSize))

	hasShadow := opts.Style != StyleFlatSquare

	if labelWidth > 0 {
		if hasShadow {
			sb.WriteString(fmt.Sprintf(`<text aria-hidden="true" x="%.0f" y="%d" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="%d">%s</text>`,
				labelTextX*10, shadowY, labelTextW, escXML(opts.Label)))
		}
		sb.WriteString(fmt.Sprintf(`<text x="%.0f" y="%d" transform="scale(.1)" textLength="%d">%s</text>`,
			labelTextX*10, textY, labelTextW, escXML(opts.Label)))
	}

	if hasShadow {
		sb.WriteString(fmt.Sprintf(`<text aria-hidden="true" x="%.0f" y="%d" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="%d">%s</text>`,
			msgTextX*10, shadowY, msgTextW, escXML(opts.Message)))
	}
	sb.WriteString(fmt.Sprintf(`<text x="%.0f" y="%d" transform="scale(.1)" textLength="%d">%s</text>`,
		msgTextX*10, textY, msgTextW, escXML(opts.Message)))

	sb.WriteString(`</g></svg>`)
	return sb.String()
}

func FormatNumber(n int64) string {
	switch {
	case n >= 1_000_000:
		v := float64(n) / 1_000_000
		if v >= 10 {
			return fmt.Sprintf("%.0fM", v)
		}
		return fmt.Sprintf("%.1fM", v)
	case n >= 1_000:
		v := float64(n) / 1_000
		if v >= 10 {
			return fmt.Sprintf("%.0fk", v)
		}
		return fmt.Sprintf("%.1fk", v)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func escXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
