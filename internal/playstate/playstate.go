package playstate

import (
	"maps"
	"time"

	"github.com/chriserin/sq/internal/arrangement"
	"github.com/chriserin/sq/internal/grid"
)

type Iterations map[*arrangement.Arrangement]int

type PlayState struct {
	Playing            bool
	AllowAdvance       bool
	HasSolo            bool
	RecordPreRollBeats uint8
	PlayMode           PlayMode
	LoopMode           LoopMode
	BeatTime           time.Time
	LineStates         []LineState
	Iterations         *Iterations
	Baseline           *Iterations
	LoopedArrangement  *arrangement.Arrangement
	BoundedLoop        BoundedLoop
}

type BoundedLoop struct {
	Active     bool
	LeftBound  uint8
	RightBound uint8
}

func (b *BoundedLoop) ContractRight() {
	if b.RightBound > b.LeftBound {
		b.RightBound--
	}
}

func (b *BoundedLoop) ContractLeft() {
	if b.LeftBound < b.RightBound {
		b.LeftBound++
	}
}

func (b *BoundedLoop) ExpandRight(beats uint8) {
	if b.RightBound < beats {
		b.RightBound++
	}
}

func (b *BoundedLoop) ExpandLeft() {
	if b.LeftBound > 0 {
		b.LeftBound--
	}
}

func (i *Iterations) IsFull(cursor *arrangement.ArrCursor) bool {
	for index, node := range *cursor {
		if index == 0 {
			continue
		}
		if node.IsGroup() {
			count := (*i)[node]
			if node.Iterations > count+1 {
				return false
			}
		} else {
			section := node.Section
			count := (*i)[node]
			if section.Cycles+section.StartCycles > count {
				return false
			}
		}
	}
	return true
}

func (i *Iterations) ResetIterations(cursor arrangement.ArrCursor) {
	for _, arrRef := range cursor {
		if (*i)[arrRef] == arrRef.Iterations {
			(*i)[arrRef] = 0
		}
	}
}

type PlayMode uint8

const (
	PlayStandard PlayMode = iota
	PlayReceiver
)

type LoopMode uint8

const (
	OneTimeWholeSequence LoopMode = iota
	LoopWholeSequence
	LoopPart
	LoopOverlay
)

type GroupPlayState uint8

const (
	PlayStatePlay GroupPlayState = iota
	PlayStateMute
	PlayStateSolo
)

type LineState struct {
	Index               uint8
	CurrentBeat         uint8
	ResetLocation       uint8
	ResetActionLocation uint8
	ResetAction         grid.Action
	GroupPlayState      GroupPlayState
	Direction           int8
	ResetDirection      int8
}

func (ls LineState) IsMuted() bool {
	return ls.GroupPlayState == PlayStateMute
}

func (ls LineState) IsSolo() bool {
	return ls.GroupPlayState == PlayStateSolo
}

func (ls LineState) GridKey() grid.GridKey {
	return grid.GridKey{Line: ls.Index, Beat: ls.CurrentBeat}
}

func InitLineStates(lines int, previousPlayState []LineState, startBeat uint8) []LineState {
	linestates := make([]LineState, 0, lines)

	for i := range uint8(lines) {
		var previousGroupPlayState = PlayStatePlay
		if len(previousPlayState) > int(i) {
			previousState := previousPlayState[i]
			previousGroupPlayState = previousState.GroupPlayState
		}

		linestates = append(linestates, InitLineState(previousGroupPlayState, i, startBeat))
	}
	return linestates
}

func InitLineState(previousGroupPlayState GroupPlayState, index uint8, startBeat uint8) LineState {
	return LineState{
		Index:               index,
		CurrentBeat:         startBeat,
		Direction:           1,
		ResetDirection:      1,
		ResetLocation:       0,
		ResetActionLocation: 0,
		ResetAction:         grid.ActionNothing,
		GroupPlayState:      previousGroupPlayState,
	}
}

func (ls *LineState) AdvancePlayState(combinedPattern grid.Pattern, lineIndex int, beats uint8, lineStates []LineState, boundedLoop BoundedLoop, loopMode LoopMode) bool {

	advancedBeat := int8(ls.CurrentBeat) + ls.Direction
	currentState := *ls

	startBound := uint8(0)
	endBound := uint8(beats)

	if boundedLoop.Active && loopMode == LoopOverlay {
		startBound = uint8(boundedLoop.LeftBound)
		endBound = uint8(boundedLoop.RightBound) + 1
	}

	if advancedBeat >= int8(endBound) || advancedBeat < int8(startBound) {
		// reset locations should be 1 time use.  Reset back to 0.
		if ls.ResetLocation != 0 && combinedPattern[grid.GK(uint8(lineIndex), currentState.ResetActionLocation)].Action == currentState.ResetAction {
			ls.CurrentBeat = max(currentState.ResetLocation, startBound)
			advancedBeat = int8(currentState.ResetLocation)
		} else {
			ls.CurrentBeat = startBound
			advancedBeat = int8(startBound)
		}
		ls.Direction = currentState.ResetDirection
		ls.ResetLocation = 0
	} else {
		ls.CurrentBeat = uint8(advancedBeat)
	}

	switch combinedPattern[grid.GK(uint8(lineIndex), uint8(advancedBeat))].Action {
	case grid.ActionNothing:
		return true
	case grid.ActionLineReset:
		ls.CurrentBeat = startBound
	case grid.ActionLineReverse:
		ls.CurrentBeat = uint8(max(advancedBeat-2, int8(startBound)))
		ls.Direction = -1
		ls.ResetLocation = uint8(max(advancedBeat-1, int8(startBound)))
		ls.ResetActionLocation = uint8(advancedBeat)
		ls.ResetAction = grid.ActionLineReverse
	case grid.ActionLineBounce:
		ls.CurrentBeat = uint8(max(advancedBeat-1, int8(startBound)))
		ls.Direction = -1
	case grid.ActionLineSkipBeat:
		ls.AdvancePlayState(combinedPattern, lineIndex, beats, lineStates, boundedLoop, loopMode)
	case grid.ActionLineDelay:
		ls.CurrentBeat = uint8(max(advancedBeat-1, int8(startBound)))
	case grid.ActionLineResetAll:
		for i := range lineStates {
			lineStates[i].CurrentBeat = startBound
			lineStates[i].Direction = 1
			lineStates[i].ResetLocation = startBound
			lineStates[i].ResetDirection = 1
		}
		return false
	case grid.ActionLineBounceAll:
		for i := range lineStates {
			if i <= lineIndex {
				lineStates[i].CurrentBeat = uint8(max(lineStates[i].CurrentBeat-1, startBound))
			}
			lineStates[i].Direction = -1
		}
		return false
	}

	return true
}

func BuildIterationsMap(arr *arrangement.Arrangement, iterations *Iterations) {
	if arr.IsGroup() {
		(*iterations)[arr] = 0
	} else {
		(*iterations)[arr] = arr.Section.StartCycles
	}
	for _, node := range arr.Nodes {
		BuildIterationsMap(node, iterations)
	}
}

func Copy(playState PlayState) PlayState {
	newIterations := make(Iterations)
	maps.Copy(newIterations, *playState.Iterations)
	playState.Iterations = &newIterations

	newBaseline := make(Iterations)
	maps.Copy(newBaseline, *playState.Baseline)
	playState.Baseline = &newBaseline

	newLineStates := make([]LineState, len(playState.LineStates))
	copy(newLineStates, playState.LineStates)
	playState.LineStates = newLineStates

	return playState
}
