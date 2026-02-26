package themes

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
)

func Darken(lgColor color.Color, percent float64) color.Color {
	return Lighten(lgColor, -1.0*percent)
}
func Lighten(lgColor color.Color, percent float64) color.Color {
	// Extract RGB components using color.Color interface
	rr, gg, bb, _ := lgColor.RGBA()
	r := int(rr >> 8)
	g := int(gg >> 8)
	b := int(bb >> 8)

	// Calculate the factor to increase the brightness
	factor := 1 + percent/100.0

	// Increase each component by the factor and ensure it does not exceed 255
	r = int(float64(r) * factor)
	g = int(float64(g) * factor)
	b = int(float64(b) * factor)

	if r > 255 {
		r = 255
	}
	if r < 0 {
		r = 0
	}
	if g > 255 {
		g = 255
	}
	if g < 0 {
		g = 0
	}
	if b > 255 {
		b = 255
	}
	if b < 0 {
		b = 0
	}

	// Convert the adjusted RGB values back to a hex string
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, b))
}
