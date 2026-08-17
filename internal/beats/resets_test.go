package beats

import (
	"maps"
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

// groupTwoPartsSequence builds root -> groupA -> [nodeA, nodeB] — a group
// with two sibling parts, and groupA's own target (2) requires two full
// laps before it's exhausted, so an ordinary nodeA -> nodeB move leaves
// groupA still resting on its first, not-yet-completed pass.
func groupTwoPartsSequence() (sequence.Sequence, arrangement.ArrCursor) {
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
	groupA := &arrangement.Arrangement{
		Iterations: 2,
		Nodes:      []*arrangement.Arrangement{nodeA, nodeB},
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

// advanceAndCompute mirrors the two-call sequence production code uses
// (Beat(): AdvancePlayState then computeResetEvents) as a single call, for
// tests that only care about the resulting events, not the intermediate
// separation.
func advanceAndCompute(playState *playstate.PlayState, sequence sequence.Sequence, cursor *arrangement.ArrCursor) []ResetEvent {
	wrapped := AdvancePlayState(playState, sequence, cursor)
	if wrapped {
		return computeResetEvents(*cursor, playState.Iterations, playState.Baseline)
	} else {
		return nil
	}
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
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
			Baseline:     &baseline,
		}

		events := advanceAndCompute(&playState, sequence, &cursor)
		assert.Equal(t, []ResetKind{ResetKindPartStart, ResetKindPartLoop}, kinds(events))
		assert.True(t, playState.Playing)

		events = advanceAndCompute(&playState, sequence, &cursor)
		assert.Nil(t, events, "an ordinary end-of-song stop must not fire a false-positive song-start reset")
		assert.False(t, playState.Playing)
	})

	t.Run("a part repeating its own cycles fires PartLoop, not PartStart", func(t *testing.T) {
		sequence, cursor := SimpleSequence()
		(*sequence.Parts)[0].Beats = 1
		cursor[len(cursor)-1].Section.Cycles = 2

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
			Baseline:     &baseline,
		}

		events := advanceAndCompute(&playState, sequence, &cursor)
		assert.Equal(t, []ResetKind{ResetKindPartLoop}, kinds(events))
		assert.True(t, playState.Playing)

		events = advanceAndCompute(&playState, sequence, &cursor)
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
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
			Baseline:     &baseline,
		}

		events := advanceAndCompute(&playState, sequence, &cursor)
		assert.Equal(t, []ResetKind{ResetKindGroupLoop, ResetKindPartStart, ResetKindPartLoop}, kinds(events))
		assert.True(t, playState.Playing)

		events = advanceAndCompute(&playState, sequence, &cursor)
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
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
			Baseline:     &baseline,
		}

		events := advanceAndCompute(&playState, sequence, &cursor)
		assert.Equal(t, []ResetKind{ResetKindGroupStart, ResetKindGroupLoop, ResetKindPartStart, ResetKindPartLoop}, kinds(events))
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
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
			Baseline:     &baseline,
		}

		events := advanceAndCompute(&playState, sequence, &cursor)
		assert.Equal(t, []ResetKind{ResetKindGroupStart, ResetKindGroupLoop, ResetKindPartStart, ResetKindPartLoop}, kinds(events))
		assert.True(t, playState.Playing)
	})

	t.Run("moving from a plain part into a sibling group fires GroupLoop, GroupStart, PartLoop, and PartStart together", func(t *testing.T) {
		sequence, cursor := PartGroupSiblingSequence()
		(*sequence.Parts)[0].Beats = 1
		(*sequence.Parts)[1].Beats = 1
		sequence.Arrangement.Nodes[1].Iterations = 1

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
			Baseline:     &baseline,
		}

		events := advanceAndCompute(&playState, sequence, &cursor)
		assert.Equal(t, []ResetKind{ResetKindGroupStart, ResetKindGroupLoop, ResetKindPartStart, ResetKindPartLoop}, kinds(events))
		assert.True(t, playState.Playing)
	})

	t.Run("a looped multi-part song only fires Song on the beat that actually loops back", func(t *testing.T) {
		// An ordinary A-to-B move never lands the cursor on root's actual
		// first child (MoveToSibling only ever targets index+1), so Song
		// must not fire on it — only a genuine loop-back does.
		sequence, cursor := SiblingSectionSequence()
		(*sequence.Parts)[0].Beats = 1
		(*sequence.Parts)[1].Beats = 1

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		playState := playstate.PlayState{
			Playing:           true,
			AllowAdvance:      true,
			LineStates:        playstate.InitLineStates(1, nil, 0),
			Iterations:        &iterations,
			Baseline:          &baseline,
			LoopedArrangement: sequence.Arrangement,
		}

		expected := [][]ResetKind{
			{ResetKindPartStart, ResetKindPartLoop},                // beat 0: A -> B, ordinary move
			{ResetKindSong, ResetKindPartStart, ResetKindPartLoop}, // beat 1: B -> A, genuine loop-back
			{ResetKindPartStart, ResetKindPartLoop},                // beat 2: A -> B again, ordinary move — must NOT also fire Song
			{ResetKindSong, ResetKindPartStart, ResetKindPartLoop}, // beat 3: B -> A, genuine loop-back again
		}
		for beat, want := range expected {
			events := advanceAndCompute(&playState, sequence, &cursor)
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
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		playState := playstate.PlayState{
			Playing:           true,
			AllowAdvance:      true,
			LineStates:        playstate.InitLineStates(1, nil, 0),
			Iterations:        &iterations,
			Baseline:          &baseline,
			LoopedArrangement: sequence.Arrangement,
		}

		for lap := 0; lap < 3; lap++ {
			events := advanceAndCompute(&playState, sequence, &cursor)
			assert.Equal(t, []ResetKind{ResetKindSong, ResetKindPartStart, ResetKindPartLoop}, kinds(events), "lap %d", lap)
			assert.True(t, playState.Playing)
		}
	})

	t.Run("not playing or not yet allowed to advance produces no events", func(t *testing.T) {
		sequence, cursor := SimpleSequence()
		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)

		notPlaying := playstate.PlayState{Playing: false, AllowAdvance: true, Iterations: &iterations, Baseline: &baseline, LineStates: playstate.InitLineStates(1, nil, 0)}
		assert.Empty(t, advanceAndCompute(&notPlaying, sequence, &cursor))

		notAllowed := playstate.PlayState{Playing: true, AllowAdvance: false, Iterations: &iterations, Baseline: &baseline, LineStates: playstate.InitLineStates(1, nil, 0)}
		assert.Empty(t, advanceAndCompute(&notAllowed, sequence, &cursor))
	})

	t.Run("PlayReceiver on a single-part arrangement keeps relooping in place instead of stopping", func(t *testing.T) {
		// IsDone's own PlayReceiver/AllLastSiblings/IsFull guard (beats.go:133)
		// deliberately holds off calling PlayMove here — a receiver relies on
		// an external Stop rather than PlayMove's cursor.IsRoot() fallback to
		// end playback, so this just keeps firing PartLoop every beat.
		//
		// That guard also makes the cursor.IsRoot() fallback itself
		// unreachable in general: IsFull's leaf check uses the same
		// threshold as IsDone's own "done" check, so the guard is already
		// satisfied on the very first beat that would otherwise trigger
		// PlayMove, for any arrangement shape. No dedicated test covers
		// that fallback directly — "only fires Song on the beat that
		// actually loops back" below exercises the same climb logic
		// through the reachable, ordinary loop-back path instead.
		sequence, cursor := SimpleSequence()
		(*sequence.Parts)[0].Beats = 1

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
			Baseline:     &baseline,
			PlayMode:     playstate.PlayReceiver,
		}

		for beat := 0; beat < 3; beat++ {
			events := advanceAndCompute(&playState, sequence, &cursor)
			assert.Equal(t, []ResetKind{ResetKindPartLoop}, kinds(events), "beat %d", beat)
			assert.True(t, playState.Playing)
		}
	})

	t.Run("only the beat where the keyline actually wraps produces events, not every beat a node happens to sit at baseline", func(t *testing.T) {
		// Regression test for a bug caught while designing this rework: a
		// bare "is this node at its baseline value" check can't tell "this
		// just became true this beat" from "has been resting true for many
		// beats already" — a part sits at its baseline for every beat of its
		// first pass through its own grid, not just the one entry beat.
		// Callers must gate computeResetEvents on wrapped (AdvancePlayState's
		// return value) to avoid re-firing PartStart/Song on every one of
		// those beats.
		sequence, cursor := SimpleSequence()
		(*sequence.Parts)[0].Beats = 4 // several beats per pass, so most beats aren't wrap beats

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		playState := playstate.PlayState{
			Playing:           true,
			AllowAdvance:      true,
			LineStates:        playstate.InitLineStates(1, nil, 0),
			Iterations:        &iterations,
			Baseline:          &baseline,
			LoopedArrangement: sequence.Arrangement,
		}

		expected := [][]ResetKind{
			{}, // beat 0: mid-pass, not a wrap beat — leaf still sits at baseline the whole time
			{}, // beat 1: mid-pass
			{}, // beat 2: mid-pass
			{ResetKindSong, ResetKindPartStart, ResetKindPartLoop}, // beat 3: the keyline wraps — genuine loop-back
		}
		for beat, want := range expected {
			events := advanceAndCompute(&playState, sequence, &cursor)
			assert.Equal(t, want, kinds(events), "beat %d", beat)
			assert.True(t, playState.Playing)
		}
	})

	t.Run("a KeepCycles part at a group's first-child position still fires GroupLoop/PartStart on genuine entry", func(t *testing.T) {
		// KeepCycles deliberately skips resetting a part's Iterations to
		// StartCycles on re-entry (beats.go), so comparing against the fixed
		// StartCycles constant can't tell a genuine fresh/looping entry apart
		// from an ordinary local repeat for such a part — and because the
		// climb's gate applies the same test to whichever child is being
		// examined at each level, this would otherwise silently suppress
		// GroupLoop/GroupStart for everything above it too, not just this
		// part's own PartStart. Baseline (updated to whatever value the part
		// actually carries at each entry, not a fixed constant) fixes this.
		sequence, cursor := SimpleGroupedSequence()
		(*sequence.Parts)[0].Beats = 1
		cursor[1].Iterations = 2 // groupA needs 2 laps before it's exhausted
		cursor[2].Section.KeepCycles = true

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		iterations[cursor[2]] = 5 // simulate a carried-over, non-StartCycles value
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		baseline[cursor[2]] = 1 // stale baseline, deliberately mismatched from the carried-over value above

		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
			Baseline:     &baseline,
		}

		events := advanceAndCompute(&playState, sequence, &cursor)
		assert.Equal(t, []ResetKind{ResetKindGroupLoop, ResetKindPartStart, ResetKindPartLoop}, kinds(events))
		assert.True(t, playState.Playing)
	})

	t.Run("an ordinary sibling move inside a group still on its first pass fires PartStart/PartLoop only", func(t *testing.T) {
		// The group hasn't completed even one lap yet (still resting at its
		// initial baseline of 0), so before the Nodes[0]-position fix this
		// could be mistaken for the group itself being freshly entered. An
		// ordinary sibling move can never land on Nodes[0] (MoveToSibling
		// only ever targets index+1), so the climb's gate must fail here and
		// stop before ever consulting the group's own value.
		sequence, cursor := groupTwoPartsSequence()
		(*sequence.Parts)[0].Beats = 1
		(*sequence.Parts)[1].Beats = 1

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		playState := playstate.PlayState{
			Playing:      true,
			AllowAdvance: true,
			LineStates:   playstate.InitLineStates(1, nil, 0),
			Iterations:   &iterations,
			Baseline:     &baseline,
		}

		events := advanceAndCompute(&playState, sequence, &cursor)
		assert.Equal(t, []ResetKind{ResetKindPartStart, ResetKindPartLoop}, kinds(events))
		assert.True(t, playState.Playing)
	})
}

func TestInitialResetEvents(t *testing.T) {
	t.Run("song, PartLoop, and PartStart for a flat, ungrouped cursor", func(t *testing.T) {
		// The very first entry is still a "content began a fresh pass" event,
		// so PartLoop fires alongside PartStart here too, not just on later
		// repeats.
		sequence, cursor := SimpleSequence()
		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		events := InitialResetEvents(cursor, &iterations, &baseline)
		assert.Equal(t, []ResetKind{ResetKindSong, ResetKindPartStart, ResetKindPartLoop}, kinds(events))
	})

	t.Run("song, one group, and part for a single-nested cursor", func(t *testing.T) {
		sequence, cursor := SimpleGroupedSequence()
		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		events := InitialResetEvents(cursor, &iterations, &baseline)
		assert.Equal(t, []ResetKind{ResetKindSong, ResetKindGroupStart, ResetKindGroupLoop, ResetKindPartStart, ResetKindPartLoop}, kinds(events))
	})

	t.Run("song, two nested groups, and part fires only one GroupLoop/GroupStart pair", func(t *testing.T) {
		// There's only one groupStart/groupLoop channel/note configured each
		// — no notion of depth — so entering two nested groups at once must
		// not produce two separate NoteOn events on the same channel/note.
		sequence, cursor := NestedGroupsSequence()
		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(sequence.Arrangement, &iterations)
		baseline := make(playstate.Iterations, len(iterations))
		maps.Copy(baseline, iterations)
		events := InitialResetEvents(cursor, &iterations, &baseline)
		assert.Equal(t, []ResetKind{ResetKindSong, ResetKindGroupStart, ResetKindGroupLoop, ResetKindPartStart, ResetKindPartLoop}, kinds(events))
	})
}
