package main

import (
	"testing"

	"github.com/chriserin/sq/internal/config"
	"github.com/chriserin/sq/internal/grid"
	"github.com/stretchr/testify/assert"
)

func TestTempoChangeNoteDoesNotDrawGateTail(t *testing.T) {
	m := createTestModel(WithGridSize(4, 1))
	// GateIndex 25 is a value that must fall within the "long gate" tail
	// range (len(ShortGates)..len(ShortGates)+len(LongGates)) so it would
	// have been misread as a gate length before this note's GateIndex was
	// excluded from gate-tail rendering.
	m.currentOverlay.SetNote(grid.GridKey{Line: 0, Beat: 0}, grid.Note{Action: grid.ActionTempoChange, GateIndex: 25})

	pattern := m.CombinedOverlayPattern(m.currentOverlay)
	output := lineView(0, m, pattern)

	// Each rune of the tail glyph is emitted into its own grid cell,
	// separated by ANSI styling codes, so the glyph never appears as one
	// contiguous substring even when the tail bug is present — check for the
	// first rune, which is what the cell immediately after the note (beat 1)
	// would render if the tail leaked through.
	gateTailFirstRune := string([]rune(config.LongGates[25-len(config.ShortGates)].Shape)[0])
	assert.NotContains(t, output, gateTailFirstRune, "a tempo-change note's GateIndex must not be drawn as a gate tail on the grid")
}
