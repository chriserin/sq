package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math/rand"
	"os"
	"runtime"
	"slices"
	"strconv"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chriserin/sq/internal/arrangement"
	"github.com/chriserin/sq/internal/beats"
	"github.com/chriserin/sq/internal/config"
	"github.com/chriserin/sq/internal/grid"
	"github.com/chriserin/sq/internal/mappings"
	"github.com/chriserin/sq/internal/notereg"
	"github.com/chriserin/sq/internal/operation"
	"github.com/chriserin/sq/internal/overlaykey"
	"github.com/chriserin/sq/internal/overlays"
	"github.com/chriserin/sq/internal/playstate"
	"github.com/chriserin/sq/internal/seqmidi"
	"github.com/chriserin/sq/internal/sequence"
	themes "github.com/chriserin/sq/internal/themes"
	"github.com/chriserin/sq/internal/theory"
	"github.com/chriserin/sq/internal/timing"
)

var timingChannel chan timing.TimingMsg
var updateChannel chan beats.ModelMsg

func init() {
	timingChannel = timing.GetTimingChannel()
}

func Key(help string, keyboardKey ...string) key.Binding {
	return key.NewBinding(key.WithKeys(keyboardKey...), key.WithHelp(keyboardKey[0], help))
}

type overlayKey = overlaykey.OverlayPeriodicity

type gridKey = grid.GridKey

func GK(line uint8, beat uint8) gridKey {
	return gridKey{
		Line: line,
		Beat: beat,
	}
}

type note = grid.Note
type action = grid.Action

var zeronote note

type VisualSelection struct {
	visualMode operation.VisualMode
	anchor     gridKey
	lead       gridKey
}

func (va VisualSelection) Bounds(totalBeats uint8) grid.Bounds {
	switch va.visualMode {
	case operation.VisualBlock:
		return grid.InitBounds(va.anchor, va.lead)
	case operation.VisualLine:
		return grid.InitBounds(GK(va.anchor.Line, 0), GK(va.lead.Line, totalBeats-1))
	}
	panic(fmt.Sprintf("no visual area for mode: %d", va.visualMode))
}

type model struct {
	firstDigitApplied     bool
	hasUIFocus            bool
	transmitterConnected  bool
	logFileAvailable      bool
	playEditing           bool
	showArrangementView   bool
	hideEmptyLines        bool
	modifyKey             bool
	transmitting          bool
	clockPreRoll          bool
	euclideanHits         uint8
	ratchetCursor         uint8
	temporaryNoteValue    uint8
	sectionSideIndicator  SectionSide
	selectionIndicator    operation.Selection
	patternMode           operation.PatternMode
	midiLoopMode          timing.MidiLoopMode
	gridCursor            gridKey
	visualSelection       VisualSelection
	partSelectorIndex     int
	needsWrite            int
	editingPartID         int
	focus                 operation.Focus
	cancel                context.CancelFunc
	currentOverlay        *overlays.Overlay
	logFile               *os.File
	undoStack             UndoStack
	redoStack             UndoStack
	yankBuffer            Buffer
	lockReceiverChannel   chan bool
	unlockReceiverChannel chan bool
	errChan               chan error
	currentError          error
	currentViewError      error
	theme                 string
	filename              string
	help                  help.Model
	cursor                cursor.Model
	overlayKeyEdit        overlaykey.Model
	arrangement           arrangement.Model
	textInput             textinput.Model
	midiConnection        *seqmidi.MidiConnection
	activeChord           overlays.OverlayChord
	temporaryState        temporaryState
	// play state
	playState playstate.PlayState
	// save everything below here
	definition sequence.Sequence
}

func (m *model) SetGridCursor(key gridKey) {
	currentNote, currentExists := m.CurrentNote()
	currentPosition := m.gridCursor
	m.gridCursor = key
	newNote, noteExists := m.CurrentNote()
	if currentExists && currentNote.Action == grid.ActionSpecificValue && currentPosition != key {
		m.PushSpecificValueUndoable(currentPosition, m.temporaryNoteValue, currentNote.AccentIndex)
		m.SetSelectionIndicator(operation.SelectGrid)
	} else if noteExists && newNote.Action == grid.ActionSpecificValue {
		m.RecordSpecificValue(newNote.AccentIndex)
		m.SetSelectionIndicator(operation.SelectSpecificValue)
	}
}

func (m *model) RecordSpecificValue(value uint8) {
	m.temporaryNoteValue = value
}

func (m *model) RecordSpecificValueUndo() {
	currentNote, noteExists := m.CurrentNote()
	if noteExists && currentNote.Action == grid.ActionSpecificValue {
		m.PushSpecificValueUndoable(m.gridCursor, m.temporaryNoteValue, currentNote.AccentIndex)
	}
}

func (m *model) PushSpecificValueUndoable(position gridKey, oldValue, newValue uint8) {
	if oldValue != newValue {
		m.PushUndoables(
			UndoSpecificValue{
				ArrCursor:      m.arrangement.Cursor,
				overlayKey:     m.currentOverlay.Key,
				cursorPosition: position,
				specificValue:  oldValue,
			},
			UndoSpecificValue{
				ArrCursor:      m.arrangement.Cursor,
				overlayKey:     m.currentOverlay.Key,
				cursorPosition: position,
				specificValue:  newValue,
			},
		)
	}
}

func (m *model) UnsetActiveChord() {
	m.activeChord = overlays.OverlayChord{}
}

func (m *model) SetCurrentError(err error) {
	m.currentError = err
	m.selectionIndicator = operation.SelectError
	m.LogError(err)
}

func (m *model) SetCurrentViewError(err error) {
	m.currentViewError = err
	m.LogError(err)
}

func (m *model) ResetCurrentOverlay() {
	if m.playState.Playing && m.playEditing {
		return
	}
	currentNode := m.arrangement.Cursor.GetCurrentNode()
	if currentNode != nil && currentNode.IsEndNode() {
		partID := currentNode.Section.Part
		if len(*m.definition.Parts) > partID {
			currentCycles := (*m.playState.Iterations)[m.arrangement.CurrentNode()]
			playingOverlay := m.CurrentPart().Overlays.HighestMatchingOverlay(currentCycles)
			m.currentOverlay = playingOverlay
			if m.focus != operation.FocusOverlayKey {
				m.overlayKeyEdit.SetOverlayKey(m.currentOverlay.Key)
			}
		}
	}
}

type SectionSide uint8

const (
	SectionAfter SectionSide = iota
	SectionBefore
)

type GridNote struct {
	gridKey gridKey
	note    note
}

func (m *model) PushArrUndo(arrundo arrangement.Undo) {
	m.PushUndoables(UndoArrangement{arrUndo: arrundo.Undo}, UndoArrangement{arrUndo: arrundo.Redo})
}

func (m *model) PushUndoables(undo Undoable, redo Undoable) {
	if m.undoStack == EmptyStack {
		m.undoStack = UndoStack{
			undo: undo,
			redo: redo,
			next: nil,
			id:   rand.Int(),
		}
	} else {
		pusheddown := m.undoStack
		lastin := UndoStack{
			undo: undo,
			redo: redo,
			next: &pusheddown,
			id:   rand.Int(),
		}
		m.undoStack = lastin
	}
}

func (m *model) PushUndo(undo UndoStack) {
	if m.undoStack == EmptyStack {
		undo.next = nil
		m.undoStack = undo
	} else {
		pusheddown := m.undoStack
		undo.next = &pusheddown
		m.undoStack = undo
	}
}

func (m *model) PushRedo(redo UndoStack) {
	if m.redoStack == EmptyStack {
		redo.next = nil
		m.redoStack = redo
	} else {
		pusheddown := m.redoStack
		redo.next = &pusheddown
		m.redoStack = redo
	}
}

func (m *model) ResetRedo() {
	m.redoStack = EmptyStack
}

func (m *model) PopUndo() UndoStack {
	firstout := m.undoStack
	if firstout != EmptyStack && firstout.next != nil {
		m.undoStack = *m.undoStack.next
	} else {
		m.undoStack = EmptyStack
	}
	return firstout
}

func (m *model) PopRedo() UndoStack {
	firstout := m.redoStack
	if firstout != EmptyStack && firstout.next != nil {
		m.redoStack = *m.redoStack.next
	} else {
		m.redoStack = EmptyStack
	}
	return firstout
}

func (m *model) Undo() UndoStack {
	undoStack := m.PopUndo()
	if undoStack != EmptyStack {
		location := undoStack.undo.ApplyUndo(m)
		if location.ApplyLocation {
			m.ApplyLocation(location)
		}
	}
	return undoStack
}

func (m *model) Redo() UndoStack {
	undoStack := m.PopRedo()
	if undoStack != EmptyStack {
		location := undoStack.redo.ApplyUndo(m)
		if location.ApplyLocation {
			m.ApplyLocation(location)
		}
	}
	return undoStack
}

func (m *model) ApplyLocation(location Location) {
	m.SetGridCursor(location.GridKey)
	overlay := m.CurrentPart().Overlays.FindOverlay(location.OverlayKey)
	if overlay == nil {
		m.currentOverlay = m.CurrentPart().Overlays
	} else {
		m.currentOverlay = overlay
	}
	m.overlayKeyEdit.SetOverlayKey(m.currentOverlay.Key)
	m.focus = operation.FocusGrid
	m.arrangement.Focus = false
}

type temporaryState struct {
	lines        []grid.LineDefinition
	tempo        int
	subdivisions int
	accents      sequence.PatternAccents
	beats        uint8
	active       bool
}

type StateDiff struct {
	LinesChanged        bool
	TempoChanged        bool
	SubdivisionsChanged bool
	AccentsChanged      bool
	OldTempo            int
	NewTempo            int
	OldSubdivisions     int
	NewSubdivisions     int
	OldAccents          sequence.PatternAccents
	NewAccents          sequence.PatternAccents
	OldLines            []grid.LineDefinition
	NewLines            []grid.LineDefinition
}

func (s StateDiff) Changed() bool {
	return s.LinesChanged || s.TempoChanged || s.SubdivisionsChanged || s.AccentsChanged
}

func (s StateDiff) Reverse() StateDiff {
	return StateDiff{
		LinesChanged:        s.LinesChanged,
		TempoChanged:        s.TempoChanged,
		SubdivisionsChanged: s.SubdivisionsChanged,
		AccentsChanged:      s.AccentsChanged,
		OldTempo:            s.NewTempo,
		NewTempo:            s.OldTempo,
		OldSubdivisions:     s.NewSubdivisions,
		NewSubdivisions:     s.OldSubdivisions,
		OldAccents:          s.NewAccents,
		NewAccents:          s.OldAccents,
		OldLines:            s.NewLines,
		NewLines:            s.OldLines,
	}
}

func (s StateDiff) Apply(m *model) {
	if s.AccentsChanged {
		m.definition.Accents = s.NewAccents
	}
	if s.LinesChanged {
		m.definition.Lines = s.NewLines
	}
	if s.TempoChanged {
		m.definition.Tempo = s.NewTempo
		m.SyncTempo()
	}
	if s.SubdivisionsChanged {
		m.definition.Subdivisions = s.NewSubdivisions
	}
}

func createStateDiff(definition *sequence.Sequence, temporary *temporaryState) StateDiff {
	diff := StateDiff{}

	if !slices.Equal(definition.Lines, temporary.lines) {
		diff.LinesChanged = true
		diff.OldLines = definition.Lines
		diff.NewLines = temporary.lines
	}

	if definition.Tempo != temporary.tempo {
		diff.TempoChanged = true
		diff.OldTempo = definition.Tempo
		diff.NewTempo = temporary.tempo
	}

	if definition.Subdivisions != temporary.subdivisions {
		diff.SubdivisionsChanged = true
		diff.OldSubdivisions = definition.Subdivisions
		diff.NewSubdivisions = temporary.subdivisions
	}

	if !definition.Accents.Equal(&temporary.accents) {
		diff.AccentsChanged = true
		diff.OldAccents = definition.Accents
		diff.NewAccents = temporary.accents
	}

	return diff
}

type errorMsg struct {
	error error
}
type viewPanicMsg struct {
	error error
}

func (m model) SyncTempo() {
	go func() {
		timingChannel <- timing.TempoMsg{
			Tempo:        m.definition.Tempo,
			Subdivisions: m.definition.Subdivisions,
		}
	}()
}

func (m *model) EnsureOverlay() {
	m.EnsureOverlayWithKey(m.overlayKeyEdit.GetKey())
}

func (m model) FindOverlay(key overlayKey) *overlays.Overlay {
	return m.CurrentPart().Overlays.FindOverlay(key)
}

func (m *model) EnsureOverlayWithKey(key overlayKey) {
	overlay := m.FindOverlay(key)
	if overlay != nil {
		m.currentOverlay = overlay
		return
	}

	partID := m.CurrentPartID()
	var overlayToInsert *overlays.Overlay
	if m.modifyKey && m.currentOverlay.Key != overlaykey.ROOT {
		overlayToInsert = m.currentOverlay
		(*m.definition.Parts)[partID].Overlays = (*m.definition.Parts)[partID].Overlays.Remove(m.currentOverlay.Key)
		m.modifyKey = false
	} else {
		overlayToInsert = overlays.InitOverlay(key, nil)
	}

	overlayToInsert.Key = key
	if m.CurrentPart().Overlays != nil {
		var newOverlay = m.CurrentPart().Overlays.Insert(key, overlayToInsert)
		(*m.definition.Parts)[partID].Overlays = newOverlay
		m.currentOverlay = newOverlay.FindOverlay(key)
	} else {
		(*m.definition.Parts)[partID].Overlays = overlayToInsert
	}
}

func (m model) VisualSelectionBounds() grid.Bounds {
	return m.visualSelection.Bounds(m.CurrentPart().Beats)
}

func (m model) PatternBounds() grid.Bounds {
	return grid.Bounds{
		Top:    0,
		Right:  m.CurrentPart().Beats - 1,
		Bottom: uint8(len(m.definition.Lines)),
		Left:   0,
	}
}

func (m model) PasteBounds() grid.Bounds {
	if m.visualSelection.visualMode == operation.VisualNone {
		return m.PatternBounds()
	} else {
		return m.VisualSelectionBounds()
	}
}

func (m model) YankBounds() grid.Bounds {
	if m.visualSelection.visualMode == operation.VisualNone {
		return grid.InitBounds(m.gridCursor, m.gridCursor)
	} else {
		return m.VisualSelectionBounds()
	}
}

func (m model) InVisualSelection(key gridKey) bool {
	return m.VisualSelectionBounds().InBounds(key)
}

func (m model) VisualSelectedGridKeys() []gridKey {
	if m.visualSelection.visualMode == operation.VisualNone {
		return []gridKey{m.gridCursor}
	} else {
		return m.visualSelection.Bounds(m.CurrentPart().Beats).GridKeys()
	}
}

func (m *model) AddNote() {
	keys := m.VisualSelectedGridKeys()
	for _, k := range keys {
		m.currentOverlay.SetNote(k, grid.InitNote())
	}
}

func (m *model) AddAction(act action) {
	keys := m.VisualSelectedGridKeys()
	for _, k := range keys {
		m.currentOverlay.SetNote(k, grid.InitActionNote(act))
	}
}

func (m *model) RemoveNote() {
	keys := m.VisualSelectedGridKeys()
	for _, k := range keys {
		m.currentOverlay.SetNote(k, zeronote)
	}
}

func (m *model) OverlayRemoveNote() {
	keys := m.VisualSelectedGridKeys()
	for _, gridKey := range keys {
		m.currentOverlay.RemoveNote(gridKey)
	}
}

func (m *model) RatchetModify(modifier int8) {
	modifyFunc := func(key gridKey, currentNote note) {
		m.currentOverlay.SetNote(key, currentNote.IncrementRatchet(modifier))
	}
	m.Modify(modifyFunc)
}

func (m *model) EnsureRatchetCursorVisible() {
	currentNote, _ := m.CurrentNote()
	if m.ratchetCursor > currentNote.Ratchets.Length {
		m.ratchetCursor = m.ratchetCursor - 1
	}
}

func (m *model) IncrementCC() {
	note := m.definition.Lines[m.gridCursor.Line].Note
	for i := note + 1; i <= 127; i++ {
		_, exists := config.FindCC(i, m.definition.Instrument)
		if exists {
			m.definition.Lines[m.gridCursor.Line].Note = i
			return
		}
	}
}

func (m *model) DecrementCC() {
	note := m.definition.Lines[m.gridCursor.Line].Note
	for i := note - 1; i != 255; i-- {
		_, exists := config.FindCC(i, m.definition.Instrument)
		if exists {
			m.definition.Lines[m.gridCursor.Line].Note = i
			return
		}
	}
}

func (m *model) IncreaseSpan() {
	currentNote, _ := m.CurrentNote()
	if currentNote != zeronote && currentNote.Action == grid.ActionNothing {
		span := currentNote.Ratchets.Span
		if span < 8 {
			currentNote.Ratchets.Span = span + 1
		}
		m.currentOverlay.SetNote(m.gridCursor, currentNote)
	}
}

func (m *model) DecreaseSpan() {
	currentNote, _ := m.CurrentNote()
	if currentNote != zeronote && currentNote.Action == grid.ActionNothing {
		span := currentNote.Ratchets.Span
		if span > 0 {
			currentNote.Ratchets.Span = span - 1
		}
		m.currentOverlay.SetNote(m.gridCursor, currentNote)
	}
}

func (m *model) IncreaseAccentEnd() {
	end := m.definition.Accents.End

	if end < 127 && end < m.definition.Accents.Start-1 {
		m.definition.Accents.End = m.definition.Accents.End + 1
		m.definition.Accents.ReCalc()
	}
}

func (m *model) DecreaseAccentEnd() {
	end := m.definition.Accents.End

	if end > 0 {
		m.definition.Accents.End = m.definition.Accents.End - 1
		m.definition.Accents.ReCalc()
	}
}

func (m *model) IncreaseAccentStart() {
	if m.definition.Accents.Start < 127 {
		m.definition.Accents.Start = m.definition.Accents.Start + 1
		m.definition.Accents.ReCalc()
	}
}

func (m *model) DecreaseAccentStart() {
	start := m.definition.Accents.Start

	if start > 2 && start > m.definition.Accents.End+1 {
		m.definition.Accents.Start = m.definition.Accents.Start - 1
		m.definition.Accents.ReCalc()
	}
}

func (m *model) DecreaseAccentTarget() {
	m.definition.Accents.Target = (m.definition.Accents.Target + 1) % 2
}

func (m *model) IncreaseTempo(amount int) {
	newAmount := m.definition.Tempo + amount
	if newAmount <= 300 {
		m.definition.Tempo = newAmount
		m.SyncTempo()
	} else if m.definition.Tempo == 300 {
		// do nothing if already at 300
	} else if newAmount > 300 {
		m.definition.Tempo = 300
		m.SyncTempo()
	}
}

func (m *model) DecreaseTempo(amount int) {
	newAmount := m.definition.Tempo - amount
	if newAmount > 30 {
		m.definition.Tempo = newAmount
		m.SyncTempo()
	} else if m.definition.Tempo == 30 {
		// do nothing if already at 30
	} else if newAmount < 30 {
		m.definition.Tempo = 30
		m.SyncTempo()
	}
}

func (m *model) IncreaseBeats() {
	newBeats := m.CurrentPart().Beats + 1
	if newBeats < 128 {
		(*m.definition.Parts)[m.CurrentPartID()].Beats = newBeats
	}
}

func (m *model) DecreaseBeats() {
	newBeats := int(m.CurrentPart().Beats) - 1
	if newBeats >= 1 {
		(*m.definition.Parts)[m.CurrentPartID()].Beats = uint8(newBeats)
		if m.gridCursor.Beat >= uint8(newBeats) {
			m.SetGridCursor(gridKey{
				Line: m.gridCursor.Line,
				Beat: uint8(newBeats - 1),
			})
		}
	}
}

func (m *model) IncreaseStartBeats() {
	currentNode := m.arrangement.Cursor.GetCurrentNode()
	partBeats := m.CurrentPart().Beats
	currentNode.Section.IncreaseStartBeats(int(partBeats))
}

func (m *model) DecreaseStartBeats() {
	currentNode := m.arrangement.Cursor.GetCurrentNode()
	currentNode.Section.DecreaseStartBeats()
}

func (m *model) IncreaseStartCycles() {
	currentNode := m.arrangement.Cursor.GetCurrentNode()
	currentNode.Section.IncreaseStartCycles()
}

func (m *model) DecreaseStartCycles() {
	currentNode := m.arrangement.Cursor.GetCurrentNode()
	currentNode.Section.DecreaseStartCycles()
}

func (m *model) IncreaseCycles() {
	currentNode := m.arrangement.Cursor.GetCurrentNode()
	currentNode.Section.IncreaseCycles()
}

func (m *model) DecreaseCycles() {
	currentNode := m.arrangement.Cursor.GetCurrentNode()
	currentNode.Section.DecreaseCycles()
}

func (m *model) IncreasePartSelector() {
	newIndex := m.partSelectorIndex + 1
	if newIndex < len(*m.definition.Parts) {
		m.partSelectorIndex = newIndex
	}
}

func (m *model) DecreasePartSelector() {
	newIndex := m.partSelectorIndex - 1
	if newIndex > -2 {
		m.partSelectorIndex = newIndex
	}
}

func (m *model) ToggleRatchetMute() {
	currentNote, _ := m.CurrentNote()
	currentNote.Ratchets.Toggle(m.ratchetCursor)
	m.currentOverlay.SetNote(m.gridCursor, currentNote)
}

func LoadFile(filename string, template string, instrument string) (sequence.Sequence, error) {
	var definition sequence.Sequence
	var fileErr error

	fileInfo, fileInfoErr := os.Stat(filename)
	fileExists := fileInfoErr == nil && !fileInfo.IsDir()

	if filename != "" && fileExists {
		definition, fileErr = sequence.Read(filename)
		gridTemplate, exists := config.GetTemplate(definition.Template)
		definition.TemplateSequencerType = gridTemplate.SequencerType
		// TODO: Give these hardcoded values a run for a bit before consolidating
		if exists {
			config.LongGates = config.GetGateLengths(32)
		} else {
			//maxGateLength := config.GetDefaultTemplate().MaxGateLength
			config.LongGates = config.GetGateLengths(32)
		}
	}

	if !fileExists || fileErr != nil {
		newDefinition := sequence.InitSequence(template, instrument)
		definition = newDefinition
	}

	return definition, fileErr
}

func InitModel(filename string, midiConnection *seqmidi.MidiConnection, options ProgramOptions, cancel context.CancelFunc) model {

	newCursor := cursor.New()
	newCursor.BlinkSpeed = 600 * time.Millisecond
	newCursor.Style = lipgloss.NewStyle().Background(lipgloss.AdaptiveColor{Light: "255", Dark: "0"})

	lockReceiverChannel := make(chan bool)
	unlockReceiverChannel := make(chan bool)
	errorChannel := make(chan error)

	logFile, logFileErr := tea.LogToFile("debug.log", "debug")
	definition, err := LoadFile(filename, options.gridTemplate, options.instrument)
	if err == nil {
		// No capacity for multiple errors currently
		err = logFileErr
	}

	return model{
		cancel:                cancel,
		transmitting:          true,
		currentError:          err,
		theme:                 options.theme,
		filename:              filename,
		textInput:             InitTextInput(),
		partSelectorIndex:     -1,
		lockReceiverChannel:   lockReceiverChannel,
		unlockReceiverChannel: unlockReceiverChannel,
		errChan:               errorChannel,
		help:                  help.New(),
		cursor:                newCursor,
		midiConnection:        midiConnection,
		logFile:               logFile,
		selectionIndicator:    operation.SelectGrid,
		focus:                 operation.FocusGrid,
		patternMode:           operation.PatternFill,
		logFileAvailable:      logFileErr == nil,
		gridCursor:            GK(0, 0),
		currentOverlay:        (*definition.Parts)[0].Overlays,
		overlayKeyEdit:        overlaykey.InitModel(),
		arrangement:           arrangement.InitModel(definition.Arrangement, definition.Parts),
		definition:            definition,
		playState: playstate.PlayState{
			LineStates: playstate.InitLineStates(len(definition.Lines), []playstate.LineState{}, 0),
		},
	}
}

func InitTextInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "------"
	ti.Prompt = ""
	ti.CharLimit = 20
	ti.Width = 20
	ti.Focus()
	return ti
}

func (m model) LogTeaMsg(msg tea.Msg) {
	switch msg := msg.(type) {
	case beats.BeatMsg:
		m.LogString(fmt.Sprintf("beatMsg %d %d\n", msg.Interval, m.definition.Tempo))
	case tea.KeyMsg:
		m.LogString(fmt.Sprintf("keyMsg %s\n", msg.String()))
	case cursor.BlinkMsg:
	default:
		m.LogString(fmt.Sprintf("%T\n", msg))
	}
}

func (m *model) LogString(message string) {
	if m.logFileAvailable {
		_, err := m.logFile.WriteString(message + "\n")
		if err != nil {
			m.logFileAvailable = false
			panic("could not write to log file")
		}
	}
}

func (m *model) LogError(err error) {
	if m.logFileAvailable {
		m.LogString("ERROR --------------------- \n")

		_, writeErr := fmt.Fprintf(m.logFile, "%+v", err)
		if writeErr != nil {
			m.logFileAvailable = false
			panic("could not write to log file")
		}

		m.LogString("\n")
	}
}

func (m *model) LogFromBeatTime() {
	if m.logFileAvailable {
		_, err := fmt.Fprintf(m.logFile, "%d\n", time.Since(m.playState.BeatTime))
		if err != nil {
			m.logFileAvailable = false
			panic("could not write to log file")
		}
	}
}

var beatsLooper beats.BeatsLooper

func RunProgram(filename string, options ProgramOptions) (*tea.Program, error) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	midiConnection := seqmidi.InitMidiConnection(options.outport, options.midiout, ctx)
	config.Init()
	themes.ChooseTheme(options.theme)
	model := InitModel(filename, midiConnection, options, cancel)
	model.ResetIterations()
	beatsLooper = beats.InitBeatsLooper()
	updateChannel = beatsLooper.UpdateChannel
	midiConnection.DeviceLoop(ctx)

	if model.midiConnection.HasTransmitter() {
		model.midiLoopMode = timing.MlmReceiver
	} else {
		model.midiLoopMode = timing.MlmTransmitter
	}

	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithReportFocus())
	SetupTimingLoop(model, beatsLooper, program.Send, ctx)
	beatsLooper.Loop(program.Send, midiConnection, ctx)
	midiConnection.LoopMidi(ctx)
	if options.outport {
		err := midiConnection.CreateOutport()
		if err != nil {
			return nil, err
		}
	}
	return program, nil
}

func SetupTimingLoop(model model, beatsLooper beats.BeatsLooper, sendFn func(tea.Msg), ctx context.Context) {
	err := timing.Loop(model.midiLoopMode, model.lockReceiverChannel, model.unlockReceiverChannel, ctx, beatsLooper, sendFn, model.midiConnection)
	if err != nil {
		go func() {
			sendFn(errorMsg{fault.Wrap(err, fmsg.With("could not setup midi event loop"))})
			// for {
			// 	// NOTE: Eat the messages sent on the program channel so that we can quit and still send the stopMsg
			// 	<-model.programChannel
			// }
		}()
	} else {
		ErrorLoop(sendFn, model.errChan)
		model.SyncTempo()
	}
}

func ErrorLoop(sendFn func(tea.Msg), errChan chan error) {
	go func() {
		for {
			programError := <-errChan
			tag := ftag.Get(programError)
			switch tag {
			case "view_panic":
				sendFn(viewPanicMsg{programError})
			default:
				sendFn(errorMsg{programError})
			}
		}
	}()
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg { return tea.FocusMsg{} }
}

func Is(msg tea.KeyMsg, k ...key.Binding) bool {
	return key.Matches(msg, k...)
}

func IsNot(msg tea.KeyMsg, k ...key.Binding) bool {
	return !key.Matches(msg, k...)
}

type panicMsg struct {
	message    string
	stacktrace []byte
}

func (m model) Update(msg tea.Msg) (rModel tea.Model, rCmd tea.Cmd) {
	defer func() {
		if r := recover(); r != nil {
			rModel = m
			stackTrace := make([]byte, 4096)
			n := runtime.Stack(stackTrace, false)
			rCmd = func() tea.Msg {
				return panicMsg{message: fmt.Sprintf("Caught Update Panic: %v", r), stacktrace: stackTrace[:n]}
			}
		}
	}()

	// NOTE: Processing enter/escape keys before anything else
	switch msg := msg.(type) {
	case errorMsg:
		m.SetCurrentError(msg.error)
	case viewPanicMsg:
		m.SetCurrentViewError(msg.error)
	case panicMsg:
		m.SetCurrentError(errors.New(msg.message))
		m.LogString(fmt.Sprintf(" ------ Panic Message ------- \n%s\n", msg.message))
		m.LogString(fmt.Sprintf(" ------ Stacktrace ---------- \n%s\n", msg.stacktrace))
	case tea.KeyMsg:

		mapping := mappings.ProcessKey(msg, m.focus, m.selectionIndicator, m.definition.TemplateSequencerType, m.patternMode)

		// NOTE: Finally process the mapping
		switch mapping.Command {
		case mappings.ToggleBoundedLoop:
			if m.visualSelection.visualMode != operation.VisualNone {
				m.playState.BoundedLoop.Active = !m.playState.BoundedLoop.Active
				m.playState.BoundedLoop.LeftBound = m.visualSelection.Bounds(m.CurrentPart().Beats).Left
				m.playState.BoundedLoop.RightBound = m.visualSelection.Bounds(m.CurrentPart().Beats).Right
			} else {
				m.playState.BoundedLoop.Active = !m.playState.BoundedLoop.Active
				m.playState.BoundedLoop.LeftBound = m.gridCursor.Beat
				m.playState.BoundedLoop.RightBound = m.gridCursor.Beat
			}
			m.SyncBeatLoop()
		case mappings.ExpandLeftLoopBound:
			m.playState.BoundedLoop.ExpandLeft()
			m.SyncBeatLoop()
		case mappings.ExpandRightLoopBound:
			m.playState.BoundedLoop.ExpandRight((*m.definition.Parts)[m.CurrentPartID()].Beats)
			m.SyncBeatLoop()
		case mappings.ContractLeftLoopBound:
			m.playState.BoundedLoop.ContractLeft()
			m.SyncBeatLoop()
		case mappings.ContractRightLoopBound:
			m.playState.BoundedLoop.ContractRight()
			m.SyncBeatLoop()
		case mappings.PurposePanic:
			return m, func() tea.Msg { panic("Panic") }
		case mappings.ToggleTransmitting:
			m.transmitting = !m.transmitting
		case mappings.ToggleClockPreRoll:
			m.clockPreRoll = !m.clockPreRoll
		case mappings.DecreaseAllChannels:
			var minChannel uint8 = 255
			for i := range m.definition.Lines {
				if m.definition.Lines[i].Channel < minChannel {
					minChannel = m.definition.Lines[i].Channel
				}
			}
			if minChannel == 1 {
				return m, nil
			}
			for i := range m.definition.Lines {
				m.definition.Lines[i].DecrementChannel()
			}
		case mappings.IncreaseAllChannels:
			var maxChannel uint8
			for i := range m.definition.Lines {
				if m.definition.Lines[i].Channel > maxChannel {
					maxChannel = m.definition.Lines[i].Channel
				}
			}
			if maxChannel >= 16 {
				return m, nil
			}
			for i := range m.definition.Lines {
				m.definition.Lines[i].IncrementChannel()
			}
		case mappings.DecreaseAllNote:
			var minNote uint8 = 255
			for i := range m.definition.Lines {
				if m.definition.Lines[i].Note < minNote {
					minNote = m.definition.Lines[i].Note
				}
			}
			if minNote == 1 {
				return m, nil
			}
			for i := range m.definition.Lines {
				m.definition.Lines[i].DecrementNote()
			}
		case mappings.IncreaseAllNote:
			var maxNote uint8
			for i := range m.definition.Lines {
				if m.definition.Lines[i].Note > maxNote {
					maxNote = m.definition.Lines[i].Note
				}
			}
			if maxNote >= 127 {
				return m, nil
			}
			for i := range m.definition.Lines {
				m.definition.Lines[i].IncrementNote()
			}
		case mappings.MidiPanic:
			channels := make([]uint8, 0)
			for _, line := range m.definition.Lines {
				if slices.Contains(channels, line.Channel) {
					continue
				}
				channels = append(channels, line.Channel)
			}
			err := m.midiConnection.Panic(channels)
			if err != nil {
				m.SetCurrentError(err)
			}
		case mappings.ToggleHideLines:
			m.hideEmptyLines = !m.hideEmptyLines
			if m.hideEmptyLines {
				m.CursorValid()
			}
		case mappings.ConfirmConfirmQuit:
			return m.Quit()
		case mappings.ConfirmConfirmReload:
			m.ReloadFile()
			m.selectionIndicator = operation.SelectGrid
		case mappings.ConfirmConfirmNew:
			newModel := m.NewSequence()
			var cmd tea.Cmd
			cursor, cmd := m.cursor.Update(tea.FocusMsg{})
			newModel.cursor = cursor
			m.SyncBeatLoop()
			return newModel, cmd
		case mappings.ConfirmChangePart:
			_, cmd := m.arrangement.Update(arrangement.ChangePart{Index: m.partSelectorIndex})
			m.currentOverlay = m.CurrentPart().Overlays
			m.selectionIndicator = operation.SelectGrid
			return m, cmd
		case mappings.ConfirmSelectPart:
			_, cmd := m.arrangement.Update(arrangement.NewPart{Index: m.partSelectorIndex, After: m.sectionSideIndicator == SectionAfter, IsPlaying: m.playState.Playing})
			if m.sectionSideIndicator == SectionAfter {
				m.arrangement.Cursor.MoveNext()
			} else {
				m.arrangement.Cursor.MovePrev()
			}
			(*m.playState.Iterations)[m.arrangement.CurrentNode()] = m.CurrentSongSection().StartCycles
			m.ResetCurrentOverlay()
			m.selectionIndicator = operation.SelectGrid
			return m, cmd
		case mappings.ConfirmFileName:
			if m.textInput.Value() != "" {
				m.filename = fmt.Sprintf("%s.sq", m.textInput.Value())
				m.textInput.Reset()
				m.selectionIndicator = operation.SelectGrid
				m.Save()
				return m, nil
			} else {
				m.selectionIndicator = operation.SelectGrid
				return m, nil
			}
		case mappings.ConfirmRenamePart:
			m.RenamePart(m.textInput.Value())
			m.textInput.Reset()
			m.selectionIndicator = operation.SelectGrid
			return m, nil
		case mappings.ConfirmOverlayKey:
			currentKey := m.currentOverlay.Key
			m.focus = operation.FocusGrid
			m.overlayKeyEdit.Focus(false)
			m.EnsureOverlay()
			if currentKey != m.currentOverlay.Key {
				undoable := UndoNewOverlay{
					overlayKey:     m.overlayKeyEdit.GetKey(),
					cursorPosition: m.gridCursor,
					ArrCursor:      m.arrangement.Cursor,
				}
				redoable := UndoRemoveOverlay{
					overlayKey:     m.overlayKeyEdit.GetKey(),
					cursorPosition: m.gridCursor,
					ArrCursor:      m.arrangement.Cursor,
				}
				m.PushUndoables(undoable, redoable)
			}
		case mappings.RemoveOverlay:
			undoable := UndoRemoveOverlay{
				overlay:        m.currentOverlay,
				overlayKey:     m.overlayKeyEdit.GetKey(),
				cursorPosition: m.gridCursor,
				ArrCursor:      m.arrangement.Cursor,
			}
			m.RemoveOverlay()
			redoable := UndoNewOverlay{
				overlayKey:     undoable.overlayKey,
				cursorPosition: m.gridCursor,
				ArrCursor:      m.arrangement.Cursor,
			}
			m.PushUndoables(undoable, redoable)
		case mappings.Quit:
			if m.currentViewError == nil {
				m.SetSelectionIndicator(operation.SelectConfirmQuit)
			} else {
				return m.Quit()
			}
		case mappings.ArrKeyMessage:
			arrangmementModel, cmd := m.arrangement.Update(msg)
			m.arrangement = arrangmementModel
			m.ResetCurrentOverlay()
			return m, cmd
		case mappings.TextInputMessage:
			tiModel, cmd := m.textInput.Update(msg)
			m.textInput = tiModel
			return m, cmd
		case mappings.OverlayKeyMessage:
			okModel, cmd := m.overlayKeyEdit.Update(msg)
			m.overlayKeyEdit = okModel
			return m, cmd
		case mappings.ReloadFile:
			m.SetSelectionIndicator(operation.SelectConfirmReload)
		case mappings.HoldingKeys:
			return m, nil
		case mappings.CursorDown:
			if slices.Contains([]operation.Selection{operation.SelectGrid, operation.SelectSetupChannel, operation.SelectSetupMessageType, operation.SelectSetupValue, operation.SelectSpecificValue}, m.selectionIndicator) {
				m.CursorDown()
				m.UnsetActiveChord()
				m.SetVisualArea()
			}
		case mappings.CursorUp:
			if slices.Contains([]operation.Selection{operation.SelectGrid, operation.SelectSetupChannel, operation.SelectSetupMessageType, operation.SelectSetupValue, operation.SelectSpecificValue}, m.selectionIndicator) {
				m.CursorUp()
				m.UnsetActiveChord()
				m.SetVisualArea()
			}
		case mappings.CursorLeft:
			if m.selectionIndicator == operation.SelectRatchets {
				if m.ratchetCursor > 0 {
					m.ratchetCursor--
				}
			} else if m.selectionIndicator > operation.SelectGrid && m.selectionIndicator != operation.SelectSpecificValue {
				// Do Nothing
			} else {
				m.CursorLeft()
				m.UnsetActiveChord()
				m.SetVisualArea()
			}
		case mappings.CursorRight:
			if m.selectionIndicator == operation.SelectRatchets {
				currentNote, _ := m.CurrentNote()
				if m.ratchetCursor < currentNote.Ratchets.Length {
					m.ratchetCursor++
				}
			} else if m.selectionIndicator > operation.SelectGrid && m.selectionIndicator != operation.SelectSpecificValue {
				// Do Nothing
			} else {
				m.CursorRight()
				m.UnsetActiveChord()
				m.SetVisualArea()
			}
		case mappings.CursorLineStart:
			m.SetGridCursor(gridKey{
				Line: m.gridCursor.Line,
				Beat: 0,
			})
			m.SetVisualArea()
		case mappings.CursorLineEnd:
			m.SetGridCursor(gridKey{
				Line: m.gridCursor.Line,
				Beat: m.CurrentPart().Beats - 1,
			})
			m.SetVisualArea()
		case mappings.CursorLastLine:
			var newLine uint8
			if m.hideEmptyLines {
				pattern := m.CombinedOverlayPattern(m.currentOverlay)
				showLines := GetShowLines(len(m.definition.Lines), pattern, m.CurrentPart().Beats)
				newLine = showLines[len(showLines)-1]
			} else {
				newLine = uint8(len(m.definition.Lines) - 1)
			}
			m.SetGridCursor(gridKey{
				Line: newLine,
				Beat: m.gridCursor.Beat,
			})
			m.SetVisualArea()
		case mappings.CursorFirstLine:
			var newLine uint8
			if m.hideEmptyLines {
				pattern := m.CombinedOverlayPattern(m.currentOverlay)
				showLines := GetShowLines(len(m.definition.Lines), pattern, m.CurrentPart().Beats)
				newLine = showLines[0]
			} else {
				newLine = 0
			}
			m.SetGridCursor(gridKey{
				Line: newLine,
				Beat: m.gridCursor.Beat,
			})
			m.SetVisualArea()
		case mappings.Escape:
			m.PushUndoableDefinitionState()
			m.Escape()
		case mappings.PlayStop:
			if !m.playState.Playing {
				if m.clockPreRoll {
					m.playState.RecordPreRollBeats = 1
				}
				if m.visualSelection.visualMode == operation.VisualNone {
					m.playState.LoopMode = playstate.OneTimeWholeSequence
				} else {
					m.playState.LoopMode = playstate.LoopOverlay
				}
			} else {
				err := m.midiConnection.SendStopMessage()
				if err != nil {
					m.SetCurrentError(err)
				}
			}
			m.StartStop(0)
		case mappings.PlayPart:
			if !m.playState.Playing {
				if m.clockPreRoll {
					m.playState.RecordPreRollBeats = 1
				}
				m.playState.LoopMode = playstate.LoopPart
			}
			m.StartStop(0)
		case mappings.PlayLoop:
			if !m.playState.Playing {
				if m.clockPreRoll {
					m.playState.RecordPreRollBeats = 1
				}
				m.playState.LoopMode = playstate.LoopWholeSequence
			}
			m.StartStop(0)
		case mappings.PlayOverlayLoop:
			if !m.playState.Playing {
				if m.clockPreRoll {
					m.playState.RecordPreRollBeats = 1
				}
				m.playState.LoopMode = playstate.LoopOverlay
			}
			m.StartStop(0)
		case mappings.PlayAlong:
			if !m.playState.Playing {
				m.playState.RecordPreRollBeats = 8
				err := m.midiConnection.SendPlayMessage()
				if err != nil {
					m.SetCurrentError(err)
				} else {
					m.StartStop(28650 * time.Microsecond)
				}
			} else {
				m.StartStop(0)
			}
		case mappings.PlayRecord:
			if !m.playState.Playing {
				m.playState.RecordPreRollBeats = 8
				err := m.midiConnection.SendRecordMessage()
				if err != nil {
					m.SetCurrentError(err)
				} else {
					m.StartStop(28650 * time.Microsecond)
				}
			} else {
				m.StartStop(0)
			}
		case mappings.ModifyKeyInputSwitch:
			m.SetSelectionIndicator(operation.SelectGrid)
			m.focus = operation.FocusOverlayKey
			m.overlayKeyEdit.Focus(true)
			m.modifyKey = true
		case mappings.OverlayInputSwitch:
			// NOTE: This component handles getting into the overlay key edit mode
			// the overlaykey component handles getting out of it
			m.modifyKey = false
			m.SetSelectionIndicator(operation.SelectGrid)
			if m.focus == operation.FocusOverlayKey {
				m.focus = operation.FocusGrid
				m.overlayKeyEdit.Focus(false)
				m.EnsureOverlay()
			} else {
				m.focus = operation.FocusOverlayKey
				m.overlayKeyEdit.Focus(true)
			}
		case mappings.TempoInputSwitch:
			states := []operation.Selection{operation.SelectGrid, operation.SelectTempo, operation.SelectTempoSubdivision}
			m.ClampAndSyncTempo()
			if m.selectionIndicator == states[0] {
				m.CaptureTemporaryState()
			}
			if m.selectionIndicator == states[len(states)-1] {
				m.PushUndoableDefinitionState()
			}
			m.SetSelectionIndicator(AdvanceSelectionState(states, m.selectionIndicator))
		case mappings.SetupInputSwitch:
			var states []operation.Selection
			if m.definition.Lines[m.gridCursor.Line].MsgType == grid.MessageTypeProgramChange {
				states = []operation.Selection{operation.SelectGrid, operation.SelectSetupChannel, operation.SelectSetupMessageType}
			} else {
				states = []operation.Selection{operation.SelectGrid, operation.SelectSetupChannel, operation.SelectSetupMessageType, operation.SelectSetupValue}
			}
			if m.selectionIndicator == states[0] {
				m.CaptureTemporaryState()
			}
			if m.selectionIndicator == states[len(states)-1] {
				m.PushUndoableDefinitionState()
			}
			m.SetSelectionIndicator(AdvanceSelectionState(states, m.selectionIndicator))
		case mappings.AccentInputSwitch:
			states := []operation.Selection{operation.SelectGrid, operation.SelectAccentTarget, operation.SelectAccentStart, operation.SelectAccentEnd}
			if m.selectionIndicator == states[0] {
				m.CaptureTemporaryState()
			}
			if m.selectionIndicator == states[len(states)-1] {
				m.PushUndoableDefinitionState()
			}

			m.ClampAccentValues()
			m.SetSelectionIndicator(AdvanceSelectionState(states, m.selectionIndicator))
		case mappings.RatchetInputSwitch:
			currentNote, _ := m.CurrentNote()
			if currentNote.AccentIndex > 0 {
				states := []operation.Selection{operation.SelectGrid, operation.SelectRatchets, operation.SelectRatchetSpan}
				if m.selectionIndicator == states[0] {
					m.CaptureTemporaryState()
				}
				if m.selectionIndicator == states[len(states)-1] {
					m.PushUndoableDefinitionState()
				}
				m.SetSelectionIndicator(AdvanceSelectionState(states, m.selectionIndicator))
				m.ratchetCursor = 0
			}
		case mappings.BeatInputSwitch:
			states := []operation.Selection{operation.SelectGrid, operation.SelectBeats, operation.SelectStartBeats}
			if m.selectionIndicator == states[0] {
				m.CaptureTemporaryState()
			}
			if m.selectionIndicator == states[len(states)-1] {
				m.PushUndoableDefinitionState()
			}
			m.SetSelectionIndicator(AdvanceSelectionState(states, m.selectionIndicator))
		case mappings.CyclesInputSwitch:
			states := []operation.Selection{operation.SelectGrid, operation.SelectCycles, operation.SelectStartCycles}
			if m.selectionIndicator == states[0] {
				m.CaptureTemporaryState()
			}
			if m.selectionIndicator == states[len(states)-1] {
				m.PushUndoableDefinitionState()
			}
			m.SetSelectionIndicator(AdvanceSelectionState(states, m.selectionIndicator))
		case mappings.ToggleArrangementView:
			m.showArrangementView = !m.showArrangementView || m.focus != operation.FocusArrangementEditor
			if m.showArrangementView {
				m.SetSelectionIndicator(operation.SelectGrid)
				m.focus = operation.FocusArrangementEditor
				m.arrangement.Focus = true
			} else {
				m.Escape()
			}
		case mappings.Increase:
			switch m.selectionIndicator {
			case operation.SelectTempo:
				m.IncreaseTempo(1)
			case operation.SelectTempoSubdivision:
				if m.definition.Subdivisions < 8 {
					m.definition.Subdivisions++
				}
				m.SyncTempo()
			case operation.SelectSetupChannel:
				m.definition.Lines[m.gridCursor.Line].IncrementChannel()
			case operation.SelectSetupMessageType:
				m.definition.Lines[m.gridCursor.Line].IncrementMessageType()
			case operation.SelectSetupValue:
				switch m.definition.Lines[m.gridCursor.Line].MsgType {
				case grid.MessageTypeNote:
					m.definition.Lines[m.gridCursor.Line].IncrementNote()
				case grid.MessageTypeCc:
					m.IncrementCC()
				}
			case operation.SelectRatchetSpan:
				m.IncreaseSpan()
			case operation.SelectAccentEnd:
				m.IncreaseAccentEnd()
			case operation.SelectAccentTarget:
				// Only two options right now, so increase and decrease would do the
				// same thing
				m.DecreaseAccentTarget()
			case operation.SelectAccentStart:
				m.IncreaseAccentStart()
			case operation.SelectBeats:
				m.IncreaseBeats()
			case operation.SelectCycles:
				m.IncreaseCycles()
			case operation.SelectStartBeats:
				m.IncreaseStartBeats()
			case operation.SelectStartCycles:
				m.IncreaseStartCycles()
			case operation.SelectPart:
				m.IncreasePartSelector()
			case operation.SelectChangePart:
				m.IncreasePartSelector()
			case operation.SelectSpecificValue:
				note, _ := m.CurrentNote()
				m.IncrementSpecificValue(note)
			default:
				m.IncreaseTempo(5)
			}
			m.SyncBeatLoop()
		case mappings.Decrease:
			switch m.selectionIndicator {
			case operation.SelectTempo:
				m.DecreaseTempo(1)
			case operation.SelectTempoSubdivision:
				if m.definition.Subdivisions > 1 {
					m.definition.Subdivisions--
				}
				m.SyncTempo()
			case operation.SelectSetupChannel:
				m.definition.Lines[m.gridCursor.Line].DecrementChannel()
			case operation.SelectSetupMessageType:
				m.definition.Lines[m.gridCursor.Line].DecrementMessageType()
			case operation.SelectSetupValue:
				switch m.definition.Lines[m.gridCursor.Line].MsgType {
				case grid.MessageTypeNote:
					m.definition.Lines[m.gridCursor.Line].DecrementNote()
				case grid.MessageTypeCc:
					m.DecrementCC()
				}
			case operation.SelectRatchetSpan:
				m.DecreaseSpan()
			case operation.SelectAccentEnd:
				m.DecreaseAccentEnd()
			case operation.SelectAccentTarget:
				m.DecreaseAccentTarget()
			case operation.SelectAccentStart:
				m.DecreaseAccentStart()
			case operation.SelectBeats:
				m.DecreaseBeats()
			case operation.SelectCycles:
				m.DecreaseCycles()
			case operation.SelectStartBeats:
				m.DecreaseStartBeats()
			case operation.SelectStartCycles:
				m.DecreaseStartCycles()
			case operation.SelectPart:
				m.DecreasePartSelector()
			case operation.SelectChangePart:
				m.DecreasePartSelector()
			case operation.SelectSpecificValue:
				note, _ := m.CurrentNote()
				m.DecrementSpecificValue(note)
			default:
				m.DecreaseTempo(5)
			}
			m.SyncBeatLoop()
		case mappings.ToggleGateMode:
			m.SetPatternMode(operation.PatternGate)
		case mappings.ToggleGateNoteMode:
			m.SetPatternMode(operation.PatternNoteGate)
		case mappings.ToggleWaitMode:
			m.SetPatternMode(operation.PatternWait)
		case mappings.ToggleWaitNoteMode:
			m.SetPatternMode(operation.PatternNoteWait)
		case mappings.ToggleAccentMode:
			m.SetPatternMode(operation.PatternAccent)
		case mappings.ToggleAccentNoteMode:
			m.SetPatternMode(operation.PatternNoteAccent)
		case mappings.ToggleRatchetMode:
			m.SetPatternMode(operation.PatternRatchet)
		case mappings.ToggleRatchetNoteMode:
			m.SetPatternMode(operation.PatternNoteRatchet)
		case mappings.ToggleChordMode:
			if m.definition.TemplateSequencerType == operation.SeqModeChord {
				m.definition.TemplateSequencerType = operation.SeqModeLine
			} else {
				m.definition.TemplateSequencerType = operation.SeqModeChord
			}
		case mappings.ToggleMonoMode:
			if m.definition.TemplateSequencerType == operation.SeqModeMono {
				m.definition.TemplateSequencerType = operation.SeqModeLine
			} else {
				m.definition.TemplateSequencerType = operation.SeqModeMono
			}
		case mappings.PrevOverlay:
			m.NextOverlay(-1)
			m.overlayKeyEdit.SetOverlayKey(m.currentOverlay.Key)
		case mappings.NextOverlay:
			m.NextOverlay(+1)
			m.overlayKeyEdit.SetOverlayKey(m.currentOverlay.Key)
		case mappings.Save:
			m.Escape()
			if m.filename == "" {
				m.selectionIndicator = operation.SelectFileName
			} else {
				m.Save()
			}
		case mappings.SaveAs:
			m.Escape()
			m.selectionIndicator = operation.SelectFileName
		case mappings.Undo:
			tempo, subdiv := m.definition.Tempo, m.definition.Subdivisions
			undoStack := m.Undo()
			if undoStack != EmptyStack {
				m.PushRedo(undoStack)
			}
			if tempo != m.definition.Tempo || subdiv != m.definition.Subdivisions {
				m.SyncTempo()
			}
		case mappings.Redo:
			tempo, subdiv := m.definition.Tempo, m.definition.Subdivisions
			undoStack := m.Redo()
			if undoStack != EmptyStack {
				m.PushUndo(undoStack)
			}
			if tempo != m.definition.Tempo || subdiv != m.definition.Subdivisions {
				m.SyncTempo()
			}
		case mappings.New:
			m.selectionIndicator = operation.SelectConfirmNew
		case mappings.ToggleVisualMode:
			switch m.visualSelection.visualMode {
			case operation.VisualBlock:
				m.visualSelection.visualMode = operation.VisualNone
			case operation.VisualLine:
				m.visualSelection.visualMode = operation.VisualBlock
			default:
				m.visualSelection = VisualSelection{anchor: m.gridCursor, lead: m.gridCursor, visualMode: operation.VisualBlock}
			}
		case mappings.ToggleVisualLineMode:
			switch m.visualSelection.visualMode {
			case operation.VisualLine:
				m.visualSelection.visualMode = operation.VisualNone
			case operation.VisualBlock:
				m.visualSelection.visualMode = operation.VisualLine
			default:
				m.visualSelection = VisualSelection{anchor: m.gridCursor, lead: m.gridCursor, visualMode: operation.VisualLine}
			}
		case mappings.TogglePlayEdit:
			m.playEditing = !m.playEditing
			if m.playEditing {
				m.editingPartID = m.CurrentPartID()
			}
		case mappings.NewLine:
			if len(m.definition.Lines) < 100 {
				lastLine := m.definition.Lines[len(m.definition.Lines)-1]
				secondLastLine := m.definition.Lines[len(m.definition.Lines)-2]
				var difference int8
				if int8(lastLine.Note)-int8(secondLastLine.Note) > 0 {
					difference = 1
				} else {
					difference = -1
				}

				m.definition.Lines = append(m.definition.Lines, grid.LineDefinition{
					Channel: lastLine.Channel,
					Note:    uint8(int8(lastLine.Note) + difference),
				})
				m.playState.LineStates = playstate.InitLineStates(len(m.definition.Lines), m.playState.LineStates, 0)
			}
			m.SyncBeatLoop()
		case mappings.NewSectionAfter:
			m.SetSelectionIndicator(operation.SelectPart)
			m.sectionSideIndicator = SectionAfter
		case mappings.NewSectionBefore:
			m.SetSelectionIndicator(operation.SelectPart)
			m.sectionSideIndicator = SectionBefore
		case mappings.ChangePart:
			m.SetSelectionIndicator(operation.SelectChangePart)
		case mappings.NextTheme:
			m.NextTheme()
		case mappings.PrevTheme:
			m.PrevTheme()
		case mappings.NextSection:
			m.NextSection()
		case mappings.PrevSection:
			m.PrevSection()
		case mappings.Yank:
			m.yankBuffer = m.Yank()
			m.SetGridCursor(m.YankBounds().TopLeft())
			m.visualSelection.visualMode = operation.VisualNone
		case mappings.Mute:
			if m.IsRatchetSelector() {
				m.ToggleRatchetMute()
			} else {
				m.playState.LineStates = Mute(m.playState.LineStates, m.gridCursor.Line)
				m.playState.HasSolo = m.HasSolo()
			}
			m.SyncBeatLoop()
		case mappings.MuteAll:
			m.playState.LineStates = MuteAll(m.playState.LineStates, m.gridCursor.Line)
			m.playState.HasSolo = false
			m.SyncBeatLoop()
		case mappings.UnMuteAll:
			m.playState.LineStates = UnMuteAll(m.playState.LineStates, m.gridCursor.Line)
			m.playState.HasSolo = false
			m.SyncBeatLoop()
		case mappings.Solo:
			m.playState.LineStates = Solo(m.playState.LineStates, m.gridCursor.Line)
			m.playState.HasSolo = m.HasSolo()
			m.SyncBeatLoop()
		case mappings.Enter:
			m.RecordSpecificValueUndo()
			m.PushUndoableDefinitionState()
			m.Escape()
			m.SetSelectionIndicator(operation.SelectGrid)
		case mappings.ClearAllOverlays:
			m.ClearAllOverlays()
			m.UnsetActiveChord()
		default:
			// NOTE: Assuming that every other mapping updates the definition
			m = m.UpdateDefinition(mapping)
			// NOTE: Only sync beat loop on definition changes
			m.SyncBeatLoop()
		}
	case tea.FocusMsg:
		m.hasUIFocus = true
	case tea.BlurMsg:
		m.hasUIFocus = false
	case overlaykey.UpdatedOverlayKey:
		if !msg.HasFocus {
			m.focus = operation.FocusGrid
			m.selectionIndicator = operation.SelectGrid
		}
	case timing.UIStartMsg:
		if !m.playState.Playing {
			m.playState.LoopMode = msg.LoopMode
			m.playState.Playing = true
			// NOTE: Getting a start message from midieventloop means we are in receiver mode
			m.playState.PlayMode = playstate.PlayReceiver
		} else {
			m.SetCurrentError(errors.New("cannot start when already started"))
		}
		m.Start(0)
	case timing.UIStopMsg:
		m.playState.Playing = false
		m.playState.PlayMode = playstate.PlayStandard
		m.Stop()
	case timing.TransmitterConnectedMsg:
		m.transmitterConnected = true
	case timing.TransmitterNotConnectedMsg:
		m.transmitterConnected = false
	case arrangement.GiveBackFocus:
		m.selectionIndicator = operation.SelectGrid
		m.focus = operation.FocusGrid
	case arrangement.RenamePart:
		m.SetSelectionIndicator(operation.SelectRenamePart)
	case arrangement.Undo:
		m.PushArrUndo(msg)
	case beats.ModelPlayedMsg:
		if m.playState.Playing {
			m.playState = msg.PlayState
			m.arrangement.Cursor = msg.Cursor
		}
		if msg.PerformStop {
			m.playState.LineStates = playstate.InitLineStates(len(m.definition.Lines), m.playState.LineStates, 0)
			m.ResetIterations()
			m.SafeStop()
			m.SyncBeatLoop()
		} else {
			m.ResetCurrentOverlay()
		}
		m.arrangement.ResetDepth()
		return m, nil
	case beats.AnticipatoryStop:
		if m.midiLoopMode == timing.MlmTransmitter {
			timingChannel <- timing.AnticipatoryStopMsg{}
		}
	}

	var cmd tea.Cmd
	cursor, cmd := m.cursor.Update(msg)
	m.cursor = cursor
	return m, cmd
}

func (m *model) SyncBeatLoop() {
	go func() {
		updateChannel <- beats.ModelMsg{Sequence: m.definition, PlayState: m.playState, Cursor: m.arrangement.Cursor}
	}()
}

func (m model) Quit() (tea.Model, tea.Cmd) {

	timingChannel <- timing.QuitMsg{}
	m.cancel()
	err := m.logFile.Close()
	if err != nil {
		// NOTE: no good way to display this error when quitting, just panic
		panic("Unable to close logfile")
	}
	return m, tea.Quit
}

func (m *model) SetVisualArea() {
	m.visualSelection.lead = m.gridCursor
}

func (m *model) CursorDown() {
	pattern := m.CombinedOverlayPattern(m.currentOverlay)
	showLines := GetShowLines(len(m.definition.Lines), pattern, m.CurrentPart().Beats)
	if m.gridCursor.Line < uint8(len(m.definition.Lines)-1) {
		for i := range len(m.definition.Lines) - int(m.gridCursor.Line) {
			newLine := m.gridCursor.Line + (1 + uint8(i))
			if slices.Contains(showLines, newLine) || !m.hideEmptyLines {
				m.SetGridCursor(gridKey{
					Line: newLine,
					Beat: m.gridCursor.Beat,
				})
				return
			}
		}
	}
}

func (m *model) CursorUp() {
	pattern := m.CombinedOverlayPattern(m.currentOverlay)
	showLines := GetShowLines(len(m.definition.Lines), pattern, m.CurrentPart().Beats)
	if m.gridCursor.Line > 0 {
		for i := range m.gridCursor.Line + 1 {
			newLine := m.gridCursor.Line - (1 + i)
			if slices.Contains(showLines, newLine) || !m.hideEmptyLines {
				m.SetGridCursor(gridKey{
					Line: newLine,
					Beat: m.gridCursor.Beat,
				})
				return
			}
		}
	}
}

func (m *model) CursorValid() {
	pattern := m.CombinedOverlayPattern(m.currentOverlay)
	showLines := GetShowLines(len(m.definition.Lines), pattern, m.CurrentPart().Beats)
	if !slices.Contains(showLines, m.gridCursor.Line) {
		center := int(m.gridCursor.Line)
		max := len(m.definition.Lines)
		for i := range 2 * max {
			var index int
			if i%2 == 0 {
				index = center + (i / 2)
			} else {
				index = center - ((i + 1) / 2)
			}
			if index < 0 || index >= max {
				continue
			}
			if slices.Contains(showLines, uint8(index)) || !m.hideEmptyLines {
				m.SetGridCursor(gridKey{
					Line: uint8(index),
					Beat: m.gridCursor.Beat,
				})
				return
			}
		}
	}
}

func (m *model) CursorRight() {
	if m.gridCursor.Beat < m.CurrentPart().Beats-1 {
		m.SetGridCursor(gridKey{
			Line: m.gridCursor.Line,
			Beat: m.gridCursor.Beat + 1,
		})
	}
}

func (m *model) CursorLeft() {
	if m.gridCursor.Beat > 0 {
		m.SetGridCursor(gridKey{
			Line: m.gridCursor.Line,
			Beat: m.gridCursor.Beat - 1,
		})
	}
}

func (m *model) SetSelectionIndicator(desiredIndicator operation.Selection) {
	m.patternMode = operation.PatternFill
	m.selectionIndicator = desiredIndicator
	m.firstDigitApplied = false
}

func (m *model) PushUndoableDefinitionState() {
	if m.temporaryState.active {
		diff := createStateDiff(&m.definition, &m.temporaryState)
		if diff.Changed() {
			m.PushUndoables(UndoStateDiff{diff}, UndoStateDiff{diff.Reverse()})
		}
		if m.CurrentPart().Beats != m.temporaryState.beats {
			m.PushUndoables(UndoBeats{m.temporaryState.beats, m.arrangement.Cursor}, UndoBeats{m.CurrentPart().Beats, m.arrangement.Cursor})
		}
		m.temporaryState = temporaryState{}
	}
}

func (m *model) CaptureTemporaryState() {
	linesCopy := make([]grid.LineDefinition, len(m.definition.Lines))
	for i, defLine := range m.definition.Lines {
		newLine := grid.LineDefinition{
			Channel: defLine.Channel,
			Note:    defLine.Note,
			MsgType: defLine.MsgType,
			Name:    defLine.Name,
		}
		linesCopy[i] = newLine
	}
	accentDataCopy := make([]config.Accent, len(m.definition.Accents.Data))
	copy(accentDataCopy, m.definition.Accents.Data)

	m.temporaryState = temporaryState{
		lines:        linesCopy,
		tempo:        m.definition.Tempo,
		subdivisions: m.definition.Subdivisions,
		accents: sequence.PatternAccents{
			End:    m.definition.Accents.End,
			Start:  m.definition.Accents.Start,
			Target: m.definition.Accents.Target,
			Data:   accentDataCopy,
		},
		beats:  m.CurrentPart().Beats,
		active: true,
	}
}

func (m *model) SetPatternMode(mode operation.PatternMode) {
	m.patternMode = mode
	m.selectionIndicator = operation.SelectGrid
}

func (m *model) Escape() {
	// NOTE: Visual mode is only exited when pressing escape in base state
	if m.selectionIndicator == operation.SelectGrid && m.patternMode == operation.PatternFill {
		m.visualSelection.visualMode = operation.VisualNone
	}

	m.ClampAndSyncTempo()
	m.ClampAccentValues()

	if m.selectionIndicator == operation.SelectGrid {
		m.focus = operation.FocusGrid
		m.arrangement.Escape()
	}
	m.patternMode = operation.PatternFill
	m.selectionIndicator = operation.SelectGrid
	m.textInput.Reset()

	m.overlayKeyEdit.Escape(m.currentOverlay.Key)
}

func (m *model) ClampAccentValues() {
	if m.definition.Accents.Start < m.definition.Accents.End {
		m.definition.Accents.Start = m.definition.Accents.End
		m.definition.Accents.ReCalc()
	}
}

func (m *model) ClampAndSyncTempo() {
	if m.selectionIndicator == operation.SelectTempo {
		m.definition.Tempo = max(min(m.definition.Tempo, 300), 20)
		m.SyncTempo()
	}
}

func (m model) NewSequence() model {
	newModel := m

	definition, _ := LoadFile("", m.definition.Template, m.definition.Instrument)
	newModel.definition = definition
	newModel.arrangement = arrangement.InitModel(definition.Arrangement, definition.Parts)
	newModel.ResetIterations()
	newModel.SetGridCursor(GK(0, 0))
	newModel.undoStack = UndoStack{}
	newModel.redoStack = UndoStack{}
	newModel.activeChord = overlays.OverlayChord{}
	newModel.currentOverlay = newModel.CurrentPart().Overlays
	newModel.focus = operation.FocusGrid
	newModel.selectionIndicator = operation.SelectGrid
	newModel.patternMode = operation.PatternFill
	newModel.partSelectorIndex = 0
	newModel.needsWrite = 0
	newModel.overlayKeyEdit.SetOverlayKey(overlaykey.ROOT)

	return newModel
}

func (m model) NeedsWrite() bool {
	return m.needsWrite != m.undoStack.id
}

func (m *model) NextTheme() {
	index := slices.Index(themes.Themes, m.theme)
	if index+1 < len(themes.Themes) {
		m.theme = themes.Themes[index+1]
	} else {
		m.theme = themes.Themes[0]
	}
	themes.ChooseTheme(m.theme)
}

func (m *model) PrevTheme() {
	index := slices.Index(themes.Themes, m.theme)
	if index-1 >= 0 {
		m.theme = themes.Themes[index-1]
	} else {
		m.theme = themes.Themes[len(themes.Themes)-1]
	}
	themes.ChooseTheme(m.theme)
}

func (m *model) NextSection() {
	if m.arrangement.Cursor.MoveNext() {
		m.ResetCurrentOverlay()
		m.arrangement.ResetDepth()
	}
}

func (m *model) PrevSection() {
	if m.arrangement.Cursor.MovePrev() {
		m.ResetCurrentOverlay()
		m.arrangement.ResetDepth()
	}
}

func (m *model) ResetIterations() {
	iterations := make(playstate.Iterations)
	playstate.BuildIterationsMap(m.arrangement.Root, &iterations)
	m.playState.Iterations = &iterations
}

func (m *model) Start(delay time.Duration) {
	if m.midiConnection.HasOutport() {
		// DO NOTHING
	} else if m.midiConnection.HasDevices() {
		m.midiConnection.EnsureConnection()
	} else {
		m.SetCurrentError(fault.New("cannot open midi connection", fmsg.WithDesc("No MIDI devices found", "Please ensure a MIDI device is connected")))
		m.playState.Playing = false
		m.playState.PlayMode = playstate.PlayStandard
		return
	}

	m.ResetIterations()
	m.arrangement.ResetDepth()

	switch m.playState.LoopMode {
	case playstate.OneTimeWholeSequence:
		m.playState.LoopedArrangement = nil
		m.arrangement.Cursor = arrangement.ArrCursor{m.definition.Arrangement}
		m.arrangement.Cursor.MoveNext()
		section := m.CurrentSongSection()
		m.playState.LineStates = playstate.InitLineStates(len(m.definition.Lines), m.playState.LineStates, uint8(section.StartBeat))
	case playstate.LoopWholeSequence:
		m.playState.LoopedArrangement = m.arrangement.Root
		m.arrangement.Cursor = arrangement.ArrCursor{m.definition.Arrangement}
		m.arrangement.Cursor.MoveNext()
		section := m.CurrentSongSection()
		m.playState.LineStates = playstate.InitLineStates(len(m.definition.Lines), m.playState.LineStates, uint8(section.StartBeat))
	case playstate.LoopPart:
		m.playState.LoopedArrangement = m.arrangement.CurrentNode()
		section := m.CurrentSongSection()
		m.playState.LineStates = playstate.InitLineStates(len(m.definition.Lines), m.playState.LineStates, uint8(section.StartBeat))
	case playstate.LoopOverlay:
		m.playState.LoopedArrangement = m.arrangement.CurrentNode()
		(*m.playState.Iterations)[m.arrangement.CurrentNode()] = m.currentOverlay.Key.GetMinimumKeyCycle()
		if m.playState.BoundedLoop.Active {
			m.playState.LineStates = playstate.InitLineStates(len(m.definition.Lines), m.playState.LineStates, uint8(m.playState.BoundedLoop.LeftBound))
		} else {
			section := m.CurrentSongSection()
			m.playState.LineStates = playstate.InitLineStates(len(m.definition.Lines), m.playState.LineStates, uint8(section.StartBeat))
		}
	}

	if m.playState.Playing {
		time.AfterFunc(delay, func() {
			// NOTE: Order matters here, modelMsg must be sent before startMsg
			updateChannel <- beats.ModelMsg{Sequence: m.definition, PlayState: m.playState, Cursor: m.arrangement.Cursor}
			if m.playState.PlayMode != playstate.PlayReceiver {
				timingChannel <- timing.StartMsg{LoopMode: m.playState.LoopMode, Tempo: m.definition.Tempo, Subdivisions: m.definition.Subdivisions, Prerollbeats: m.playState.RecordPreRollBeats, Transmitting: m.transmitting}
			}
		})
	}
}

func (m *model) Stop() {
	m.playState.AllowAdvance = false
	m.playState.RecordPreRollBeats = 0
	m.playState.LoopMode = playstate.OneTimeWholeSequence
	m.playState.LoopedArrangement = nil
	m.arrangement.ResetDepth()
	m.ResetCurrentOverlay()
	m.SyncBeatLoop()

	notes := notereg.Clear()
	for _, n := range notes {
		beatsLooper.PlayMessage(time.Duration(0), n)
	}
}

func (m *model) StartStop(delay time.Duration) {
	m.playEditing = false
	if !m.playState.Playing {
		m.SafeStart(delay)
	} else {
		m.SafeStop()
	}
}

func (m *model) SafeStart(delay time.Duration) {
	m.playState.Playing = true
	m.playState.PlayMode = playstate.PlayStandard
	if m.midiLoopMode == timing.MlmReceiver {
		// NOTE: When instance is receiver, allow it to play alone and lock out transmitter messages
		m.lockReceiverChannel <- true
	}
	m.Start(delay)
}

func (m *model) SafeStop() {
	m.playState.Playing = false
	if m.playState.PlayMode == playstate.PlayStandard {
		go func() {
			timingChannel <- timing.StopMsg{}
			if m.midiLoopMode == timing.MlmReceiver {
				// NOTE: Unlock to allow transmitter messages
				m.unlockReceiverChannel <- true
			}
		}()
	} else {
		go func() {
			timingChannel <- timing.StopMsg{}
		}()
	}
	m.Stop()
}

func AdvanceSelectionState(states []operation.Selection, currentSelection operation.Selection) operation.Selection {
	index := slices.Index(states, currentSelection)
	var resultSelection operation.Selection
	if index < 0 {
		indexNothing := slices.Index(states, operation.SelectGrid)
		resultSelection = states[uint8(indexNothing+1)%uint8(len(states))]
	} else {
		resultSelection = states[uint8(index+1)%uint8(len(states))]
	}
	return resultSelection
}

func (m model) UpdateDefinitionKeys(mapping mappings.Mapping) model {
	chord, _ := m.currentOverlay.FindChord(m.gridCursor, m.currentOverlay.Key.GetMinimumKeyCycle())
	m.activeChord = chord

	if operation.IsNumberSelection(m.selectionIndicator) && mapping.Command == mappings.NumberPattern {
		number, isNumber := mappings.MappingToNumber(mapping)
		if isNumber {
			switch m.selectionIndicator {
			case operation.SelectSpecificValue:
				note, _ := m.CurrentNote()
				m.SetSpecificValue(note, number)
			case operation.SelectStartBeats:
				m.SetStartBeats(number)
			case operation.SelectStartCycles:
				m.SetStartCycles(number)
			case operation.SelectBeats:
				m.SetBeats(number)
			case operation.SelectCycles:
				m.SetCycles(number)
			case operation.SelectTempo:
				m.SetTempo(number)
			case operation.SelectTempoSubdivision:
				m.SetTempoSubdivision(number)
			case operation.SelectSetupChannel:
				m.SetSetupChannel(number)
			case operation.SelectSetupValue:
				m.SetSetupValue(number)
			case operation.SelectAccentStart:
				m.SetAccentStart(number)
			case operation.SelectAccentEnd:
				m.SetAccentEnd(number)
			case operation.SelectEuclideanHits:
				m.SetEuclideanHits(number)
			}
		}
		return m
	}

	switch mapping.Command {
	case mappings.ConfirmEuclidenHits:
		m.ApplyEuclidean(m.euclideanHits)
		m.SetSelectionIndicator(operation.SelectGrid)
	case mappings.NoteAdd:
		m.AddNote()
	case mappings.NoteRemove:
		m.yankBuffer = m.Yank()
		m.RemoveNote()
		m.visualSelection.visualMode = operation.VisualNone
	case mappings.AccentIncrease:
		m.AccentModify(1)
	case mappings.AccentDecrease:
		m.AccentModify(-1)
	case mappings.GateIncrease:
		m.GateModify(1)
	case mappings.GateDecrease:
		m.GateModify(-1)
	case mappings.GateBigIncrease:
		m.GateModify(8)
	case mappings.GateBigDecrease:
		m.GateModify(-8)
	case mappings.WaitIncrease:
		m.WaitModify(1)
	case mappings.WaitDecrease:
		m.WaitModify(-1)
	case mappings.OverlayNoteRemove:
		m.OverlayRemoveNote()
	case mappings.ClearLine:
		if m.visualSelection.visualMode == operation.VisualNone {
			m.ClearOverlayLine()
		} else {
			m.RemoveNote()
		}
	case mappings.ClearOverlay:
		m.ClearOverlay()
		m.UnsetActiveChord()
	case mappings.RatchetIncrease:
		m.RatchetModify(1)
	case mappings.RatchetDecrease:
		m.RatchetModify(-1)
		m.EnsureRatchetCursorVisible()
	case mappings.ActionAddLineReset:
		m.AddAction(grid.ActionLineReset)
	case mappings.ActionAddLineReverse:
		m.AddAction(grid.ActionLineReverse)
	case mappings.ActionAddSkipBeat:
		m.AddAction(grid.ActionLineSkipBeat)
	case mappings.ActionAddLineResetAll:
		m.AddAction(grid.ActionLineResetAll)
	case mappings.ActionAddLineBounce:
		m.AddAction(grid.ActionLineBounce)
	case mappings.ActionAddLineBounceAll:
		m.AddAction(grid.ActionLineBounceAll)
	case mappings.ActionAddLineDelay:
		m.AddAction(grid.ActionLineDelay)
	case mappings.ActionAddSpecificValue:
		if m.definition.Lines[m.gridCursor.Line].MsgType != grid.MessageTypeNote {
			m.AddAction(grid.ActionSpecificValue)
			m.SetSelectionIndicator(operation.SelectSpecificValue)
		}
	case mappings.SelectKeyLine:
		m.definition.Keyline = m.gridCursor.Line
	case mappings.OverlayStackToggle:
		m.currentOverlay.ToggleOverlayStackOptions()
	case mappings.RotateRight:
		switch m.definition.TemplateSequencerType {
		default:
			m.RotateRight()
		case operation.SeqModeChord:
			m.EnsureChord()
			m.MoveChordRight()
			m.CursorRight()
		}
	case mappings.RotateLeft:
		switch m.definition.TemplateSequencerType {
		default:
			m.RotateLeft()
		case operation.SeqModeChord:
			m.EnsureChord()
			m.MoveChordLeft()
			m.CursorLeft()
		}
	case mappings.RotateUp:
		switch m.definition.TemplateSequencerType {
		case operation.SeqModeChord:
			m.EnsureChord()
			m.MoveChordUp()
			m.CursorUp()
		default:
			m.RotateUp()
		}
	case mappings.RotateDown:
		switch m.definition.TemplateSequencerType {
		case operation.SeqModeChord:
			m.EnsureChord()
			m.MoveChordDown()
			m.CursorDown()
		default:
			m.RotateDown()
		}
	case mappings.Reverse:
		m.Reverse()
	case mappings.Paste:
		m.Paste()
	case mappings.Duplicate:
		m.Duplicate()
	case mappings.Euclidean:
		m.euclideanHits = 1
		m.SetSelectionIndicator(operation.SelectEuclideanHits)
	case mappings.MajorTriad:
		m.EnsureChord()
		m.ChordChange(theory.MajorTriad)
	case mappings.MinorTriad:
		m.EnsureChord()
		m.ChordChange(theory.MinorTriad)
	case mappings.AugmentedTriad:
		m.EnsureChord()
		m.ChordChange(theory.AugmentedTriad)
	case mappings.DiminishedTriad:
		m.EnsureChord()
		m.ChordChange(theory.DiminishedTriad)
	case mappings.MajorSeventh:
		m.EnsureChord()
		m.ChordChange(theory.MajorSeventh)
	case mappings.MinorSeventh:
		m.EnsureChord()
		m.ChordChange(theory.MinorSeventh)
	case mappings.AugFifth:
		m.EnsureChord()
		m.ChordChange(theory.Aug5)
	case mappings.DimFifth:
		m.EnsureChord()
		m.ChordChange(theory.Dim5)
	case mappings.PerfectFifth:
		m.EnsureChord()
		m.ChordChange(theory.Perfect5)
	case mappings.MinorSecond:
		m.EnsureChord()
		m.ChordChange(theory.Minor2)
	case mappings.MajorSecond:
		m.EnsureChord()
		m.ChordChange(theory.Major2)
	case mappings.MinorThird:
		m.EnsureChord()
		m.ChordChange(theory.Minor3)
	case mappings.MajorThird:
		m.EnsureChord()
		m.ChordChange(theory.Major3)
	case mappings.PerfectFourth:
		m.EnsureChord()
		m.ChordChange(theory.Perfect4)
	case mappings.MajorSixth:
		m.EnsureChord()
		m.ChordChange(theory.Major6)
	case mappings.Octave:
		m.EnsureChord()
		m.ChordChange(theory.Octave)
	case mappings.MinorNinth:
		m.EnsureChord()
		m.ChordChange(theory.Minor9)
	case mappings.MajorNinth:
		m.EnsureChord()
		m.ChordChange(theory.Major9)
	case mappings.IncreaseInversions:
		m.EnsureChord()
		m.NextInversion()
		m.MoveCursorToChord()
	case mappings.DecreaseInversions:
		m.EnsureChord()
		m.PreviousInversion()
		m.MoveCursorToChord()
	case mappings.NextArpeggio:
		m.EnsureChord()
		m.NextArpeggio()
		m.MoveCursorToChord()
	case mappings.PrevArpeggio:
		m.EnsureChord()
		m.PrevArpeggio()
		m.MoveCursorToChord()
	case mappings.NextDouble:
		m.EnsureChord()
		m.NextDouble()
	case mappings.PrevDouble:
		m.EnsureChord()
		m.PrevDouble()
	case mappings.OmitRoot:
		m.EnsureChord()
		m.OmitChordNote(theory.OmitRoot)
	case mappings.OmitSecond:
		m.EnsureChord()
		m.OmitChordNote(theory.OmitSecond)
	case mappings.OmitThird:
		m.EnsureChord()
		m.OmitChordNote(theory.OmitThird)
	case mappings.OmitFourth:
		m.EnsureChord()
		m.OmitChordNote(theory.OmitFourth)
	case mappings.OmitFifth:
		m.EnsureChord()
		m.OmitChordNote(theory.OmitFifth)
	case mappings.OmitSixth:
		m.EnsureChord()
		m.OmitChordNote(theory.OmitSixth)
	case mappings.OmitSeventh:
		m.EnsureChord()
		m.OmitChordNote(theory.OmitSeventh)
	case mappings.OmitOctave:
		m.EnsureChord()
		m.OmitChordNote(theory.OmitOctave)
	case mappings.OmitNinth:
		m.EnsureChord()
		m.OmitChordNote(theory.OmitNinth)
	case mappings.RemoveChord:
		m.RemoveChord()
	case mappings.ConvertToNotes:
		m.ConvertChordToNotes()
	}

	if mapping.LastValue >= "1" && mapping.LastValue <= "9" {
		beatInterval, _ := strconv.ParseInt(mapping.LastValue, 0, 8)
		switch m.patternMode {
		case operation.PatternFill:
			m.fill(uint8(beatInterval))
		case operation.PatternAccent:
			m.incrementAccent(uint8(beatInterval), -1, operation.EveryBeat)
		case operation.PatternNoteAccent:
			m.incrementAccent(uint8(beatInterval), -1, operation.EveryNote)
		case operation.PatternGate:
			m.incrementGate(uint8(beatInterval), -1, operation.EveryBeat)
		case operation.PatternNoteGate:
			m.incrementGate(uint8(beatInterval), -1, operation.EveryNote)
		case operation.PatternRatchet:
			m.incrementRatchet(uint8(beatInterval), -1, operation.EveryBeat)
		case operation.PatternNoteRatchet:
			m.incrementRatchet(uint8(beatInterval), -1, operation.EveryNote)
		case operation.PatternWait:
			m.incrementWait(uint8(beatInterval), -1, operation.EveryBeat)
		case operation.PatternNoteWait:
			m.incrementWait(uint8(beatInterval), -1, operation.EveryNote)
		}
	}

	if mappings.IsShiftSymbol(mapping.LastValue) {
		beatInterval := convertSymbolToInt(mapping.LastValue)
		switch m.patternMode {
		case operation.PatternFill:
			m.fillSpaces(uint8(beatInterval))
		case operation.PatternAccent:
			m.incrementAccent(uint8(beatInterval), 1, operation.EveryBeat)
		case operation.PatternNoteAccent:
			m.incrementAccent(uint8(beatInterval), 1, operation.EveryNote)
		case operation.PatternGate:
			m.incrementGate(uint8(beatInterval), 1, operation.EveryBeat)
		case operation.PatternNoteGate:
			m.incrementGate(uint8(beatInterval), 1, operation.EveryNote)
		case operation.PatternRatchet:
			m.incrementRatchet(uint8(beatInterval), 1, operation.EveryBeat)
		case operation.PatternNoteRatchet:
			m.incrementRatchet(uint8(beatInterval), 1, operation.EveryNote)
		case operation.PatternWait:
			m.incrementWait(uint8(beatInterval), 1, operation.EveryBeat)
		case operation.PatternNoteWait:
			m.incrementWait(uint8(beatInterval), 1, operation.EveryNote)
		}
	}

	return m
}

func (m *model) SetEuclideanHits(number int) {
	var maxSteps uint8
	if m.visualSelection.visualMode == operation.VisualNone {
		maxSteps = m.CurrentPart().Beats
	} else {
		bounds := m.visualSelection.Bounds(m.CurrentPart().Beats)
		maxSteps = bounds.Right - bounds.Left

	}
	m.euclideanHits =
		uint8(m.clamp(m.UnshiftDigit(int(m.euclideanHits), number), 1, int(maxSteps)))
}

func (m *model) SetAccentEnd(number int) {
	m.definition.Accents.End =
		uint8(m.clamp(m.UnshiftDigit(int(m.definition.Accents.End), number), 0, int(m.definition.Accents.Start)))
	m.definition.Accents.ReCalc()
}

func (m *model) SetAccentStart(number int) {
	m.definition.Accents.Start =
		uint8(m.clamp(m.UnshiftDigit(int(m.definition.Accents.Start), number), 0, 127))
	if m.definition.Accents.Start >= m.definition.Accents.End {
		m.definition.Accents.ReCalc()
	}
}

func (m *model) SetSetupValue(number int) {
	m.definition.Lines[m.gridCursor.Line].Note =
		uint8(m.clamp(m.UnshiftDigit(int(m.definition.Lines[m.gridCursor.Line].Note), number), 1, 127))
}

func (m *model) SetSetupChannel(number int) {
	m.definition.Lines[m.gridCursor.Line].Channel =
		uint8(m.clamp(m.UnshiftDigit(int(m.definition.Lines[m.gridCursor.Line].Channel), number), 1, 16))
}

func (m *model) SetTempoSubdivision(number int) {
	m.definition.Subdivisions = m.clamp(number, 1, 8)
}

func (m *model) SetTempo(number int) {
	m.definition.Tempo = m.clamp(m.UnshiftDigit(m.definition.Tempo, number), 1, 300)
}

func (m *model) SetStartCycles(number int) {
	currentNode := m.arrangement.Cursor.GetCurrentNode()
	value := currentNode.Section.StartCycles
	newValue := m.UnshiftDigit(value, number)
	clampedValue := m.clamp(newValue, 0, 127)
	currentNode.Section.StartCycles = clampedValue
}

func (m *model) SetBeats(number int) {
	currentNode := m.arrangement.Cursor.GetCurrentNode()
	partId := currentNode.Section.Part
	value := (*m.definition.Parts)[partId].Beats
	newValue := m.UnshiftDigit(int(value), number)
	clampedValue := m.clamp(newValue, 0, 127)
	(*m.definition.Parts)[partId].Beats = uint8(clampedValue)
}

func (m *model) SetCycles(number int) {
	currentNode := m.arrangement.Cursor.GetCurrentNode()
	value := currentNode.Section.Cycles
	newValue := m.UnshiftDigit(value, number)
	clampedValue := m.clamp(newValue, 0, 127)
	currentNode.Section.Cycles = clampedValue
}

func (m *model) SetStartBeats(number int) {
	currentNode := m.arrangement.Cursor.GetCurrentNode()
	value := currentNode.Section.StartBeat
	newValue := m.UnshiftDigit(value, number)
	clampedValue := m.clamp(newValue, 0, 127)
	currentNode.Section.StartBeat = clampedValue
}

func (m *model) SetSpecificValue(note note, number int) {
	value := note.AccentIndex
	newValue := m.UnshiftDigit(int(value), number)
	clampedValue := m.clamp(newValue, 0, 127)
	note.AccentIndex = uint8(clampedValue)
	m.currentOverlay.SetNote(m.gridCursor, note)
}

func (m *model) clamp(value, min, max int) int {
	if value < min {
		m.firstDigitApplied = false
		return min
	} else if value > max {
		m.firstDigitApplied = false
		return max
	} else {
		return value
	}
}

func (m *model) UnshiftDigit(digits int, newDigit int) int {
	if m.firstDigitApplied {
		return (int(digits)%100)*10 + newDigit
	} else {
		m.firstDigitApplied = true
		return newDigit
	}
}

func (m *model) MoveCursorToChord() {
	pattern := make(grid.Pattern)
	m.activeChord.GridChord.ArpeggiatedPattern(&pattern)
	keys := maps.Keys(pattern)
	collectedKeys := slices.Collect(keys)
	slices.SortFunc(collectedKeys, grid.Compare)
	m.gridCursor = collectedKeys[len(collectedKeys)-1]
}

func (m *model) EnsureChord() {
	overlayChord := m.CurrentChord()
	if overlayChord.HasValue() {
		if !overlayChord.BelongsTo(m.currentOverlay) {
			newGridChord := m.currentOverlay.SetChord(overlayChord.GridChord)
			m.activeChord = overlays.OverlayChord{GridChord: newGridChord}
		}
	}
}

func (m model) CurrentChord() overlays.OverlayChord {
	if m.definition.TemplateSequencerType == operation.SeqModeLine {
		return overlays.OverlayChord{}
	}
	overlayChord, exists := m.currentOverlay.FindChord(m.gridCursor, m.currentOverlay.Key.GetMinimumKeyCycle())
	if exists {
		return overlayChord
	} else if m.activeChord.GridChord != nil {
		return m.activeChord
	} else {
		return overlays.OverlayChord{}
	}
}

func (m *model) ChordChange(alteration uint32) {
	overlayChord := m.CurrentChord()
	if overlayChord.GridChord == nil {
		m.currentOverlay.CreateChord(m.gridCursor, alteration)
	} else {
		overlayChord.GridChord.ApplyAlteration(alteration)
	}
}

func (m *model) RemoveChord() {
	if m.activeChord.HasValue() {
		m.currentOverlay.RemoveChord(m.activeChord)
		m.UnsetActiveChord()
	}
}

func (m *model) ConvertChordToNotes() {
	if m.activeChord.HasValue() {
		pattern := make(grid.Pattern)
		m.activeChord.GridChord.ArpeggiatedPattern(&pattern)
		m.currentOverlay.RemoveChord(m.activeChord)
		for gk, note := range pattern {
			m.currentOverlay.SetNote(gk, note)
		}
		m.activeChord = overlays.OverlayChord{}
	}
}

func (m *model) OmitChordNote(omission uint32) {
	if m.activeChord.HasValue() {
		m.activeChord.GridChord.Chord.OmitNote(omission)
	}
}

func (m *model) NextInversion() {
	if m.activeChord.HasValue() {
		m.activeChord.GridChord.Chord.NextInversion()
	}
}

func (m *model) PreviousInversion() {
	if m.activeChord.HasValue() {
		m.activeChord.GridChord.Chord.PreviousInversion()
	}
}

func (m *model) NextArpeggio() {
	if m.activeChord.HasValue() {
		m.activeChord.GridChord.NextArp()
	}
}

func (m *model) PrevArpeggio() {
	if m.activeChord.HasValue() {
		m.activeChord.GridChord.PrevArp()
	}
}

func (m *model) NextDouble() {
	if m.activeChord.HasValue() {
		m.activeChord.GridChord.NextDouble()
	}
}

func (m *model) PrevDouble() {
	if m.activeChord.HasValue() {
		m.activeChord.GridChord.PrevDouble()
	}
}

func convertSymbolToInt(symbol string) int64 {
	switch symbol[0] {
	case '!':
		return 1
	case '@':
		return 2
	case '#':
		return 3
	case '$':
		return 4
	case '%':
		return 5
	case '^':
		return 6
	case '&':
		return 7
	case '*':
		return 8
	case '(':
		return 9
	}
	return 0
}

func (m model) UpdateDefinition(mapping mappings.Mapping) model {

	deepCopy := overlays.DeepCopy(m.currentOverlay)
	if !m.playEditing {
		m.EnsureOverlay()
	}
	if m.playState.Playing && !m.playEditing {
		m.playEditing = true
		m.editingPartID = m.CurrentPartID()
		currentCycles := (*m.playState.Iterations)[m.arrangement.CurrentNode()]
		playingOverlay := m.CurrentPart().Overlays.HighestMatchingOverlay(currentCycles)
		m.currentOverlay = playingOverlay
		m.overlayKeyEdit.SetOverlayKey(playingOverlay.Key)
	}
	m = m.UpdateDefinitionKeys(mapping)
	undoable := m.UndoableOverlay(m.currentOverlay, deepCopy)
	redoable := m.UndoableOverlay(deepCopy, m.currentOverlay)

	if !undoable.overlayDiff.IsEmpty() {
		m.PushUndoables(undoable, redoable)
		m.ResetRedo()
	}

	return m
}

func (m model) UndoableOverlay(overlayA, overlayB *overlays.Overlay) UndoOverlayDiff {
	diff := overlays.DiffOverlays(overlayA, overlayB)
	return UndoOverlayDiff{m.currentOverlay.Key, m.gridCursor, m.arrangement.Cursor, diff}
}

func (m *model) Save() {
	err := sequence.Write(m.definition, m.filename)
	if err != nil {
		m.SetCurrentError(fault.Wrap(err, fmsg.With("cannot write file")))
	}
	m.needsWrite = m.undoStack.id
}

func (m model) CurrentPart() arrangement.Part {
	section := m.CurrentSongSection()
	partID := section.Part
	return (*m.definition.Parts)[partID]
}

func (m model) RenamePart(value string) {
	section := m.CurrentSongSection()
	partID := section.Part
	(*m.definition.Parts)[partID].Name = value
}

func (m model) CurrentPartID() int {
	section := m.CurrentSongSection()
	return section.Part
}

func (m model) CurrentSongSection() arrangement.SongSection {
	currentNode := m.arrangement.Cursor.GetCurrentNode()
	if currentNode != nil && currentNode.IsEndNode() {
		return currentNode.Section
	}

	panic("Cursor should always be at end node")
}

func (m model) CurrentNote() (note, bool) {
	note, exists := m.currentOverlay.GetNote(m.gridCursor)
	return note, exists
}

func (m *model) NextOverlay(direction int) {
	switch direction {
	case 1:
		overlay := m.CurrentPart().Overlays.FindAboveOverlay(m.currentOverlay.Key)
		m.currentOverlay = overlay
	case -1:
		if m.currentOverlay.Below != nil {
			m.currentOverlay = m.currentOverlay.Below
		}
	default:
	}
}

func (m *model) ClearOverlayLine() {
	start := m.gridCursor.Beat
	for i := start; i < m.CurrentPart().Beats; i++ {
		m.currentOverlay.SetNote(GK(m.gridCursor.Line, i), zeronote)
	}
}

func (m *model) RemoveOverlay() {
	newOverlay := (*m.definition.Parts)[m.CurrentPartID()].Overlays.Remove(m.currentOverlay.Key)
	if newOverlay != nil {
		(*m.definition.Parts)[m.CurrentPartID()].Overlays = newOverlay
		m.currentOverlay = (*m.definition.Parts)[m.CurrentPartID()].Overlays
		m.overlayKeyEdit.SetOverlayKey(m.currentOverlay.Key)
	}
}

func (m *model) ClearOverlay() {
	m.currentOverlay.Clear()
}

func (m *model) ClearAllOverlays() {
	for i := range *m.definition.Parts {
		(*m.definition.Parts)[i].Overlays.ClearRecursive()
	}
}

type MoveFunc func()

func (m *model) RotateRight() {
	combinedPattern := m.CombinedEditPattern(m.currentOverlay)

	lineStart, lineEnd := m.PatternActionLineBoundaries()
	start, end := m.PatternActionBeatBoundaries()

	for l := lineStart; l <= lineEnd; l++ {

		moves := make([]MoveFunc, 0, (end - start))
		lastKey := GK(l, end)
		firstKey := GK(l, start)
		lastNote := combinedPattern[lastKey]
		previousNote := zeronote
		var previousKey grid.GridKey
		for i := start; i <= end; i++ {
			currentKey := GK(l, i)
			currentNote := combinedPattern[currentKey]

			chord, chordExists := m.currentOverlay.Chords.FindChordWithNote(previousKey)
			if chordExists {
				fromKey := previousKey
				moves = append(moves, func() {
					chord.Move(fromKey, currentKey)
				})
			} else {
				noteToMove := previousNote
				moves = append(moves, func() {
					m.currentOverlay.MoveNoteTo(currentKey, noteToMove)
				})
			}
			previousNote = currentNote
			previousKey = currentKey
		}
		noteToMove := lastNote
		chord, chordExists := m.currentOverlay.Chords.FindChordWithNote(previousKey)
		if chordExists {
			chord.Move(previousKey, firstKey)
		} else {
			moves = append(moves, func() {
				m.currentOverlay.MoveNoteTo(firstKey, noteToMove)
			})
		}

		for _, moveFn := range moves {
			if moveFn != nil {
				moveFn()
			}
		}
	}
}

func (m *model) RotateLeft() {
	combinedPattern := m.CombinedEditPattern(m.currentOverlay)

	lineStart, lineEnd := m.PatternActionLineBoundaries()
	start, end := m.PatternActionBeatBoundaries()
	for l := lineStart; l <= lineEnd; l++ {
		moves := make([]MoveFunc, 0, (end - start))
		firstNote := combinedPattern[GK(l, start)]
		previousNote := zeronote
		var previousKey grid.GridKey
		for i := int8(end); i >= int8(start); i-- {
			currentKey := GK(l, uint8(i))
			currentNote := combinedPattern[currentKey]

			chord, chordExists := m.currentOverlay.Chords.FindChordWithNote(previousKey)
			if chordExists {
				fromKey := previousKey
				moves = append(moves, func() {
					chord.Move(fromKey, currentKey)
				})
			} else {
				noteToMove := previousNote
				moves = append(moves, func() {
					m.currentOverlay.MoveNoteTo(currentKey, noteToMove)
				})
			}

			previousNote = currentNote
			previousKey = currentKey
		}

		chord, chordExists := m.currentOverlay.Chords.FindChordWithNote(previousKey)
		if chordExists {
			moves = append(moves, func() {
				chord.Move(GK(l, start), GK(l, end))
			})
		} else {
			moves = append(moves, func() {
				m.currentOverlay.MoveNoteTo(GK(l, end), firstNote)
			})
		}

		for _, moveFn := range moves {
			if moveFn != nil {
				moveFn()
			}
		}
	}
}

func (m *model) Reverse() {
	combinedPattern := m.CombinedEditPattern(m.currentOverlay)

	lineStart, lineEnd := m.PatternActionLineBoundaries()
	start, end := m.PatternActionBeatBoundaries()

	for l := lineStart; l <= lineEnd; l++ {
		// Collect standalone notes with their positions (not part of chords)
		type noteAtPosition struct {
			beat uint8
			note grid.Note
		}
		var standaloneNotes []noteAtPosition

		for i := start; i <= end; i++ {
			currentKey := GK(l, i)
			currentNote := combinedPattern[currentKey]
			_, chordExists := m.currentOverlay.Chords.FindChordWithNote(currentKey)

			// Collect notes that are NOT part of a chord
			if !chordExists && currentNote != zeronote {
				standaloneNotes = append(standaloneNotes, noteAtPosition{i, currentNote})
			}
		}

		// Clear only standalone notes
		for _, notePos := range standaloneNotes {
			currentKey := GK(l, notePos.beat)
			m.currentOverlay.SetNote(currentKey, zeronote)
		}

		// Place notes in mirrored positions (preserving gaps)
		// If a note was at beat B, place it at end - (B - start)
		for _, notePos := range standaloneNotes {
			mirroredBeat := end - (notePos.beat - start)
			targetKey := GK(l, mirroredBeat)
			m.currentOverlay.MoveNoteTo(targetKey, notePos.note)
		}
	}
}

func (m *model) ApplyEuclidean(hits uint8) {
	lineStart, lineEnd := m.PatternActionLineBoundaries()
	start, end := m.PatternActionBeatBoundaries()

	steps := int(end - start + 1)
	euclideanPattern := grid.GenerateEuclideanRhythm(int(hits), steps)

	for l := lineStart; l <= lineEnd; l++ {
		// Clear existing notes in the range
		for i := start; i <= end; i++ {
			key := GK(l, i)
			m.currentOverlay.RemoveNote(key)
		}

		// Apply Euclidean pattern
		for i, shouldHit := range euclideanPattern {
			if shouldHit {
				key := GK(l, start+uint8(i))
				m.currentOverlay.SetNote(key, grid.InitNote())
			}
		}
	}
}

func (m *model) RotateUp() {
	m.RotateNotesUp()
	m.RotateChordsUp()
}

func (m *model) RotateChordsUp() {
	chordPattern := make(overlays.ChordPattern)
	m.currentOverlay.CombineChords(&chordPattern, m.currentOverlay.Key.GetMinimumKeyCycle())

	lineStart, lineEnd := m.MonoModeLineBoundaries()
	beatStart, beatEnd := m.PatternActionBeatBoundaries()
	for beat := beatStart; beat <= beatEnd; beat++ {
		for l := lineStart; l <= lineEnd; l++ {
			key := GK(uint8(l), beat)
			chord, exists := chordPattern[key]
			if exists {
				var gridChord = chord.GridChord
				if !chord.BelongsTo(m.currentOverlay) {
					gridChord = m.currentOverlay.SetChord(chord.GridChord)
				}
				index := l - 1
				if l != lineStart {
					gridChord.Root.Line = uint8(index)
				} else {
					gridChord.Root.Line = lineEnd
				}
			}
		}
	}
}

func (m *model) RotateNotesUp() {
	pattern := m.CombinedNotePattern(m.currentOverlay)
	lineStart, lineEnd := m.MonoModeLineBoundaries()
	beatStart, beatEnd := m.PatternActionBeatBoundaries()
	for beat := beatStart; beat <= beatEnd; beat++ {
		for l := lineStart; l <= lineEnd; l++ {
			key := GK(uint8(l), beat)
			_, exists := pattern[key]
			if exists {
				m.currentOverlay.RemoveNote(key)
			}
			if l != lineStart {
				newKey := GK(uint8(l-1), beat)
				note := pattern[key]
				m.currentOverlay.MoveNoteTo(newKey, note)
			}
		}
		newKey := GK(lineEnd, beat)
		note := pattern[GK(lineStart, beat)]
		m.currentOverlay.MoveNoteTo(newKey, note)
	}
}

func (m *model) RotateDown() {
	m.RotateNotesDown()
	m.RotateChordsDown()
}

func (m *model) RotateChordsDown() {
	chordPattern := make(overlays.ChordPattern)
	m.currentOverlay.CombineChords(&chordPattern, m.currentOverlay.Key.GetMinimumKeyCycle())

	lineStart, lineEnd := m.MonoModeLineBoundaries()
	beatStart, beatEnd := m.PatternActionBeatBoundaries()
	for beat := beatStart; beat <= beatEnd; beat++ {
		for l := int(lineEnd); l >= int(lineStart); l-- {
			key := GK(uint8(l), beat)
			chord, exists := chordPattern[key]
			if exists {
				var gridChord = chord.GridChord
				if !chord.BelongsTo(m.currentOverlay) {
					gridChord = m.currentOverlay.SetChord(chord.GridChord)
				}
				index := l + 1
				if index <= int(lineEnd) {
					gridChord.Root.Line = uint8(index)
				} else {
					gridChord.Root.Line = lineStart
				}
			}
		}
	}
}

func (m *model) RotateNotesDown() {
	pattern := m.CombinedNotePattern(m.currentOverlay)
	lineStart, lineEnd := m.MonoModeLineBoundaries()
	beatStart, beatEnd := m.PatternActionBeatBoundaries()
	for beat := beatStart; beat <= beatEnd; beat++ {
		for l := int(lineEnd); l >= int(lineStart); l-- {
			key := GK(uint8(l), beat)
			_, exists := pattern[key]
			if exists {
				m.currentOverlay.RemoveNote(key)
			}
			index := l + 1
			if index <= int(lineEnd) {
				newKey := GK(uint8(index), beat)
				note := pattern[key]
				m.currentOverlay.MoveNoteTo(newKey, note)
			}
		}

		newKey := GK(lineStart, beat)
		note := pattern[GK(lineEnd, beat)]
		m.currentOverlay.MoveNoteTo(newKey, note)
	}
}

func (m *model) MoveChordLeft() {
	if m.activeChord.HasValue() {
		gridChord := m.activeChord.GridChord
		newBeat := gridChord.Root.Beat - 1
		gridChord.Root.Beat = newBeat
	}
}

func (m *model) MoveChordRight() {
	if m.activeChord.HasValue() {
		gridChord := m.activeChord.GridChord
		newBeat := gridChord.Root.Beat + 1
		gridChord.Root.Beat = newBeat
	}
}

func (m *model) MoveChordUp() {
	if m.activeChord.HasValue() {
		gridChord := m.activeChord.GridChord
		newLine := gridChord.Root.Line - 1
		gridChord.Root.Line = newLine
	}
}

func (m *model) MoveChordDown() {
	if m.activeChord.HasValue() {
		gridChord := m.activeChord.GridChord
		newLine := gridChord.Root.Line + 1
		gridChord.Root.Line = newLine
	}
}

func Mute(playState []playstate.LineState, line uint8) []playstate.LineState {
	switch playState[line].GroupPlayState {
	case playstate.PlayStatePlay:
		playState[line].GroupPlayState = playstate.PlayStateMute
	case playstate.PlayStateMute:
		playState[line].GroupPlayState = playstate.PlayStatePlay
	case playstate.PlayStateSolo:
		playState[line].GroupPlayState = playstate.PlayStateMute
	}
	return playState
}

func MuteAll(lineStates []playstate.LineState, line uint8) []playstate.LineState {
	for i := range lineStates {
		lineStates[i].GroupPlayState = playstate.PlayStateMute
	}
	return lineStates
}

func UnMuteAll(lineStates []playstate.LineState, line uint8) []playstate.LineState {
	for i := range lineStates {
		lineStates[i].GroupPlayState = playstate.PlayStatePlay
	}
	return lineStates
}

func Solo(playState []playstate.LineState, line uint8) []playstate.LineState {
	switch playState[line].GroupPlayState {
	case playstate.PlayStatePlay:
		playState[line].GroupPlayState = playstate.PlayStateSolo
	case playstate.PlayStateMute:
		playState[line].GroupPlayState = playstate.PlayStateSolo
	case playstate.PlayStateSolo:
		playState[line].GroupPlayState = playstate.PlayStatePlay
	}
	return playState
}

func (m model) HasSolo() bool {
	for _, state := range m.playState.LineStates {
		if state.GroupPlayState == playstate.PlayStateSolo {
			return true
		}
	}
	return false
}

type Buffer struct {
	bounds    grid.Bounds
	gridNotes []GridNote
	gridChord *overlays.GridChord
}

func (b Buffer) IsChord() bool {
	if b.gridChord != nil {
		return len(b.gridChord.Notes) > 0
	} else {
		return false
	}
}

func InitChordBuffer(gridChord *overlays.GridChord) Buffer {
	return Buffer{
		gridChord: gridChord,
	}
}

func InitBuffer(bounds grid.Bounds, notes []GridNote) Buffer {
	return Buffer{
		bounds:    bounds.Normalized(),
		gridNotes: notes,
	}
}

func (m model) Yank() Buffer {
	currentChord := m.CurrentChord()
	if m.visualSelection.visualMode == operation.VisualNone && currentChord.HasValue() {
		return InitChordBuffer(currentChord.GridChord)
	} else {
		bounds := m.YankBounds()
		combinedPattern := m.CombinedEditPattern(m.currentOverlay)
		capturedGridNotes := make([]GridNote, 0, len(combinedPattern))

		for key, note := range combinedPattern {
			if bounds.InBounds(key) {
				normalizedGridKey := GK(key.Line-bounds.Top, key.Beat-bounds.Left)
				capturedGridNotes = append(capturedGridNotes, GridNote{normalizedGridKey, note})
			}
		}

		return InitBuffer(bounds, capturedGridNotes)
	}

}

func (m *model) Paste() {
	if m.yankBuffer.IsChord() {
		m.currentOverlay.PasteChord(m.gridCursor, m.yankBuffer.gridChord)
	} else {
		bounds := m.PasteBounds()

		var keyModifier gridKey
		if m.visualSelection.visualMode == operation.VisualNone {
			keyModifier = m.gridCursor
		} else {
			keyModifier = bounds.TopLeft()
		}
		gridNotes := m.yankBuffer.gridNotes

		for _, gridNote := range gridNotes {
			key := gridNote.gridKey
			newKey := GK(key.Line+keyModifier.Line, key.Beat+keyModifier.Beat)
			note := gridNote.note
			if bounds.InBounds(newKey) {
				m.currentOverlay.SetNote(newKey, note)
			}
		}
	}
}

func (m *model) DuplicateVisualSelection() {
	bounds := m.visualSelection.Bounds(m.CurrentPart().Beats)
	originalVisualMode := m.visualSelection.visualMode
	combinedPattern := m.CombinedEditPattern(m.currentOverlay)

	// Calculate the offset for the new area (starts immediately after current area)
	beatOffset := bounds.Right + 1

	// Check if there's room for the duplicate
	if beatOffset <= m.CurrentPart().Beats-1 {
		// Copy all notes from the visual selection to the new location
		for key, note := range combinedPattern {
			if bounds.InBounds(key) {
				newKey := gridKey{
					Line: key.Line,
					Beat: key.Beat - bounds.Left + beatOffset,
				}
				// Only paste if the new location is within bounds
				if newKey.Beat < m.CurrentPart().Beats {
					m.currentOverlay.SetNote(newKey, note)
				}
			}
		}

		// Move cursor to the first beat of the new area
		m.SetGridCursor(gridKey{
			Line: bounds.Top,
			Beat: beatOffset,
		})

		// Create new visual selection in the duplicated area
		selectionWidth := bounds.Right - bounds.Left
		m.visualSelection.visualMode = originalVisualMode
		m.visualSelection.anchor = gridKey{
			Line: bounds.Top,
			Beat: beatOffset,
		}
		m.visualSelection.lead = gridKey{
			Line: bounds.Bottom,
			Beat: beatOffset + selectionWidth,
		}
	}
}

func (m *model) DuplicateLineToNextLine() {
	bounds := m.visualSelection.Bounds(m.CurrentPart().Beats)
	combinedPattern := m.CombinedEditPattern(m.currentOverlay)

	sourceLine := bounds.Top
	targetLine := sourceLine + 1

	// Check if there's room for the duplicate (not past the last line)
	if int(targetLine) < len(m.definition.Lines) {
		// Copy all notes from the source line to the target line
		for beat := bounds.Left; beat <= bounds.Right; beat++ {
			sourceKey := gridKey{Line: sourceLine, Beat: beat}
			if note, exists := combinedPattern[sourceKey]; exists {
				targetKey := gridKey{Line: targetLine, Beat: beat}
				m.currentOverlay.SetNote(targetKey, note)
			}
		}

		// Move cursor to the first beat of the new line
		m.SetGridCursor(gridKey{
			Line: targetLine,
			Beat: bounds.Left,
		})

		// Create new visual selection on the duplicated line
		m.visualSelection.visualMode = operation.VisualLine
		m.visualSelection.anchor = gridKey{
			Line: targetLine,
			Beat: 0,
		}
		m.visualSelection.lead = gridKey{
			Line: targetLine,
			Beat: 0,
		}
	}
}

func (m *model) DuplicateLinesToNextLines() {
	bounds := m.visualSelection.Bounds(m.CurrentPart().Beats)
	combinedPattern := m.CombinedEditPattern(m.currentOverlay)

	numLines := bounds.Bottom - bounds.Top + 1
	targetStartLine := bounds.Bottom + 1

	// Check if there's at least one line available below
	if int(targetStartLine) < len(m.definition.Lines) {
		// Calculate how many lines we can actually duplicate
		availableLines := uint8(len(m.definition.Lines)) - targetStartLine
		linesToDuplicate := numLines
		if availableLines < numLines {
			linesToDuplicate = availableLines
		}

		// Copy each line from the selection to the corresponding line below
		for i := uint8(0); i < linesToDuplicate; i++ {
			sourceLine := bounds.Top + i
			targetLine := targetStartLine + i

			// Copy all notes from the source line to the target line
			for beat := bounds.Left; beat <= bounds.Right; beat++ {
				sourceKey := gridKey{Line: sourceLine, Beat: beat}
				if note, exists := combinedPattern[sourceKey]; exists {
					targetKey := gridKey{Line: targetLine, Beat: beat}
					m.currentOverlay.SetNote(targetKey, note)
				}
			}
		}

		// Move cursor to the first beat of the first duplicated line
		m.SetGridCursor(gridKey{
			Line: targetStartLine,
			Beat: bounds.Left,
		})

		// Create new visual selection on the duplicated lines
		m.visualSelection.visualMode = operation.VisualLine
		m.visualSelection.anchor = gridKey{
			Line: targetStartLine,
			Beat: 0,
		}
		m.visualSelection.lead = gridKey{
			Line: targetStartLine + linesToDuplicate - 1,
			Beat: 0,
		}
	}
}

func (m *model) DuplicateChord() {
	currentChord := m.CurrentChord()
	if currentChord.HasValue() {
		// Calculate how many beats the chord spans
		// Consider both beat offsets and GateIndex of each note
		maxBeatsSpan := uint8(1)

		for _, beatNote := range currentChord.GridChord.Notes {
			// Calculate beats for this note's gate (ceiling division)
			beatsForGate := uint8(1)
			if beatNote.Note.GateIndex > 8 {
				beatsForGate = uint8((beatNote.Note.GateIndex + 7) / 8)
			}

			// Total span for this note is its beat offset + its gate length
			noteSpan := uint8(beatNote.Beat) + beatsForGate

			if noteSpan > maxBeatsSpan {
				maxBeatsSpan = noteSpan
			}
		}

		// Use the chord's root position, not the cursor position
		rootBeat := currentChord.GridChord.Root.Beat
		newBeat := rootBeat + maxBeatsSpan
		if newBeat < m.CurrentPart().Beats {
			newKey := gridKey{
				Line: currentChord.GridChord.Root.Line,
				Beat: newBeat,
			}
			m.currentOverlay.PasteChord(newKey, currentChord.GridChord)

			// Find the note on the highest line (maximum line number, lowest note)
			positions := currentChord.GridChord.Positions()

			// Move cursor to the highest line (lowest note) of the duplicated chord
			m.SetGridCursor(gridKey{
				Line: positions[0].Line,
				Beat: newBeat,
			})
		}
	}
}

func (m *model) DuplicateSingleNote() {
	// Check if we're on a chord first
	currentChord := m.CurrentChord()
	if currentChord.HasValue() {
		m.DuplicateChord()
		return
	}

	// Otherwise duplicate a single note
	if currentNote, exists := m.CurrentNote(); exists {
		// Calculate how many beats to skip based on GateIndex
		// GateIndex of 8 equals 1 beat
		beatsToSkip := uint8(1)
		if currentNote.GateIndex > 8 {
			// Calculate beats needed: ceiling(GateIndex / 8)
			beatsToSkip = uint8((currentNote.GateIndex + 7) / 8)
		}

		newBeat := m.gridCursor.Beat + beatsToSkip
		if newBeat < m.CurrentPart().Beats {
			newKey := gridKey{
				Line: m.gridCursor.Line,
				Beat: newBeat,
			}
			m.currentOverlay.SetNote(newKey, currentNote)

			// Move cursor to the duplicated note position
			m.SetGridCursor(newKey)
		}
	}
}

func (m *model) Duplicate() {
	if m.visualSelection.visualMode != operation.VisualNone {
		bounds := m.visualSelection.Bounds(m.CurrentPart().Beats)
		// Check if it's visual line mode
		if m.visualSelection.visualMode == operation.VisualLine {
			if bounds.Top == bounds.Bottom {
				// Single line selected
				m.DuplicateLineToNextLine()
			} else {
				// Multiple lines selected
				m.DuplicateLinesToNextLines()
			}
		} else {
			// Visual block mode
			m.DuplicateVisualSelection()
		}
	} else {
		m.DuplicateSingleNote()
	}
}

func (m model) CurrentBeatGridKeys(gridKeys *[]grid.GridKey) {
	for _, linestate := range m.playState.LineStates {
		if linestate.IsSolo() || (!linestate.IsMuted() && !m.playState.HasSolo) {
			*gridKeys = append(*gridKeys, linestate.GridKey())
		}
	}
}

func (m model) PlayingOverlayKeys() []overlayKey {
	keys := make([]overlayKey, 0, 10)

	currentCycles := (*m.playState.Iterations)[m.arrangement.CurrentNode()]
	m.CurrentPart().Overlays.GetMatchingOverlayKeys(&keys, currentCycles)
	return keys
}

func (m model) CombinedEditPattern(overlay *overlays.Overlay) grid.Pattern {
	pattern := make(grid.Pattern)
	overlay.CombineGridPattern(&pattern, overlay.Key.GetMinimumKeyCycle(), overlays.CombineTypeAll)
	return pattern
}

func (m model) CombinedNotePattern(overlay *overlays.Overlay) grid.Pattern {
	pattern := make(grid.Pattern)
	overlay.CombineGridPattern(&pattern, overlay.Key.GetMinimumKeyCycle(), overlays.CombineTypeNotes)
	return pattern
}

func (m model) CombinedOverlayPattern(overlay *overlays.Overlay) overlays.OverlayPattern {
	pattern := make(overlays.OverlayPattern)
	if m.playState.Playing && !m.playEditing {

		currentCycles := (*m.playState.Iterations)[m.arrangement.CurrentNode()]
		m.CurrentPart().Overlays.CombineOverlayPattern(&pattern, currentCycles)
	} else {
		overlay.CombineOverlayPattern(&pattern, overlay.Key.GetMinimumKeyCycle())
	}
	return pattern
}

func (m *model) Every(every uint8, everyFn func(gridKey)) {
	switch m.definition.TemplateSequencerType {
	case operation.SeqModeChord:
		m.EveryChordNote(every, everyFn)
	case operation.SeqModeMono:
		m.EveryMonoSpace(every, everyFn)
	default:
		m.EveryLineSpace(every, everyFn)
	}
}

func (m *model) EveryMonoSpace(every uint8, everyFn func(gridKey)) {
	lineStart, lineEnd := m.MonoModeLineBoundaries()
	start, end := m.PatternActionBeatBoundaries()

	lines := make([]uint8, 0, len(m.definition.Lines))
	for i, line := range m.definition.Lines {
		lineIndex := uint8(i)
		if line.MsgType == grid.MessageTypeNote {
			if lineIndex >= lineStart && lineIndex <= lineEnd {
				lines = append(lines, lineIndex)
			}
		}
	}

	pattern := make(grid.Pattern)
	m.currentOverlay.CombinedLineGridPattern(&pattern, m.currentOverlay.Key.GetMinimumKeyCycle(), lines)

	keys := slices.Collect(maps.Keys(pattern))

	slices.SortFunc(keys, grid.CompareBeat)

	keysWithSpaces := make([]grid.GridKey, 0, len(keys))
	for b := start; b <= end; b += every {
		index, exists := slices.BinarySearchFunc(keys, b, grid.BSearchBeatFunc)

		if exists {
			keysWithSpaces = append(keysWithSpaces, keys[index])
		} else {
			keysWithSpaces = append(keysWithSpaces, GK(lines[0], b))
		}
	}

	for _, key := range keysWithSpaces {
		everyFn(key)
	}
}

func (m *model) EveryMonoNote(every uint8, everyFn func(gridKey)) {
	lineStart, lineEnd := m.MonoModeLineBoundaries()
	start, end := m.PatternActionBeatBoundaries()

	lines := make([]uint8, 0, len(m.definition.Lines))
	for i, line := range m.definition.Lines {
		lineIndex := uint8(i)
		if line.MsgType == grid.MessageTypeNote {
			if lineIndex >= lineStart && lineIndex <= lineEnd {
				lines = append(lines, lineIndex)
			}
		}
	}

	pattern := make(grid.Pattern)
	m.currentOverlay.CombinedLineGridPattern(&pattern, m.currentOverlay.Key.GetMinimumKeyCycle(), lines)

	keys := slices.Collect(maps.Keys(pattern))

	slices.SortFunc(keys, grid.CompareBeat)
	slices.Reverse(keys)

	i := 0
	for _, key := range keys {
		if key.Beat >= start && key.Beat <= end {
			if i%int(every) == 0 {
				everyFn(key)
			}
			i++
		}
	}
}

func (m *model) EveryChordNote(every uint8, everyFn func(gridKey)) {
	bounds := m.PasteBounds()
	combinedOverlay := m.CombinedEditPattern(m.currentOverlay)
	keys := slices.Collect(maps.Keys(combinedOverlay))
	slices.SortFunc(keys, grid.Compare)
	counter := 0
	for _, gk := range keys {
		if bounds.InBounds(gk) {
			if counter%int(every) == 0 {
				everyFn(gk)
			}
			counter++
		}
	}
}

func (m *model) EveryLineSpace(every uint8, everyFn func(gridKey)) {
	lineStart, lineEnd := m.PatternActionLineBoundaries()
	start, end := m.PatternActionBeatBoundaries()

	for l := lineStart; l <= lineEnd; l++ {
		for b := start; b <= end; b += every {
			everyFn(GK(l, b))
		}
	}
}

func (m *model) EveryMonoOpenSpace(every uint8, everyFn func(gridKey)) {
	lineStart, lineEnd := m.MonoModeLineBoundaries()
	start, end := m.PatternActionBeatBoundaries()

	lines := make([]uint8, 0, len(m.definition.Lines))
	for i, line := range m.definition.Lines {
		lineIndex := uint8(i)
		if line.MsgType == grid.MessageTypeNote {
			if lineIndex >= lineStart && lineIndex <= lineEnd {
				lines = append(lines, lineIndex)
			}
		}
	}

	pattern := make(grid.Pattern)
	m.currentOverlay.CombinedLineGridPattern(&pattern, m.currentOverlay.Key.GetMinimumKeyCycle(), lines)

	keys := slices.Collect(maps.Keys(pattern))

	slices.SortFunc(keys, grid.CompareBeat)

	keysWithSpaces := make([]grid.GridKey, 0, len(keys))
	for b := start; b <= end; b += 1 {
		_, exists := slices.BinarySearchFunc(keys, b, grid.BSearchBeatFunc)

		if !exists {
			keysWithSpaces = append(keysWithSpaces, GK(lines[0], b))
		}
	}

	counter := 0
	for _, key := range keysWithSpaces {
		if counter%int(every) == 0 {
			everyFn(key)
		}
		counter++
	}
}

func (m *model) EveryOpenSpace(every uint8, everyFn func(gridKey)) {
	lineStart, _ := m.PatternActionLineBoundaries()
	start, end := m.PatternActionBeatBoundaries()

	pattern := make(grid.Pattern)
	m.currentOverlay.CombineGridPattern(&pattern, m.currentOverlay.Key.GetMinimumKeyCycle(), overlays.CombineTypeAll)

	keys := slices.Collect(maps.Keys(pattern))

	slices.SortFunc(keys, grid.CompareBeat)

	justSpaces := make([]grid.GridKey, 0, len(keys))
	for b := start; b <= end; b += 1 {
		_, exists := pattern[GK(lineStart, b)]

		if !exists {
			justSpaces = append(justSpaces, GK(lineStart, b))
		}
	}

	counter := 0
	for _, gk := range justSpaces {
		if counter%int(every) == 0 {
			everyFn(gk)
		}
		counter++
	}
}

func (m *model) EveryNote(every uint8, everyFn func(gridKey), combinedOverlay grid.Pattern) {
	switch m.definition.TemplateSequencerType {
	case operation.SeqModeLine:
		m.EveryLineNote(every, everyFn, combinedOverlay)
	case operation.SeqModeMono:
		m.EveryMonoNote(every, everyFn)
	default:
		m.EveryLineNote(every, everyFn, combinedOverlay)
	}
}

func (m *model) EveryLineNote(every uint8, everyFn func(gridKey), combinedOverlay grid.Pattern) {
	lineStart, lineEnd := m.PatternActionLineBoundaries()
	start, end := m.PatternActionBeatBoundaries()

	noteCollection := make([]grid.GridKey, 0)
	for gridKey, note := range combinedOverlay {
		if note != zeronote {
			if gridKey.Line >= lineStart && gridKey.Line <= lineEnd && gridKey.Beat >= start && gridKey.Beat <= end {
				noteCollection = append(noteCollection, gridKey)
			}
		}
	}

	slices.SortFunc(noteCollection, grid.Compare)

	for i, gridKey := range noteCollection {
		if i%int(every) == 0 {
			everyFn(gridKey)
		}
	}
}

func (m *model) fill(every uint8) {
	combinedOverlay := m.CombinedEditPattern(m.currentOverlay)

	everyFn := func(gridKey gridKey) {
		currentNote, hasNote := combinedOverlay[gridKey]
		hasNote = hasNote && currentNote != zeronote

		currentLinePos := GK(m.gridCursor.Line, gridKey.Beat)
		if hasNote {
			m.currentOverlay.SetNote(gridKey, zeronote)
			if gridKey != currentLinePos {
				m.currentOverlay.SetNote(currentLinePos, grid.InitNote())
			}
		} else {
			m.currentOverlay.SetNote(currentLinePos, grid.InitNote())
		}
	}

	m.Every(every, everyFn)
}

func (m *model) fillSpaces(every uint8) {
	combinedOverlay := m.CombinedEditPattern(m.currentOverlay)

	everyFn := func(gridKey gridKey) {
		currentNote, hasNote := combinedOverlay[gridKey]
		hasNote = hasNote && currentNote != zeronote

		currentLinePos := GK(m.gridCursor.Line, gridKey.Beat)
		if hasNote {
			m.currentOverlay.SetNote(gridKey, zeronote)
			if gridKey != currentLinePos {
				m.currentOverlay.SetNote(currentLinePos, grid.InitNote())
			}
		} else {
			m.currentOverlay.SetNote(currentLinePos, grid.InitNote())
		}
	}

	if m.definition.TemplateSequencerType == operation.SeqModeMono {
		m.EveryMonoOpenSpace(every, everyFn)
	} else {
		m.EveryOpenSpace(every, everyFn)
	}
}

func (m *model) incrementAccent(every uint8, modifier int8, everyMode operation.EveryMode) {
	combinedOverlay := m.CombinedEditPattern(m.currentOverlay)

	everyFn := func(gridKey gridKey) {
		currentNote, hasNote := combinedOverlay[gridKey]
		hasNote = hasNote && currentNote != zeronote

		if hasNote {
			m.currentOverlay.SetNote(gridKey, currentNote.IncrementAccent(modifier, uint8(len(config.Accents))))
		}
	}

	switch everyMode {
	case operation.EveryBeat:
		m.Every(every, everyFn)
	case operation.EveryNote:
		m.EveryNote(every, everyFn, combinedOverlay)
	}
}

func (m *model) incrementGate(every uint8, modifier int8, everyMode operation.EveryMode) {
	combinedOverlay := m.CombinedEditPattern(m.currentOverlay)

	everyFn := func(gridKey gridKey) {
		currentNote, hasNote := combinedOverlay[gridKey]
		hasNote = hasNote && currentNote != zeronote

		if hasNote {
			m.currentOverlay.SetNote(gridKey, currentNote.IncrementGate(modifier, len(config.ShortGates)+len(config.LongGates)))
		}
	}

	switch everyMode {
	case operation.EveryBeat:
		m.Every(every, everyFn)
	case operation.EveryNote:
		m.EveryNote(every, everyFn, combinedOverlay)
	}
}

func (m *model) incrementRatchet(every uint8, modifier int8, everyMode operation.EveryMode) {
	combinedOverlay := m.CombinedEditPattern(m.currentOverlay)

	everyFn := func(gridKey gridKey) {
		currentNote, hasNote := combinedOverlay[gridKey]
		hasNote = hasNote && currentNote != zeronote

		if hasNote {
			m.currentOverlay.SetNote(gridKey, currentNote.IncrementRatchet(modifier))
		}
	}

	switch everyMode {
	case operation.EveryBeat:
		m.Every(every, everyFn)
	case operation.EveryNote:
		m.EveryNote(every, everyFn, combinedOverlay)
	}
}

func (m *model) incrementWait(every uint8, modifier int8, everyMode operation.EveryMode) {
	combinedOverlay := m.CombinedEditPattern(m.currentOverlay)

	everyFn := func(gridKey gridKey) {
		currentNote, hasNote := combinedOverlay[gridKey]
		hasNote = hasNote && currentNote != zeronote

		if hasNote {
			m.currentOverlay.SetNote(gridKey, currentNote.IncrementWait(modifier))
		}
	}

	switch everyMode {
	case operation.EveryBeat:
		m.Every(every, everyFn)
	case operation.EveryNote:
		m.EveryNote(every, everyFn, combinedOverlay)
	}
}

func (m *model) IncrementSpecificValue(note grid.Note) {
	note.AccentIndex = note.AccentIndex + 1
	if note.AccentIndex < 128 {

		m.currentOverlay.SetNote(m.gridCursor, note)
	}
}

func (m *model) DecrementSpecificValue(note grid.Note) {
	if note.AccentIndex > 0 {
		note.AccentIndex = note.AccentIndex - 1
		m.currentOverlay.SetNote(m.gridCursor, note)
	}
}

func (m *model) AccentModify(modifier int8) {
	modifyFunc := func(key gridKey, currentNote note) {
		m.currentOverlay.SetNote(key, currentNote.IncrementAccent(modifier, uint8(len(config.Accents))))
	}
	m.Modify(modifyFunc)
}

func (m *model) GateModify(modifier int8) {
	modifyFunc := func(key gridKey, currentNote note) {
		m.currentOverlay.SetNote(key, currentNote.IncrementGate(modifier, len(config.ShortGates)+len(config.LongGates)))
	}
	m.Modify(modifyFunc)
}

type Boundable interface {
	InBounds(key grid.GridKey) bool
}

func (m *model) Modify(modifyFunc func(gridKey, note)) {

	var bounds Boundable
	currentChord := m.CurrentChord()
	if m.visualSelection.visualMode == operation.VisualNone && currentChord.HasValue() {
		bounds = currentChord.GridChord
	} else {
		bounds = m.YankBounds()
	}
	combinedOverlay := m.CombinedEditPattern(m.currentOverlay)

	for key, currentNote := range combinedOverlay {
		if bounds.InBounds(key) {
			if currentNote != zeronote {
				modifyFunc(key, currentNote)
			}
		}
	}
}

func (m *model) WaitModify(modifier int8) {
	modifyFunc := func(key gridKey, currentNote note) {
		m.currentOverlay.SetNote(key, currentNote.IncrementWait(modifier))
	}

	m.Modify(modifyFunc)
}

func (m model) PatternActionBeatBoundaries() (uint8, uint8) {
	if m.visualSelection.visualMode == operation.VisualNone {
		return m.gridCursor.Beat, m.CurrentPart().Beats - 1
	} else {
		bounds := m.visualSelection.Bounds(m.CurrentPart().Beats)
		return bounds.Left, bounds.Right
	}
}

func (m model) PatternActionLineBoundaries() (uint8, uint8) {
	if m.visualSelection.visualMode == operation.VisualNone {
		return m.gridCursor.Line, m.gridCursor.Line
	} else {
		bounds := m.visualSelection.Bounds(m.CurrentPart().Beats)
		return bounds.Top, bounds.Bottom
	}
}

func (m model) MonoModeLineBoundaries() (uint8, uint8) {
	if m.visualSelection.visualMode == operation.VisualNone {
		return 0, uint8(len(m.definition.Lines) - 1)
	} else {
		bounds := m.visualSelection.Bounds(m.CurrentPart().Beats)
		return bounds.Top, bounds.Bottom
	}
}

func (m *model) ReloadFile() {
	if m.filename != "" {
		var err error
		m.definition, err = LoadFile(m.filename, m.definition.Template, m.definition.Instrument)
		if err != nil {
			m.SetCurrentError(fault.Wrap(err, fmsg.WithDesc("could not reload file", fmt.Sprintf("Could not reload file %s", m.filename))))
		}
		m.ResetCurrentOverlay()
	}
}
