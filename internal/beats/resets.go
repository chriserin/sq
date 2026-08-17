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
	// Node is the specific Part leaf or Group node that triggered this
	// event, used to look up a Start/Loop range note (see reset_pool.go).
	// Always nil for ResetKindSong (Song doesn't participate in the pools).
	Node *arrangement.Arrangement
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
//
// Every node genuinely implicated in the cascade gets its own event (not
// collapsed onto a single outermost node), so a pool assignment (see
// reset_pool.go) can hand each one its own distinct note; EmitResets is
// responsible for still sending only one physical NoteOn per Kind on the
// scalar mappings.
func computeResetEvents(cursor arrangement.ArrCursor, iterations *playstate.Iterations, baseline *playstate.Iterations) []ResetEvent {
	var events []ResetEvent

	currentPart := cursor[len(cursor)-1]
	events = append(events, ResetEvent{Kind: ResetKindPartLoop, Node: currentPart})
	if atBaseline(currentPart, iterations, baseline) {
		events = append(events, ResetEvent{Kind: ResetKindPartStart, Node: currentPart})
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
			// Fresh entry at this level, possibly part of a cascade from
			// further out -- gets its own pair and the climb continues.
			events = append(events, ResetEvent{Kind: ResetKindGroupLoop, Node: groupNode}, ResetEvent{Kind: ResetKindGroupStart, Node: groupNode})
		} else {
			// This node's own counter just incremented: the local-loop
			// trigger. Nothing above it changed, so the climb stops here.
			events = append(events, ResetEvent{Kind: ResetKindGroupLoop, Node: groupNode})
			break
		}
	}

	slices.SortFunc(events, func(a, b ResetEvent) int {
		return int(a.Kind) - int(b.Kind)
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

// resetSend is one physical NoteOn/NoteOff pulse to send: a 0-indexed
// channel/note pair, independent of whether it came from a scalar mapping
// or a pool note.
type resetSend struct {
	channel uint8
	note    uint8
}

// resolveResetSends decides which physical pulses events should produce:
// the scalar mapping for each distinct Kind present (at most one send per
// Kind per beat, regardless of how many nodes cascaded — computeResetEvents
// no longer collapses nested groups onto one node, so this dedup happens
// here instead), plus, for every event with a Node, that node's
// individually-assigned pool note (startPoolNotes/loopPoolNotes,
// precomputed once per session — see reset_pool.go's ArrangementPoolNotes).
//
// Kept separate from EmitResets, and pure/side-effect-free, so the dedup
// and pool-lookup logic is directly unit-testable: SendReset only sends
// through an armed device (internal/seqmidi's outDevices, unexported), so
// there's no way to observe EmitResets' actual sends from this package
// without one.
func resolveResetSends(events []ResetEvent, startPoolNotes, loopPoolNotes map[*arrangement.Arrangement]uint8) []resetSend {
	var sends []resetSend
	var scalarSent [5]bool // indexed by ResetKind

	for _, ev := range events {
		if !scalarSent[ev.Kind] {
			scalarSent[ev.Kind] = true
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
			if m.Channel != 0 {
				sends = append(sends, resetSend{channel: m.Channel - 1, note: m.Note})
			}
		}

		if ev.Node == nil {
			continue // Song never participates in the pools
		}
		var poolNotes map[*arrangement.Arrangement]uint8
		var channel uint8
		switch ev.Kind {
		case ResetKindPartStart, ResetKindGroupStart:
			poolNotes, channel = startPoolNotes, config.StartResetRange.Channel
		case ResetKindPartLoop, ResetKindGroupLoop:
			poolNotes, channel = loopPoolNotes, config.LoopResetRange.Channel
		}
		if channel == 0 {
			continue // range unconfigured
		}
		if note, ok := poolNotes[ev.Node]; ok {
			sends = append(sends, resetSend{channel: channel - 1, note: note})
		}
	}
	return sends
}

func EmitResets(mc *seqmidi.MidiConnection, events []ResetEvent, startPoolNotes, loopPoolNotes map[*arrangement.Arrangement]uint8) {
	for _, s := range resolveResetSends(events, startPoolNotes, loopPoolNotes) {
		sendResetPulse(mc, s.channel, s.note)
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
