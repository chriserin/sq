// Package mappings provides keyboard input handling and command mapping functionality
// for the sequencer application. It manages key combinations, processes user input,
// and maps keyboard shortcuts to sequencer commands based on the current mode
// (trigger, polyphony, pattern mode, chord mode, etc.).
package mappings

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/chriserin/sq/internal/operation"
)

var Keycombo = make([]tea.KeyPressMsg, 0, 3)
var timer *time.Timer
var mutex = sync.Mutex{}

func KeycomboView() string {
	var buf strings.Builder
	for _, msg := range Keycombo {
		buf.WriteString(msg.String())
	}
	return buf.String()
}

type Command int
type Mapping struct {
	Command   Command
	LastValue string
}

const (
	HoldingKeys Command = iota
	Quit
	CursorUp
	CursorDown
	CursorLeft
	CursorRight
	CursorLineStart
	CursorLineEnd
	CursorLastLine
	CursorFirstLine
	Escape
	PlayStop
	PlayPart
	PlayLoop
	PlayOverlayLoop
	PlayRecord
	PlayAlong
	TempoInputSwitch
	OverlayInputSwitch
	ModifyKeyInputSwitch
	SetupInputSwitch
	AccentInputSwitch
	RatchetInputSwitch
	BeatInputSwitch
	CyclesInputSwitch
	ToggleArrangementView
	Increase
	Decrease
	Enter
	ToggleGateMode
	ToggleGateNoteMode
	ToggleWaitMode
	ToggleWaitNoteMode
	ToggleAccentMode
	ToggleAccentNoteMode
	ToggleRatchetMode
	ToggleRatchetNoteMode
	NextOverlay
	PrevOverlay
	Save
	SaveAs
	Undo
	Redo
	New
	ToggleVisualMode
	ToggleVisualLineMode
	TogglePlayEdit
	ToggleHideLines
	NewLine
	NewSectionAfter
	NewSectionBefore
	ChangePart
	NextSection
	PrevSection
	NextTheme
	PrevTheme
	Yank
	Mute
	MuteAll
	UnMuteAll
	Solo
	NoteAdd
	NoteRemove
	OverlayNoteRemove
	AccentIncrease
	AccentDecrease
	GateIncrease
	GateDecrease
	GateBigIncrease
	GateBigDecrease
	WaitIncrease
	WaitDecrease
	ClearLine
	ClearOverlay
	ClearAllOverlays
	RemoveOverlay
	RatchetIncrease
	RatchetDecrease
	ActionAddLineReset
	ActionAddLineResetAll
	ActionAddLineReverse
	ActionAddSkipBeat
	ActionAddLineBounce
	ActionAddLineBounceAll
	ActionAddLineDelay
	ActionAddSpecificValue
	SelectKeyLine
	OverlayStackToggle
	NumberPattern
	RotateRight
	RotateLeft
	RotateUp
	RotateDown
	Paste
	MajorTriad
	MinorTriad
	AugmentedTriad
	DiminishedTriad
	MinorSecond
	MajorSecond
	MinorThird
	MajorThird
	PerfectFourth
	AugFifth
	DimFifth
	PerfectFifth
	MajorSixth
	MinorSeventh
	MajorSeventh
	Octave
	MinorNinth
	MajorNinth
	IncreaseInversions
	DecreaseInversions
	ToggleChordMode
	ToggleMonoMode
	NextArpeggio
	PrevArpeggio
	NextDouble
	PrevDouble
	OmitRoot
	OmitSecond
	OmitThird
	OmitFourth
	OmitFifth
	OmitSixth
	OmitSeventh
	OmitOctave
	OmitNinth
	RemoveChord
	ConvertToNotes
	ReloadFile
	OverlayKeyMessage
	ArrKeyMessage
	TextInputMessage
	ConfirmOverlayKey
	ConfirmRenamePart
	ConfirmFileName
	ConfirmSelectPart
	ConfirmChangePart
	ConfirmConfirmNew
	ConfirmConfirmReload
	ConfirmConfirmQuit
	ConfirmEuclidenHits
	MidiPanic
	IncreaseAllChannels
	DecreaseAllChannels
	IncreaseAllNote
	DecreaseAllNote
	ToggleTransmitting
	ToggleClockPreRoll
	PurposePanic
	ToggleBoundedLoop
	ExpandLeftLoopBound
	ExpandRightLoopBound
	ContractLeftLoopBound
	ContractRightLoopBound
	Euclidean
	Reverse
	Duplicate
)

// CommandDescriptions maps each command to its human-readable description
var CommandDescriptions = map[Command]string{
	Quit:                   "Quit the application",
	CursorUp:               "Move cursor up",
	CursorDown:             "Move cursor down",
	CursorLeft:             "Move cursor left",
	CursorRight:            "Move cursor right",
	CursorLineStart:        "Move cursor to beginning of current line",
	CursorLineEnd:          "Move cursor to end of current line",
	CursorLastLine:         "Move cursor to last line",
	CursorFirstLine:        "Move cursor to first line",
	Escape:                 "Cancel current action or exit mode, move focus to the grid when elsewhere, or escape from visual mode",
	PlayStop:               "Play the full arrangement once. If playing, stop",
	PlayPart:               "Play current part in a loop",
	PlayLoop:               "Play the full arrangement in a loop",
	PlayOverlayLoop:        "Play the current overlay in a loop",
	PlayRecord:             "Play the full arrangement once and send a record message at the beginning",
	PlayAlong:              "Play along with external clock",
	TempoInputSwitch:       "Select the inputs that control the tempo and subdivision. Press once to select the tempo input, press again to select the subdivisions input",
	OverlayInputSwitch:     "This selects the inputs that control the overlay period/key",
	ModifyKeyInputSwitch:   "Modify the key of the current note",
	SetupInputSwitch:       "Select the inputs that control the midi message for each line. Pressing this key combo repeatedly will move through the channel, target and value inputs",
	AccentInputSwitch:      "This selects the controls that determine the accent values and target. Use +/- to increase and decrease the selections",
	RatchetInputSwitch:     "Select the inputs that control the ratchets for the current note. Press again to select the Span input",
	BeatInputSwitch:        "This selects the current part's beats which can be increased or decreased with +/-. Using this key combination again will move through selections Beats and Start Beats",
	CyclesInputSwitch:      "This selects the current part's cycles which can be increased or decreased with +/-. Using this key combination again will move through selections Cycles and Start Cycles",
	ToggleArrangementView:  "Open the arrangement view when closed. Focus the arrangement view while unfocused and open. Press enter to move focus back to the grid. While open and focused, close the arrangement view",
	Increase:               "Increase value of current selection (tempo, beats, cycles, accents, etc.) or tempo by 5 if no specific selection",
	Decrease:               "Decrease value of current selection (tempo, beats, cycles, accents, etc.) or tempo by 5 if no specific selection",
	Enter:                  "Confirm current action, move focus to the grid when elsewhere, or escape from visual mode",
	ToggleGateMode:         "Start Pattern Mode - Gate. Use the facilities of pattern mode to increase or decrease the gate values of the line",
	ToggleGateNoteMode:     "Toggle gate note mode",
	ToggleWaitMode:         "Start Pattern Mode - Wait. Use the facilities of pattern mode to increase or decrease the wait values of the line",
	ToggleWaitNoteMode:     "Toggle wait note mode",
	ToggleAccentMode:       "Start Pattern Mode - Accent. Use the facilities of pattern mode to increase or decrease the accent values of the line",
	ToggleAccentNoteMode:   "Toggle accent note mode",
	ToggleRatchetMode:      "Start Pattern Mode - Ratchet. Use the facilities of pattern mode to increase or decrease the ratchet values of the line",
	ToggleRatchetNoteMode:  "Toggle ratchet note mode",
	NextOverlay:            "Move to next overlay",
	PrevOverlay:            "Move to previous overlay",
	Save:                   "Save the current sequence. If not previously saved, you will be prompted to name the new file. The file will be saved in the directory from which you opened sq",
	SaveAs:                 "Save the current sequence with a new name",
	Undo:                   "Undo last action",
	Redo:                   "Redo last undone action",
	New:                    "Create a new sequence using the same template as the current sequence",
	ToggleVisualMode:       "Toggle visual selection",
	ToggleVisualLineMode:   "Toggle visual line selection",
	TogglePlayEdit:         "Toggle play edit mode. Press while playing to ensure the current overlay/part does not change while editing. Press again to allow changing",
	ToggleHideLines:        "Toggle hiding lines with no notes",
	NewLine:                "Create a new line with a value 1 greater than the previous line",
	NewSectionAfter:        "Create new section after the current section",
	NewSectionBefore:       "Create new section before the current section",
	ChangePart:             "Change the part of the section to either an existing part or a new part",
	NextSection:            "Move to the next section within the arrangement. If the next section is a group, then this mapping will move to the first section within that group",
	PrevSection:            "Move to the previous section. Move to the next section within the arrangement. If the previous section is a group, then this mapping will move to the last section within that group",
	NextTheme:              "Switch to next theme. A theme consists of the set of colors used to draw the sq application and the set of icons used to represent different accent levels",
	PrevTheme:              "Move to the previous theme. A theme consists of the set of colors used to draw the sq application and the set of icons used to represent different accent levels",
	Yank:                   "Copy current selection to buffer. Copies all values of a visual selection or the value under cursor if no visual selection",
	Mute:                   "Mute the current line. Midi messages will not be sent from this line when the line is muted",
	MuteAll:                "Mute all lines",
	UnMuteAll:              "Unmute all lines",
	Solo:                   "Solo the current line. Only midi messages from this line or other soloed lines will be sent",
	NoteAdd:                "Add note at current position",
	NoteRemove:             "Remove note at current position, and remove it from any stacked overlays if the current overlay is higher than the overlay of the current note",
	OverlayNoteRemove:      "Remove note from overlay at current position, allowing notes in lower layers to show through",
	AccentIncrease:         "Increase accent value for current note",
	AccentDecrease:         "Decrease accent value for current note",
	GateIncrease:           "Increases the gate value for current note. The gate corresponds to the length of the note",
	GateDecrease:           "Decreases the gate value for current note. The gate corresponds to the length of the note",
	GateBigIncrease:        "Increase gate value for current note by 8, or 1 full beat. The gate corresponds to the length of the note",
	GateBigDecrease:        "Decrease gate value for current note by 8, or 1 full beat. The gate corresponds to the length of the note",
	WaitIncrease:           "Increase the wait value for current note. The wait value is the time between the playback of the note's beat and the sending of the midi message. This is useful for creating a swing effect",
	WaitDecrease:           "Decrease the wait value for current note. The wait value is the time between the playback of the note's beat and the sending of the midi message. This is useful for creating a swing effect",
	ClearLine:              "Remove all notes from the current line from the current cursor position to the end",
	ClearOverlay:           "Remove all notes and actions from the current overlay layer",
	ClearAllOverlays:       "Clear all overlays",
	RemoveOverlay:          "Remove the current overlay",
	RatchetIncrease:        "Increase the number of hits evenly divided within the span of 1 beat",
	RatchetDecrease:        "Decrease the number of hits evenly divided within the span of 1 beat",
	ActionAddLineReset:     "Add line reset action to current line. When the playback cursor reaches this action, the playback cursor will reset to the first beat",
	ActionAddLineResetAll:  "Add reset action all to current line. When the playback cursor reaches this action, all playback cursors will reset to the first beat",
	ActionAddLineReverse:   "Add line reverse action to current line. When the playback cursor reaches this action, the playback will reverse for this line",
	ActionAddSkipBeat:      "Add skip beat all action to current line. When the playback cursor reaches this action, all the playback cursors will advance an additional beat",
	ActionAddLineBounce:    "Add line bounce action to current line. When the playback cursor reaches this action it will reverse direction, and reverse again when reaching the line beginning creating a bouncing effect",
	ActionAddLineBounceAll: "Add line bounce all action to current line. When the playback cursor reaches this action all playback cursors will reverse direction, and reverse again when reaching the line beginning creating a bouncing effect",
	ActionAddLineDelay:     "Add line delay action to current line",
	ActionAddSpecificValue: "Add specific value note to the grid. When cursor is above this note, +/- will affect the specific value of the note",
	SelectKeyLine:          "Selects the current line as the key line. The KeyCycle of the part is advanced when the cursor returns to the first beat",
	OverlayStackToggle:     "Toggle the behaviour of the current overlay layer between three options: No association, press up, press down",
	NumberPattern:          "Add/remove notes or increase/decrease values in a pattern",
	RotateRight:            "Rotate pattern right. On the current line shift all notes right of the cursor right by one beat. A note at the last beat will be moved to the cursor's beat",
	RotateLeft:             "Rotate pattern left. On the current line shift all notes right of the cursor left by one beat. A note at the cursor's beat will be moved to the last beat of the line",
	RotateUp:               "Rotate pattern up. In the current column shift all notes down by one line, with a note in the bottom line moving to the top line",
	RotateDown:             "Rotate pattern down. In the current column shift all notes down by one line, with a note in the bottom line moving to the top line",
	Paste:                  "Paste the buffer at the position of the cursor",
	MajorTriad:             "Add major triad chord",
	MinorTriad:             "Add minor triad chord",
	AugmentedTriad:         "Add augmented triad chord",
	DiminishedTriad:        "Add diminished triad chord",
	MinorSecond:            "Add minor second",
	MajorSecond:            "Add major second",
	MinorThird:             "Add minor third",
	MajorThird:             "Add major third",
	PerfectFourth:          "Add perfect fourth",
	AugFifth:               "Add augmented fifth",
	DimFifth:               "Add diminished fifth",
	PerfectFifth:           "Add perfect fifth",
	MajorSixth:             "Add major sixth",
	MinorSeventh:           "Add minor seventh",
	MajorSeventh:           "Add major seventh",
	Octave:                 "Add octave",
	MinorNinth:             "Add minor ninth",
	MajorNinth:             "Add major ninth",
	IncreaseInversions:     "Increase chord inversions",
	DecreaseInversions:     "Decrease chord inversions",
	ToggleChordMode:        "Toggle chord mode",
	ToggleMonoMode:         "Toggle mono mode",
	NextArpeggio:           "Next arpeggio pattern",
	PrevArpeggio:           "Previous arpeggio pattern",
	NextDouble:             "Next double pattern",
	PrevDouble:             "Previous double pattern",
	OmitRoot:               "Omit root note from chord",
	OmitSecond:             "Omit second note from chord",
	OmitThird:              "Omit third note from chord",
	OmitFourth:             "Omit fourth note from chord",
	OmitFifth:              "Omit fifth note from chord",
	OmitSixth:              "Omit sixth note from chord",
	OmitSeventh:            "Omit seventh note from chord",
	OmitOctave:             "Omit eighth note from chord",
	OmitNinth:              "Omit ninth note from chord",
	RemoveChord:            "Remove chord at current position",
	ConvertToNotes:         "Convert chord to individual notes",
	ReloadFile:             "Reload current file, any changes since the last save will be lost",
	MidiPanic:              "Send MIDI panic (all notes off)",
	IncreaseAllChannels:    "Increase all channels",
	DecreaseAllChannels:    "Decrease all channels",
	IncreaseAllNote:        "Increase all note values",
	DecreaseAllNote:        "Decrease all note values",
	ToggleTransmitting:     "Toggle transmitting MIDI messages",
	ToggleClockPreRoll:     "Toggle clock pre-roll",
	Euclidean:              "Apply Euclidean rhythm pattern to selection or pattern",
	Reverse:                "Reverse notes from cursor to end of line, or reverse notes within visual selection",
	Duplicate:              "Duplicate what is under the cursor to the next beat in the current line",
	ToggleBoundedLoop:      "Toggle bounded loop mode. When enabled, overlay playback loops between left and right bounds instead of the full sequence",
	ExpandLeftLoopBound:    "Expand the left loop bound one beat to the left, increasing the loop region size",
	ExpandRightLoopBound:   "Expand the right loop bound one beat to the right, increasing the loop region size",
	ContractLeftLoopBound:  "Contract the left loop bound one beat to the right, decreasing the loop region size",
	ContractRightLoopBound: "Contract the right loop bound one beat to the left, decreasing the loop region size",
}

type mappingKey [3]string
type registry map[OperationKey]Command

// KeysForCommand Function that gets keys for mappings by looking at the looping
// through the registry and returning the keys for the given command
// If the command is not found, it returns an empty string slice.
// This is used to get the keys for a given command in the mappings.
func KeysForCommand(command Command) []string {
	var keys []string
	for key, cmd := range allCommands() {
		if cmd == command {
			for _, k := range key.key {
				if k != "" {
					keys = append(keys, k)
				}
			}
			break
		}
	}
	return keys
}

// MappingInfo contains information about a single key mapping
type MappingInfo struct {
	Command     Command
	Name        string
	Keys        []string
	Description string
}

func (mi MappingInfo) GetKeys() string {
	if mi.Command == NumberPattern {
		return "0-9, !-("
	} else {
		return strings.Join(mi.Keys, ", ")
	}
}

// GetAllMappings returns all mappings with their names, keys, and descriptions
func GetAllMappings() []MappingInfo {
	// Create a map to collect all key bindings for each command
	commandKeys := make(map[Command][][3]string)

	for opKey, cmd := range allCommands() {
		// Skip commands with empty keys (they're catch-all handlers)
		if opKey.key == [3]string{} {
			continue
		}
		commandKeys[cmd] = append(commandKeys[cmd], opKey.key)
	}

	// Build the result slice
	var result []MappingInfo
	for cmd := range PurposePanic {
		keys, hasKeys := commandKeys[cmd]
		desc, hasDesc := CommandDescriptions[cmd]

		if hasKeys && hasDesc {
			// Format the keys
			var formattedKeys []string
			for _, key := range keys {
				var parts []string
				for _, k := range key {
					if k != "" {
						parts = append(parts, strings.Replace(k, " ", "<space>", 1))
					}
				}
				if len(parts) > 0 {
					formattedKeys = append(formattedKeys, strings.Join(parts, " + "))
				}
			}

			if len(formattedKeys) > 0 {
				result = append(result, MappingInfo{
					Name:        cmd.String(),
					Keys:        formattedKeys,
					Description: desc,
				})
			}
		}
	}

	return result
}

// String returns a string representation of the Command
func (c Command) String() string {
	names := []string{
		"HoldingKeys",
		"Quit",
		"CursorUp",
		"CursorDown",
		"CursorLeft",
		"CursorRight",
		"CursorLineStart",
		"CursorLineEnd",
		"CursorLastLine",
		"CursorFirstLine",
		"Escape",
		"PlayStop",
		"PlayPart",
		"PlayLoop",
		"PlayOverlayLoop",
		"PlayRecord",
		"PlayAlong",
		"TempoInputSwitch",
		"OverlayInputSwitch",
		"ModifyKeyInputSwitch",
		"SetupInputSwitch",
		"AccentInputSwitch",
		"RatchetInputSwitch",
		"BeatInputSwitch",
		"CyclesInputSwitch",
		"ToggleArrangementView",
		"Increase",
		"Decrease",
		"Enter",
		"ToggleGateMode",
		"ToggleGateNoteMode",
		"ToggleWaitMode",
		"ToggleWaitNoteMode",
		"ToggleAccentMode",
		"ToggleAccentNoteMode",
		"ToggleRatchetMode",
		"ToggleRatchetNoteMode",
		"NextOverlay",
		"PrevOverlay",
		"Save",
		"SaveAs",
		"Undo",
		"Redo",
		"New",
		"ToggleVisualMode",
		"ToggleVisualLineMode",
		"TogglePlayEdit",
		"ToggleHideLines",
		"NewLine",
		"NewSectionAfter",
		"NewSectionBefore",
		"ChangePart",
		"NextSection",
		"PrevSection",
		"NextTheme",
		"PrevTheme",
		"Yank",
		"Mute",
		"MuteAll",
		"UnMuteAll",
		"Solo",
		"NoteAdd",
		"NoteRemove",
		"OverlayNoteRemove",
		"AccentIncrease",
		"AccentDecrease",
		"GateIncrease",
		"GateDecrease",
		"GateBigIncrease",
		"GateBigDecrease",
		"WaitIncrease",
		"WaitDecrease",
		"ClearLine",
		"ClearOverlay",
		"ClearAllOverlays",
		"RemoveOverlay",
		"RatchetIncrease",
		"RatchetDecrease",
		"ActionAddLineReset",
		"ActionAddLineResetAll",
		"ActionAddLineReverse",
		"ActionAddSkipBeat",
		"ActionAddLineBounce",
		"ActionAddLineBounceAll",
		"ActionAddLineDelay",
		"ActionAddSpecificValue",
		"SelectKeyLine",
		"OverlayStackToggle",
		"NumberPattern",
		"RotateRight",
		"RotateLeft",
		"RotateUp",
		"RotateDown",
		"Paste",
		"MajorTriad",
		"MinorTriad",
		"AugmentedTriad",
		"DiminishedTriad",
		"MinorSecond",
		"MajorSecond",
		"MinorThird",
		"MajorThird",
		"PerfectFourth",
		"AugFifth",
		"DimFifth",
		"PerfectFifth",
		"MajorSixth",
		"MinorSeventh",
		"MajorSeventh",
		"Octave",
		"MinorNinth",
		"MajorNinth",
		"IncreaseInversions",
		"DecreaseInversions",
		"ToggleChordMode",
		"ToggleMonoMode",
		"NextArpeggio",
		"PrevArpeggio",
		"NextDouble",
		"PrevDouble",
		"OmitRoot",
		"OmitSecond",
		"OmitThird",
		"OmitFourth",
		"OmitFifth",
		"OmitSixth",
		"OmitSeventh",
		"OmitOctave",
		"OmitNinth",
		"RemoveChord",
		"ConvertToNotes",
		"ReloadFile",
		"OverlayKeyMessage",
		"ArrKeyMessage",
		"TextInputMessage",
		"ConfirmOverlayKey",
		"ConfirmRenamePart",
		"ConfirmFileName",
		"ConfirmSelectPart",
		"ConfirmChangePart",
		"ConfirmConfirmNew",
		"ConfirmConfirmReload",
		"ConfirmConfirmQuit",
		"ConfirmEuclidenHits",
		"MidiPanic",
		"IncreaseAllChannels",
		"DecreaseAllChannels",
		"IncreaseAllNote",
		"DecreaseAllNote",
		"ToggleTransmitting",
		"ToggleClockPreRoll",
		"PurposePanic",
		"ToggleBoundedLoop",
		"ExpandLeftLoopBound",
		"ExpandRightLoopBound",
		"ContractLeftLoopBound",
		"ContractRightLoopBound",
		"Euclidean",
		"Reverse",
		"Duplicate",
	}

	if c >= 0 && int(c) < len(names) {
		return names[c]
	}
	return "Unknown"
}

func allCommands() registry {
	// Combine all mappings into a single registry
	all := make(registry)
	for _, m := range []registry{mappings} {
		maps.Copy(all, m)
	}
	return all
}

type OperationKey struct {
	focus       operation.Focus
	selection   operation.Selection
	mode        operation.SequencerMode
	patternMode operation.PatternMode
	key         mappingKey
}

var mappings = registry{
	OperationKey{key: k("b", "p")}:                                          MidiPanic,
	OperationKey{focus: operation.FocusAny, key: k("space")}:                PlayStop,
	OperationKey{focus: operation.FocusAny, key: k("'", "space")}:           PlayOverlayLoop,
	OperationKey{focus: operation.FocusAny, key: k(":", "space")}:           PlayRecord,
	OperationKey{focus: operation.FocusAny, key: k(";", "space")}:           PlayAlong,
	OperationKey{focus: operation.FocusGrid, key: k("+")}:                   Increase,
	OperationKey{focus: operation.FocusGrid, key: k("=")}:                   Increase,
	OperationKey{focus: operation.FocusGrid, key: k("-")}:                   Decrease,
	OperationKey{selection: operation.SelectTempo, key: k("+")}:             Increase,
	OperationKey{selection: operation.SelectTempo, key: k("=")}:             Increase,
	OperationKey{selection: operation.SelectTempo, key: k("-")}:             Decrease,
	OperationKey{selection: operation.SelectTempoSubdivision, key: k("+")}:  Increase,
	OperationKey{selection: operation.SelectTempoSubdivision, key: k("=")}:  Increase,
	OperationKey{selection: operation.SelectTempoSubdivision, key: k("-")}:  Decrease,
	OperationKey{selection: operation.SelectPart, key: k("+")}:              Increase,
	OperationKey{selection: operation.SelectPart, key: k("=")}:              Increase,
	OperationKey{selection: operation.SelectPart, key: k("-")}:              Decrease,
	OperationKey{selection: operation.SelectChangePart, key: k("+")}:        Increase,
	OperationKey{selection: operation.SelectChangePart, key: k("=")}:        Increase,
	OperationKey{selection: operation.SelectChangePart, key: k("-")}:        Decrease,
	OperationKey{focus: operation.FocusGrid, key: k("Z")}:                   CursorLineStart,
	OperationKey{focus: operation.FocusGrid, key: k("z")}:                   CursorLineEnd,
	OperationKey{focus: operation.FocusGrid, key: k("b", "l")}:              CursorLastLine,
	OperationKey{focus: operation.FocusGrid, key: k("b", "f")}:              CursorFirstLine,
	OperationKey{focus: operation.FocusGrid, key: k("b", "d")}:              Duplicate,
	OperationKey{focus: operation.FocusGrid, key: k("b", "h")}:              ToggleHideLines,
	OperationKey{focus: operation.FocusGrid, key: k("b", "t")}:              ToggleTransmitting,
	OperationKey{focus: operation.FocusGrid, key: k("b", "c")}:              ToggleClockPreRoll,
	OperationKey{focus: operation.FocusGrid, key: k("b", "u")}:              Euclidean,
	OperationKey{focus: operation.FocusGrid, key: k("A")}:                   AccentIncrease,
	OperationKey{focus: operation.FocusGrid, key: k("C")}:                   ClearOverlay,
	OperationKey{focus: operation.FocusGrid, key: k("b", "C")}:              ClearAllOverlays,
	OperationKey{focus: operation.FocusGrid, key: k("D")}:                   RemoveOverlay,
	OperationKey{focus: operation.FocusGrid, key: k("G")}:                   GateIncrease,
	OperationKey{focus: operation.FocusGrid, key: k("E")}:                   GateBigIncrease,
	OperationKey{focus: operation.FocusGrid, key: k("J")}:                   RotateDown,
	OperationKey{focus: operation.FocusGrid, key: k("K")}:                   RotateUp,
	OperationKey{focus: operation.FocusGrid, key: k("H")}:                   RotateLeft,
	OperationKey{focus: operation.FocusGrid, key: k("L")}:                   RotateRight,
	OperationKey{focus: operation.FocusGrid, key: k("Y")}:                   SelectKeyLine,
	OperationKey{focus: operation.FocusGrid, key: k("M")}:                   Solo,
	OperationKey{focus: operation.FocusGrid, key: k("R")}:                   RatchetIncrease,
	OperationKey{focus: operation.FocusAny, key: k("U")}:                    Redo,
	OperationKey{focus: operation.FocusGrid, key: k("W")}:                   WaitIncrease,
	OperationKey{focus: operation.FocusGrid, key: k("[", "c")}:              PrevTheme,
	OperationKey{focus: operation.FocusAny, key: k("[", "s")}:               PrevSection,
	OperationKey{focus: operation.FocusGrid, key: k("]", "c")}:              NextTheme,
	OperationKey{focus: operation.FocusAny, key: k("]", "s")}:               NextSection,
	OperationKey{focus: operation.FocusGrid, key: k("g")}:                   GateDecrease,
	OperationKey{focus: operation.FocusGrid, key: k("e")}:                   GateBigDecrease,
	OperationKey{focus: operation.FocusAny, key: k("alt+space")}:            PlayLoop,
	OperationKey{focus: operation.FocusAny, key: k("ctrl+space")}:           PlayPart,
	OperationKey{focus: operation.FocusAny, key: k("ctrl+]")}:               NewSectionAfter,
	OperationKey{focus: operation.FocusGrid, key: k("ctrl+b")}:              BeatInputSwitch,
	OperationKey{focus: operation.FocusGrid, key: k("ctrl+k")}:              CyclesInputSwitch,
	OperationKey{focus: operation.FocusAny, key: k("ctrl+c")}:               ChangePart,
	OperationKey{focus: operation.FocusGrid, key: k("ctrl+e")}:              AccentInputSwitch,
	OperationKey{focus: operation.FocusGrid, key: k("ctrl+a")}:              ToggleArrangementView,
	OperationKey{focus: operation.FocusArrangementEditor, key: k("ctrl+a")}: ToggleArrangementView,
	OperationKey{focus: operation.FocusAny, key: k("ctrl+l")}:               NewLine,
	OperationKey{focus: operation.FocusAny, key: k("ctrl+n")}:               New,
	OperationKey{focus: operation.FocusGrid, key: k("ctrl+o")}:              OverlayInputSwitch,
	OperationKey{focus: operation.FocusGrid, key: k("ctrl+x")}:              ModifyKeyInputSwitch,
	OperationKey{focus: operation.FocusOverlayKey, key: k("ctrl+o")}:        OverlayInputSwitch,
	OperationKey{focus: operation.FocusAny, key: k("ctrl+p")}:               NewSectionBefore,
	OperationKey{focus: operation.FocusGrid, key: k("ctrl+d")}:              SetupInputSwitch,
	OperationKey{focus: operation.FocusAny, key: k("ctrl+t")}:               TempoInputSwitch,
	OperationKey{focus: operation.FocusAny, key: k("ctrl+u")}:               OverlayStackToggle,
	OperationKey{focus: operation.FocusAny, key: k("ctrl+s")}:               Save,
	OperationKey{focus: operation.FocusAny, key: k("ctrl+w")}:               SaveAs,
	OperationKey{focus: operation.FocusGrid, key: k("ctrl+y")}:              RatchetInputSwitch,
	OperationKey{focus: operation.FocusGrid, key: k("a")}:                   AccentDecrease,
	OperationKey{focus: operation.FocusGrid, key: k("c")}:                   ClearLine,
	OperationKey{focus: operation.FocusGrid, key: k("d")}:                   NoteRemove,
	OperationKey{focus: operation.FocusGrid, key: k("b", "e")}:              TogglePlayEdit,
	OperationKey{focus: operation.FocusGrid, key: k("f")}:                   NoteAdd,
	OperationKey{focus: operation.FocusGrid, key: k("b", "r")}:              ReloadFile,
	OperationKey{focus: operation.FocusGrid, key: k("b", "v")}:              ActionAddSpecificValue,
	OperationKey{focus: operation.FocusGrid, key: k("h")}:                   CursorLeft,
	OperationKey{focus: operation.FocusGrid, key: k("j")}:                   CursorDown,
	OperationKey{focus: operation.FocusGrid, key: k("k")}:                   CursorUp,
	OperationKey{focus: operation.FocusGrid, key: k("l")}:                   CursorRight,
	OperationKey{focus: operation.FocusGrid, key: k("m")}:                   Mute,
	OperationKey{focus: operation.FocusGrid, key: k("b", "m")}:              MuteAll,
	OperationKey{focus: operation.FocusGrid, key: k("b", "M")}:              UnMuteAll,
	OperationKey{focus: operation.FocusGrid, key: k("o")}:                   ToggleChordMode,
	OperationKey{focus: operation.FocusGrid, key: k("O")}:                   ToggleMonoMode,
	OperationKey{focus: operation.FocusGrid, key: k("n", "a")}:              ToggleAccentMode,
	OperationKey{focus: operation.FocusGrid, key: k("n", "A")}:              ToggleAccentNoteMode,
	OperationKey{focus: operation.FocusGrid, key: k("n", "w")}:              ToggleWaitMode,
	OperationKey{focus: operation.FocusGrid, key: k("n", "W")}:              ToggleWaitNoteMode,
	OperationKey{focus: operation.FocusGrid, key: k("n", "g")}:              ToggleGateMode,
	OperationKey{focus: operation.FocusGrid, key: k("n", "G")}:              ToggleGateNoteMode,
	OperationKey{focus: operation.FocusGrid, key: k("n", "r")}:              ToggleRatchetMode,
	OperationKey{focus: operation.FocusGrid, key: k("n", "R")}:              ToggleRatchetNoteMode,
	OperationKey{focus: operation.FocusGrid, key: k("n", "P")}:              PurposePanic,
	OperationKey{focus: operation.FocusGrid, key: k("n", "v")}:              Reverse,
	OperationKey{focus: operation.FocusGrid, key: k("p")}:                   Paste,
	OperationKey{focus: operation.FocusAny, key: k("q")}:                    Quit,
	OperationKey{focus: operation.FocusGrid, key: k("r")}:                   RatchetDecrease,
	OperationKey{focus: operation.FocusGrid, key: k("s", "s")}:              ActionAddLineReset,
	OperationKey{focus: operation.FocusGrid, key: k("s", "S")}:              ActionAddLineResetAll,
	OperationKey{focus: operation.FocusGrid, key: k("s", "b")}:              ActionAddLineBounce,
	OperationKey{focus: operation.FocusGrid, key: k("s", "B")}:              ActionAddLineBounceAll,
	OperationKey{focus: operation.FocusGrid, key: k("s", "k")}:              ActionAddSkipBeat,
	OperationKey{focus: operation.FocusGrid, key: k("s", "r")}:              ActionAddLineReverse,
	OperationKey{focus: operation.FocusGrid, key: k("s", "z")}:              ActionAddLineDelay,
	OperationKey{focus: operation.FocusAny, key: k("u")}:                    Undo,
	OperationKey{focus: operation.FocusGrid, key: k("v")}:                   ToggleVisualMode,
	OperationKey{focus: operation.FocusGrid, key: k("V")}:                   ToggleVisualLineMode,
	OperationKey{focus: operation.FocusGrid, key: k("w")}:                   WaitDecrease,
	OperationKey{focus: operation.FocusGrid, key: k("x")}:                   OverlayNoteRemove,
	OperationKey{focus: operation.FocusGrid, key: k("y")}:                   Yank,
	OperationKey{focus: operation.FocusGrid, key: k("{")}:                   NextOverlay,
	OperationKey{focus: operation.FocusGrid, key: k("}")}:                   PrevOverlay,
	OperationKey{focus: operation.FocusGrid, key: k("enter")}:               Enter,
	OperationKey{focus: operation.FocusGrid, key: k("n", "l")}:              ToggleBoundedLoop,
	OperationKey{focus: operation.FocusGrid, key: k("<")}:                   ExpandLeftLoopBound,
	OperationKey{focus: operation.FocusGrid, key: k(">")}:                   ExpandRightLoopBound,
	OperationKey{focus: operation.FocusGrid, key: k(",")}:                   ContractLeftLoopBound,
	OperationKey{focus: operation.FocusGrid, key: k(".")}:                   ContractRightLoopBound,
	OperationKey{focus: operation.FocusArrangementEditor, key: k("enter")}:  Enter,
	OperationKey{focus: operation.FocusOverlayKey, key: k("enter")}:         ConfirmOverlayKey,
	OperationKey{selection: operation.SelectRenamePart, key: k("enter")}:    ConfirmRenamePart,
	OperationKey{selection: operation.SelectFileName, key: k("enter")}:      ConfirmFileName,
	OperationKey{selection: operation.SelectPart, key: k("enter")}:          ConfirmSelectPart,
	OperationKey{selection: operation.SelectChangePart, key: k("enter")}:    ConfirmChangePart,
	OperationKey{selection: operation.SelectConfirmNew, key: k("enter")}:    ConfirmConfirmNew,
	OperationKey{selection: operation.SelectConfirmReload, key: k("enter")}: ConfirmConfirmReload,
	OperationKey{selection: operation.SelectConfirmQuit, key: k("enter")}:   ConfirmConfirmQuit,
	OperationKey{selection: operation.SelectEuclideanHits, key: k("enter")}: ConfirmEuclidenHits,
	OperationKey{selection: operation.SelectFileName, key: k("esc")}:        Escape,
	OperationKey{selection: operation.SelectSetupChannel, key: k("J")}:      DecreaseAllChannels,
	OperationKey{selection: operation.SelectSetupChannel, key: k("K")}:      IncreaseAllChannels,
	OperationKey{selection: operation.SelectSetupValue, key: k("J")}:        DecreaseAllNote,
	OperationKey{selection: operation.SelectSetupValue, key: k("K")}:        IncreaseAllNote,
	OperationKey{key: k("esc")}:                                             Escape,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("t", "M")}: MajorTriad,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("t", "m")}: MinorTriad,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("t", "d")}: DiminishedTriad,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("t", "a")}: AugmentedTriad,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("7", "m")}: MinorSeventh,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("7", "M")}: MajorSeventh,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("5", "a")}: AugFifth,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("5", "d")}: DimFifth,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("5", "p")}: PerfectFifth,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("2", "m")}: MinorSecond,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("2", "M")}: MajorSecond,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("3", "m")}: MinorThird,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("3", "M")}: MajorThird,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("4", "p")}: PerfectFourth,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("6", "M")}: MajorSixth,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("8", "p")}: Octave,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("9", "m")}: MinorNinth,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("9", "M")}: MajorNinth,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("[", "i")}: DecreaseInversions,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("]", "i")}: IncreaseInversions,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("1", "o")}: OmitRoot,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("2", "o")}: OmitSecond,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("3", "o")}: OmitThird,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("4", "o")}: OmitFourth,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("5", "o")}: OmitFifth,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("6", "o")}: OmitSixth,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("7", "o")}: OmitSeventh,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("8", "o")}: OmitOctave,
	OperationKey{focus: operation.FocusGrid, selection: operation.SelectGrid, patternMode: operation.PatternFill, mode: operation.SeqModeChord, key: k("9", "o")}: OmitNinth,
	OperationKey{focus: operation.FocusGrid, mode: operation.SeqModeChord, key: k("X")}:                                                                           RemoveChord,
	OperationKey{focus: operation.FocusGrid, mode: operation.SeqModeChord, key: k("]", "p")}:                                                                      NextArpeggio,
	OperationKey{focus: operation.FocusGrid, mode: operation.SeqModeChord, key: k("[", "p")}:                                                                      PrevArpeggio,
	OperationKey{focus: operation.FocusGrid, mode: operation.SeqModeChord, key: k("]", "d")}:                                                                      NextDouble,
	OperationKey{focus: operation.FocusGrid, mode: operation.SeqModeChord, key: k("[", "d")}:                                                                      PrevDouble,
	OperationKey{focus: operation.FocusGrid, mode: operation.SeqModeChord, key: k("n", "n")}:                                                                      ConvertToNotes,
	OperationKey{focus: operation.FocusOverlayKey, key: [3]string{}}:                                                                                              OverlayKeyMessage,
	OperationKey{selection: operation.SelectRenamePart, key: [3]string{}}:                                                                                         TextInputMessage,
	OperationKey{selection: operation.SelectFileName, key: [3]string{}}:                                                                                           TextInputMessage,
	OperationKey{focus: operation.FocusArrangementEditor, selection: operation.SelectFileName, key: [3]string{}}:                                                  TextInputMessage,
	OperationKey{focus: operation.FocusArrangementEditor, key: [3]string{}}:                                                                                       ArrKeyMessage,
	OperationKey{focus: operation.FocusArrangementEditor, key: k("'")}:                                                                                            HoldingKeys,
}

func k(x ...string) [3]string {
	if len(x) <= 3 {
		combo := [3]string{}
		copy(combo[:], x)
		return combo
	} else {
		panic("Can't have key combos longer than 3")
	}
}

var holdKeysTime = time.Millisecond * 500

func ResetKeycombo() {
	mutex.Lock()
	defer mutex.Unlock()
	if timer != nil {
		timer.Stop()
	}
	Keycombo = make([]tea.KeyPressMsg, 0, 3)
}

func ProcessKey(key tea.KeyPressMsg, focus operation.Focus, selection operation.Selection, seqtype operation.SequencerMode, patternMode operation.PatternMode) Mapping {
	mutex.Lock()
	defer mutex.Unlock()
	if len(Keycombo) < 3 {
		Keycombo = append(Keycombo, key)
	} else {
		Keycombo = slices.Delete(Keycombo, 0, 1)
		Keycombo = append(Keycombo, key)
	}

	if timer != nil {
		timer.Stop()
	}

	var mk mappingKey
	for i, msg := range Keycombo {
		mk[i] = msg.String()
	}

	operationKeys := []OperationKey{
		// Route enter and esc to the textInput mappings
		ToMappingKey(mk, operation.FocusAny, selection, operation.SeqModeAny, operation.PatternAny),
		// Route any text keys to the textInput
		ToMappingKey([3]string{}, operation.FocusAny, selection, operation.SeqModeAny, operation.PatternAny),
		// For mappings good in any focus, like Play
		ToMappingKey(mk, operation.FocusAny, operation.SelectAny, operation.SeqModeAny, operation.PatternAny),
		// Handle focus specific commands handled by ui
		ToMappingKey(mk, focus, operation.SelectAny, operation.SeqModeAny, operation.PatternAny),
		// Route any text keys to other focuses
		ToMappingKey([3]string{}, focus, operation.SelectAny, operation.SeqModeAny, operation.PatternAny),
		// Route selection specific mappings to those selections.
		ToMappingKey(mk, focus, selection, operation.SeqModeAny, operation.PatternAny),
		ToMappingKey(mk, focus, operation.SelectAny, seqtype, patternMode),
		ToMappingKey(mk, focus, operation.SelectAny, seqtype, operation.PatternAny),
		ToMappingKey(mk, operation.FocusGrid, operation.SelectGrid, seqtype, operation.PatternFill),
	}

	var command Command
	var exists bool

	for _, opKey := range operationKeys {
		command, exists = mappings[opKey]
		if exists {
			break
		}
	}

	if !exists && len(Keycombo) == 1 && seqtype != operation.SeqModeChord && key.String() >= "0" && key.String() <= "9" {
		command = NumberPattern
		exists = true
	}

	if !exists && len(Keycombo) == 1 && seqtype != operation.SeqModeChord && IsShiftSymbol(key.String()) {
		command = NumberPattern
		exists = true
	}

	if !exists {
		timer = time.AfterFunc(holdKeysTime, func() {
			mutex.Lock()
			defer mutex.Unlock()
			Keycombo = make([]tea.KeyPressMsg, 0, 3)
		})
	}

	if exists && HoldingKeys == command {
		return Mapping{HoldingKeys, key.String()}
	} else if exists {
		Keycombo = make([]tea.KeyPressMsg, 0, 3)
		return Mapping{command, key.String()}
	} else {
		return Mapping{HoldingKeys, key.String()}
	}
}

func IsShiftSymbol(symbol string) bool {
	return slices.Contains([]string{"!", "@", "#", "$", "%", "^", "&", "*", "("}, symbol)
}

func ToMappingKey(mk [3]string, focus operation.Focus, selection operation.Selection, seqMode operation.SequencerMode, patternMode operation.PatternMode) OperationKey {

	return OperationKey{
		key:         mk,
		focus:       focus,
		selection:   selection,
		mode:        seqMode,
		patternMode: patternMode,
	}
}

func MappingToNumber(mapping Mapping) (int, bool) {
	if mapping.LastValue >= "0" && mapping.LastValue <= "9" {
		beatInterval, err := strconv.ParseInt(mapping.LastValue, 0, 8)
		if err != nil {
			return 0, false
		}
		return int(beatInterval), true
	}
	return 0, false
}
