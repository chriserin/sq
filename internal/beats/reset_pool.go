package beats

import (
	"github.com/chriserin/sq/internal/arrangement"
	"github.com/chriserin/sq/internal/config"
)

// resetPoolIdentity is the dedup key for assigning arrangement nodes to pool
// notes. Part leaves are deduped by Section.Part (the named Part definition
// — every placement of the same part shares one note); Group nodes are
// deduped by their own tree-node pointer (groups have no reusable
// "definition", so each group node is its own identity).
type resetPoolIdentity struct {
	isGroup   bool
	partIndex int
	groupNode *arrangement.Arrangement
}

// assignPoolNote returns identity's assigned note in r. The first time an
// identity is seen it claims *next and advances it, wrapping back to
// r.StartNote once r.EndNote is exhausted; later encounters of the same
// identity return the note already recorded in assigned.
func assignPoolNote(r config.ResetRange, assigned map[resetPoolIdentity]uint8, next *uint8, id resetPoolIdentity) uint8 {
	if note, ok := assigned[id]; ok {
		return note
	}
	note := *next
	assigned[id] = note
	if note == r.EndNote {
		*next = r.StartNote
	} else {
		*next = note + 1
	}
	return note
}

// ArrangementPoolNotes walks root once, depth-first in Nodes slice order
// (the arrangement's authored/saved order), and resolves every Part/Group
// node's Start-range and Loop-range note directly, keyed by node pointer
// (Part leaves that share an identity via Section.Part get the same note
// under every one of their tree placements). Computed once per play
// session (ui.go's ResetIterations, called from Start()) — the arrangement
// and the configured ranges are both static for a session's lifetime, so
// this is safe to compute once and reuse for every beat.
func ArrangementPoolNotes(root *arrangement.Arrangement, startRange, loopRange config.ResetRange) (startNotes, loopNotes map[*arrangement.Arrangement]uint8) {
	startNotes = make(map[*arrangement.Arrangement]uint8)
	loopNotes = make(map[*arrangement.Arrangement]uint8)
	startAssigned := make(map[resetPoolIdentity]uint8)
	loopAssigned := make(map[resetPoolIdentity]uint8)
	nextStart := startRange.StartNote
	nextLoop := loopRange.StartNote

	var walk func(n *arrangement.Arrangement)
	walk = func(n *arrangement.Arrangement) {
		for _, child := range n.Nodes {
			var id resetPoolIdentity
			if child.IsEndNode() {
				id = resetPoolIdentity{partIndex: child.Section.Part}
			} else {
				id = resetPoolIdentity{isGroup: true, groupNode: child}
			}
			if startRange.Channel != 0 {
				startNotes[child] = assignPoolNote(startRange, startAssigned, &nextStart, id)
			}
			if loopRange.Channel != 0 {
				loopNotes[child] = assignPoolNote(loopRange, loopAssigned, &nextLoop, id)
			}
			if child.IsGroup() {
				walk(child)
			}
		}
	}
	walk(root)
	return startNotes, loopNotes
}
