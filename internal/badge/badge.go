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
	opts = normalizeOptions(opts)
	style := resolveStyle(opts)
	layout := measureLayout(opts, style.height)

	var sb strings.Builder
	renderSVGHeader(&sb, opts, style, layout)
	renderGradients(&sb, opts.Style)
	renderBackground(&sb, opts, style, layout)
	renderText(&sb, opts, style, layout)
	sb.WriteString(`</svg>`)
	return sb.String()
}

type styleConfig struct {
	height    int
	radius    int
	hasShadow bool
}

type badgeLayout struct {
	labelWidth float64
	msgWidth   float64
	totalWidth float64
	labelTextX float64
	msgTextX   float64
	fontSize   int
	textY      int
	shadowY    int
	labelTextW int
	msgTextW   int
}

func normalizeOptions(opts Options) Options {
	if opts.Style == "" {
		opts.Style = StyleFlat
	}
	if opts.Color == "" {
		opts.Color = "#007ec6"
	}
	if opts.LabelColor == "" {
		opts.LabelColor = "#555"
	}
	return opts
}

func resolveStyle(opts Options) styleConfig {
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
	return styleConfig{
		height:    height,
		radius:    radius,
		hasShadow: opts.Style != StyleFlatSquare,
	}
}

func measureLayout(opts Options, height int) badgeLayout {
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

	return badgeLayout{
		labelWidth: labelWidth,
		msgWidth:   msgWidth,
		totalWidth: totalWidth,
		labelTextX: labelTextX,
		msgTextX:   msgTextX,
		fontSize:   fontSize,
		textY:      textY,
		shadowY:    shadowY,
		labelTextW: labelTextW,
		msgTextW:   msgTextW,
	}
}

func renderSVGHeader(sb *strings.Builder, opts Options, style styleConfig, layout badgeLayout) {
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%d" role="img" aria-label="%s: %s">`,
		layout.totalWidth, style.height, escXML(opts.Label), escXML(opts.Message)))
	sb.WriteString(fmt.Sprintf(`<title>%s: %s</title>`, escXML(opts.Label), escXML(opts.Message)))
}

func renderGradients(sb *strings.Builder, style Style) {
	switch style {
	case StyleFlat:
		sb.WriteString(`<linearGradient id="s" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient>`)
	case StylePlastic:
		sb.WriteString(`<linearGradient id="s" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient>`)
	}
}

func renderBackground(sb *strings.Builder, opts Options, style styleConfig, layout badgeLayout) {
	if style.radius > 0 {
		sb.WriteString(fmt.Sprintf(`<clipPath id="r"><rect width="%.0f" height="%d" rx="%d" fill="#fff"/></clipPath>`, layout.totalWidth, style.height, style.radius))
		sb.WriteString(`<g clip-path="url(#r)">`)
	} else {
		sb.WriteString(`<g shape-rendering="crispEdges">`)
	}

	if layout.labelWidth > 0 {
		sb.WriteString(fmt.Sprintf(`<rect width="%.0f" height="%d" fill="%s"/>`, layout.labelWidth, style.height, opts.LabelColor))
	}
	sb.WriteString(fmt.Sprintf(`<rect x="%.0f" width="%.0f" height="%d" fill="%s"/>`, layout.labelWidth, layout.msgWidth, style.height, opts.Color))

	if opts.Style == StyleFlat || opts.Style == StylePlastic {
		sb.WriteString(fmt.Sprintf(`<rect width="%.0f" height="%d" fill="url(#s)"/>`, layout.totalWidth, style.height))
	}
	sb.WriteString(`</g>`)
}

func renderText(sb *strings.Builder, opts Options, style styleConfig, layout badgeLayout) {
	sb.WriteString(fmt.Sprintf(`<g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="%d">`, layout.fontSize))

	if layout.labelWidth > 0 {
		if style.hasShadow {
			sb.WriteString(fmt.Sprintf(`<text aria-hidden="true" x="%.0f" y="%d" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="%d">%s</text>`,
				layout.labelTextX*10, layout.shadowY, layout.labelTextW, escXML(opts.Label)))
		}
		sb.WriteString(fmt.Sprintf(`<text x="%.0f" y="%d" transform="scale(.1)" textLength="%d">%s</text>`,
			layout.labelTextX*10, layout.textY, layout.labelTextW, escXML(opts.Label)))
	}

	if style.hasShadow {
		sb.WriteString(fmt.Sprintf(`<text aria-hidden="true" x="%.0f" y="%d" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="%d">%s</text>`,
			layout.msgTextX*10, layout.shadowY, layout.msgTextW, escXML(opts.Message)))
	}
	sb.WriteString(fmt.Sprintf(`<text x="%.0f" y="%d" transform="scale(.1)" textLength="%d">%s</text>`,
		layout.msgTextX*10, layout.textY, layout.msgTextW, escXML(opts.Message)))

	sb.WriteString(`</g>`)
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
