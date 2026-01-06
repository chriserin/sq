package main

import (
	"fmt"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/chriserin/sq/internal/config"
	"github.com/chriserin/sq/internal/grid"
	"github.com/chriserin/sq/internal/mappings"
	"github.com/chriserin/sq/internal/operation"
	"github.com/chriserin/sq/internal/overlaykey"
	"github.com/chriserin/sq/internal/overlays"
	"github.com/chriserin/sq/internal/playstate"
	"github.com/chriserin/sq/internal/sequence"
	themes "github.com/chriserin/sq/internal/themes"
	"github.com/chriserin/sq/internal/timing"
	midi "gitlab.com/gomidi/midi/v2"
)

func (m model) View() (output string) {
	defer func() {
		if r := recover(); r != nil {
			stackTrace := make([]byte, 4096)
			n := runtime.Stack(stackTrace, false)
			msg := panicMsg{message: fmt.Sprintf("Caught View Panic: %v", r), stacktrace: stackTrace[:n]}
			m.LogString(fmt.Sprintf(" ------ Panic Message ------- \n%s\n", msg.message))
			m.LogString(fmt.Sprintf(" ------ Stacktrace ---------- \n%s\n", msg.stacktrace))
			output = fmt.Sprintf("View Layer Panic: %v\n", r)
			m.errChan <- fault.New(fmt.Sprintf("caught view panic: %v", r), ftag.With("view_panic"), fmsg.WithDesc(fmt.Sprintf("%v", r), "View Panic"))
		}
	}()

	if m.currentViewError != nil {
		result := fmt.Sprintf("There was an error rendering the view: %s", m.currentViewError)
		result += "\n\n"
		result += "Details in debug.log\n"
		result += "Please consider reporting this issue\n"
		result += "Ctrl+C or q to exit\n"
		return result
	}

	var buf strings.Builder
	var sideView string

	visualCombinedPattern := m.CombinedOverlayPattern(m.currentOverlay)

	showLines := GetShowLines(len(m.definition.Lines), visualCombinedPattern, m.CurrentPart().Beats)

	if m.patternMode == operation.PatternAccent || m.IsAccentSelector() {
		sideView = m.AccentKeyView()
	} else if (m.CurrentPart().Overlays.Key == overlaykey.ROOT && m.CurrentPart().Overlays.IsFresh() && len(*m.definition.Parts) == 1 && m.CurrentPartID() == 0) ||
		slices.Contains([]operation.Selection{operation.SelectSetupValue, operation.SelectSetupMessageType, operation.SelectSetupChannel}, m.selectionIndicator) {
		// NOTE: We want to show the setupView on the very initial screen,
		// before any sequencing has begun OR a setup value is selected
		sideView = m.SetupView(showLines)
	} else {
		sideView = m.OverlaysView()

		var chordView string
		switch m.definition.TemplateSequencerType {
		case operation.SeqModeChord:
			currentChord := m.CurrentChord()
			chordView = m.ChordView(currentChord.GridChord)
		}
		sideView = lipgloss.JoinVertical(lipgloss.Left, sideView, chordView)
	}

	sideView = sideView[:len(sideView)-1] // remove last newline

	seqView := m.SeqView(showLines)
	seqView = seqView[:len(seqView)-1] // remove last newline

	intraborder := "  "
	if m.playState.BoundedLoop.Active && m.CurrentPart().Beats == m.playState.BoundedLoop.RightBound+1 {
		intraborder = " "

	}
	seqAndSide := lipgloss.JoinHorizontal(0, "  ", seqView, intraborder, sideView)
	buf.WriteString(lipgloss.JoinVertical(lipgloss.Left, seqAndSide, m.CurrentOverlayView()))

	if m.currentError != nil && m.selectionIndicator == operation.SelectError {
		buf.WriteString("\n")
		style := lipgloss.NewStyle().Width(50)
		style = style.Border(lipgloss.NormalBorder())
		style = style.Padding(1)
		style = style.BorderForeground(lipgloss.Color("#880000"))
		style = style.MarginLeft(2)
		var errorBuf strings.Builder
		errorBuf.WriteString("ERROR: ")
		issue := fmsg.GetIssue(m.currentError)
		if issue != "" {
			errorBuf.WriteString(issue)
		} else {
			chain := fault.Flatten(m.currentError)
			errorBuf.WriteString(chain[0].Message)
		}
		buf.WriteString(style.Render(errorBuf.String()))
	} else {
		buf.WriteString("\n")
	}
	if m.showArrangementView {
		buf.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, "  ", m.arrangement.View(m.playState.LoopedArrangement)))
	}
	buf.WriteString("\n")
	return buf.String()
}

func GetShowLines(totalLinesCount int, visualCombinedPattern overlays.OverlayPattern, beats uint8) []uint8 {
	showLines := make([]uint8, 0, totalLinesCount)
	for lineNumber := uint8(0); lineNumber < uint8(totalLinesCount); lineNumber++ {
		for i := range beats {
			gridKey := GK(uint8(lineNumber), i)
			_, exists := visualCombinedPattern[gridKey]
			if exists {
				showLines = append(showLines, lineNumber)
			}
		}
	}
	return showLines
}

func (m model) CyclesEditView() string {
	var buf strings.Builder
	cycles := m.CurrentSongSection().Cycles
	startCycles := m.CurrentSongSection().StartCycles
	cyclesInput := themes.NumberStyle.Render(strconv.Itoa(int(cycles)))
	startCyclesInput := themes.NumberStyle.Render(strconv.Itoa(int(startCycles)))
	switch m.selectionIndicator {
	case operation.SelectCycles:
		cyclesInput = themes.SelectedStyle.Render(strconv.Itoa(int(cycles)))
	case operation.SelectStartCycles:
		startCyclesInput = themes.SelectedStyle.Render(strconv.Itoa(int(startCycles)))
	}
	buf.WriteString(themes.AltArtStyle.Render(" ⟳ Amount "))
	buf.WriteString(cyclesInput)
	buf.WriteString(themes.AltArtStyle.Render("    ⟳ Start "))
	buf.WriteString(startCyclesInput)
	buf.WriteString("\n")
	return buf.String()
}

func (m model) BeatsEditView() string {
	var buf strings.Builder
	beats := m.CurrentPart().Beats
	startBeats := m.CurrentSongSection().StartBeat

	beatsInput := themes.NumberStyle.Render(strconv.Itoa(int(beats)))
	startBeatsInput := themes.NumberStyle.Render(strconv.Itoa(int(startBeats)))
	switch m.selectionIndicator {
	case operation.SelectBeats:
		beatsInput = themes.SelectedStyle.Render(strconv.Itoa(int(beats)))
	case operation.SelectStartBeats:
		startBeatsInput = themes.SelectedStyle.Render(strconv.Itoa(int(startBeats)))
	}
	buf.WriteString(themes.AltArtStyle.Render(" Beats "))
	buf.WriteString(beatsInput)
	buf.WriteString(themes.AltArtStyle.Render("  Start Beat "))
	buf.WriteString(startBeatsInput)
	buf.WriteString("\n")
	return buf.String()
}

func (m model) TempoEditView() string {
	var tempo, division string
	tempo = themes.NumberStyle.Render(strconv.Itoa(m.definition.Tempo))
	division = themes.NumberStyle.Render(strconv.Itoa(m.definition.Subdivisions))
	switch m.selectionIndicator {
	case operation.SelectTempo:
		tempo = themes.SelectedStyle.Render(strconv.Itoa(m.definition.Tempo))
	case operation.SelectTempoSubdivision:
		division = themes.SelectedStyle.Render(strconv.Itoa(m.definition.Subdivisions))
	}
	var buf strings.Builder
	buf.WriteString(themes.AltArtStyle.Render(" Tempo "))
	buf.WriteString(tempo)
	buf.WriteString(themes.AltArtStyle.Render("  Subdivisions "))
	buf.WriteString(division)
	buf.WriteString("\n")
	return buf.String()
}

func (m model) SpecificValueEditView(note grid.Note) string {
	var specificValue = themes.SelectedStyle.Render(fmt.Sprintf("%d", note.AccentIndex))
	var buf strings.Builder
	buf.WriteString(themes.AltArtStyle.Render(" Specific Value "))
	buf.WriteString(specificValue)
	buf.WriteString("\n")
	return buf.String()
}

func (m model) EuclideanHitsEditView() string {
	var buf strings.Builder
	lineStart, lineEnd := m.PatternActionLineBoundaries()
	start, end := m.PatternActionBeatBoundaries()
	steps := int(end - start + 1)
	lines := int(lineEnd - lineStart + 1)

	buf.WriteString(themes.AltArtStyle.Render(" Euclidean Hits "))
	buf.WriteString(themes.SelectedStyle.Render(fmt.Sprintf("%d", m.euclideanHits)))
	buf.WriteString(themes.AltArtStyle.Render(fmt.Sprintf(" / %d steps", steps)))
	if lines > 1 {
		buf.WriteString(themes.AltArtStyle.Render(fmt.Sprintf(" × %d lines", lines)))
	}
	buf.WriteString("\n")
	return buf.String()
}

func (m model) WriteView() string {
	if m.NeedsWrite() {
		return " [+]"
	} else {
		return "    "
	}
}

func (m model) IsAccentSelector() bool {
	states := []operation.Selection{operation.SelectAccentEnd, operation.SelectAccentTarget, operation.SelectAccentStart}
	return slices.Contains(states, m.selectionIndicator)
}

func (m model) IsRatchetSelector() bool {
	states := []operation.Selection{operation.SelectRatchets, operation.SelectRatchetSpan}
	return slices.Contains(states, m.selectionIndicator)
}

func (m model) AccentKeyView() string {
	var buf strings.Builder
	var accentEnd = m.definition.Accents.End
	var accentStart = m.definition.Accents.Start

	var accentTarget string
	switch m.definition.Accents.Target {
	case sequence.AccentTargetNote:
		accentTarget = "N"
	case sequence.AccentTargetVelocity:
		accentTarget = "V"
	}

	var accentTargetString string
	if m.selectionIndicator == operation.SelectAccentTarget {
		accentTargetString = themes.SelectedStyle.Render(fmt.Sprintf(" %s", accentTarget))
	} else {
		accentTargetString = themes.NumberStyle.Render(fmt.Sprintf(" %s", accentTarget))
	}

	title := themes.AppDescriptorStyle.Render("Accents")
	buf.WriteString(fmt.Sprintf(" %s - %s\n", title, accentTargetString))
	buf.WriteString(themes.SeqBorderStyle.Render("──────────────"))
	buf.WriteString("\n")

	var accentStartString string
	if m.selectionIndicator == operation.SelectAccentStart {
		accentStartString = themes.SelectedStyle.Render(fmt.Sprintf("%2d", accentStart))
	} else {
		accentStartString = themes.NumberStyle.Render(fmt.Sprintf("%2d", accentStart))
	}

	var accentEndString string
	if m.selectionIndicator == operation.SelectAccentEnd {
		accentEndString = themes.SelectedStyle.Render(fmt.Sprintf("%2d", accentEnd))
	} else {
		accentEndString = themes.NumberStyle.Render(fmt.Sprintf("%2d", accentEnd))
	}

	style := lipgloss.NewStyle().Foreground(lipgloss.Color(themes.AccentColors[1]))
	buf.WriteString(fmt.Sprintf("  %s  -  %s\n", style.Render(string(themes.AccentIcons[1])), accentStartString))
	for i, accent := range m.definition.Accents.Data[2 : len(m.definition.Accents.Data)-1] {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(themes.AccentColors[i+2]))
		buf.WriteString(fmt.Sprintf("  %s  -  %d\n", style.Render(string(themes.AccentIcons[i+2])), accent))
	}
	style = lipgloss.NewStyle().Foreground(lipgloss.Color(themes.AccentColors[len(themes.AccentIcons)-1]))
	buf.WriteString(fmt.Sprintf("  %s  -  %s\n", style.Render(string(themes.AccentIcons[len(themes.AccentIcons)-1])), accentEndString))
	return buf.String()
}

func (m model) SetupView(showLines []uint8) string {
	var buf strings.Builder
	buf.WriteString(themes.AppDescriptorStyle.Render("Setup"))
	buf.WriteString("\n")
	buf.WriteString(themes.SeqBorderStyle.Render("──────────────"))
	buf.WriteString("\n")
	for i, line := range m.definition.Lines {
		if m.hideEmptyLines && !slices.Contains(showLines, uint8(i)) {
			continue
		}

		buf.WriteString("CH ")
		if uint8(i) == m.gridCursor.Line && m.selectionIndicator == operation.SelectSetupChannel {
			buf.WriteString(themes.SelectedStyle.Render(fmt.Sprintf("%2d", line.Channel)))
		} else {
			buf.WriteString(themes.NumberStyle.Render(fmt.Sprintf("%2d", line.Channel)))
		}

		var messageType string
		switch line.MsgType {
		case grid.MessageTypeNote:
			messageType = "NOTE"
		case grid.MessageTypeCc:
			messageType = "CC"
		case grid.MessageTypeProgramChange:
			messageType = "Program Change"
		}

		if uint8(i) == m.gridCursor.Line && m.selectionIndicator == operation.SelectSetupMessageType {
			messageType = fmt.Sprintf(" %s ", themes.SelectedStyle.Render(messageType))
		} else {
			messageType = fmt.Sprintf(" %s ", messageType)
		}

		buf.WriteString(messageType)

		if line.MsgType == grid.MessageTypeProgramChange {
			buf.WriteString("")
		} else {
			if uint8(i) == m.gridCursor.Line && m.selectionIndicator == operation.SelectSetupValue {
				buf.WriteString(themes.SelectedStyle.Render(strconv.Itoa(int(line.Note))))
			} else {
				buf.WriteString(themes.NumberStyle.Render(strconv.Itoa(int(line.Note))))
			}
		}
		buf.WriteString(fmt.Sprintf(" %s\n", LineValueName(line, m.definition.Instrument)))
	}
	return buf.String()
}

func NoteName(note uint8) string {
	return fmt.Sprintf("%s%d", strings.ReplaceAll(midi.Note(note).Name(), "b", "♭"), int(midi.Note(note).Octave())-2)
}

func LineValueName(ld grid.LineDefinition, instrument string) string {
	switch ld.MsgType {
	case grid.MessageTypeNote:
		return NoteName(ld.Note)
	case grid.MessageTypeCc:
		cc, _ := config.FindCC(ld.Note, instrument)
		return cc.Name
	}
	return ""
}

func (m model) ChordView(gridChord *overlays.GridChord) string {

	var buf strings.Builder
	buf.WriteString(themes.AppDescriptorStyle.Render("Chord"))
	if gridChord == nil {
		buf.WriteString("\n")
		buf.WriteString(themes.SeqBorderStyle.Render("──────────────"))
		return buf.String()
	}
	chord := gridChord.Chord
	pattern := make(grid.Pattern)
	gridChord.ChordNotes(&pattern)
	baseNote := m.definition.Lines[0].Note
	buf.WriteString("\n")
	buf.WriteString(themes.SeqBorderStyle.Render("──────────────"))
	buf.WriteString("\n")
	buf.WriteString(chord.Name())
	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf("Foundation: %s", NoteName(baseNote-gridChord.Root.Line)))
	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf("Inversions: %d", chord.Inversions()))
	buf.WriteString("\n")
	buf.WriteString("\n")

	intervals := chord.NamedIntervals()
	uninvertedNotes := chord.UninvertedNotes()
	slices.Reverse(uninvertedNotes)
	for i, n := range uninvertedNotes {
		buf.WriteString(fmt.Sprintf("%3s - %s", intervals[i], NoteName(baseNote-gridChord.Root.Line+n)))
		buf.WriteString("\n")
	}

	return buf.String()
}

func (m model) OverlaysView() string {
	var buf strings.Builder
	buf.WriteString(themes.AppDescriptorStyle.Render("Overlays"))
	buf.WriteString("\n")
	buf.WriteString(themes.SeqBorderStyle.Render("──────────────"))
	buf.WriteString("\n")
	playingStyle := lipgloss.NewStyle().Background(themes.SeqOverlayColor).Foreground(themes.AppDescriptorColor)
	notPlayingStyle := themes.AppDescriptorStyle
	var playingOverlayKeys = m.PlayingOverlayKeys()
	for currentOverlay := m.CurrentPart().Overlays; currentOverlay != nil; currentOverlay = currentOverlay.Below {
		var playingSpacer = "   "
		var playing = ""
		if m.playState.LoopMode == playstate.LoopOverlay && m.playState.Playing && playingOverlayKeys[0] == currentOverlay.Key {
			playing = themes.OverlayCurrentlyLoopingSymbol
			buf.WriteString(playing)
			playingSpacer = ""
		} else if m.playState.Playing && playingOverlayKeys[0] == currentOverlay.Key {
			playing = themes.OverlayCurrentlyPlayingSymbol
			buf.WriteString(playing)
			playingSpacer = ""
		} else if m.playState.Playing && slices.Contains(playingOverlayKeys, currentOverlay.Key) {
			playing = themes.ActiveSymbol
			buf.WriteString(playing)
			playingSpacer = ""
		}
		var editing = ""
		if m.currentOverlay.Key == currentOverlay.Key {
			editing = " E"
		}
		var stackModifier = ""
		if currentOverlay.PressDown {
			stackModifier = " \u2193\u0332"
		} else if currentOverlay.PressUp {
			stackModifier = " \u2191\u0305"
		}

		overlayLine := fmt.Sprintf("%s%2s%2s", overlaykey.View(currentOverlay.Key), stackModifier, editing)

		buf.WriteString(playingSpacer)
		if m.playState.Playing && slices.Contains(playingOverlayKeys, currentOverlay.Key) {
			buf.WriteString(playingStyle.Render(overlayLine))
		} else {
			buf.WriteString(notPlayingStyle.Render(overlayLine))
		}
		buf.WriteString(playing)
		buf.WriteString(playingSpacer)
		buf.WriteString("\n")
	}
	return buf.String()
}

func PatternMode(mode string) string {
	return fmt.Sprintf(" %s  %s\n", themes.AccentModeStyle.Render(" PATTERN MODE "), themes.AccentModeStyle.Render(mode))
}

func (m model) SeqView(showLines []uint8) string {
	var buf strings.Builder
	var mode string

	visualCombinedPattern := m.CombinedOverlayPattern(m.currentOverlay)
	currentNote := visualCombinedPattern[m.gridCursor]

	buf.WriteString(m.WriteView())
	if m.selectionIndicator == operation.SelectGrid && m.patternMode == operation.PatternAccent {
		buf.WriteString(PatternMode(" Accent "))
	} else if m.selectionIndicator == operation.SelectGrid && m.patternMode == operation.PatternNoteAccent {
		mode = " Accent Note "
		buf.WriteString(PatternMode(mode))
	} else if m.selectionIndicator == operation.SelectGrid && m.patternMode == operation.PatternGate {
		mode = " Gate "
		buf.WriteString(PatternMode(mode))
	} else if m.selectionIndicator == operation.SelectGrid && m.patternMode == operation.PatternNoteGate {
		mode = " Gate Note "
		buf.WriteString(PatternMode(mode))
	} else if m.selectionIndicator == operation.SelectGrid && m.patternMode == operation.PatternWait {
		mode = " Wait "
		buf.WriteString(PatternMode(mode))
	} else if m.selectionIndicator == operation.SelectGrid && m.patternMode == operation.PatternNoteWait {
		mode = " Wait Note "
		buf.WriteString(PatternMode(mode))
	} else if m.selectionIndicator == operation.SelectGrid && m.patternMode == operation.PatternRatchet {
		mode = " Ratchet "
		buf.WriteString(PatternMode(mode))
	} else if m.selectionIndicator == operation.SelectGrid && m.patternMode == operation.PatternNoteRatchet {
		mode = " Ratchet Note "
		buf.WriteString(PatternMode(mode))
	} else if m.selectionIndicator == operation.SelectRatchets || m.selectionIndicator == operation.SelectRatchetSpan {
		buf.WriteString(m.RatchetEditView())
	} else if m.selectionIndicator == operation.SelectTempo || m.selectionIndicator == operation.SelectTempoSubdivision {
		buf.WriteString(m.TempoEditView())
	} else if slices.Contains([]operation.Selection{operation.SelectBeats, operation.SelectStartBeats}, m.selectionIndicator) {
		buf.WriteString(m.BeatsEditView())
	} else if slices.Contains([]operation.Selection{operation.SelectCycles, operation.SelectStartCycles}, m.selectionIndicator) {
		buf.WriteString(m.CyclesEditView())
	} else if m.selectionIndicator == operation.SelectPart {
		buf.WriteString(m.ChoosePartView())
	} else if m.selectionIndicator == operation.SelectChangePart {
		buf.WriteString(m.ChoosePartView())
	} else if m.selectionIndicator == operation.SelectRenamePart {
		buf.WriteString(m.RenamePartView())
	} else if m.selectionIndicator == operation.SelectFileName {
		buf.WriteString(m.FileNameView())
	} else if m.selectionIndicator == operation.SelectConfirmNew {
		buf.WriteString(m.ConfirmNewSequenceView())
	} else if m.selectionIndicator == operation.SelectConfirmQuit {
		buf.WriteString(m.ConfirmQuitView())
	} else if m.selectionIndicator == operation.SelectConfirmReload {
		buf.WriteString(m.ConfirmReloadView())
	} else if m.selectionIndicator == operation.SelectSpecificValue {
		buf.WriteString(m.SpecificValueEditView(currentNote.Note))
	} else if m.selectionIndicator == operation.SelectEuclideanHits {
		buf.WriteString(m.EuclideanHitsEditView())
	} else if m.playState.Playing {
		buf.WriteString(playstate.View(m.playState, m.arrangement.Cursor))
	} else if len(*m.definition.Parts) > 1 {
		buf.WriteString(themes.AppTitleStyle.Render(" sq "))
		buf.WriteString(themes.AppDescriptorStyle.Render(fmt.Sprintf("- %s", m.CurrentPart().GetName())))
		buf.WriteString("\n")
	} else {
		buf.WriteString(themes.AppTitleStyle.Render(" sq "))
		buf.WriteString(themes.AppDescriptorStyle.Render("- A sequencer for your cli"))
		buf.WriteString("\n")
	}

	beats := m.CurrentPart().Beats
	topLine := m.TopLine(beats)
	if m.midiLoopMode == timing.MlmTransmitter && m.transmitting {
		buf.WriteString("  T")
	} else if m.midiLoopMode == timing.MlmTransmitter && !m.transmitting {
		buf.WriteString("  ⊥")
	} else if m.midiLoopMode == timing.MlmReceiver && !m.transmitterConnected {
		buf.WriteString("  X")
	} else if m.midiLoopMode == timing.MlmReceiver && m.transmitterConnected {
		buf.WriteString("  ☨")
	} else {
		buf.WriteString("   ")
	}
	buf.WriteString(topLine)
	buf.WriteString("\n")

	for i := uint8(0); i < uint8(len(m.definition.Lines)); i++ {
		if m.hideEmptyLines && !slices.Contains(showLines, i) {
			continue
		}
		buf.WriteString(lineView(i, m, visualCombinedPattern))
	}

	return buf.String()
}

func (m model) TopLine(beats uint8) string {
	buf := strings.Builder{}
	if m.playState.BoundedLoop.Active {
		if m.playState.BoundedLoop.LeftBound == 0 {
			buf.WriteString(themes.NumberStyle.Render(" >"))
		} else {
			buf.WriteString(themes.SeqBorderStyle.Render(" ┌"))
			buf.WriteString(themes.SeqBorderStyle.Render(strings.Repeat("─", int(m.playState.BoundedLoop.LeftBound-1))))
			buf.WriteString(themes.NumberStyle.Render(">"))
		}
		buf.WriteString(themes.NumberStyle.Render(strings.Repeat("─", (int(m.playState.BoundedLoop.RightBound) - int(m.playState.BoundedLoop.LeftBound) + 1))))
		buf.WriteString(themes.NumberStyle.Render("<"))
		buf.WriteString(themes.SeqBorderStyle.Render(strings.Repeat("─", max(int(beats)-int(m.playState.BoundedLoop.RightBound+2), 0))))
		return buf.String()
	} else {
		return fmt.Sprintf(" %s%s", themes.SeqBorderStyle.Render("┌"), themes.SeqBorderStyle.Render(strings.Repeat("─", max(32, int(beats)))))
	}
}

func (m model) RenamePartView() string {
	var buf strings.Builder
	buf.WriteString(" Rename Part: ")
	buf.WriteString(m.textInput.View())
	buf.WriteString("\n")
	return buf.String()
}

func (m model) FileNameView() string {
	var buf strings.Builder
	buf.WriteString(" File Name: ")
	buf.WriteString(m.textInput.View())
	buf.WriteString("\n")
	return buf.String()
}

func (m model) ChoosePartView() string {
	var buf strings.Builder
	buf.WriteString(" Choose Part: ")
	var name string
	if m.partSelectorIndex < 0 {
		name = "New Part"
	} else {
		name = (*m.definition.Parts)[m.partSelectorIndex].GetName()
	}
	buf.WriteString(themes.SelectedStyle.Render(name))
	buf.WriteString("\n")
	return buf.String()
}

func (m model) ConfirmNewSequenceView() string {
	var buf strings.Builder
	buf.WriteString(" New Sequence: ")
	buf.WriteString(themes.SelectedStyle.Render("Confirm"))
	buf.WriteString("\n")
	return buf.String()
}

func (m model) ConfirmQuitView() string {
	var buf strings.Builder
	buf.WriteString(" Quit: ")
	buf.WriteString(themes.SelectedStyle.Render("Confirm"))
	buf.WriteString("\n")
	return buf.String()
}

func (m model) ConfirmReloadView() string {
	var buf strings.Builder
	buf.WriteString(" Reload: ")
	buf.WriteString(themes.SelectedStyle.Render("Confirm"))
	buf.WriteString("\n")
	return buf.String()
}

func (m model) RatchetEditView() string {
	currentNote, _ := m.CurrentNote()

	var buf strings.Builder
	var ratchetsBuf strings.Builder
	buf.WriteString(" Ratchets ")
	for i := range uint8(8) {
		var backgroundColor lipgloss.Color
		if i <= currentNote.Ratchets.Length {
			if m.ratchetCursor == i && m.selectionIndicator == operation.SelectRatchets {
				backgroundColor = themes.SelectedAttributeColor
			}
			if currentNote.Ratchets.HitAt(i) {
				ratchetsBuf.WriteString(themes.ActiveStyle.Background(backgroundColor).Render("\u25CF"))
			} else {
				ratchetsBuf.WriteString(themes.MutedStyle.Background(backgroundColor).Render("\u25C9"))
			}
			ratchetsBuf.WriteString(" ")
		}
	}
	buf.WriteString(ensureStringLengthWc(ratchetsBuf.String(), 16, lipgloss.Left))
	if m.selectionIndicator == operation.SelectRatchetSpan {
		buf.WriteString(fmt.Sprintf(" Span %s ", themes.SelectedStyle.Render(strconv.Itoa(int(currentNote.Ratchets.GetSpan())))))
	} else {
		buf.WriteString(fmt.Sprintf(" Span %s ", themes.NumberStyle.Render(strconv.Itoa(int(currentNote.Ratchets.GetSpan())))))
	}
	buf.WriteString("\n")

	return buf.String()
}

func (m model) ViewOverlay() string {
	return m.overlayKeyEdit.ViewOverlay()
}

func (m model) CurrentOverlayView() string {
	var matchedKey overlayKey
	if m.playState.Playing {
		cycles := (*m.playState.Iterations)[m.arrangement.CurrentNode()]
		matchedKey = m.CurrentPart().Overlays.HighestMatchingOverlay(cycles).Key
	} else {
		matchedKey = overlaykey.ROOT
	}

	var editOverlayTitle string
	if m.modifyKey && m.focus == operation.FocusOverlayKey {
		editOverlayTitle = lipgloss.NewStyle().Foreground(themes.AppTitleColor).Render(" Mod")
	} else if m.focus == operation.FocusOverlayKey {
		editOverlayTitle = lipgloss.NewStyle().Foreground(themes.AppTitleColor).Render(" New")
	} else if m.playEditing {
		editOverlayTitle = lipgloss.NewStyle().Background(themes.SeqOverlayColor).Foreground(themes.AppTitleColor).Render("Edit")
	} else {
		editOverlayTitle = lipgloss.NewStyle().Foreground(themes.AppTitleColor).Render("Edit")
	}

	var monoIndicator = "  "
	switch m.definition.TemplateSequencerType {
	case operation.SeqModeMono:
		monoIndicator = "MN"
	case operation.SeqModeChord:
		monoIndicator = "CH"
	}

	playOverlayTitle := lipgloss.NewStyle().Foreground(themes.AppTitleColor).Render("Play")

	editOverlay := fmt.Sprintf("%s %s", editOverlayTitle, lipgloss.PlaceHorizontal(11, 0, m.ViewOverlay()))
	playOverlay := fmt.Sprintf("%s %s", playOverlayTitle, lipgloss.PlaceHorizontal(11, 0, overlaykey.View(matchedKey)))
	var name = ""
	if m.playEditing {
		name = (*m.definition.Parts)[m.editingPartID].GetName()
	}
	styled := lipgloss.NewStyle().Background(themes.SeqOverlayColor).Foreground(themes.AppTitleColor).Render(name)
	var styledPlayingPart = ""
	if m.playState.Playing {
		styledPlayingPart = lipgloss.NewStyle().Foreground(themes.AppTitleColor).Render(m.CurrentPart().GetName())
	}
	secondLine := fmt.Sprintf("       %s %s", lipgloss.PlaceHorizontal(17, 0, styled), lipgloss.PlaceHorizontal(11, 0, styledPlayingPart))
	return fmt.Sprintf("   %s  %s  %s %s\n%s", monoIndicator, editOverlay, playOverlay, mappings.KeycomboView(), secondLine)
}

func KeyLineIndicator(k uint8, l uint8) string {
	if k == l {
		return themes.AltArtStyle.Render("K")
	} else {
		return " "
	}
}

var blackNotes = []uint8{1, 3, 6, 8, 10}

func ensureStringLengthWc(s string, length int, pos lipgloss.Position) string {
	if ansi.StringWidthWc(s) <= length {
		padding := length - ansi.StringWidthWc(s)
		switch pos {
		case lipgloss.Left:
			return s + strings.Repeat(" ", padding)
		case lipgloss.Right:
			return strings.Repeat(" ", padding) + s
		}
	}

	return ansi.CutWc(s, 0, length)
}

func (m model) LineIndicator(lineNumber uint8) string {
	indicator := themes.SeqBorderStyle.Render("│")
	if lineNumber == m.gridCursor.Line {
		indicator = themes.LineCursorStyle.Render("┤")
	}
	if len(m.playState.LineStates) > int(lineNumber) && m.playState.LineStates[lineNumber].GroupPlayState == playstate.PlayStateMute {
		indicator = "M"
	}
	if len(m.playState.LineStates) > int(lineNumber) && m.playState.LineStates[lineNumber].GroupPlayState == playstate.PlayStateSolo {
		indicator = "S"
	}

	var lineName string
	if m.definition.Lines[lineNumber].Name != "" {
		lineName = themes.LineNumberStyle.Render(m.definition.Lines[lineNumber].Name)
	} else if m.definition.TemplateUIStyle == "blackwhite" && m.definition.Lines[lineNumber].MsgType == grid.MessageTypeNote {
		notename := NoteName(m.definition.Lines[lineNumber].Note)
		if slices.Contains(blackNotes, m.definition.Lines[lineNumber].Note%12) {
			lineName = themes.BlackKeyStyle.Render(notename[0:4])
		} else {
			lineName = themes.WhiteKeyStyle.Render(notename)
		}
	} else if m.definition.Lines[lineNumber].MsgType == grid.MessageTypeCc {
		lineName = themes.LineNumberStyle.Render(fmt.Sprintf("C%2d", m.definition.Lines[lineNumber].Note))
	} else if m.definition.Lines[lineNumber].MsgType == grid.MessageTypeProgramChange {
		lineName = themes.LineNumberStyle.Render("PC")
	} else {
		lineName = themes.LineNumberStyle.Render(fmt.Sprintf("%2d", lineNumber))
	}

	return fmt.Sprintf("%3s%s%s", ensureStringLengthWc(lineName, 3, lipgloss.Right), KeyLineIndicator(m.definition.Keyline, lineNumber), indicator)

}

type GateSpace struct {
	StringValue []rune
	Color       lipgloss.Color
}

func (gs GateSpace) HasMore() bool {
	return len(gs.StringValue) > 0
}
func (gs *GateSpace) ShiftString() string {
	if len(gs.StringValue) == 1 {
		v := gs.StringValue
		gs.StringValue = []rune{}
		return string(v)
	} else if len(gs.StringValue) > 1 {
		v := gs.StringValue[0]
		gs.StringValue = gs.StringValue[1:]
		return string(v)
	} else {
		return ""
	}
}

func lineView(lineNumber uint8, m model, visualCombinedPattern overlays.OverlayPattern) string {
	var buf strings.Builder
	buf.WriteString(m.LineIndicator(lineNumber))

	gateSpace := GateSpace{}
	currentChord := m.CurrentChord()
	for i := uint8(0); i < m.CurrentPart().Beats; i++ {
		currentGridKey := GK(uint8(lineNumber), i)
		overlayNote, hasNote := visualCombinedPattern[currentGridKey]

		var backgroundSeqColor lipgloss.Color
		var isCurrentBeat = m.playState.Playing && m.playState.LineStates[lineNumber].CurrentBeat == i
		if isCurrentBeat {
			backgroundSeqColor = themes.SeqCursorColor
		} else if m.visualSelection.visualMode != operation.VisualNone && m.InVisualSelection(currentGridKey) {
			backgroundSeqColor = themes.SeqVisualColor
		} else if hasNote && overlayNote.HighestOverlay && overlayNote.OverlayKey != overlaykey.ROOT {
			backgroundSeqColor = themes.SeqOverlayColor
		} else if hasNote && !overlayNote.HighestOverlay && overlayNote.OverlayKey != overlaykey.ROOT {
			backgroundSeqColor = themes.SeqMiddleOverlayColor
		} else if i%8 > 3 {
			backgroundSeqColor = themes.AltSeqBackgroundColor
		} else {
			backgroundSeqColor = themes.SeqBackgroundColor
		}

		char, foregroundColor := ViewNoteComponents(overlayNote.Note)
		var hasGateTail = false
		if (!hasNote || overlayNote.Note == zeronote) && gateSpace.HasMore() {
			char = gateSpace.ShiftString()
			hasGateTail = true
		} else if gateSpace.HasMore() {
			gateSpace = GateSpace{}
		}

		style := lipgloss.NewStyle().Background(backgroundSeqColor)
		cursorMatch := m.gridCursor.Line == uint8(lineNumber) && m.gridCursor.Beat == i
		if cursorMatch {
			if hasGateTail {
				m.cursor.Style = m.cursor.Style.Background(backgroundSeqColor).Foreground(gateSpace.Color)
			}
			m.cursor.SetChar(char)
			char = m.cursor.View()
		} else if m.visualSelection.visualMode != operation.VisualNone && m.InVisualSelection(currentGridKey) {
			style = style.Foreground(themes.Black)
		} else if hasGateTail {
			if isCurrentBeat {
				style = style.Foreground(backgroundSeqColor)
			} else {
				style = style.Foreground(gateSpace.Color)
			}
		} else {
			style = style.Foreground(foregroundColor)
		}

		if overlayNote.Note.GateIndex > int16(len(config.ShortGates))-1 && int(overlayNote.Note.GateIndex) < int(len(config.ShortGates)+len(config.LongGates)) {
			gateSpaceValue := config.LongGates[overlayNote.Note.GateIndex-8].Shape
			gateSpace.StringValue = []rune(gateSpaceValue)
			gateSpace.Color = foregroundColor
		}

		gridChord, exists := m.currentOverlay.FindChordWithNote(currentGridKey)

		if exists && gridChord == currentChord.GridChord && !cursorMatch {
			fg := style.GetForeground()
			bg := style.GetBackground()
			style = style.Background(fg).Foreground(bg)
		}

		if cursorMatch {
			buf.WriteString(char)
		} else {
			buf.WriteString(style.Render(char))
		}
	}

	buf.WriteString("\n")
	return buf.String()
}

func ViewNoteComponents(currentNote grid.Note) (string, lipgloss.Color) {
	currentAction := currentNote.Action
	var char string
	var foregroundColor lipgloss.Color
	var waitShape string

	if currentNote.WaitIndex > 0 {
		waitShape = "\u0320"
	}

	if currentAction == grid.ActionNothing && currentNote != zeronote {
		currentAccentShape := themes.AccentIcons[currentNote.AccentIndex]
		currentAccentColor := themes.AccentColors[currentNote.AccentIndex]
		char = string(currentAccentShape) +
			string(config.Ratchets[currentNote.Ratchets.Length]) +
			ShortGate(currentNote) +
			waitShape
		foregroundColor = lipgloss.Color(currentAccentColor)
	} else {
		lineaction := config.Lineactions[currentAction]
		lineActionColor := themes.ActionColors[currentAction]
		char = string(lineaction.Shape)
		foregroundColor = lipgloss.Color(lineActionColor)
	}

	return char, foregroundColor
}

func ShortGate(note note) string {
	if note.GateIndex < int16(len(config.ShortGates)) {
		return string(config.ShortGates[note.GateIndex].Shape)
	} else {
		return ""
	}
}
