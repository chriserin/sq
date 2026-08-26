package sequence

import (
	"fmt"
	"io"
	"os"
	"slices"
	"sort"

	"github.com/charmbracelet/log"

	"github.com/chriserin/sq/internal/arrangement"
	"github.com/chriserin/sq/internal/grid"
	"github.com/chriserin/sq/internal/overlays"
)

// Write saves all attributes of the model's sequence struct to a file
// using a custom format that is easy to diff with tools like git diff
func Write(sequence Sequence, filename string) error {

	log.SetLevel(log.FatalLevel)

	f, err := os.Create(filename)
	if err != nil {
		log.Error("Failed to create file", "filename", filename, "error", err)
		return err
	}
	defer f.Close()

	if sequence.Parts == nil || len(*sequence.Parts) == 0 {
		log.Warn("No parts to write", "filename", filename)
		return nil
	}

	// Write global sequencer settings
	if err := writeSettings(f, &sequence); err != nil {
		return err
	}

	// Write line sequences
	if err := writeLineSequences(f, sequence.Lines); err != nil {
		return err
	}

	// Write accents
	if err := writeAccents(f, sequence.Accents); err != nil {
		return err
	}

	// Write parts
	if err := writeParts(f, *sequence.Parts); err != nil {
		return err
	}

	// Write arrangement
	if sequence.Arrangement != nil {
		if err := writeArrangement(f, sequence.Arrangement); err != nil {
			return err
		}
	}

	return nil
}

// writeSettings writes global sequencer settings
func writeSettings(w io.Writer, def *Sequence) error {
	fmt.Fprintln(w, "------------------------ GLOBAL SETTINGS ------------------------")
	fmt.Fprintf(w, "Tempo: %d\n", def.Tempo)
	fmt.Fprintf(w, "Subdivisions: %d\n", def.Subdivisions)
	fmt.Fprintf(w, "Keyline: %d\n", def.Keyline)
	fmt.Fprintf(w, "Instrument: %s\n", def.Instrument)
	fmt.Fprintf(w, "Template: %s\n", def.Template)
	fmt.Fprintf(w, "TemplateUIStyle: %s\n", def.TemplateUIStyle)
	fmt.Fprintln(w, "")

	return nil
}

// writeLineSequences writes all line sequences
func writeLineSequences(w io.Writer, lines []grid.LineDefinition) error {
	if len(lines) == 0 {
		return nil
	}

	fmt.Fprintln(w, "------------------------- LINES -------------------------")
	for i, line := range lines {
		fmt.Fprintf(w, "Line %d: Channel=%d, Note=%d, MessageType=%d, Name=%s, MidiOutput=%s\n",
			i, line.Channel, line.Note, line.MsgType, line.Name, line.MidiOutput)
	}
	fmt.Fprintln(w, "")

	return nil
}

// writeAccents writes the accents configuration
func writeAccents(w io.Writer, accents PatternAccents) error {
	fmt.Fprintln(w, "------------------------- ACCENTS -------------------------")

	// Convert accentTarget to string for better readability
	targetStr := "UNKNOWN"
	switch accents.Target {
	case AccentTargetNote:
		targetStr = "NOTE"
	case AccentTargetVelocity:
		targetStr = "VELOCITY"
	}
	fmt.Fprintf(w, "Target: %s\n", targetStr)

	fmt.Fprintf(w, "Start: %d\n", accents.Start)
	fmt.Fprintf(w, "End: %d\n", accents.End)

	// Write accent data
	if len(accents.Data) > 0 {
		fmt.Fprintln(w, "----------------------- ACCENT DATA -----------------------")
		for i, accent := range accents.Data {
			fmt.Fprintf(w, "Accent %d: Value=%d\n",
				i, accent)
		}
	}
	fmt.Fprintln(w, "")

	return nil
}

// writeParts writes all parts to the provided writer
func writeParts(w io.Writer, parts []arrangement.Part) error {
	fmt.Fprintln(w, "--------------------------- PARTS ---------------------------")

	for i, part := range parts {
		partName := part.Name
		if partName == "" {
			partName = fmt.Sprintf("Part%d", i+1)
		}

		separator := fmt.Sprintf("------------------------ PART %s ------------------------", partName)
		fmt.Fprintln(w, separator)
		fmt.Fprintf(w, "Name: %s\n", part.Name)
		fmt.Fprintf(w, "Beats: %d\n", part.Beats)

		if err := writeOverlays(w, part.Overlays); err != nil {
			return err
		}

		fmt.Fprintln(w, "") // Extra newline for better readability between parts
	}

	return nil
}

// writeArrangement writes the arrangement tree structure recursively
func writeArrangement(w io.Writer, arr *arrangement.Arrangement) error {
	if arr == nil {
		return nil
	}

	fmt.Fprintln(w, "------------------------ ARRANGEMENT ------------------------")

	// Write arrangement tree recursively using depth-first traversal
	return writeArrangementNode(w, arr, 0, -1) // Pass -1 as childIndex to indicate root node
}

// writeArrangementNode writes a single arrangement node and its children
func writeArrangementNode(w io.Writer, node *arrangement.Arrangement, depth int, childIndex ...int) error {
	// Create indentation based on depth
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	// Write node identifier
	if depth == 0 {
		fmt.Fprintln(w, "------------------------ ROOT NODE ------------------------")
	} else {
		isGroup := len(node.Nodes) > 0
		nodeType := "SECTION"
		if isGroup {
			nodeType = "GROUP"
		}

		// Include child index in the separator if provided
		indexStr := ""
		if len(childIndex) > 0 && childIndex[0] >= 0 {
			indexStr = fmt.Sprintf(" #%d", childIndex[0]+1)
		}

		fmt.Fprintf(w, "%s------------------------ %s%s NODE ------------------------\n",
			indent, nodeType, indexStr)
	}

	// Write node properties
	fmt.Fprintf(w, "%sIterations: %d\n", indent, node.Iterations)

	// If it's an end node (contains a section), write section data
	if len(node.Nodes) == 0 {
		fmt.Fprintf(w, "%sPart: %d\n", indent, node.Section.Part)
		fmt.Fprintf(w, "%sCycles: %d\n", indent, node.Section.Cycles)
		fmt.Fprintf(w, "%sStartBeat: %d\n", indent, node.Section.StartBeat)
		fmt.Fprintf(w, "%sStartCycles: %d\n", indent, node.Section.StartCycles)
		fmt.Fprintf(w, "%sKeepCycles: %t\n", indent, node.Section.KeepCycles)
	}

	// Write child nodes recursively
	if len(node.Nodes) > 0 {
		fmt.Fprintf(w, "%sChildren: %d\n", indent, len(node.Nodes))
		fmt.Fprintf(w, "%s------------------------ CHILDREN ------------------------\n", indent)

		for i, child := range node.Nodes {
			// Add empty line between children for readability
			if i > 0 {
				fmt.Fprintf(w, "%s\n", indent)
			}

			// Pass the child index to the recursive call
			if err := writeArrangementNode(w, child, depth+1, i); err != nil {
				return err
			}
		}
	}

	return nil
}

// writeOverlays writes the overlay tree structure recursively
func writeOverlays(w io.Writer, overlay *overlays.Overlay) error {
	if overlay == nil {
		return nil
	}

	// Write the current overlay
	fmt.Fprintln(w, "----------------------- OVERLAY -------------------------")

	// Write overlay key properties
	fmt.Fprintf(w, "Shift: %d\n", overlay.Key.Shift)
	fmt.Fprintf(w, "Interval: %d\n", overlay.Key.Interval)
	fmt.Fprintf(w, "Width: %d\n", overlay.Key.Width)
	fmt.Fprintf(w, "StartCycle: %d\n", overlay.Key.StartCycle)
	fmt.Fprintf(w, "PressUp: %t\n", overlay.PressUp)
	fmt.Fprintf(w, "PressDown: %t\n", overlay.PressDown)

	fmt.Fprintln(w, "------------------------ CHORDS --------------------------")
	if len(overlay.Chords) > 0 {
		slices.SortFunc(overlay.Chords, func(a, b *overlays.GridChord) int {
			return grid.Compare(a.Root, b.Root)
		})

		for _, gridChord := range overlay.Chords {
			fmt.Fprintln(w, "------------------------ CHORD --------------------------")
			fmt.Fprintln(w, "ID:", fmt.Sprintf("%p", gridChord))
			fmt.Fprintf(w, "GridKey(%d,%d): Arpeggio=%d, Notes=%d\n", gridChord.Root.Line, gridChord.Root.Beat, gridChord.Arpeggio, gridChord.Chord.Notes)

			fmt.Fprintln(w, "------------------------ BEATNOTES --------------------------")
			for _, beatNote := range gridChord.Notes {
				note := beatNote.Note
				fmt.Fprintf(w, "Beat(%d): AccentIndex=%d, Ratchets={Hits:%d,Length:%d,Span:%d}, Action=%d, GateIndex=%d, WaitIndex=%d\n",
					beatNote.Beat,
					note.AccentIndex, note.Ratchets.Hits, note.Ratchets.Length, note.Ratchets.Span,
					note.Action, note.GateIndex, note.WaitIndex)
			}
		}
	} else {
		fmt.Fprintln(w, "(empty)")
	}

	fmt.Fprintln(w, "------------------------ BLOCKERS --------------------------")
	if len(overlay.Blockers) > 0 {
		slices.SortFunc(overlay.Chords, func(a, b *overlays.GridChord) int {
			return grid.Compare(a.Root, b.Root)
		})

		for _, gridChord := range overlay.Blockers {
			fmt.Fprintln(w, "ID:", fmt.Sprintf("%p", gridChord))
		}
	} else {
		fmt.Fprintln(w, "(empty)")
	}

	// Write notes in a formatted way
	fmt.Fprintln(w, "------------------------ NOTES --------------------------")
	if len(overlay.Notes) > 0 {
		// Get all GridKeys and sort them for stable output
		gridKeys := make([]grid.GridKey, 0, len(overlay.Notes))
		for k := range overlay.Notes {
			gridKeys = append(gridKeys, k)
		}

		// Sort GridKeys by Line then Beat for consistent output
		sort.Slice(gridKeys, func(i, j int) bool {
			if gridKeys[i].Line != gridKeys[j].Line {
				return gridKeys[i].Line < gridKeys[j].Line
			}
			return gridKeys[i].Beat < gridKeys[j].Beat
		})

		for _, k := range gridKeys {
			note := overlay.Notes[k]
			fmt.Fprintf(w, "GridKey(%d,%d): AccentIndex=%d, Ratchets={Hits:%d,Length:%d,Span:%d}, Action=%d, GateIndex=%d, WaitIndex=%d\n",
				k.Line, k.Beat,
				note.AccentIndex, note.Ratchets.Hits, note.Ratchets.Length, note.Ratchets.Span,
				note.Action, note.GateIndex, note.WaitIndex)
		}
	} else {
		fmt.Fprintln(w, "(empty)")
	}

	// Recursively process the overlay below this one
	if overlay.Below != nil {
		writeOverlays(w, overlay.Below)
	}

	return nil
}
