// Package grid provides the core data structures and functionality for the sequencer's
// pattern grid system. It defines notes, patterns, ratchets, line definitions, and
// various actions that can be performed on sequencer steps. The grid represents
// the fundamental building blocks for creating and manipulating musical sequences.
package grid

import (
	"fmt"
	"time"
)

type Action uint8

const (
	ActionNothing Action = iota
	ActionLineReset
	ActionLineResetAll
	ActionLineReverse
	// ActionLineReverseAll
	ActionLineSkipBeat
	ActionLineBounce
	ActionLineBounceAll
	ActionLineDelay
	ActionSpecificValue
)

func GK(line uint8, beat uint8) GridKey {
	return GridKey{
		Line: line,
		Beat: beat,
	}
}

type GridKey struct {
	Line uint8
	Beat uint8
}

func (gk GridKey) String() string {
	return fmt.Sprintf("Grid-%0.2d-%0.2d", gk.Line, gk.Beat)
}

func Compare(a GridKey, b GridKey) int {
	if a.Beat == b.Beat {
		return int(b.Line) - int(a.Line)
	} else {
		return int(a.Beat) - int(b.Beat)
	}
}

func CompareBeat(a GridKey, b GridKey) int {
	if a.Beat == b.Beat {
		return int(a.Line) - int(b.Line)
	} else {
		return int(b.Beat) - int(a.Beat)
	}
}
func BSearchBeatFunc(a GridKey, target uint8) int {
	return int(target) - int(a.Beat)
}

type Note struct {
	AccentIndex uint8
	Ratchets    Ratchet
	Action      Action
	WaitIndex   uint8
	GateIndex   int16
}

var ZeroNote = Note{}

func InitNote() Note {
	return Note{5, InitRatchet(), ActionNothing, 0, 0}
}

func InitActionNote(act Action) Note {
	return Note{0, InitRatchet(), act, 0, 0}
}

func (n Note) IncrementAccent(modifier int8, accentsLength uint8) Note {
	var newAccent = int8(n.AccentIndex) - modifier
	if newAccent >= 1 && newAccent < int8(accentsLength) {
		n.AccentIndex = uint8(newAccent)
	}
	return n
}

func (n Note) IncrementGate(modifier int8, gatesLength int) Note {
	var newGate = int16(n.GateIndex) + int16(modifier)
	if n.GateIndex == 0 && modifier == 8 {
		newGate = 7
	}
	if newGate >= 0 && int(newGate) < gatesLength {
		n.GateIndex = int16(newGate)
	} else if int(newGate) >= gatesLength {
		n.GateIndex = int16(gatesLength - 1)
	} else if newGate < 0 {
		n.GateIndex = 0
	}
	return n
}

func (n Note) IncrementWait(modifier int8) Note {
	var newWait = int8(n.WaitIndex) + modifier
	// TODO: remove hardcoded waits length
	waitslength := 8
	if newWait >= 0 && newWait < int8(waitslength) {
		n.WaitIndex = uint8(newWait)
	}
	return n
}

func (n Note) IncrementRatchet(modifier int8) Note {
	currentRatchet := n.Ratchets.Length
	// TODO: remove hardcoded ratchets length
	var ratchetsLength int8 = 8
	newRatchet := int8(currentRatchet) + modifier
	if n.AccentIndex > 0 && n.Action == ActionNothing && newRatchet < ratchetsLength && int8(currentRatchet)+modifier >= 0 {
		n.Ratchets.SetLength(uint8(int8(currentRatchet) + modifier))
	}
	return n
}

func (r *Ratchet) SetLength(length uint8) {
	r.Length = length

	r.SetRatchet(true, length)
	//Ensure no hits are set beyond the new length
	for i := range uint8(8) {
		if i > length {
			r.SetRatchet(false, i)
		}
	}
}

type Pattern map[GridKey]Note

type Ratchet struct {
	Hits   uint8
	Length uint8
	Span   uint8
}

func InitRatchet() Ratchet {
	return Ratchet{
		// We always start with one Ratchet enabled
		Hits: boolsToUint8([8]bool{true, false, false, false, false, false, false, false}),
	}
}

func boolsToUint8(bools [8]bool) uint8 {
	var result uint8
	for i := range 8 {
		if bools[i] {
			result = result | (1 << i)
		}
	}
	return result
}

func (r Ratchet) Interval(beatInterval time.Duration) time.Duration {
	return (beatInterval * time.Duration(r.GetSpan())) / time.Duration(r.Length+1)
}

func (r Ratchet) GetSpan() uint8 {
	return r.Span + 1
}

func (r *Ratchet) SetRatchet(value bool, index uint8) {
	if value {
		// (1 << 1) == 010
		// 001 | 010 == 011
		r.Hits = r.Hits | (1 << index)
	} else {
		// 011 & 101 == 001
		r.Hits = r.Hits & ((1 << index) ^ 255)
	}
}

func (r *Ratchet) Toggle(index uint8) {
	// 011 ^ 010  == 001
	r.Hits = (r.Hits ^ (1 << index))
}

func (r Ratchet) HitAt(index uint8) bool {
	return (r.Hits & (1 << index)) != 0
}

type MessageType uint8

const (
	MessageTypeNote MessageType = iota
	MessageTypeCc
	MessageTypeProgramChange
)

type LineDefinition struct {
	Channel    uint8
	Note       uint8
	MsgType    MessageType
	Name       string
	MidiOutput string
}

func (l *LineDefinition) IncrementChannel() {
	if l.Channel < 16 {
		l.Channel++
	}
}

func (l *LineDefinition) DecrementChannel() {
	if l.Channel > 1 {
		l.Channel--
	}
}

func (l *LineDefinition) IncrementNote() {
	if l.Note < 127 {
		l.Note++
	}
}

func (l *LineDefinition) DecrementNote() {
	if l.Note > 0 {
		l.Note--
	}
}

func (l *LineDefinition) IncrementMessageType() {
	l.MsgType = (l.MsgType + 1) % 3
}

func (l *LineDefinition) DecrementMessageType() {
	if l.MsgType == 0 {
		l.MsgType = 2
	} else {
		l.MsgType--
	}
}

// GenerateEuclideanRhythm generates a Euclidean rhythm pattern using Bjorklund's algorithm.
// It distributes 'hits' as evenly as possible over 'steps'.
// Returns a slice of booleans where true indicates a hit.
func GenerateEuclideanRhythm(hits, steps int) []bool {
	if hits <= 0 || steps <= 0 || hits > steps {
		result := make([]bool, steps)
		return result
	}

	// Initialize pattern with hits and rests
	pattern := make([][]bool, steps)
	for i := 0; i < hits; i++ {
		pattern[i] = []bool{true}
	}
	for i := hits; i < steps; i++ {
		pattern[i] = []bool{false}
	}

	// Bjorklund's algorithm
	counts := make([]int, steps)
	for i := range counts {
		counts[i] = 1
	}

	numGroups := steps
	for {
		// Count how many we can pair
		minCount := hits
		if steps-hits < hits {
			minCount = steps - hits
		}

		if minCount <= 1 {
			break
		}

		// Concatenate pairs
		for i := 0; i < minCount; i++ {
			pattern[i] = append(pattern[i], pattern[hits+i]...)
			counts[i] += counts[hits+i]
		}

		// Update group counts
		remainder := numGroups - minCount*2
		numGroups = minCount + remainder
		hits = minCount
		steps = numGroups
	}

	// Flatten the pattern
	result := make([]bool, 0)
	for i := 0; i < numGroups; i++ {
		result = append(result, pattern[i]...)
	}

	return result
}
