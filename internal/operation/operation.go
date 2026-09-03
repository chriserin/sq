// Package operation defines the types and values used to represent the operational state of the application.
package operation

import "slices"

type SequencerMode uint8

const (
	SeqModeAny SequencerMode = iota
	SeqModeLine
	SeqModeChord
	SeqModeMono
)

type VisualMode uint8

const (
	VisualNone VisualMode = iota
	VisualBlock
	VisualLine
)

type Focus uint8

// NOTE: Focus is necessary because the selection can be focused for a part operation
// at the same time the arrangement editor is focused.  `FocusOverlayKey` could be switched
// out for a selection indicator, but the pattern is to focus components that have their
// own key event responses.
const (
	FocusAny Focus = iota
	FocusGrid
	FocusOverlayKey
	FocusArrangementEditor
)

type Selection uint8

const (
	// Value used for matching mappings
	SelectAny Selection = iota
	SelectGrid
	// Definition Change
	SelectTempo
	SelectTempoSubdivision
	SelectSetupChannel
	SelectSetupMessageType
	SelectSetupValue
	SelectSetupOutput
	SelectAccentTarget
	SelectAccentStart
	SelectAccentEnd

	// Part Change
	SelectBeats

	// Arrangement Change
	SelectPart
	SelectChangePart
	SelectRenamePart
	SelectCycles
	SelectStartBeats
	SelectStartCycles

	// Note Change
	SelectRatchets
	SelectRatchetSpan
	SelectSpecificValue
	SelectTempoChangeValue
	SelectEuclideanHits

	// Program Level Operation
	SelectConfirmNew
	SelectConfirmQuit
	SelectConfirmReload
	SelectFileName
	SelectError
)

type PatternMode uint8

const (
	PatternAny PatternMode = iota
	PatternFill
	PatternAccent
	PatternGate
	PatternWait
	PatternRatchet
	PatternNoteAccent
	PatternNoteGate
	PatternNoteWait
	PatternNoteRatchet
)

type EveryMode uint8

const (
	EveryBeat EveryMode = iota
	EveryNote
)

var NumberSelections = []Selection{
	SelectBeats,
	SelectCycles,
	SelectStartBeats,
	SelectStartCycles,
	SelectSpecificValue,
	SelectTempoChangeValue,
	SelectTempo,
	SelectTempoSubdivision,
	SelectSetupChannel,
	SelectSetupValue,
	SelectAccentStart,
	SelectAccentEnd,
	SelectEuclideanHits,
}

func IsNumberSelection(sel Selection) bool {
	return slices.Contains(NumberSelections, sel)
}
