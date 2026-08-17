package beats

import (
	"slices"
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
	ResetKindGroupStart
	ResetKindGroupLoop
	ResetKindPartStart
	ResetKindPartLoop
)

type ResetEvent struct {
	Kind ResetKind
}

// computeResetEvents inspects the cursor and Iterations map left by the
// most recent PlayMove and reports which Song/Group/Part start/loop events
// fired. Only meaningful on a beat where the keyline actually wrapped
// (callers must gate on that themselves, see AdvancePlayState) — a bare
// baseline check can't tell "just became true this beat" from "has been
// resting true for many beats already".
//
// Climbs from the leaf's immediate parent toward root. At each level, the
// cursor must have descended into it via its actual first child (Nodes[0])
// for that level to be implicated: MoveToSibling only ever targets
// parent.Nodes[index+1] for index >= 0, so an ordinary sibling move can
// never land on Nodes[0] — only a genuine loop-back or a fresh descent into
// a newly-entered subtree does, and both recurse through Nodes[0] at every
// level below (MoveToFirstChild). The climb continues past i==0 (root):
// reaching root with the chain unbroken is what a genuine Song restart
// looks like, checked via root's own Nodes[0]/child baseline rather than
// root's own value, which (unlike every other node) never returns to 0
// after its first loop — its target is fixed at 1 by InitRoot.
func computeResetEvents(cursor arrangement.ArrCursor, iterations *playstate.Iterations, baseline *playstate.Iterations) []ResetEvent {

	var events []ResetEvent
	var highestZeroIterGroup *arrangement.Arrangement // shallowest still-fresh ancestor group seen so far

	currentPart := cursor[len(cursor)-1]
	events = append(events, ResetEvent{Kind: ResetKindPartLoop})
	if atBaseline(currentPart, iterations, baseline) {
		events = append(events, ResetEvent{Kind: ResetKindPartStart})
	}

	for i := len(cursor) - 2; i >= 0; i-- {
		groupNode := cursor[i]
		child := cursor[i+1]
		if child != groupNode.Nodes[0] || !atBaseline(child, iterations, baseline) {
			break
		}
		if i == 0 {
			events = append(events, ResetEvent{Kind: ResetKindSong})
			break
		}
		if (*iterations)[groupNode] == 0 {
			highestZeroIterGroup = groupNode
		} else {
			// This node's own counter just incremented — it's the
			// local-loop trigger, nothing above it changed. A deeper fresh
			// node, if any, still gets GroupStart, collapsed onto this
			// same outer node since there's only one groupStart/groupLoop
			// channel/note, no notion of depth.
			events = append(events, ResetEvent{Kind: ResetKindGroupLoop})
			if highestZeroIterGroup != nil {
				events = append(events, ResetEvent{Kind: ResetKindGroupStart})
			}
			highestZeroIterGroup = nil
			break
		}
	}

	if highestZeroIterGroup != nil {
		events = append(events, ResetEvent{Kind: ResetKindGroupLoop}, ResetEvent{Kind: ResetKindGroupStart})
	}

	slices.SortFunc(events, func(a, b ResetEvent) int {
		return int(int(a.Kind) - int(b.Kind))
	})
	return events
}

// atBaseline reports whether node's current Iterations value is its own
// "just (re-)entered, no lap completed yet" resting value.
//
// One comparison covers both Group and Part: Baseline starts as a full copy
// of the initial Iterations map, and the only place it's written afterward
// is AdvancePlayState's leaf-entry site — so a group's Baseline entry never
// changes and stays 0 permanently (Group has no KeepCycles-equivalent
// escape hatch, and ResetIterations/BuildIterationsMap both use 0 for
// groups unconditionally). For a Part, this is baseline[node] rather than
// the fixed Section.StartCycles constant, since a KeepCycles part's resting
// value can diverge from it.
func atBaseline(node *arrangement.Arrangement, iterations *playstate.Iterations, baseline *playstate.Iterations) bool {
	return (*iterations)[node] == (*baseline)[node]
}

// InitialResetEvents builds the "everything just started fresh" event set
// for the very first Play press (ui.go's Start()), which never goes through
// AdvancePlayState/PlayMove. Every ancestor is freshly descended (via
// Nodes[0] throughout) and every node sits at its BuildIterationsMap
// baseline, so computeResetEvents' normal climb reports the whole cascade.
func InitialResetEvents(cursor arrangement.ArrCursor, iterations *playstate.Iterations, baseline *playstate.Iterations) []ResetEvent {
	return computeResetEvents(cursor, iterations, baseline)
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
