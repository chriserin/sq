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
		assert.Equal(t, "", instrument.Output, "instruments with no output field should not be tagged to any device")
	})

	t.Run("adds instruments tagged to a specific output device", func(t *testing.T) {
		ProcessConfig("./testdata/CCPerOutput.lua")
		instrument, ok := GetInstrumentForOutput("cc-test-device")
		assert.True(t, ok)
		assert.Equal(t, "Device Specific For CC Test", instrument.Name)
		assert.Equal(t, "cc-test-device", instrument.Output)
		assert.Equal(t, 2, len(instrument.CCs))

		_, ok = GetInstrumentForOutput("some-other-device")
		assert.False(t, ok, "a device with no tagged instrument should not match")
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

	t.Run("sets resets", func(t *testing.T) {
		ProcessConfig("./testdata/AddResets.lua")
		assert.Equal(t, "gategrid-test", ResetDeviceName)
		assert.Equal(t, ResetMapping{Channel: 1, Note: 60}, SongResetMapping)
		assert.Equal(t, ResetMapping{Channel: 1, Note: 61}, PartStartResetMapping)
		assert.Equal(t, ResetMapping{Channel: 1, Note: 63}, PartLoopResetMapping)
		assert.Equal(t, ResetMapping{Channel: 1, Note: 62}, GroupStartResetMapping)
		assert.Equal(t, ResetMapping{Channel: 1, Note: 64}, GroupLoopResetMapping)
	})

	t.Run("leaves loop mappings unconfigured when omitted", func(t *testing.T) {
		ProcessConfig("./testdata/AddResetsFirstEntryOnly.lua")
		assert.Equal(t, ResetMapping{Channel: 1, Note: 61}, PartStartResetMapping)
		assert.Equal(t, ResetMapping{}, PartLoopResetMapping)
		assert.Equal(t, ResetMapping{Channel: 1, Note: 62}, GroupStartResetMapping)
		assert.Equal(t, ResetMapping{}, GroupLoopResetMapping)
	})

	t.Run("detects a clock gate and reset colliding on the same device/channel/note", func(t *testing.T) {
		ProcessConfig("./testdata/ClockGateResetCollision.lua")
		err := ValidateClockGateResetOverlap()
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "partStart")
			assert.Contains(t, err.Error(), "collision-test")
			assert.Contains(t, err.Error(), "channel 1, note 50")
		}
	})

	t.Run("no error when clock gates and resets target different devices", func(t *testing.T) {
		ProcessConfig("./testdata/ClockGateResetDifferentDevices.lua")
		assert.NoError(t, ValidateClockGateResetOverlap())
	})

	t.Run("no error when channel/notes don't overlap on the same device", func(t *testing.T) {
		ProcessConfig("./testdata/ClockGateResetNoCollision.lua")
		assert.NoError(t, ValidateClockGateResetOverlap())
	})

	t.Run("sets reset pool ranges", func(t *testing.T) {
		ProcessConfig("./testdata/AddResetsRanges.lua")
		assert.Equal(t, ResetRange{Channel: 1, StartNote: 20, EndNote: 21}, StartResetRange)
		assert.Equal(t, ResetRange{Channel: 2, StartNote: 30, EndNote: 30}, LoopResetRange)
	})

	t.Run("rejects a reset range with endnote before startnote", func(t *testing.T) {
		before := StartResetRange
		ProcessConfig("./testdata/AddResetsRangeInvalidOrder.lua")
		// Same swallowed-panic convention as the clock gate subdivision
		// check above: the entry is never committed.
		assert.Equal(t, before, StartResetRange)
	})

	t.Run("allows a reset range spanning more than 8 notes", func(t *testing.T) {
		ProcessConfig("./testdata/AddResetsRangeWideSpan.lua")
		assert.Equal(t, ResetRange{Channel: 1, StartNote: 20, EndNote: 40}, StartResetRange)
	})

	t.Run("rejects a reset range note above 127", func(t *testing.T) {
		before := StartResetRange
		ProcessConfig("./testdata/AddResetsRangeNoteTooHigh.lua")
		assert.Equal(t, before, StartResetRange)
	})

	t.Run("rejects a reset range note below 0", func(t *testing.T) {
		before := StartResetRange
		ProcessConfig("./testdata/AddResetsRangeNoteNegative.lua")
		assert.Equal(t, before, StartResetRange)
	})

	t.Run("detects a clock gate and reset range colliding on the same device/channel/note", func(t *testing.T) {
		ProcessConfig("./testdata/ClockGateResetRangeCollision.lua")
		err := ValidateClockGateResetOverlap()
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), `"starts"`)
			assert.Contains(t, err.Error(), "range-collision-test")
			assert.Contains(t, err.Error(), "channel 1, note 71")
		}
	})
}

func TestFindCCForOutput(t *testing.T) {
	ProcessConfig("./testdata/CCPerOutput.lua")

	t.Run("uses the output-specific instrument's CCs when the output has one tagged", func(t *testing.T) {
		cc, ok := FindCCForOutput(10, "cc-test-device", "Fallback Instrument For CC Test")
		assert.True(t, ok)
		assert.Equal(t, "Device Pan", cc.Name)
	})

	t.Run("does not fall back to the named instrument once the output has its own CC set", func(t *testing.T) {
		// CC 1 only exists on the fallback instrument, not on the device-specific one.
		_, ok := FindCCForOutput(1, "cc-test-device", "Fallback Instrument For CC Test")
		assert.False(t, ok)
	})

	t.Run("falls back to the named instrument when the output has no tagged CC set", func(t *testing.T) {
		cc, ok := FindCCForOutput(1, "unknown-output", "Fallback Instrument For CC Test")
		assert.True(t, ok)
		assert.Equal(t, "Fallback Mod", cc.Name)
	})

	t.Run("falls back to StandardCCs when neither the output nor the named instrument match", func(t *testing.T) {
		cc, ok := FindCCForOutput(1, "unknown-output", "unknown-instrument")
		assert.True(t, ok)
		assert.Equal(t, "Modulation Wheel or Lever", cc.Name)
	})
}

func TestNearestCCForOutput(t *testing.T) {
	ProcessConfig("./testdata/CCPerOutput.lua")

	t.Run("returns the exact match when present", func(t *testing.T) {
		cc, ok := NearestCCForOutput(10, "cc-test-device", "Fallback Instrument For CC Test")
		assert.True(t, ok)
		assert.Equal(t, uint8(10), cc.Value)
	})

	t.Run("returns the closest value when the exact one is absent", func(t *testing.T) {
		// device-specific CCs are at 10 and 20.
		cc, ok := NearestCCForOutput(13, "cc-test-device", "Fallback Instrument For CC Test")
		assert.True(t, ok)
		assert.Equal(t, uint8(10), cc.Value, "13 is closer to 10 than to 20")

		cc, ok = NearestCCForOutput(17, "cc-test-device", "Fallback Instrument For CC Test")
		assert.True(t, ok)
		assert.Equal(t, uint8(20), cc.Value, "17 is closer to 20 than to 10")
	})
}
