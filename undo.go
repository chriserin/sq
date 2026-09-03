package main

import (
	"github.com/chriserin/sq/internal/arrangement"
	"github.com/chriserin/sq/internal/grid"
	"github.com/chriserin/sq/internal/operation"
	"github.com/chriserin/sq/internal/overlays"
)

type Undoable interface {
	ApplyUndo(m *model) Location
}

type Location struct {
	OverlayKey    overlayKey
	GridKey       gridKey
	ApplyLocation bool
}

type UndoStack struct {
	undo Undoable
	redo Undoable
	next *UndoStack
	id   int
}

var EmptyStack = UndoStack{}

type UndoBeats struct {
	beats     uint8
	ArrCursor arrangement.ArrCursor
}

func (ub UndoBeats) ApplyUndo(m *model) Location {
	m.arrangement.Cursor = ub.ArrCursor
	partID := m.CurrentPartID()
	(*m.definition.Parts)[partID].Beats = ub.beats
	return Location{ApplyLocation: false}
}

// UndoActionValue undoes/redoes the stored value of a value-carrying grid
// action (e.g. ActionSpecificValue's AccentIndex or ActionTempoChange's
// GateIndex).
type UndoActionValue struct {
	overlayKey     overlayKey
	cursorPosition gridKey
	ArrCursor      arrangement.ArrCursor
	action         grid.Action
	value          int16
}

func (uav UndoActionValue) ApplyUndo(m *model) Location {
	m.arrangement.Cursor = uav.ArrCursor
	overlay := m.CurrentPart().Overlays.FindOverlay(uav.overlayKey)
	switch uav.action {
	case grid.ActionSpecificValue:
		overlay.SetNote(uav.cursorPosition, note{Action: uav.action, AccentIndex: uint8(uav.value)})
		m.selectionIndicator = operation.SelectSpecificValue
	case grid.ActionTempoChange:
		overlay.SetNote(uav.cursorPosition, note{Action: uav.action, GateIndex: uav.value})
		m.selectionIndicator = operation.SelectTempoChangeValue
	}
	return Location{
		OverlayKey:    uav.overlayKey,
		GridKey:       uav.cursorPosition,
		ApplyLocation: true,
	}
}

type UndoNewOverlay struct {
	overlayKey     overlayKey
	cursorPosition gridKey
	ArrCursor      arrangement.ArrCursor
}

func (uno UndoNewOverlay) ApplyUndo(m *model) Location {
	m.arrangement.Cursor = uno.ArrCursor
	currentPartID := m.CurrentPartID()
	newOverlay := m.CurrentPart().Overlays.Remove(uno.overlayKey)
	(*m.definition.Parts)[currentPartID].Overlays = newOverlay
	return Location{uno.overlayKey, uno.cursorPosition, true}
}

type UndoRemoveOverlay struct {
	overlayKey     overlayKey
	overlay        *overlays.Overlay
	cursorPosition gridKey
	ArrCursor      arrangement.ArrCursor
}

func (uro UndoRemoveOverlay) ApplyUndo(m *model) Location {
	m.arrangement.Cursor = uro.ArrCursor
	m.EnsureOverlayWithKey(uro.overlayKey)
	if uro.overlay != nil {
		diff := overlays.DiffOverlays(m.currentOverlay, uro.overlay)
		diff.Apply(m.currentOverlay)
	}
	return Location{uro.overlayKey, uro.cursorPosition, true}
}

type UndoArrangement struct {
	arrUndo arrangement.Undoable
}

func (ua UndoArrangement) ApplyUndo(m *model) Location {
	m.arrangement.ApplyArrUndo(ua.arrUndo)
	m.focus = operation.FocusArrangementEditor
	m.showArrangementView = true
	m.arrangement.Focus = true
	return Location{ApplyLocation: false}
}

type UndoOverlayDiff struct {
	overlayKey     overlayKey
	cursorPosition gridKey
	ArrCursor      arrangement.ArrCursor
	overlayDiff    overlays.OverlayDiff
}

func (uod UndoOverlayDiff) ApplyUndo(m *model) Location {
	m.arrangement.Cursor = uod.ArrCursor
	overlay := m.CurrentPart().Overlays.FindOverlay(uod.overlayKey)
	if overlay == nil {
		m.currentOverlay = m.CurrentPart().Overlays
	} else {
		m.currentOverlay = overlay
	}
	uod.overlayDiff.Apply(m.currentOverlay)
	if len(uod.overlayDiff.RemovedChords) > 0 {
		m.UnsetActiveChord()
	}
	return Location{uod.overlayKey, uod.cursorPosition, true}
}

type UndoStateDiff struct {
	stateDiff StateDiff
}

func (usd UndoStateDiff) ApplyUndo(m *model) Location {
	usd.stateDiff.Apply(m)

	if usd.stateDiff.AccentsChanged {
		m.selectionIndicator = operation.SelectAccentStart
	}
	if usd.stateDiff.LinesChanged {
		m.selectionIndicator = operation.SelectSetupChannel
	}
	if usd.stateDiff.SubdivisionsChanged {
		m.selectionIndicator = operation.SelectTempoSubdivision
	}
	if usd.stateDiff.TempoChanged {
		m.selectionIndicator = operation.SelectTempo
	}
	return Location{ApplyLocation: false}
}
