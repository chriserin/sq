package beats

import (
	"time"

	"github.com/chriserin/sq/internal/arrangement"
	"github.com/chriserin/sq/internal/config"
	"github.com/chriserin/sq/internal/playstate"
	"github.com/chriserin/sq/internal/seqmidi"
	midi "gitlab.com/gomidi/midi/v2"
)

type ResetKind uint8

const (
	ResetKindSong ResetKind = iota
	ResetKindPartStart
	ResetKindPartLoop
	ResetKindGroupStart
	ResetKindGroupLoop
)

type ResetEvent struct {
	Kind ResetKind
}

// diffResetEvents compares the arrangement cursor and Iterations map before
// and after AdvancePlayState mutates them, to determine which song/part/group
// start or loop-around events happened this beat.
//
// Node identity (pointer comparison), not cursor index or raw Iterations
// value, is what's sound here: PlayMove can change the cursor's length (not
// just its tail) in a single call. Node identity is still comparable
// depth-by-depth though (every node has exactly one parent, so the same
// pointer is always at the same depth in both cursors), which is what the
// positional walk below relies on.
//
// This cascades deliberately: once any level (Song, or a Group) has been
// determined to be a fresh entry or a genuine loop-back, every level below it
// is *always* reported as a fresh "Start", even if that deeper node happens
// to be pointer-identical to what was already playing (which only happens
// when a level has just one child, e.g. a single-part song or a group with
// one part in it). Concretely: repeating the song must repeat every part
// inside it too, even a song with just one part in it.
//
// "Loop" and "Start" are not mutually exclusive: `*Loop` is the broader
// signal — "this node's content just began a fresh pass, for any reason" —
// and fires every time `*Start` does, plus on a purely local repeat with
// nothing above it also changing. `*Start` is the narrower "this was
// specifically a fresh entry" subset. A node repeating on its own (no
// ancestor involved) only fires `*Loop`; `*Start` never fires without
// `*Loop` also firing alongside it, including the very first entry.
//
// `forcedSongRestart` covers PlayMove's cursor.IsRoot() fallback (see
// AdvancePlayState) — a rare case where a restart happens without root's own
// Iterations counter changing at all, so the loop below can't detect it on
// its own; treating it as an already-started cascade handles it uniformly.
func diffResetEvents(preCursor arrangement.ArrCursor, wrapped bool, postCursor arrangement.ArrCursor, iterations *playstate.Iterations, preIterations playstate.Iterations, forcedSongRestart bool) []ResetEvent {
	var events []ResetEvent
	cascading := false
	groupLoopEmitted := false
	groupStartEmitted := false

	if forcedSongRestart {
		events = append(events, ResetEvent{Kind: ResetKindSong})
		cascading = true
	}

	ancestorDepth := len(postCursor) - 1
	for i := 0; i < ancestorDepth; i++ {
		n := postCursor[i]
		samePosition := !cascading && i < len(preCursor)-1 && preCursor[i] == n

		switch {
		case samePosition && (*iterations)[n] > preIterations[n]:
			// Strictly greater, not just "changed": PlayMove's plain
			// sibling-move branch (an ordinary child transition, unrelated to
			// this ancestor) also calls Iterations.ResetIterations on the new
			// cursor, which can zero this ancestor's counter as a side effect
			// whenever it happens to already equal its own target — a value
			// change with no genuine lap behind it. The only place an
			// ancestor's counter is ever incremented is the "did this node
			// loop back" site in PlayMove, so ">" (not "!=") is the precise
			// signal for a real lap; resets only ever move a value down to 0.
			// This is a purely local repeat (no ancestor above n changed to
			// reach it), so only *Loop fires here, never *Start.
			if i == 0 {
				events = append(events, ResetEvent{Kind: ResetKindSong})
			} else if !groupLoopEmitted {
				events = append(events, ResetEvent{Kind: ResetKindGroupLoop})
				groupLoopEmitted = true
			}
			cascading = true
		case samePosition:
			// Unchanged at this depth — no event yet, keep checking deeper
			// levels; this node might still be abandoned or start cascading
			// further down.
		default:
			// Already cascading (an ancestor above already started/looped),
			// or this depth's node simply differs from before — either way a
			// fresh entry, which is always also a "content began a fresh
			// pass" event, so both GroupLoop and GroupStart fire together.
			// Root itself (i == 0) never gets either — its own event, if
			// any, is always the "Song" case above (or forcedSongRestart).
			// Groups can nest inside groups, and the cascade can run through
			// several levels in one beat (e.g. entering a group-of-groups
			// from a plain sibling move) — but there's only one
			// groupStart/groupLoop channel/note configured each, with no
			// notion of "depth", so each fires at most once per beat;
			// deeper levels are part of the same event, not additional ones.
			if i > 0 {
				if !groupLoopEmitted {
					events = append(events, ResetEvent{Kind: ResetKindGroupLoop})
					groupLoopEmitted = true
				}
				if !groupStartEmitted {
					events = append(events, ResetEvent{Kind: ResetKindGroupStart})
					groupStartEmitted = true
				}
			}
			cascading = true
		}
	}

	preLeaf := preCursor[len(preCursor)-1]
	postLeaf := postCursor[len(postCursor)-1]
	partStarted := cascading || postLeaf != preLeaf
	if wrapped {
		// wrapped is a necessary condition of partStarted (AdvancePlayState
		// only calls PlayMove — the only way partStarted can be true — on a
		// beat where the keyline just wrapped), so this always fires
		// alongside PartStart below, plus on its own for a purely local
		// repeat.
		events = append(events, ResetEvent{Kind: ResetKindPartLoop})
	}
	if partStarted {
		events = append(events, ResetEvent{Kind: ResetKindPartStart})
	}

	return events
}

// InitialResetEvents builds the "everything just started fresh" event set
// for the very first Play press (ui.go's Start()), which never goes through
// AdvancePlayState/PlayMove at all. The very first entry is still a "content
// began a fresh pass" event, same as diffResetEvents' cascade case, so each
// Start is paired with its Loop counterpart.
func InitialResetEvents(cursor arrangement.ArrCursor) []ResetEvent {
	events := []ResetEvent{{Kind: ResetKindSong}}
	// One GroupLoop/GroupStart pair at most, regardless of how many nested
	// group levels sit between root and the leaf — same reasoning as
	// diffResetEvents: there's only one groupStart/groupLoop channel/note
	// each, with no notion of depth.
	if len(cursor) > 2 {
		events = append(events, ResetEvent{Kind: ResetKindGroupLoop}, ResetEvent{Kind: ResetKindGroupStart})
	}
	return append(events, ResetEvent{Kind: ResetKindPartLoop}, ResetEvent{Kind: ResetKindPartStart})
}

func EmitResets(mc *seqmidi.MidiConnection, events []ResetEvent) {
	for _, ev := range events {
		var m config.ResetMapping
		switch ev.Kind {
		case ResetKindSong:
			m = config.SongResetMapping
		case ResetKindPartStart:
			m = config.PartStartResetMapping
		case ResetKindPartLoop:
			m = config.PartLoopResetMapping
		case ResetKindGroupStart:
			m = config.GroupStartResetMapping
		case ResetKindGroupLoop:
			m = config.GroupLoopResetMapping
		}
		if m.Channel == 0 {
			continue // unconfigured — e.g. no partLoop set, so PartLoop events never fire
		}
		sendResetPulse(mc, m.Channel-1, m.Note)
	}
}

const resetHoldDuration = 10 * time.Millisecond

func sendResetPulse(mc *seqmidi.MidiConnection, channel, note uint8) {
	if err := mc.SendReset(midi.NoteOn(channel, note, 100)); err != nil {
		return
	}
	time.AfterFunc(resetHoldDuration, func() {
		mc.SendReset(midi.NoteOff(channel, note))
	})
}
