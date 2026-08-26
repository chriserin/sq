package sequence

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/chriserin/sq/internal/arrangement"
	"github.com/chriserin/sq/internal/config"
	"github.com/chriserin/sq/internal/grid"
	"github.com/chriserin/sq/internal/overlaykey"
	"github.com/chriserin/sq/internal/overlays"
)

// Read loads the model's sequence struct from a file
// The file format should match the format created by the Write function
func Read(filename string) (Sequence, error) {
	file, err := os.Open(filename)
	if err != nil {
		log.Error("Failed to open file", "filename", filename, "error", err)
		return Sequence{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	sequence := Sequence{
		Parts:           &[]arrangement.Part{},
		Lines:           []grid.LineDefinition{},
		Tempo:           120, // Default values
		Subdivisions:    4,
		Keyline:         0,
		Instrument:      "piano",
		Template:        "default",
		TemplateUIStyle: "dark",
		Accents: PatternAccents{
			End:    10,
			Start:  64,
			Target: AccentTargetVelocity,
			Data:   []config.Accent{},
		},
	}

	sequence = Scan(scanner, sequence)
	// Check if we got a scanner error
	if err := scanner.Err(); err != nil {
		log.Error("Error reading file", "filename", filename, "error", err)
		return Sequence{}, err
	}

	return sequence, nil
}

func Scan(scanner *bufio.Scanner, sequence Sequence) Sequence {
	var currentSection string
	var currentPart *arrangement.Part
	var currentOverlay *overlays.Overlay
	var currentChord *overlays.GridChord

	var blockersList = make(map[overlaykey.OverlayPeriodicity][]string)
	var chordsList = make(map[string]*overlays.GridChord)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for section headers
		if strings.Contains(line, "----------------------") {
			sectionLine := strings.TrimSpace(line)
			sectionLine = strings.Trim(sectionLine, "- ")

			// Handle other section markers
			switch {
			case strings.Contains(sectionLine, "GLOBAL SETTINGS"):
				currentSection = "GLOBAL_SETTINGS"
			case strings.Contains(sectionLine, "LINES"):
				currentSection = "LINES"
			case strings.Contains(sectionLine, "ACCENTS"):
				currentSection = "ACCENTS"
			case strings.Contains(sectionLine, "ACCENT DATA"):
				currentSection = "ACCENT_DATA"
			case strings.Contains(sectionLine, "PARTS"):
				currentSection = "PARTS"
			case strings.Contains(sectionLine, "PART "):
				currentSection = "PART"

				if currentPart != nil {
					finalizePreviousPart(currentPart, blockersList, chordsList)

					blockersList = make(map[overlaykey.OverlayPeriodicity][]string)
					chordsList = make(map[string]*overlays.GridChord)
				}

				partName := strings.TrimPrefix(sectionLine, "PART ")
				partName = strings.TrimSpace(partName)

				// Create new part
				newPart := arrangement.Part{
					Name:  partName,
					Beats: 16, // Default value, will be overwritten if specified
				}
				*sequence.Parts = append(*sequence.Parts, newPart)
				currentPart = &(*sequence.Parts)[len(*sequence.Parts)-1]

			case strings.Contains(sectionLine, "OVERLAY"):
				currentSection = "OVERLAY"

				// Create a default overlay if none exists yet
				if currentPart.Overlays == nil {
					key := overlaykey.OverlayPeriodicity{
						Shift:      0,
						Interval:   0,
						Width:      0,
						StartCycle: 0,
					}
					currentPart.Overlays = overlays.InitOverlay(key, nil)
					currentOverlay = currentPart.Overlays
				} else if currentOverlay == nil {
					currentOverlay = currentPart.Overlays
				} else if currentOverlay.Below == nil {
					// Create a default overlay below the current one
					key := overlaykey.OverlayPeriodicity{
						Shift:      0,
						Interval:   0,
						Width:      0,
						StartCycle: 0,
					}
					currentOverlay.Below = overlays.InitOverlay(key, nil)
					currentOverlay = currentOverlay.Below
				} else {
					// Move to the overlay below
					currentOverlay = currentOverlay.Below
				}

			case strings.Contains(sectionLine, "BEATNOTES"):
				currentSection = "BEATNOTES"

			case strings.Contains(sectionLine, "NOTES"):
				currentSection = "NOTES"

			case strings.Contains(sectionLine, "CHORDS"):
				currentSection = "CHORDS"

			case strings.Contains(sectionLine, "CHORD"):
				currentSection = "CHORD"
				currentChord = &overlays.GridChord{}
				currentOverlay.Chords = append(currentOverlay.Chords, currentChord)

			case strings.Contains(sectionLine, "BLOCKERS"):
				currentSection = "BLOCKERS"

			case strings.Contains(sectionLine, "ARRANGEMENT"):
				currentSection = "ARRANGEMENT"

			case strings.Contains(sectionLine, "ROOT NODE"):
				currentSection = "ARRANGEMENT_NODE"
				sequence.Arrangement = &arrangement.Arrangement{
					Iterations: 1,
					Nodes:      []*arrangement.Arrangement{},
				}
				ScanArrangement(scanner, sequence.Arrangement, 0)

			default:
				// Unknown section
				continue
			}
			continue
		}

		// Process line based on current section
		switch currentSection {
		case "GLOBAL_SETTINGS":
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "Tempo":
				if tempo, err := strconv.Atoi(value); err == nil {
					sequence.Tempo = tempo
				}
			case "Subdivisions":
				if subdiv, err := strconv.Atoi(value); err == nil {
					sequence.Subdivisions = subdiv
				}
			case "Keyline":
				if keyline, err := strconv.ParseUint(value, 10, 8); err == nil {
					sequence.Keyline = uint8(keyline)
				}
			case "Instrument":
				sequence.Instrument = value
			case "Template":
				sequence.Template = value
			case "TemplateUIStyle":
				sequence.TemplateUIStyle = value
			}

		case "LINES":
			if !strings.HasPrefix(line, "Line ") {
				continue
			}

			// Parse line sequence
			// Format: Line X: Channel=Y, Note=Z, MessageType=W
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			paramStr := strings.TrimSpace(parts[1])
			params := strings.Split(paramStr, ", ")

			lineDef := grid.LineDefinition{}

			for _, param := range params {
				keyVal := strings.SplitN(param, "=", 2)
				if len(keyVal) != 2 {
					continue
				}

				key := strings.TrimSpace(keyVal[0])
				value := strings.TrimSpace(keyVal[1])

				switch key {
				case "Channel":
					if channel, err := strconv.ParseUint(value, 10, 8); err == nil {
						lineDef.Channel = uint8(channel)
					}
				case "Note":
					if note, err := strconv.ParseUint(value, 10, 8); err == nil {
						lineDef.Note = uint8(note)
					}
				case "MessageType":
					if msgType, err := strconv.ParseUint(value, 10, 8); err == nil {
						lineDef.MsgType = grid.MessageType(msgType)
					}
				case "Name":
					lineDef.Name = value
				case "MidiOutput":
					lineDef.MidiOutput = value
				}
			}

			sequence.Lines = append(sequence.Lines, lineDef)

		case "ACCENTS":
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "End":
				if end, err := strconv.ParseUint(value, 10, 8); err == nil {
					sequence.Accents.End = uint8(end)
				}
			case "Start":
				if start, err := strconv.ParseUint(value, 10, 8); err == nil {
					sequence.Accents.Start = uint8(start)
				}
			case "Target":
				switch value {
				case "NOTE":
					sequence.Accents.Target = AccentTargetNote
				case "VELOCITY":
					sequence.Accents.Target = AccentTargetVelocity
				}
			}

		case "ACCENT_DATA":
			if !strings.HasPrefix(line, "Accent ") {
				continue
			}

			// Parse accent data
			// Format: Accent X: Shape='Y', Color=Z, Value=W
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			paramStr := strings.TrimSpace(parts[1])
			params := strings.Split(paramStr, ", ")

			var accent config.Accent

			for _, param := range params {
				keyVal := strings.SplitN(param, "=", 2)
				if len(keyVal) != 2 {
					continue
				}

				key := strings.TrimSpace(keyVal[0])
				value := strings.TrimSpace(keyVal[1])

				switch key {
				case "Value":
					if val, err := strconv.Atoi(value); err == nil {
						accent = config.Accent(val)
					}
				}
			}

			sequence.Accents.Data = append(sequence.Accents.Data, accent)

		case "PART":
			if currentPart == nil {
				continue
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "Name":
				currentPart.Name = value
			case "Beats":
				if beats, err := strconv.ParseUint(value, 10, 8); err == nil {
					currentPart.Beats = uint8(beats)
				}
			}

		case "OVERLAY":
			if currentOverlay == nil {
				continue
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "Shift":
				if shift, err := strconv.ParseUint(value, 10, 8); err == nil {
					currentOverlay.Key.Shift = uint8(shift)
				}
			case "Interval":
				if interval, err := strconv.ParseUint(value, 10, 8); err == nil {
					currentOverlay.Key.Interval = uint8(interval)
				}
			case "Width":
				if width, err := strconv.ParseUint(value, 10, 8); err == nil {
					currentOverlay.Key.Width = uint8(width)
				}
			case "StartCycle":
				if startCycle, err := strconv.ParseUint(value, 10, 8); err == nil {
					currentOverlay.Key.StartCycle = uint8(startCycle)
				}
			case "PressUp":
				if pressUp, err := strconv.ParseBool(value); err == nil {
					currentOverlay.PressUp = pressUp
				}
			case "PressDown":
				if pressDown, err := strconv.ParseBool(value); err == nil {
					currentOverlay.PressDown = pressDown
				}
			}

		case "BEATNOTES":
			if currentOverlay == nil || line == "(empty)" {
				continue
			}

			beat, coordEnd := GetBeat(line)

			// Parse note properties
			propStr := line[coordEnd+2:] // "AccentIndex=Z, ..."

			note := noteprops(propStr)

			currentChord.Notes = append(currentChord.Notes, overlays.BeatNote{Beat: beat, Note: note})

		case "NOTES":
			if currentOverlay == nil || line == "(empty)" {
				continue
			}

			gridKey, coordEnd := GetGridKey(line)

			// Parse note properties
			propStr := line[coordEnd+2:] // "AccentIndex=Z, ..."

			note := noteprops(propStr)

			currentOverlay.AddNote(gridKey, note)

		case "ARRANGEMENT":
			// This is just the arrangement header section, handled in section detection
		case "BLOCKERS":
			if line == "(empty)" {
				continue
			} else {
				id := GetID(line)
				blockersList[currentOverlay.Key] = append(blockersList[currentOverlay.Key], id)
			}
		case "CHORD":
			id := GetID(line)
			if id != "" {
				chordsList[id] = currentChord
				scanner.Scan()
				line = scanner.Text()
			}
			gridKey, coordEnd := GetGridKey(line)
			currentChord.Root = gridKey
			propStr := line[coordEnd+2:] // "AccentIndex=Z, ..."
			props := strings.Split(propStr, ", ")

			for _, prop := range props {
				keyVal := strings.SplitN(prop, "=", 2)
				if len(keyVal) != 2 {
					continue
				}

				key := strings.TrimSpace(keyVal[0])
				value := strings.TrimSpace(keyVal[1])

				switch key {
				case "Arpeggio":
					if arppegio, err := strconv.ParseInt(value, 10, 8); err == nil {
						currentChord.Arpeggio = overlays.Arp(arppegio)
					}
				case "Notes":
					if notes, err := strconv.ParseUint(value, 10, 32); err == nil {
						currentChord.Chord.Notes = uint32(notes)
					}
				}
			}

		default:
			// Unknown section, ignore
		}
	}

	if currentPart != nil {
		finalizePreviousPart(currentPart, blockersList, chordsList)
	}

	return sequence
}

func finalizePreviousPart(currentPart *arrangement.Part, blockersList map[overlaykey.OverlayPeriodicity][]string, chordsList map[string]*overlays.GridChord) {
	currentOverlay := currentPart.Overlays
	for currentOverlay != nil {
		if ids, exists := blockersList[currentOverlay.Key]; exists {
			for _, id := range ids {
				blocker := chordsList[id]
				fmt.Fprintln(os.Stderr, "KEYS", slices.Collect(maps.Keys(chordsList)))
				fmt.Fprintln(os.Stderr, "Adding blocker", id, "to overlay", currentOverlay.Key, "blocker:", blocker)
				currentOverlay.Blockers = append(currentOverlay.Blockers, blocker)
			}
		}
		currentOverlay = currentOverlay.Below
	}
}

func GetID(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return ""
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	if key != "ID" {
		return ""
	}

	return value
}

func ScanArrangement(scanner *bufio.Scanner, currentArrangement *arrangement.Arrangement, indentLevel int) bool {
	for scanner.Scan() {
	REREAD:
		line := scanner.Text()

		if strings.Contains(line, "------------------------") {
			sectionLine := strings.TrimSpace(line)
			if strings.Contains(sectionLine, "CHILDREN") {
				continue
			}
			sectionLineIndentLevel := len(line) - len(sectionLine)

			if sectionLineIndentLevel <= indentLevel {
				return true
			}

			switch {
			case strings.Contains(sectionLine, "GROUP"):
				fallthrough
			case strings.Contains(sectionLine, "SECTION"):
				newArrangement := &arrangement.Arrangement{}
				currentArrangement.Nodes = append(currentArrangement.Nodes, newArrangement)
				reread := ScanArrangement(scanner, newArrangement, sectionLineIndentLevel)
				if reread {
					goto REREAD
				} else {
					continue
				}
			}
		} else if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return false
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Iterations":
			if iterations, err := strconv.Atoi(value); err == nil {
				if currentArrangement != nil {
					currentArrangement.Iterations = iterations
				}
			}
		case "Part":
			if part, err := strconv.Atoi(value); err == nil {
				if currentArrangement != nil {
					currentArrangement.Section.Part = part
				}
			}
		case "Cycles":
			if cycles, err := strconv.Atoi(value); err == nil {
				if currentArrangement != nil {
					currentArrangement.Section.Cycles = cycles
				}
			}
		case "StartBeat":
			if startBeat, err := strconv.Atoi(value); err == nil {
				if currentArrangement != nil {
					currentArrangement.Section.StartBeat = startBeat
				}
			}
		case "StartCycles":
			if startCycles, err := strconv.Atoi(value); err == nil {
				if currentArrangement != nil {
					currentArrangement.Section.StartCycles = startCycles
				}
			}
		case "KeepCycles":
			if keepCycles, err := strconv.ParseBool(value); err == nil {
				if currentArrangement != nil {
					currentArrangement.Section.KeepCycles = keepCycles
					return false
				}
			}
		case "Children":
			// We'll create child nodes in the CHILDREN section
			childCount, err := strconv.Atoi(value)
			if err != nil || childCount == 0 {
				return false
			}

			if currentArrangement != nil {
				// Set up for receiving children
				currentArrangement.Nodes = make([]*arrangement.Arrangement, 0, childCount)
			}
		}
	}
	return false
}

func GetBeat(line string) (int, int) {
	if !strings.HasPrefix(line, "Beat(") {
		return 0, -1
	}

	coordEnd := strings.Index(line, ")")
	if coordEnd == -1 {
		return 0, -1
	}

	coordStr := line[5:coordEnd]

	beat, err := strconv.ParseInt(coordStr, 10, 8)

	if err == nil {
		return int(beat), coordEnd
	}

	return 0, -1
}

func GetGridKey(line string) (grid.GridKey, int) {

	// Parse grid key and note
	// Format: GridKey(X,Y): AccentIndex=Z, Ratchets={Hits:A,Length:B,Span:C}, Action=D, GateIndex=E, WaitIndex=F
	if !strings.HasPrefix(line, "GridKey(") {
		return grid.GridKey{}, -1
	}

	// Extract grid key coordinates
	coordEnd := strings.Index(line, ")")
	if coordEnd == -1 {
		return grid.GridKey{}, -1
	}

	coordStr := line[8:coordEnd] // "X,Y"
	coords := strings.Split(coordStr, ",")
	if len(coords) != 2 {
		return grid.GridKey{}, -1
	}

	line8, err1 := strconv.ParseUint(coords[0], 10, 8)
	beat8, err2 := strconv.ParseUint(coords[1], 10, 8)
	if err1 != nil || err2 != nil {
		return grid.GridKey{}, -1
	}

	gridKey := grid.GridKey{
		Line: uint8(line8),
		Beat: uint8(beat8),
	}

	return gridKey, coordEnd
}

func noteprops(propStr string) grid.Note {

	props := strings.Split(propStr, ", ")
	note := grid.InitNote()

	for _, prop := range props {
		if strings.Contains(prop, "Ratchets={") {
			// Handle ratchets property separately
			ratchetsStr := prop[strings.Index(prop, "{")+1 : strings.Index(prop, "}")]
			ratchetsProps := strings.Split(ratchetsStr, ",")

			for _, rProp := range ratchetsProps {
				rKeyVal := strings.SplitN(rProp, ":", 2)
				if len(rKeyVal) != 2 {
					continue
				}

				rKey := strings.TrimSpace(rKeyVal[0])
				rValue := strings.TrimSpace(rKeyVal[1])

				switch rKey {
				case "Hits":
					if hits, err := strconv.ParseUint(rValue, 10, 8); err == nil {
						note.Ratchets.Hits = uint8(hits)
					}
				case "Length":
					if length, err := strconv.ParseUint(rValue, 10, 8); err == nil {
						note.Ratchets.Length = uint8(length)
					}
				case "Span":
					if span, err := strconv.ParseUint(rValue, 10, 8); err == nil {
						note.Ratchets.Span = uint8(span)
					}
				}
			}
		} else {
			keyVal := strings.SplitN(prop, "=", 2)
			if len(keyVal) != 2 {
				continue
			}

			key := strings.TrimSpace(keyVal[0])
			value := strings.TrimSpace(keyVal[1])

			switch key {
			case "AccentIndex":
				if accentIdx, err := strconv.ParseUint(value, 10, 8); err == nil {
					note.AccentIndex = uint8(accentIdx)
				}
			case "Action":
				if action, err := strconv.ParseUint(value, 10, 8); err == nil {
					note.Action = grid.Action(action)
				}
			case "GateIndex":
				if gateIdx, err := strconv.ParseUint(value, 10, 8); err == nil {
					note.GateIndex = int16(gateIdx)
				}
			case "WaitIndex":
				if waitIdx, err := strconv.ParseUint(value, 10, 8); err == nil {
					note.WaitIndex = uint8(waitIdx)
				}
			}
		}
	}
	return note
}
