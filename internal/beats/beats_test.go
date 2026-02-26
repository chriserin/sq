package beats

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/chriserin/sq/internal/arrangement"
	"github.com/chriserin/sq/internal/config"
	"github.com/chriserin/sq/internal/grid"
	"github.com/chriserin/sq/internal/playstate"
	"github.com/chriserin/sq/internal/seqmidi"
	"github.com/chriserin/sq/internal/sequence"
	"github.com/stretchr/testify/assert"
	midi "gitlab.com/gomidi/midi/v2"
)

func TestSimpleSequenceBeats(t *testing.T) {
	tests := []struct {
		name                string
		partBeats           uint8
		expectedBeatsPlayed int
	}{
		{"Part with 1 beat", 1, 1},
		{"Part with 3 beats", 3, 3},
		{"Part with 7 beats", 7, 7},
		{"Part with 13 beats", 13, 13},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sequence, cursor := SimpleSequence()

			(*sequence.Parts)[0].Beats = tt.partBeats

			beatsPlayed, _ := PlayTestLoop(sequence, cursor, int(tt.partBeats)+3, playstate.PlayState{Playing: true}, t.Context())
			assert.Equal(t, tt.expectedBeatsPlayed, beatsPlayed)
		})
	}
}

func TestOneNote(t *testing.T) {
	tests := []struct {
		name                 string
		partBeats            uint8
		expectedBeatsPlayed  int
		expectedMidiMessages []seqmidi.Message
	}{
		{
			"Part with 1 note",
			1,
			1,
			[]seqmidi.Message{{Msg: midi.NoteOn(4, 5, 5), Delay: 0}, {Msg: midi.NoteOff(4, 5), Delay: 20 * time.Millisecond}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sequence, cursor := SimpleSequence()

			(*sequence.Parts)[0].Beats = tt.partBeats
			(*sequence.Parts)[0].Overlays.AddNote(grid.GridKey{Line: 0, Beat: 0}, grid.Note{AccentIndex: 5})

			beatsPlayed, testMessages := PlayTestLoop(sequence, cursor, int(tt.partBeats)+3, playstate.PlayState{Playing: true}, t.Context())
			assert.Equal(t, tt.expectedBeatsPlayed, beatsPlayed)

			if assert.Len(t, testMessages, len(tt.expectedMidiMessages), "Number of MIDI messages") {
				for i, msg := range tt.expectedMidiMessages {
					assert.Equal(t, msg.Delay, testMessages[i].Delay, "Delay")

					switch msg.Msg.Type() {
					case midi.NoteOnMsg:
						var expectedChannel, expectedNote, expectedVelocity uint8 = 0, 0, 0
						var testChannel, testNote, testVelocity uint8 = 0, 0, 0
						assert.True(t, msg.Msg.GetNoteOn(&expectedChannel, &expectedNote, &expectedVelocity), "Note On Parsing expected message")
						assert.True(t, testMessages[i].Msg.GetNoteOn(&testChannel, &testNote, &testVelocity), "Note On Parsing test message")
						assert.Equal(t, expectedChannel, testChannel, "Note On Channel")
						assert.Equal(t, expectedNote, testNote, "Note On Note")
						assert.Equal(t, expectedVelocity, testVelocity, "Note On Velocity")
					case midi.NoteOffMsg:
						var expectedChannel, expectedNote, expectedVelocity uint8 = 0, 0, 0
						var testChannel, testNote, testVelocity uint8 = 0, 0, 0
						assert.True(t, msg.Msg.GetNoteOff(&expectedChannel, &expectedNote, &expectedVelocity), "Note On Parsing expected message")
						assert.True(t, testMessages[i].Msg.GetNoteOff(&testChannel, &testNote, &testVelocity), "Note On Parsing test message")
						assert.Equal(t, expectedChannel, testChannel, "Note Off Channel")
						assert.Equal(t, expectedNote, testNote, "Note Off Note")
						assert.Equal(t, expectedVelocity, testVelocity, "Note Off Velocity")
					}
				}
			}
		})
	}
}

func TestRatchet(t *testing.T) {
	tests := []struct {
		name                 string
		partBeats            uint8
		expectedBeatsPlayed  int
		expectedMidiMessages []seqmidi.Message
	}{
		{
			"Part with 1 note ratcheted twice",
			1,
			1,
			[]seqmidi.Message{
				{Msg: midi.NoteOn(4, 5, 5), Delay: 0},
				{Msg: midi.NoteOff(4, 5), Delay: 20 * time.Millisecond},
				{Msg: midi.NoteOn(4, 5, 5), Delay: 125 * time.Millisecond},
				{Msg: midi.NoteOff(4, 5), Delay: 145 * time.Millisecond},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sequence, cursor := SimpleSequence()

			(*sequence.Parts)[0].Beats = tt.partBeats
			note := grid.Note{AccentIndex: 5}
			note.Ratchets.SetLength(0)
			note.Ratchets.SetLength(1)
			(*sequence.Parts)[0].Overlays.AddNote(grid.GridKey{Line: 0, Beat: 0}, note)

			beatsPlayed, testMessages := PlayTestLoop(sequence, cursor, int(tt.partBeats)+3, playstate.PlayState{Playing: true}, t.Context())
			assert.Equal(t, tt.expectedBeatsPlayed, beatsPlayed)

			if assert.Len(t, testMessages, len(tt.expectedMidiMessages), "Number of MIDI messages") {
				for i, msg := range tt.expectedMidiMessages {
					assert.Equal(t, msg.Delay, testMessages[i].Delay, "Delay")

					switch msg.Msg.Type() {
					case midi.NoteOnMsg:
						var expectedChannel, expectedNote, expectedVelocity uint8 = 0, 0, 0
						var testChannel, testNote, testVelocity uint8 = 0, 0, 0
						assert.True(t, msg.Msg.GetNoteOn(&expectedChannel, &expectedNote, &expectedVelocity), "Note On Parsing expected message")
						assert.True(t, testMessages[i].Msg.GetNoteOn(&testChannel, &testNote, &testVelocity), "Note On Parsing test message")
						assert.Equal(t, expectedChannel, testChannel, "Note On Channel")
						assert.Equal(t, expectedNote, testNote, "Note On Note")
						assert.Equal(t, expectedVelocity, testVelocity, "Note On Velocity")
					case midi.NoteOffMsg:
						var expectedChannel, expectedNote, expectedVelocity uint8 = 0, 0, 0
						var testChannel, testNote, testVelocity uint8 = 0, 0, 0
						assert.True(t, msg.Msg.GetNoteOff(&expectedChannel, &expectedNote, &expectedVelocity), "Note On Parsing expected message")
						assert.True(t, testMessages[i].Msg.GetNoteOff(&testChannel, &testNote, &testVelocity), "Note On Parsing test message")
						assert.Equal(t, expectedChannel, testChannel, "Note Off Channel")
						assert.Equal(t, expectedNote, testNote, "Note Off Note")
						assert.Equal(t, expectedVelocity, testVelocity, "Note Off Velocity")
					}
				}
			}
		})
	}
}

func TestSimpleSequenceLoopSong(t *testing.T) {
	tests := []struct {
		name                string
		partBeats           uint8
		expectedBeatsPlayed int
	}{
		{"Part with 1 beat", 1, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sequence, cursor := SimpleSequence()

			(*sequence.Parts)[0].Beats = tt.partBeats

			beatsPlayed, _ := PlayTestLoop(sequence, cursor, int(tt.partBeats)+3, playstate.PlayState{Playing: true, LoopedArrangement: sequence.Arrangement}, t.Context())
			assert.Equal(t, tt.expectedBeatsPlayed, beatsPlayed)
		})
	}
}

func TestGroupedSequenceBeats(t *testing.T) {
	tests := []struct {
		name                string
		partBeats           uint8
		groupIterations     int
		expectedBeatsPlayed int
	}{
		{"Part with 1 beat", 1, 1, 1},
		{"Part with 2 beat", 2, 2, 4},
		{"Part with 3 beats", 3, 3, 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sequence, cursor := SimpleGroupedSequence()

			(*sequence.Parts)[0].Beats = tt.partBeats
			cursor[1].Iterations = tt.groupIterations

			beatsPlayed, _ := PlayTestLoop(sequence, cursor, tt.expectedBeatsPlayed+3, playstate.PlayState{Playing: true}, t.Context())
			assert.Equal(t, tt.expectedBeatsPlayed, beatsPlayed)
		})
	}
}

func TestSiblingSections(t *testing.T) {
	tests := []struct {
		name                string
		partABeats          uint8
		partBBeats          uint8
		expectedBeatsPlayed int
	}{
		{"Parts with 1 beat", 1, 1, 2},
		{"Parts with 2 beats", 2, 2, 4},
		{"Parts with different beats 1/2", 1, 2, 3},
		{"Parts with different beats 2/1", 2, 1, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sequence, cursor := SiblingSectionSequence()

			(*sequence.Parts)[0].Beats = tt.partABeats
			(*sequence.Parts)[1].Beats = tt.partBBeats

			beatsPlayed, _ := PlayTestLoop(sequence, cursor, tt.expectedBeatsPlayed+3, playstate.PlayState{Playing: true}, t.Context())
			assert.Equal(t, tt.expectedBeatsPlayed, beatsPlayed)
		})
	}
}

func TestNestedGroups(t *testing.T) {
	tests := []struct {
		name                string
		partBeats           uint8
		groupAIterations    int
		groupBIterations    int
		expectedBeatsPlayed int
	}{
		{"Part with 1 beat", 1, 2, 2, 4},
		{"Part with 2 beats", 2, 2, 2, 8},
		{"Part with 3 beats", 3, 2, 2, 12},
		{"Part with 3 beats and different iterations", 3, 2, 3, 18},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sequence, cursor := NestedGroupsSequence()

			(*sequence.Parts)[0].Beats = tt.partBeats
			cursor[1].Iterations = tt.groupAIterations
			cursor[2].Iterations = tt.groupBIterations

			beatsPlayed, _ := PlayTestLoop(sequence, cursor, tt.expectedBeatsPlayed+3, playstate.PlayState{Playing: true}, t.Context())
			assert.Equal(t, tt.expectedBeatsPlayed, beatsPlayed)
		})
	}
}

func TestGroupPartSiblingSequence(t *testing.T) {
	tests := []struct {
		name                string
		partABeats          uint8
		partBBeats          uint8
		groupIterations     int
		expectedBeatsPlayed int
	}{
		{"Parts with 1 beat", 1, 1, 1, 2},
		{"Parts with 2 beats", 2, 2, 2, 6},
		{"Parts with different beats 1/2", 1, 2, 2, 4},
		{"Parts with different beats 2/1", 2, 1, 2, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sequence, cursor := GroupPartSiblingSequence()

			(*sequence.Parts)[0].Beats = tt.partABeats
			(*sequence.Parts)[1].Beats = tt.partBBeats
			sequence.Arrangement.Nodes[0].Iterations = tt.groupIterations

			beatsPlayed, _ := PlayTestLoop(sequence, cursor, tt.expectedBeatsPlayed+3, playstate.PlayState{Playing: true}, t.Context())
			assert.Equal(t, tt.expectedBeatsPlayed, beatsPlayed)
		})
	}
}

func TestPartGroupSiblingSequence(t *testing.T) {
	tests := []struct {
		name                string
		partABeats          uint8
		partBBeats          uint8
		groupIterations     int
		expectedBeatsPlayed int
	}{
		{"Parts with 1 beat", 1, 1, 1, 2},
		{"Parts with 2 beats", 1, 1, 2, 3},
		{"Parts with different beats 1/2", 1, 2, 2, 4},
		{"Parts with different beats 2/1", 2, 1, 2, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sequence, cursor := PartGroupSiblingSequence()

			(*sequence.Parts)[0].Beats = tt.partABeats
			(*sequence.Parts)[1].Beats = tt.partBBeats
			sequence.Arrangement.Nodes[1].Iterations = tt.groupIterations

			beatsPlayed, _ := PlayTestLoop(sequence, cursor, tt.expectedBeatsPlayed+3, playstate.PlayState{Playing: true}, t.Context())
			assert.Equal(t, tt.expectedBeatsPlayed, beatsPlayed)
		})
	}
}

func PlayTestLoop(sequence sequence.Sequence, cursor arrangement.ArrCursor, limit int, playState playstate.PlayState, ctx context.Context) (int, []seqmidi.Message) {
	testMessageChan := make(chan ModelPlayedMsg)
	beatsPlayedCounter := 0
	var update = ModelPlayedMsg{PlayState: playState}
	sendFn := func(msg tea.Msg) {
		switch msg := msg.(type) {
		case ModelPlayedMsg:
			testMessageChan <- msg
		case AnticipatoryStop:
		}
	}

	beatsLooper := InitBeatsLooper()

	updateChannel := beatsLooper.UpdateChannel
	beatChannel := beatsLooper.BeatChannel

	midiConnection := seqmidi.MidiConnection{Test: true, TestQueue: &[]seqmidi.Message{}}
	beatsLooper.Loop(sendFn, &midiConnection, ctx)

	iterations := make(playstate.Iterations)
	playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
	playState.LineStates = playstate.InitLineStates(1, []playstate.LineState{}, 0)
	playState.Iterations = &iterations

	updateChannel <- ModelMsg{
		PlayState: playState,
		Sequence:  sequence,
		Cursor:    cursor,
	}

	for update.PlayState.Playing && beatsPlayedCounter < limit {
		beatChannel <- BeatMsg{Interval: 250 * time.Millisecond}
		update = <-testMessageChan
		if update.PerformStop {
			break
		} else {
			beatsPlayedCounter++
		}
		updateChannel <- ModelMsg{PlayState: update.PlayState, Sequence: sequence, Cursor: update.Cursor}
	}

	return beatsPlayedCounter, *midiConnection.TestQueue
}

func SimpleSequence() (sequence.Sequence, arrangement.ArrCursor) {
	var parts = sequence.InitParts()

	nodeA := &arrangement.Arrangement{
		Section:    arrangement.SongSection{Part: 0, Cycles: 1, StartBeat: 0, StartCycles: 1},
		Iterations: 1,
	}

	root := &arrangement.Arrangement{
		Iterations: 1,
		Nodes:      make([]*arrangement.Arrangement, 0),
	}

	root.Nodes = append(root.Nodes, nodeA)

	testSequence := sequence.Sequence{
		Arrangement: root,
		Parts:       &parts,
		Keyline:     0,
		Lines:       []grid.LineDefinition{{Channel: 5, Note: 5, MsgType: grid.MessageTypeNote, Name: "Line 1"}},
		Accents: sequence.PatternAccents{
			Start:  0,
			End:    8,
			Data:   []config.Accent{0, 1, 2, 3, 4, 5, 6, 7},
			Target: sequence.AccentTargetVelocity,
		},
	}

	return testSequence, arrangement.ArrCursor{root, nodeA}
}

func SiblingSectionSequence() (sequence.Sequence, arrangement.ArrCursor) {
	var parts = sequence.InitParts()
	parts = append(parts, arrangement.InitPart("Part 2"))

	nodeA := &arrangement.Arrangement{
		Section:    arrangement.SongSection{Part: 0, Cycles: 1, StartBeat: 0, StartCycles: 1},
		Iterations: 1,
	}

	nodeB := &arrangement.Arrangement{
		Section:    arrangement.SongSection{Part: 1, Cycles: 1, StartBeat: 0, StartCycles: 1},
		Iterations: 1,
	}

	root := &arrangement.Arrangement{
		Iterations: 1,
		Nodes:      make([]*arrangement.Arrangement, 0),
	}

	root.Nodes = append(root.Nodes, nodeA, nodeB)

	testSequence := sequence.Sequence{
		Arrangement: root,
		Parts:       &parts,
		Keyline:     0,
		Lines:       make([]grid.LineDefinition, 1),
	}

	return testSequence, arrangement.ArrCursor{root, nodeA}
}

func SimpleGroupedSequence() (sequence.Sequence, arrangement.ArrCursor) {
	var parts = sequence.InitParts()

	nodeA := &arrangement.Arrangement{
		Section:    arrangement.SongSection{Part: 0, Cycles: 1, StartBeat: 0, StartCycles: 1},
		Iterations: 1,
	}

	groupA := &arrangement.Arrangement{
		Iterations: 1,
		Nodes:      []*arrangement.Arrangement{nodeA},
	}

	root := &arrangement.Arrangement{
		Iterations: 1,
		Nodes:      make([]*arrangement.Arrangement, 0),
	}

	root.Nodes = append(root.Nodes, groupA)

	testSequence := sequence.Sequence{
		Arrangement: root,
		Parts:       &parts,
		Keyline:     0,
		Lines:       make([]grid.LineDefinition, 1),
	}

	return testSequence, arrangement.ArrCursor{root, groupA, nodeA}
}

func NestedGroupsSequence() (sequence.Sequence, arrangement.ArrCursor) {
	var parts = sequence.InitParts()

	nodeA := &arrangement.Arrangement{
		Section:    arrangement.SongSection{Part: 0, Cycles: 1, StartBeat: 0, StartCycles: 1},
		Iterations: 1,
	}

	groupA := &arrangement.Arrangement{
		Iterations: 1,
		Nodes:      []*arrangement.Arrangement{nodeA},
	}

	groupB := &arrangement.Arrangement{
		Iterations: 1,
		Nodes:      []*arrangement.Arrangement{groupA},
	}

	root := &arrangement.Arrangement{
		Iterations: 1,
		Nodes:      make([]*arrangement.Arrangement, 0),
	}

	root.Nodes = append(root.Nodes, groupB)

	testSequence := sequence.Sequence{
		Arrangement: root,
		Parts:       &parts,
		Keyline:     0,
		Lines:       make([]grid.LineDefinition, 1),
	}

	return testSequence, arrangement.ArrCursor{root, groupB, groupA, nodeA}
}

func GroupPartSiblingSequence() (sequence.Sequence, arrangement.ArrCursor) {
	var parts = sequence.InitParts()
	parts = append(parts, arrangement.InitPart("Part 2"))

	nodeA := &arrangement.Arrangement{
		Section:    arrangement.SongSection{Part: 0, Cycles: 1, StartBeat: 0, StartCycles: 1},
		Iterations: 1,
	}

	nodeB := &arrangement.Arrangement{
		Section:    arrangement.SongSection{Part: 1, Cycles: 1, StartBeat: 0, StartCycles: 1},
		Iterations: 1,
	}

	groupA := &arrangement.Arrangement{
		Iterations: 1,
		Nodes:      []*arrangement.Arrangement{nodeA},
	}

	root := &arrangement.Arrangement{
		Iterations: 1,
		Nodes:      make([]*arrangement.Arrangement, 0),
	}

	root.Nodes = append(root.Nodes, groupA, nodeB)

	testSequence := sequence.Sequence{
		Arrangement: root,
		Parts:       &parts,
		Keyline:     0,
		Lines:       make([]grid.LineDefinition, 1),
	}

	return testSequence, arrangement.ArrCursor{root, groupA, nodeA}
}

func PartGroupSiblingSequence() (sequence.Sequence, arrangement.ArrCursor) {
	var parts = sequence.InitParts()
	parts = append(parts, arrangement.InitPart("Part 2"))

	nodeA := &arrangement.Arrangement{
		Section:    arrangement.SongSection{Part: 0, Cycles: 1, StartBeat: 0, StartCycles: 1},
		Iterations: 1,
	}

	nodeB := &arrangement.Arrangement{
		Section:    arrangement.SongSection{Part: 1, Cycles: 1, StartBeat: 0, StartCycles: 1},
		Iterations: 1,
	}

	groupA := &arrangement.Arrangement{
		Iterations: 1,
		Nodes:      []*arrangement.Arrangement{nodeA},
	}

	root := &arrangement.Arrangement{
		Iterations: 1,
		Nodes:      make([]*arrangement.Arrangement, 0),
	}

	root.Nodes = append(root.Nodes, nodeB, groupA)

	testSequence := sequence.Sequence{
		Arrangement: root,
		Parts:       &parts,
		Keyline:     0,
		Lines:       make([]grid.LineDefinition, 1),
	}

	return testSequence, arrangement.ArrCursor{root, nodeB}
}

func TestCCMessage(t *testing.T) {
	tests := []struct {
		name          string
		line          grid.LineDefinition
		note          grid.Note
		accents       []config.Accent
		delay         time.Duration
		instrument    string
		expectedChan  uint8
		expectedCtrl  uint8
		expectedValue uint8
		expectedDelay time.Duration
	}{
		{
			name:          "Specific value action",
			line:          grid.LineDefinition{Channel: 1, Note: 10},
			note:          grid.Note{Action: grid.ActionSpecificValue, AccentIndex: 42},
			accents:       []config.Accent{0, 30, 60, 90, 120},
			delay:         100 * time.Millisecond,
			instrument:    "",
			expectedChan:  0,
			expectedCtrl:  10,
			expectedValue: 42,
			expectedDelay: 100 * time.Millisecond,
		},
		{
			name:          "Accent-based value mid-range",
			line:          grid.LineDefinition{Channel: 3, Note: 7},
			note:          grid.Note{Action: grid.ActionNothing, AccentIndex: 2},
			accents:       []config.Accent{0, 30, 60, 90, 120},
			delay:         0,
			instrument:    "",
			expectedChan:  2,
			expectedCtrl:  7,
			expectedValue: 95, // (5-2)/4 * 127 = 95.25 -> 95
			expectedDelay: 0,
		},
		{
			name:          "Accent-based value at end",
			line:          grid.LineDefinition{Channel: 4, Note: 11},
			note:          grid.Note{Action: grid.ActionNothing, AccentIndex: 4},
			accents:       []config.Accent{0, 30, 60, 90, 120},
			delay:         10 * time.Millisecond,
			instrument:    "",
			expectedChan:  3,
			expectedCtrl:  11,
			expectedValue: 31, // (5-4)/4 * 127 = 31.75 -> 31
			expectedDelay: 10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CCMessage(tt.line, tt.note, tt.accents, tt.delay, true, tt.instrument)

			assert.Equal(t, tt.expectedChan, result.channel, "Channel")
			assert.Equal(t, tt.expectedCtrl, result.control, "Control")
			assert.Equal(t, tt.expectedValue, result.ccValue, "CC Value")
			assert.Equal(t, tt.expectedDelay, result.delay, "Delay")

			// Test that MidiMessage generates correct MIDI message
			msg := result.MidiMessage()
			var channel, control, value uint8
			assert.True(t, msg.GetControlChange(&channel, &control, &value), "MIDI message should be control change")
			assert.Equal(t, tt.expectedChan, channel, "MIDI channel")
			assert.Equal(t, tt.expectedCtrl, control, "MIDI control")
			assert.Equal(t, tt.expectedValue, value, "MIDI value")
		})
	}
}

func TestPCMessage(t *testing.T) {
	tests := []struct {
		name          string
		line          grid.LineDefinition
		note          grid.Note
		accents       []config.Accent
		delay         time.Duration
		instrument    string
		expectedChan  uint8
		expectedValue uint8
		expectedDelay time.Duration
	}{
		{
			name:          "Specific value action",
			line:          grid.LineDefinition{Channel: 1, Note: 10},
			note:          grid.Note{Action: grid.ActionSpecificValue, AccentIndex: 64},
			accents:       []config.Accent{0, 30, 60, 90, 120},
			delay:         100 * time.Millisecond,
			instrument:    "",
			expectedChan:  0,
			expectedValue: 64,
			expectedDelay: 100 * time.Millisecond,
		},
		{
			name:          "Accent-based value mid-range",
			line:          grid.LineDefinition{Channel: 3, Note: 30},
			note:          grid.Note{Action: grid.ActionNothing, AccentIndex: 2},
			accents:       []config.Accent{0, 30, 60, 90, 120},
			delay:         0,
			instrument:    "",
			expectedChan:  2,
			expectedValue: 95, // (5-2)/4 * 127 = 95.25 -> 95
			expectedDelay: 0,
		},
		{
			name:          "Accent-based value at end",
			line:          grid.LineDefinition{Channel: 4, Note: 40},
			note:          grid.Note{Action: grid.ActionNothing, AccentIndex: 4},
			accents:       []config.Accent{0, 30, 60, 90, 120},
			delay:         10 * time.Millisecond,
			instrument:    "",
			expectedChan:  3,
			expectedValue: 31, // (5-4)/4 * 127 = 31.75 -> 31
			expectedDelay: 10 * time.Millisecond,
		},
		{
			name:          "Zero delay",
			line:          grid.LineDefinition{Channel: 5, Note: 50},
			note:          grid.Note{Action: grid.ActionNothing, AccentIndex: 1},
			accents:       []config.Accent{0, 30, 60, 90, 120},
			delay:         0,
			instrument:    "",
			expectedChan:  4,
			expectedValue: 127, // (5-1)/4 * 127 = 127 exactly
			expectedDelay: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PCMessage(tt.line, tt.note, tt.accents, tt.delay, true, tt.instrument)

			assert.Equal(t, tt.expectedChan, result.channel, "Channel")
			assert.Equal(t, tt.expectedValue, result.pcValue, "PC Value")
			assert.Equal(t, tt.expectedDelay, result.delay, "Delay")

			// Test that MidiMessage generates correct MIDI message
			msg := result.MidiMessage()
			var channel, value uint8
			assert.True(t, msg.GetProgramChange(&channel, &value), "MIDI message should be program change")
			assert.Equal(t, tt.expectedChan, channel, "MIDI channel")
			assert.Equal(t, tt.expectedValue, value, "MIDI value")
		})
	}
}
