package beats

import (
	"testing"

	"github.com/chriserin/sq/internal/arrangement"
	"github.com/chriserin/sq/internal/config"
	"github.com/chriserin/sq/internal/grid"
	"github.com/chriserin/sq/internal/sequence"
	"github.com/stretchr/testify/assert"
)

// poolTestSequence builds:
//
//	root
//	├── groupA -> [nodeVerse1, nodeVerse2]  (same Section.Part index -- one identity, two placements)
//	├── groupB -> [nodeChorus]
//	├── groupC -> [nodeBridge]
//	└── nodeOutro
//
// 4 parts (Verse reused, Chorus, Bridge, Outro), 3 groups -> 7 distinct pool
// identities in first-appearance order: groupA, Verse, groupB, Chorus,
// groupC, Bridge, Outro.
func poolTestSequence() (sequence.Sequence, *arrangement.Arrangement, *arrangement.Arrangement, *arrangement.Arrangement, *arrangement.Arrangement, *arrangement.Arrangement, *arrangement.Arrangement, *arrangement.Arrangement) {
	parts := sequence.InitParts() // Part 0: Verse
	parts = append(parts,
		arrangement.InitPart("Chorus"), // Part 1
		arrangement.InitPart("Bridge"), // Part 2
		arrangement.InitPart("Outro"),  // Part 3
	)

	nodeVerse1 := &arrangement.Arrangement{Section: arrangement.SongSection{Part: 0, Cycles: 1, StartCycles: 1}, Iterations: 1}
	nodeVerse2 := &arrangement.Arrangement{Section: arrangement.SongSection{Part: 0, Cycles: 1, StartCycles: 1}, Iterations: 1}
	nodeChorus := &arrangement.Arrangement{Section: arrangement.SongSection{Part: 1, Cycles: 1, StartCycles: 1}, Iterations: 1}
	nodeBridge := &arrangement.Arrangement{Section: arrangement.SongSection{Part: 2, Cycles: 1, StartCycles: 1}, Iterations: 1}
	nodeOutro := &arrangement.Arrangement{Section: arrangement.SongSection{Part: 3, Cycles: 1, StartCycles: 1}, Iterations: 1}

	groupA := &arrangement.Arrangement{Iterations: 1, Nodes: []*arrangement.Arrangement{nodeVerse1, nodeVerse2}}
	groupB := &arrangement.Arrangement{Iterations: 1, Nodes: []*arrangement.Arrangement{nodeChorus}}
	groupC := &arrangement.Arrangement{Iterations: 1, Nodes: []*arrangement.Arrangement{nodeBridge}}

	root := &arrangement.Arrangement{Iterations: 1, Nodes: []*arrangement.Arrangement{groupA, groupB, groupC, nodeOutro}}

	testSequence := sequence.Sequence{
		Arrangement: root,
		Parts:       &parts,
		Keyline:     0,
		Lines:       make([]grid.LineDefinition, 1),
	}

	return testSequence, root, groupA, groupB, groupC, nodeVerse1, nodeVerse2, nodeChorus
}

func TestAssignPoolNote(t *testing.T) {
	r := config.ResetRange{Channel: 1, StartNote: 20, EndNote: 22} // 3-note range

	idA := resetPoolIdentity{partIndex: 0}
	idB := resetPoolIdentity{partIndex: 1}
	idC := resetPoolIdentity{partIndex: 2}
	idD := resetPoolIdentity{partIndex: 3}

	assigned := make(map[resetPoolIdentity]uint8)
	next := r.StartNote

	t.Run("first identity claims StartNote", func(t *testing.T) {
		assert.Equal(t, uint8(20), assignPoolNote(r, assigned, &next, idA))
		assert.Equal(t, uint8(21), next)
	})

	t.Run("a different identity claims the next note", func(t *testing.T) {
		assert.Equal(t, uint8(21), assignPoolNote(r, assigned, &next, idB))
		assert.Equal(t, uint8(22), next)
	})

	t.Run("a repeated identity reuses its note without advancing next", func(t *testing.T) {
		assert.Equal(t, uint8(20), assignPoolNote(r, assigned, &next, idA))
		assert.Equal(t, uint8(22), next, "next must not move for a repeat")
	})

	t.Run("reaching EndNote wraps the next assignment back to StartNote", func(t *testing.T) {
		assert.Equal(t, uint8(22), assignPoolNote(r, assigned, &next, idC))
		assert.Equal(t, uint8(20), next, "next wraps after claiming EndNote")

		assert.Equal(t, uint8(20), assignPoolNote(r, assigned, &next, idD))
		assert.Equal(t, uint8(21), next)
	})
}

func TestArrangementPoolNotes(t *testing.T) {
	t.Run("reused part shares one note, distinct groups get distinct notes, no wraparound", func(t *testing.T) {
		_, root, groupA, groupB, _, nodeVerse1, nodeVerse2, nodeChorus := poolTestSequence()
		startRange := config.ResetRange{Channel: 1, StartNote: 20, EndNote: 27} // 8 slots, more than the 7 identities
		loopRange := config.ResetRange{Channel: 2, StartNote: 30, EndNote: 37}

		startNotes, loopNotes := ArrangementPoolNotes(root, startRange, loopRange)

		assert.Equal(t, startNotes[nodeVerse1], startNotes[nodeVerse2], "both placements of the reused part share one note")
		assert.Equal(t, uint8(20), startNotes[groupA])
		assert.Equal(t, uint8(21), startNotes[nodeVerse1])
		assert.Equal(t, uint8(22), startNotes[groupB])
		assert.Equal(t, uint8(23), startNotes[nodeChorus])
		assert.NotEqual(t, startNotes[groupA], startNotes[groupB], "distinct groups get distinct notes")

		// Loop range mirrors the same identity order, offset by its own StartNote.
		assert.Equal(t, uint8(30), loopNotes[groupA])
		assert.Equal(t, uint8(31), loopNotes[nodeVerse1])
		assert.Equal(t, loopNotes[nodeVerse1], loopNotes[nodeVerse2])
	})

	t.Run("wraparound when the range is smaller than the identity count", func(t *testing.T) {
		_, root, groupA, groupB, groupC, _, _, _ := poolTestSequence()
		startRange := config.ResetRange{Channel: 1, StartNote: 20, EndNote: 23} // 4 slots for 7 identities

		startNotes, _ := ArrangementPoolNotes(root, startRange, config.ResetRange{})

		// Identity order: groupA(0) Verse(1) groupB(2) Chorus(3) groupC(4) Bridge(5) Outro(6)
		// -> notes:        20        21        22        23        20(wrap) 21       22
		assert.Equal(t, uint8(20), startNotes[groupA])
		assert.Equal(t, uint8(22), startNotes[groupB])
		assert.Equal(t, uint8(20), startNotes[groupC], "wraps back to the range's first note")
	})

	t.Run("unconfigured range (Channel == 0) produces no entries", func(t *testing.T) {
		_, root, groupA, _, _, _, _, _ := poolTestSequence()

		startNotes, loopNotes := ArrangementPoolNotes(root, config.ResetRange{}, config.ResetRange{})

		assert.Empty(t, startNotes)
		assert.Empty(t, loopNotes)
		_, ok := startNotes[groupA]
		assert.False(t, ok)
	})
}
