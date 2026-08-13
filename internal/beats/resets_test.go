package beats

import (
	"testing"

	"github.com/chriserin/sq/internal/arrangement"
	"github.com/chriserin/sq/internal/grid"
	"github.com/chriserin/sq/internal/playstate"
	"github.com/chriserin/sq/internal/sequence"
	"github.com/stretchr/testify/assert"
)

// nestedGroupSiblingSequence builds root -> [nodeB (plain part), groupOuter
// -> groupInner -> nodeA] — a plain sibling part followed by two levels of
// nested groups, so moving from nodeB into the group chain enters both
// groupOuter and groupInner in a single beat.
func nestedGroupSiblingSequence() (sequence.Sequence, arrangement.ArrCursor) {
	parts := sequence.InitParts()
	parts = append(parts, arrangement.InitPart("Part 2"))

	nodeA := &arrangement.Arrangement{
		Section:    arrangement.SongSection{Part: 0, Cycles: 1, StartBeat: 0, StartCycles: 1},
		Iterations: 1,
	}
	nodeB := &arrangement.Arrangement{
		Section:    arrangement.SongSection{Part: 1, Cycles: 1, StartBeat: 0, StartCycles: 1},
		Iterations: 1,
	}
	groupInner := &arrangement.Arrangement{
		Iterations: 1,
		Nodes:      []*arrangement.Arrangement{nodeA},
	}
	groupOuter := &arrangement.Arrangement{
		Iterations: 1,
		Nodes:      []*arrangement.Arrangement{groupInner},
	}
	root := &arrangement.Arrangement{
		Iterations: 1,
		Nodes:      make([]*arrangement.Arrangement, 0),
	}
	root.Nodes = append(root.Nodes, nodeB, groupOuter)

	testSequence := sequence.Sequence{
		Arrangement: root,
		Parts:       &parts,
		Keyline:     0,
		Lines:       make([]grid.LineDefinition, 1),
	}

	return testSequence, arrangement.ArrCursor{root, nodeB}
}

// kinds extracts just the ResetKind values, in order, for easier assertions.
func kinds(events []ResetEvent) []ResetKind {
	result := make([]ResetKind, len(events))
	for i, ev := range events {
		result[i] = ev.Kind
	}
	return result
}

func TestAdvancePlayState_ResetEvents(t *testing.T) {
	t.Run("plain part-to-part move fires PartLoop and PartStart together, then stops with no false-positive song start", func(t *testing.T) {
		sequence, cursor := SiblingSectionSequence()
		(*sequence.Parts)[0].Beats = 1
		(*sequence.Parts)[1].Beats = 1

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
		}

		events := AdvancePlayState(&playState, sequence, &cursor)
		assert.Equal(t, []ResetKind{ResetKindPartLoop, ResetKindPartStart}, kinds(events))
		assert.True(t, playState.Playing)

		events = AdvancePlayState(&playState, sequence, &cursor)
		assert.Nil(t, events, "an ordinary end-of-song stop must not fire a false-positive song-start reset")
		assert.False(t, playState.Playing)
	})

	t.Run("a part repeating its own cycles fires PartLoop, not PartStart", func(t *testing.T) {
		sequence, cursor := SimpleSequence()
		(*sequence.Parts)[0].Beats = 1
		cursor[len(cursor)-1].Section.Cycles = 2

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
		}

		events := AdvancePlayState(&playState, sequence, &cursor)
		assert.Equal(t, []ResetKind{ResetKindPartLoop}, kinds(events))
		assert.True(t, playState.Playing)

		events = AdvancePlayState(&playState, sequence, &cursor)
		assert.Nil(t, events)
		assert.False(t, playState.Playing)
	})

	t.Run("a group looping through its only child cascades to PartLoop and PartStart together", func(t *testing.T) {
		// The group repeating means everything inside it repeats too — the
		// part is being freshly re-entered as a consequence of the group's
		// own loop-back, not independently continuing on its own, even though
		// it happens to be the exact same part node (the group's only child).
		// PartLoop fires alongside PartStart, not instead of it — PartLoop is
		// "content began a fresh pass, for any reason", a superset of
		// PartStart's "specifically a fresh entry".
		sequence, cursor := SimpleGroupedSequence()
		(*sequence.Parts)[0].Beats = 1
		cursor[1].Iterations = 2

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
		}

		events := AdvancePlayState(&playState, sequence, &cursor)
		assert.Equal(t, []ResetKind{ResetKindGroupLoop, ResetKindPartLoop, ResetKindPartStart}, kinds(events))
		assert.True(t, playState.Playing)

		events = AdvancePlayState(&playState, sequence, &cursor)
		assert.Nil(t, events)
		assert.False(t, playState.Playing)
	})

	t.Run("an outer group looping cascades Start through an inner, pointer-identical group", func(t *testing.T) {
		// groupB (outer, iterations=2) contains only groupA (inner,
		// iterations=1), which contains only nodeA. groupB genuinely loops
		// back to its own first child — which is groupA again, the exact
		// same pointer, since it's groupB's only child. Even so, groupA must
		// report GroupStart (not GroupLoop, not nothing), because it's being
		// freshly re-entered as a consequence of groupB's own loop-back.
		sequence, cursor := NestedGroupsSequence()
		(*sequence.Parts)[0].Beats = 1
		cursor[1].Iterations = 2 // groupB
		cursor[2].Iterations = 1 // groupA

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
		}

		events := AdvancePlayState(&playState, sequence, &cursor)
		assert.Equal(t, []ResetKind{ResetKindGroupLoop, ResetKindGroupStart, ResetKindPartLoop, ResetKindPartStart}, kinds(events))
		assert.True(t, playState.Playing)
	})

	t.Run("entering two nested groups at once fires only one GroupLoop/GroupStart pair, not one per level", func(t *testing.T) {
		// Regression test for a real bug: moving from a plain sibling part
		// into a chain of nested groups (groupOuter -> groupInner) used to
		// fire GroupStart once per group level, producing multiple NoteOn
		// events on the exact same channel/note in the same beat — there's
		// only one groupStart mapping configured, with no notion of depth,
		// so entering N nested groups simultaneously must still be reported
		// as a single GroupStart event.
		sequence, cursor := nestedGroupSiblingSequence()
		(*sequence.Parts)[0].Beats = 1
		(*sequence.Parts)[1].Beats = 1

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
		}

		events := AdvancePlayState(&playState, sequence, &cursor)
		assert.Equal(t, []ResetKind{ResetKindGroupLoop, ResetKindGroupStart, ResetKindPartLoop, ResetKindPartStart}, kinds(events))
		assert.True(t, playState.Playing)
	})

	t.Run("moving from a plain part into a sibling group fires GroupLoop, GroupStart, PartLoop, and PartStart together", func(t *testing.T) {
		sequence, cursor := PartGroupSiblingSequence()
		(*sequence.Parts)[0].Beats = 1
		(*sequence.Parts)[1].Beats = 1
		sequence.Arrangement.Nodes[1].Iterations = 1

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
		}

		events := AdvancePlayState(&playState, sequence, &cursor)
		assert.Equal(t, []ResetKind{ResetKindGroupLoop, ResetKindGroupStart, ResetKindPartLoop, ResetKindPartStart}, kinds(events))
		assert.True(t, playState.Playing)
	})

	t.Run("a looped multi-part song only fires Song on the beat that actually loops back", func(t *testing.T) {
		// Regression test: PlayMove's plain sibling-move branch (an ordinary
		// A-to-B transition, unrelated to root) also calls
		// Iterations.ResetIterations on the new cursor, which can zero root's
		// counter as a side effect whenever it happens to already equal its
		// own target. An earlier version of diffResetEvents treated any value
		// change as a lap and fired a spurious Song event on that ordinary
		// transition beat. Only beats where the cursor genuinely loops back
		// from the last part to the first (root's counter actually
		// increasing) should fire Song.
		sequence, cursor := SiblingSectionSequence()
		(*sequence.Parts)[0].Beats = 1
		(*sequence.Parts)[1].Beats = 1

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		playState := playstate.PlayState{
			Playing:           true,
			AllowAdvance:      true,
			LineStates:        playstate.InitLineStates(1, nil, 0),
			Iterations:        &iterations,
			LoopedArrangement: sequence.Arrangement,
		}

		expected := [][]ResetKind{
			{ResetKindPartLoop, ResetKindPartStart},                // beat 0: A -> B, ordinary move
			{ResetKindSong, ResetKindPartLoop, ResetKindPartStart}, // beat 1: B -> A, genuine loop-back
			{ResetKindPartLoop, ResetKindPartStart},                // beat 2: A -> B again, ordinary move — must NOT also fire Song
			{ResetKindSong, ResetKindPartLoop, ResetKindPartStart}, // beat 3: B -> A, genuine loop-back again
		}
		for beat, want := range expected {
			events := AdvancePlayState(&playState, sequence, &cursor)
			assert.Equal(t, want, kinds(events), "beat %d", beat)
			assert.True(t, playState.Playing)
		}
	})

	t.Run("a looped single-part song fires Song, PartLoop, and PartStart together on every lap", func(t *testing.T) {
		// Same principle as the group case above: repeating the song must
		// repeat everything inside it, including the part, even when the
		// whole song is just that one part — and PartLoop fires alongside
		// PartStart (not instead of it), including on this, the very first
		// beat of every lap.
		sequence, cursor := SimpleSequence()
		(*sequence.Parts)[0].Beats = 1

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		playState := playstate.PlayState{
			Playing:           true,
			AllowAdvance:      true,
			LineStates:        playstate.InitLineStates(1, nil, 0),
			Iterations:        &iterations,
			LoopedArrangement: sequence.Arrangement,
		}

		for lap := 0; lap < 3; lap++ {
			events := AdvancePlayState(&playState, sequence, &cursor)
			assert.Equal(t, []ResetKind{ResetKindSong, ResetKindPartLoop, ResetKindPartStart}, kinds(events), "lap %d", lap)
			assert.True(t, playState.Playing)
		}
	})

	t.Run("not playing or not yet allowed to advance produces no events", func(t *testing.T) {
		sequence, cursor := SimpleSequence()
		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)

		notPlaying := playstate.PlayState{Playing: false, AllowAdvance: true, Iterations: &iterations, LineStates: playstate.InitLineStates(1, nil, 0)}
		assert.Nil(t, AdvancePlayState(&notPlaying, sequence, &cursor))

		notAllowed := playstate.PlayState{Playing: true, AllowAdvance: false, Iterations: &iterations, LineStates: playstate.InitLineStates(1, nil, 0)}
		assert.Nil(t, AdvancePlayState(&notAllowed, sequence, &cursor))
	})

	t.Run("PlayReceiver on a single-part arrangement keeps relooping in place instead of stopping", func(t *testing.T) {
		// IsDone's own PlayReceiver/AllLastSiblings/IsFull guard (beats.go:133)
		// deliberately holds off calling PlayMove at all here — a receiver
		// session relies on an external Stop (or the transmitter's pulseLimit
		// mechanism) rather than PlayMove's cursor.IsRoot() fallback to end
		// playback, so this never reaches AdvancePlayState's forcedSongRestart
		// handling; it just keeps firing PartLoop every beat, matching a real
		// receiver that keeps replaying the last part until told to stop.
		sequence, cursor := SimpleSequence()
		(*sequence.Parts)[0].Beats = 1

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
			PlayMode:     playstate.PlayReceiver,
		}

		for beat := 0; beat < 3; beat++ {
			events := AdvancePlayState(&playState, sequence, &cursor)
			assert.Equal(t, []ResetKind{ResetKindPartLoop}, kinds(events), "beat %d", beat)
			assert.True(t, playState.Playing)
		}
	})
}

func TestInitialResetEvents(t *testing.T) {
	t.Run("song, PartLoop, and PartStart for a flat, ungrouped cursor", func(t *testing.T) {
		// The very first entry is still a "content began a fresh pass" event,
		// so PartLoop fires alongside PartStart here too, not just on later
		// repeats.
		_, cursor := SimpleSequence()
		events := InitialResetEvents(cursor)
		assert.Equal(t, []ResetKind{ResetKindSong, ResetKindPartLoop, ResetKindPartStart}, kinds(events))
	})

	t.Run("song, one group, and part for a single-nested cursor", func(t *testing.T) {
		_, cursor := SimpleGroupedSequence()
		events := InitialResetEvents(cursor)
		assert.Equal(t, []ResetKind{ResetKindSong, ResetKindGroupLoop, ResetKindGroupStart, ResetKindPartLoop, ResetKindPartStart}, kinds(events))
	})

	t.Run("song, two nested groups, and part fires only one GroupLoop/GroupStart pair", func(t *testing.T) {
		// There's only one groupStart/groupLoop channel/note configured each
		// — no notion of depth — so entering two nested groups at once must
		// not produce two separate NoteOn events on the same channel/note.
		_, cursor := NestedGroupsSequence()
		events := InitialResetEvents(cursor)
		assert.Equal(t, []ResetKind{ResetKindSong, ResetKindGroupLoop, ResetKindGroupStart, ResetKindPartLoop, ResetKindPartStart}, kinds(events))
	})
}
