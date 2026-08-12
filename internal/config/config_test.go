package config

import (
	"testing"

	"github.com/chriserin/sq/internal/grid"
	"github.com/stretchr/testify/assert"
)

func TestProcessConfig(t *testing.T) {
	t.Run("creates templates", func(t *testing.T) {
		ProcessConfig("./testdata/AddTemplate.lua")
		template, exists := GetTemplate("Drums")
		assert.True(t, exists)
		assert.Equal(t, "Drums", template.Name)
		assert.Equal(t, 8, len(template.Lines))
		assert.Equal(t, uint8(41), template.Lines[5].Note)
		assert.Equal(t, grid.MessageTypeNote, template.Lines[4].MsgType)
		assert.Equal(t, "plain", template.UIStyle)
		assert.Equal(t, 32, template.MaxGateLength)
	})

	t.Run("adds instruments", func(t *testing.T) {
		ProcessConfig("./testdata/AddInstrument.lua")
		instrument := GetInstrument("Prophet 10")
		assert.Equal(t, "Prophet 10", instrument.Name)
		assert.Equal(t, 59, len(instrument.CCs))
		assert.Equal(t, "GLIDE RATE", instrument.CCs[11].Name)
		assert.Equal(t, uint8(26), instrument.CCs[11].Value)
		assert.Equal(t, uint8(120), instrument.CCs[11].UpperLimit)
	})

	t.Run("sets clock gates", func(t *testing.T) {
		ProcessConfig("./testdata/AddClockGates.lua")
		assert.Equal(t, "gategrid-test", ClockGateDeviceName)
		assert.Contains(t, ClockGateMappings, ClockGateMapping{Subdivision: 1, Channel: 1, Note: 1})
		assert.Contains(t, ClockGateMappings, ClockGateMapping{Subdivision: 2, Channel: 1, Note: 2})
		assert.Contains(t, ClockGateMappings, ClockGateMapping{Subdivision: 2, Channel: 2, Note: 9})
	})

	t.Run("rejects out of range subdivision", func(t *testing.T) {
		before := len(ClockGateMappings)
		ProcessConfig("./testdata/AddClockGatesInvalidSubdivision.lua")
		// Matches the existing addinstrument/addtemplate convention (config.go:383,459):
		// a plain-string panic from a Lua-registered Go function is swallowed by
		// golua's callEx (it only converts panics implementing `error`), so
		// ProcessConfig doesn't itself panic here — the malformed entry is just
		// never appended.
		assert.Equal(t, before, len(ClockGateMappings))
	})
}
