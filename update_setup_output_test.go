package main

import (
	"testing"

	"github.com/chriserin/sq/internal/grid"
	"github.com/chriserin/sq/internal/mappings"
	"github.com/chriserin/sq/internal/seqmidi"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/drivers/testdrv"
)

type fakeOutputsDriver struct {
	outs []drivers.Out
}

func (d *fakeOutputsDriver) Ins() ([]drivers.In, error)   { return nil, nil }
func (d *fakeOutputsDriver) Outs() ([]drivers.Out, error) { return d.outs, nil }
func (d *fakeOutputsDriver) String() string               { return "fake" }
func (d *fakeOutputsDriver) Close() error                 { return nil }

// WithMidiOutputs replaces the model's MidiConnection with one that has the
// given named devices registered (each shows up as "<name>-out"), so tests
// can exercise per-line Output cycling without a real MIDI backend.
func WithMidiOutputs(names ...string) modelFunc {
	return func(m *model) model {
		mc := &seqmidi.MidiConnection{}
		var outs []drivers.Out
		for _, name := range names {
			o, _ := testdrv.New(name).Outs()
			outs = append(outs, o[0])
		}
		_ = mc.UpdateOutDeviceList(&fakeOutputsDriver{outs: outs})
		m.midiConnection = mc
		return *m
	}
}

// setupInputSwitchesTo returns the number of SetupInputSwitch presses needed
// to move the cursor from Grid to Output for the given message type.
func setupInputSwitchesToOutput(msgType grid.MessageType) []any {
	count := 4
	if msgType == grid.MessageTypeProgramChange {
		count = 3
	}
	commands := make([]any, count)
	for i := range commands {
		commands[i] = mappings.SetupInputSwitch
	}
	return commands
}

func TestIncrementDecrementLineOutput(t *testing.T) {
	tests := []struct {
		name           string
		outputs        []string
		initialOutput  string
		extraCommand   any
		expectedOutput string
		description    string
	}{
		{
			name:           "Increase cycles to the next device alphabetically",
			outputs:        []string{"Alpha", "Zeta"},
			initialOutput:  "Alpha-out",
			extraCommand:   mappings.Increase,
			expectedOutput: "Zeta-out",
			description:    "should move from Alpha-out to Zeta-out",
		},
		{
			name:           "Increase wraps from the last device back to the first",
			outputs:        []string{"Alpha", "Zeta"},
			initialOutput:  "Zeta-out",
			extraCommand:   mappings.Increase,
			expectedOutput: "Alpha-out",
			description:    "should wrap from Zeta-out back to Alpha-out",
		},
		{
			name:           "Decrease wraps from the first device to the last",
			outputs:        []string{"Alpha", "Zeta"},
			initialOutput:  "Alpha-out",
			extraCommand:   mappings.Decrease,
			expectedOutput: "Zeta-out",
			description:    "should wrap from Alpha-out back to Zeta-out",
		},
		{
			name:           "Decrease from the middle moves backward",
			outputs:        []string{"Alpha", "Middle", "Zeta"},
			initialOutput:  "Middle-out",
			extraCommand:   mappings.Decrease,
			expectedOutput: "Alpha-out",
			description:    "should move from Middle-out back to Alpha-out",
		},
		{
			name:           "Increase from a disconnected device lands on the first available device",
			outputs:        []string{"Alpha", "Zeta"},
			initialOutput:  "Missing-out",
			extraCommand:   mappings.Increase,
			expectedOutput: "Alpha-out",
			description:    "a stale/missing selection should land on the alphabetically-first device rather than an arbitrary offset",
		},
		{
			name:           "Decrease from a disconnected device also lands on the first available device",
			outputs:        []string{"Alpha", "Zeta"},
			initialOutput:  "Missing-out",
			extraCommand:   mappings.Decrease,
			expectedOutput: "Alpha-out",
			description:    "both directions treat a stale selection the same way",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createTestModel(WithMidiOutputs(tt.outputs...), func(m *model) model {
				m.definition.Lines[m.gridCursor.Line].MidiOutput = tt.initialOutput
				return *m
			})

			commands := append(setupInputSwitchesToOutput(grid.MessageTypeNote), tt.extraCommand)
			m, _ = processCommands(commands, m)

			assert.Equal(t, tt.expectedOutput, m.definition.Lines[m.gridCursor.Line].MidiOutput, tt.description)
		})
	}
}

func TestIncrementLineOutput_NoDevicesDoesNotPanic(t *testing.T) {
	m := createTestModel(WithMidiOutputs(), func(m *model) model {
		m.definition.Lines[m.gridCursor.Line].MidiOutput = ""
		return *m
	})

	assert.NotPanics(t, func() {
		m.IncrementLineOutput()
		m.DecrementLineOutput()
	}, "cycling with zero registered devices should no-op, not panic")
}

func TestSnapLineCCToNearest(t *testing.T) {
	t.Run("snaps to the nearest CC value when the current one doesn't exist in the effective set", func(t *testing.T) {
		m := createTestModel(WithMidiOutputs("Alpha"))
		line := &m.definition.Lines[m.gridCursor.Line]
		line.MsgType = grid.MessageTypeCc
		line.Note = 3 // absent from StandardCCs; nearest present value is 2 (a tie with 4 broken toward the earlier one)
		line.MidiOutput = "Alpha-out"

		m.snapLineCCToNearest(line)

		assert.Equal(t, uint8(2), line.Note)
	})

	t.Run("leaves an already-valid CC value untouched", func(t *testing.T) {
		m := createTestModel(WithMidiOutputs("Alpha"))
		line := &m.definition.Lines[m.gridCursor.Line]
		line.MsgType = grid.MessageTypeCc
		line.Note = 7 // present in StandardCCs
		line.MidiOutput = "Alpha-out"

		m.snapLineCCToNearest(line)

		assert.Equal(t, uint8(7), line.Note)
	})

	t.Run("never touches Note-type lines", func(t *testing.T) {
		m := createTestModel(WithMidiOutputs("Alpha"))
		line := &m.definition.Lines[m.gridCursor.Line]
		line.MsgType = grid.MessageTypeNote
		line.Note = 3
		line.MidiOutput = "Alpha-out"

		m.snapLineCCToNearest(line)

		assert.Equal(t, uint8(3), line.Note)
	})
}

func TestOutputCycling_SnapsCCValueOnRealUse(t *testing.T) {
	m := createTestModel(WithMidiOutputs("Alpha", "Zeta"), func(m *model) model {
		line := &m.definition.Lines[m.gridCursor.Line]
		line.MsgType = grid.MessageTypeCc
		line.Note = 3
		line.MidiOutput = "Alpha-out"
		return *m
	})

	commands := append(setupInputSwitchesToOutput(grid.MessageTypeCc), mappings.Increase)
	m, _ = processCommands(commands, m)

	line := m.definition.Lines[m.gridCursor.Line]
	assert.Equal(t, "Zeta-out", line.MidiOutput)
	assert.Equal(t, uint8(2), line.Note, "changing Output through the real ctrl+d/+ flow should trigger the same CC snapping as calling snapLineCCToNearest directly")
}

func TestUndoPreservesMidiOutputOnUnrelatedEdit(t *testing.T) {
	m := createTestModel(WithMidiOutputs("Alpha", "Zeta"), func(m *model) model {
		m.definition.Lines[m.gridCursor.Line].MidiOutput = "Zeta-out"
		return *m
	})

	assert.Equal(t, "Zeta-out", m.definition.Lines[m.gridCursor.Line].MidiOutput, "sanity check on initial state")

	// Change an unrelated field (Channel), then bail out via Escape+Undo.
	// Regression guard for CaptureTemporaryState: it must copy MidiOutput
	// into its snapshot, or this undo would blank MidiOutput on every line.
	commands := []any{
		mappings.SetupInputSwitch,
		mappings.Increase,
		mappings.Escape,
		mappings.Undo,
	}
	m, _ = processCommands(commands, m)

	assert.Equal(t, "Zeta-out", m.definition.Lines[m.gridCursor.Line].MidiOutput, "undoing an unrelated Setup edit should not wipe out MidiOutput")
}
