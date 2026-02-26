package beats

import (
	"context"
	"fmt"
	"maps"
	"math/rand"
	"os"
	"runtime/debug"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/chriserin/sq/internal/arrangement"
	"github.com/chriserin/sq/internal/config"
	"github.com/chriserin/sq/internal/grid"
	"github.com/chriserin/sq/internal/notereg"
	"github.com/chriserin/sq/internal/overlays"
	"github.com/chriserin/sq/internal/playstate"
	"github.com/chriserin/sq/internal/seqmidi"
	"github.com/chriserin/sq/internal/sequence"
	midi "gitlab.com/gomidi/midi/v2"
)

type ModelMsg struct {
	PlayState playstate.PlayState
	Sequence  sequence.Sequence
	Cursor    arrangement.ArrCursor
}

type ModelPlayedMsg struct {
	PerformStop bool
	PlayState   playstate.PlayState
	Cursor      arrangement.ArrCursor
}

type AnticipatoryStop struct{}

type BeatsLooper struct {
	ClockChannel  chan ClockMsg
	BeatChannel   chan BeatMsg
	UpdateChannel chan ModelMsg
	PlayQueue     chan seqmidi.Message
	ErrChan       chan error
}

func InitBeatsLooper() BeatsLooper {
	beatChannel := make(chan BeatMsg)
	clockChannel := make(chan ClockMsg)
	updateChannel := make(chan ModelMsg)
	playQueue := make(chan seqmidi.Message)
	errChan := make(chan error)

	return BeatsLooper{
		ClockChannel:  clockChannel,
		BeatChannel:   beatChannel,
		UpdateChannel: updateChannel,
		PlayQueue:     playQueue,
		ErrChan:       errChan,
	}
}

func (bl BeatsLooper) Loop(sendFn func(tea.Msg), midiConn *seqmidi.MidiConnection, ctx context.Context) {
	// NOTE: Create a log file for debug information
	logFile, _ := tea.LogToFile("debug.log", "debug")

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Recovered in beats loop from panic: %v\n", r)
				debug.PrintStack()
			}
		}()
		var playState playstate.PlayState
		var definition sequence.Sequence
		var cursor arrangement.ArrCursor
		for {
			if !playState.Playing {
				// NOTE: Wait for a model update that puts us into a playing state.
				select {
				case modelMsg := <-bl.UpdateChannel:
					copiedPlayState := playstate.Copy(modelMsg.PlayState)
					playState = copiedPlayState
					definition = modelMsg.Sequence
					cursor = modelMsg.Cursor
				case <-ctx.Done():
					return
				}
			} else {
				// NOTE: In a playing state, respond to beat messages
				select {
				case modelMsg := <-bl.UpdateChannel:
					playState = modelMsg.PlayState
					definition = modelMsg.Sequence
					cursor = modelMsg.Cursor
				case <-bl.ClockChannel:
					bl.PlayQueue <- seqmidi.Message{Msg: midi.TimingClock(), Delay: 0}
				case BeatMsg := <-bl.BeatChannel:
					bl.Beat(BeatMsg, playState, definition, cursor, sendFn)
				case <-ctx.Done():
					return
				case err := <-bl.ErrChan:
					_, logErr := fmt.Fprintf(logFile, "Error: %v", err)
					if logErr != nil {
						fmt.Println("An error occurred while writing the original error to the log file", err, logErr)
					}
				}
			}
		}
	}()
	go func() {
		for {
			defer func() {
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "Recovered in beats loop -> midiConn from panic: %v\n", r)
					debug.PrintStack()
				}
			}()
			select {
			case midiMessage := <-bl.PlayQueue:
				midiConn.Send(midiMessage)
			case <-ctx.Done():
				return
			}

		}
	}()
}

func IsDone(playState playstate.PlayState, currentNode *arrangement.Arrangement, currentSection arrangement.SongSection, cursor *arrangement.ArrCursor) bool {
	return playState.LoopedArrangement != currentNode &&
		currentSection.Cycles+currentSection.StartCycles <= (*playState.Iterations)[currentNode] &&
		!(cursor.AllLastSiblings() && playState.Iterations.IsFull(cursor) && playState.PlayMode == playstate.PlayReceiver && playState.LoopMode != playstate.LoopWholeSequence)
}

func (bl BeatsLooper) Beat(msg BeatMsg, playState playstate.PlayState, definition sequence.Sequence, cursor arrangement.ArrCursor, sendFn func(tea.Msg)) {
	if playState.Playing {
		AdvancePlayState(&playState, definition, &cursor)
	}

	if !playState.Playing {
		sendFn(ModelPlayedMsg{PerformStop: true, PlayState: playState, Cursor: cursor})
		return
	} else {
		bl.PlaySequence(&playState, definition, cursor, msg)
		go func() {
			sendFn(ModelPlayedMsg{PlayState: playState, Cursor: cursor})
		}()
		go func() {
			bl.UpdateChannel <- ModelMsg{PlayState: playState, Sequence: definition, Cursor: cursor}
		}()
	}

	// NOTE: Looking at future state to determine if we need to prevent sending the
	// receiver a final pulse
	copiedPlayState := playstate.Copy(playState)
	copiedCursor := make(arrangement.ArrCursor, len(cursor))
	copy(copiedCursor, cursor)
	AdvancePlayState(&copiedPlayState, definition, &copiedCursor)
	if !copiedPlayState.Playing {
		sendFn(AnticipatoryStop{})
	}
}

func (bl BeatsLooper) PlaySequence(playState *playstate.PlayState, definition sequence.Sequence, cursor arrangement.ArrCursor, msg BeatMsg) {

	currentNode := cursor[len(cursor)-1]
	currentSection := cursor[len(cursor)-1].Section
	var partID int
	var currentCycles int
	var currentPart arrangement.Part
	var playingOverlay *overlays.Overlay

	partID = currentSection.Part
	currentPart = (*definition.Parts)[partID]
	currentCycles = (*playState.Iterations)[currentNode]
	playingOverlay = currentPart.Overlays.HighestMatchingOverlay(currentCycles)

	noteLineStates := make([]playstate.LineState, 0, len(playState.LineStates))
	metaLineStates := make([]playstate.LineState, 0, len(playState.LineStates))
	for i, ls := range playState.LineStates {
		if definition.Lines[i].MsgType == grid.MessageTypeNote {
			noteLineStates = append(noteLineStates, ls)
		} else {
			metaLineStates = append(metaLineStates, ls)
		}
	}

	// Play the CC/PC Messages
	gridKeys := make([]grid.GridKey, 0, len(playState.LineStates))
	CurrentBeatGridKeys(&gridKeys, metaLineStates, playState.HasSolo)

	pattern := make(grid.Pattern)
	playingOverlay.CurrentBeatOverlayPattern(&pattern, currentCycles, gridKeys)

	bl.PlayBeat(msg.Interval, pattern, definition)

	// Play the Note Messages
	gridKeys = make([]grid.GridKey, 0, len(playState.LineStates))
	CurrentBeatGridKeys(&gridKeys, noteLineStates, playState.HasSolo)

	pattern = make(grid.Pattern)
	playingOverlay.CurrentBeatOverlayPattern(&pattern, currentCycles, gridKeys)

	bl.PlayBeat(msg.Interval, pattern, definition)

	if !playState.AllowAdvance {
		playState.AllowAdvance = true
	}
}

func AdvancePlayState(playState *playstate.PlayState, definition sequence.Sequence, cursor *arrangement.ArrCursor) {
	currentNode := (*cursor)[len(*cursor)-1]
	currentSection := (*cursor)[len(*cursor)-1].Section
	partID := currentSection.Part
	currentPart := (*definition.Parts)[partID]
	currentCycles := (*playState.Iterations)[currentNode]
	playingOverlay := currentPart.Overlays.HighestMatchingOverlay(currentCycles)

	if playState.Playing {
		// NOTE: Only advance if we've already played the first beat.
		if playState.AllowAdvance {
			advanceCurrentBeat(currentCycles, *playingOverlay, playState.LineStates, currentPart.Beats, playState.BoundedLoop, playState.LoopMode)
			advanceKeyCycle(definition.Keyline, playState.LineStates, playState.LoopMode, currentNode, playState.Iterations)
			if IsDone(*playState, currentNode, currentSection, cursor) && playState.LoopMode != playstate.LoopOverlay {
				if PlayMove(cursor, playState.Iterations, playState.LoopedArrangement) || playState.PlayMode == playstate.PlayReceiver {
					currentSection = (*cursor)[len(*cursor)-1].Section
					currentNode = (*cursor)[len(*cursor)-1]
					if !currentSection.KeepCycles {
						(*playState.Iterations)[currentNode] = currentSection.StartCycles
					}
					playState.LineStates = playstate.InitLineStates(len(definition.Lines), playState.LineStates, uint8((*cursor)[len(*cursor)-1].Section.StartBeat))
				} else {
					playState.Playing = false
					return
				}
			}
		}
	}

}

func CurrentBeatGridKeys(gridKeys *[]grid.GridKey, lineStates []playstate.LineState, hasSolo bool) {
	for _, linestate := range lineStates {
		if linestate.IsSolo() || (!linestate.IsMuted() && !hasSolo) {
			*gridKeys = append(*gridKeys, linestate.GridKey())
		}
	}
}

func advanceCurrentBeat(keyCycles int, playingOverlay overlays.Overlay, lineStates []playstate.LineState, partBeats uint8, boundedLoop playstate.BoundedLoop, loopMode playstate.LoopMode) {
	pattern := make(grid.Pattern)
	playingOverlay.CombineActionPattern(&pattern, keyCycles)
	for i := range lineStates {
		doContinue := lineStates[i].AdvancePlayState(pattern, i, partBeats, lineStates, boundedLoop, loopMode)
		if !doContinue {
			break
		}
	}
}

func advanceKeyCycle(keyline uint8, lineStates []playstate.LineState, loopMode playstate.LoopMode, node *arrangement.Arrangement, iterations *playstate.Iterations) {
	if lineStates[keyline].CurrentBeat == 0 && loopMode != playstate.LoopOverlay {
		(*iterations)[node]++
	}
}

func PlayMove(cursor *arrangement.ArrCursor, iterations *playstate.Iterations, loopNode *arrangement.Arrangement) bool {
	if cursor.IsRoot() {
		cursor.MoveNext()
		return false
	} else if cursor.IsLastSibling() {
		(*iterations)[cursor.GetParentNode()]++
		hasParentIterations := (*iterations)[cursor.GetParentNode()] < cursor.GetParentNode().Iterations
		if hasParentIterations || loopNode == cursor.GetParentNode() {
			cursor.MoveToFirstSibling()
			if cursor.GetCurrentNode().IsGroup() {
				cursor.MoveNext()
			}
		} else {
			iterations.ResetIterations(*cursor)
			cursor.Up()
			return PlayMove(cursor, iterations, loopNode)
		}
	} else {
		cursor.MoveToSibling()
		iterations.ResetIterations(*cursor)
	}
	return true
}

func (bl BeatsLooper) PlayBeat(beatInterval time.Duration, pattern grid.Pattern, definition sequence.Sequence) {
	lines := definition.Lines

	keys := maps.Keys(pattern)
	sortedKeys := slices.SortedFunc(keys, grid.Compare)

	for _, gridKey := range sortedKeys {
		note := pattern[gridKey]
		if note.Action != grid.ActionNothing {
			continue
		}
		line := lines[gridKey.Line]
		if note.Ratchets.Length > 0 {
			bl.ProcessRatchets(note, beatInterval, line, definition)
		} else if note != grid.ZeroNote {
			accents := definition.Accents

			delay := Delay(note.WaitIndex, beatInterval)
			gateLength := GateLength(note.GateIndex, beatInterval)

			switch line.MsgType {
			case grid.MessageTypeNote:
				onMessage, offMessage := NoteMessages(
					line,
					uint8(definition.Accents.Data[note.AccentIndex]),
					gateLength,
					accents.Target,
					delay,
				)
				bl.PlayOnMessage(onMessage)
				bl.PlayOffMessage(offMessage)
			case grid.MessageTypeCc:
				ccMessage := CCMessage(line, note, accents.Data, delay, true, definition.Instrument)

				bl.PlayMessage(ccMessage.delay, ccMessage.MidiMessage())
			case grid.MessageTypeProgramChange:
				pcMessage := PCMessage(line, note, accents.Data, delay, true, definition.Instrument)
				bl.PlayMessage(pcMessage.delay, pcMessage.MidiMessage())
			}
		}
	}
}

func (bl BeatsLooper) ProcessRatchets(note grid.Note, beatInterval time.Duration, line grid.LineDefinition, definition sequence.Sequence) {
	ratchetInterval := note.Ratchets.Interval(beatInterval)
	for i := range note.Ratchets.Length + 1 {
		if note.Ratchets.HitAt(i) {
			shortGateLength := 20 * time.Millisecond
			ratchetDelay := time.Duration(i) * ratchetInterval
			onMessage, offMessage := NoteMessages(line, uint8(definition.Accents.Data[note.AccentIndex]), shortGateLength, definition.Accents.Target, ratchetDelay)
			bl.PlayOnMessage(onMessage)
			bl.PlayOffMessage(offMessage)
		}
	}
}

func (bl BeatsLooper) PlayMessage(delay time.Duration, message midi.Message) {
	bl.PlayQueue <- seqmidi.Message{Msg: message, Delay: delay}
}

func (bl BeatsLooper) PlayOnMessage(nm NoteMsg) {
	key := notereg.GetKey(nm.GetOnMidi())
	if notereg.HasKey(key) {
		bl.PlayQueue <- seqmidi.Message{Msg: nm.GetOffMidi(), Delay: nm.delay}
		bl.PlayQueue <- seqmidi.Message{Msg: nm.GetOnMidi(), Delay: nm.delay}
	} else {
		bl.PlayQueue <- seqmidi.Message{Msg: nm.GetOnMidi(), Delay: nm.delay}
	}
}

func (bl BeatsLooper) PlayOffMessage(nm NoteMsg) {
	bl.PlayQueue <- seqmidi.Message{Msg: nm.GetOffMidi(), Delay: nm.delay}
}

type BeatMsg struct {
	Interval time.Duration
}

type ClockMsg struct{}

func NoteMessages(l grid.LineDefinition, accentValue uint8, gateLength time.Duration, accentTarget sequence.AccentTarget, delay time.Duration) (NoteMsg, NoteMsg) {
	var noteValue uint8
	var velocityValue uint8

	switch accentTarget {
	case sequence.AccentTargetNote:
		noteValue = l.Note + accentValue
		velocityValue = 96
	case sequence.AccentTargetVelocity:
		noteValue = l.Note
		velocityValue = accentValue
	}

	id := rand.Int()
	onMsg := NoteMsg{id: id, midiType: midi.NoteOnMsg, channel: l.Channel - 1, noteValue: noteValue, velocity: velocityValue, delay: delay}
	offMsg := NoteMsg{id: id, midiType: midi.NoteOffMsg, channel: l.Channel - 1, noteValue: noteValue, velocity: 0, delay: delay + gateLength}

	return onMsg, offMsg
}

func CCMessage(l grid.LineDefinition, note grid.Note, accents []config.Accent, delay time.Duration, includeDelay bool, instrument string) controlChangeMsg {
	if note.Action == grid.ActionSpecificValue {
		return controlChangeMsg{l.Channel - 1, l.Note, note.AccentIndex, delay}
	} else {
		cc, _ := config.FindCC(l.Note, instrument)
		var ccValue uint8
		if cc.UpperLimit == 1 && note.AccentIndex > 4 {
			ccValue = 0
		} else if cc.UpperLimit == 1 {
			ccValue = 1
		} else {
			ccValue = uint8((float32((len(accents))-int(note.AccentIndex)) / float32(len(accents)-1)) * float32(cc.UpperLimit))
		}

		return controlChangeMsg{l.Channel - 1, l.Note, ccValue, delay}
	}
}

func PCMessage(l grid.LineDefinition, note grid.Note, accents []config.Accent, delay time.Duration, includeDelay bool, instrument string) programChangeMsg {
	if note.Action == grid.ActionSpecificValue {
		return programChangeMsg{l.Channel - 1, note.AccentIndex, delay}
	} else {
		pcValue := uint8((float32((len(accents))-int(note.AccentIndex)) / float32(len(accents)-1)) * float32(127))
		return programChangeMsg{l.Channel - 1, pcValue, delay}
	}
}

type Delayable interface {
	Delay() time.Duration
}

type NoteMsg struct {
	channel   uint8
	noteValue uint8
	velocity  uint8
	midiType  midi.Type
	delay     time.Duration
	id        int
}

func (nm NoteMsg) Delay() time.Duration {
	return nm.delay
}

type programChangeMsg struct {
	channel uint8
	pcValue uint8
	delay   time.Duration
}

func (pcm programChangeMsg) MidiMessage() midi.Message {
	return midi.ProgramChange(pcm.channel, pcm.pcValue)
}

func (pcm programChangeMsg) Delay() time.Duration {
	return pcm.delay
}

type controlChangeMsg struct {
	channel uint8
	control uint8
	ccValue uint8
	delay   time.Duration
}

func (ccm controlChangeMsg) MidiMessage() midi.Message {
	return midi.ControlChange(ccm.channel, ccm.control, ccm.ccValue)
}

func (ccm controlChangeMsg) Delay() time.Duration {
	return ccm.delay
}

func (nm NoteMsg) GetKey() notereg.NoteRegKey {
	return notereg.NoteRegKey{
		Channel: nm.channel,
		Note:    nm.noteValue,
	}
}

func (nm NoteMsg) GetID() int {
	return nm.id
}

func (nm NoteMsg) GetOnMidi() midi.Message {
	return midi.NoteOn(nm.channel, nm.noteValue, nm.velocity)
}

func (nm NoteMsg) GetOffMidi() midi.Message {
	return midi.NoteOff(nm.channel, nm.noteValue)
}

func (nm NoteMsg) OffMessage() midi.Message {
	return midi.NoteOff(nm.channel, nm.noteValue)
}

func Delay(waitIndex uint8, beatInterval time.Duration) time.Duration {
	var delay time.Duration
	if waitIndex != 0 {
		delay = time.Duration((float64(config.WaitPercentages[waitIndex])) / float64(100) * float64(beatInterval))
	} else {
		delay = 0
	}
	return delay
}

func GateLength(gateIndex int16, beatInterval time.Duration) time.Duration {
	var delay time.Duration
	if gateIndex < 8 {
		var delay time.Duration
		var value = config.ShortGates[gateIndex].Value
		if value > 1 {
			delay = time.Duration(config.ShortGates[gateIndex].Value) * time.Millisecond
		} else {
			delay = time.Duration(config.ShortGates[gateIndex].Value * float32(beatInterval))
		}
		return delay
	} else if gateIndex >= 8 {
		shortGatesLen := int16(len(config.ShortGates))
		return time.Duration(float64(config.LongGates[gateIndex-shortGatesLen].Value) * float64(beatInterval))
	}
	return delay
}
