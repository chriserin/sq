package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/chriserin/sq/internal/arrangement"
	"github.com/chriserin/sq/internal/operation"
	"github.com/chriserin/sq/internal/overlaykey"
	"github.com/chriserin/sq/internal/playstate"
	"github.com/chriserin/sq/internal/sequence"
	"github.com/stretchr/testify/assert"
)

var OK = overlaykey.InitOverlayKey

func TestUpdateArrangementFocus(t *testing.T) {
	t.Run("switch to arrangement view and create part", func(t *testing.T) {
		// Setup a model with a basic arrangement
		var parts = sequence.InitParts()
		var arr = sequence.InitArrangement(parts)
		def := sequence.Sequence{
			Arrangement: arr,
			Parts:       &parts,
			Keyline:     0,
		}

		iterations := make(playstate.Iterations)
		playstate.BuildIterationsMap(arr, &iterations)
		m := model{
			arrangement: arrangement.InitModel(def.Arrangement, def.Parts),
			definition:  def,
			playState: playstate.PlayState{
				LineStates: playstate.InitLineStates(1, []playstate.LineState{}, 0),
				Iterations: &iterations,
			},
			focus: operation.FocusGrid, // Start with grid focus
		}

		initialNodeCount := m.arrangement.Root.CountEndNodes()
		updatedModel, _ := m.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
		modelPtr := updatedModel.(model)

		assert.Equal(t, operation.FocusArrangementEditor, modelPtr.focus, "Model should have arrangement editor focus")
		assert.True(t, modelPtr.arrangement.Focus, "Arrangement model should have focus flag set to true")

		updatedModel, _ = updatedModel.Update(tea.KeyPressMsg{Code: ']', Mod: tea.ModCtrl})
		modelPtr = updatedModel.(model)

		assert.Equal(t, operation.FocusArrangementEditor, modelPtr.focus, "Model should have arrangement editor focus")
		assert.True(t, modelPtr.arrangement.Focus, "Arrangement model should have focus flag set to true")

		updatedModelAfterPart, _ := updatedModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		finalModel := updatedModelAfterPart.(model)

		finalNodeCount := finalModel.arrangement.Root.CountEndNodes()
		assert.Greater(t, finalNodeCount, initialNodeCount, "Arrangement should have more end nodes after part creation")

		assert.Equal(t, operation.FocusArrangementEditor, finalModel.focus, "Model should still have arrangement editor focus")
		assert.True(t, finalModel.arrangement.Focus, "Arrangement model should still have focus flag set to true")
		assert.Equal(t, operation.SelectGrid, finalModel.selectionIndicator, "Selection indicator should be reset to nothing")
	})
}

func TestSolo(t *testing.T) {
	t.Run("First Solo", func(t *testing.T) {
		playStates := []playstate.LineState{
			{GroupPlayState: playstate.PlayStatePlay},
			{GroupPlayState: playstate.PlayStatePlay},
		}
		newPlayStates := Solo(playStates, 0)
		assert.Equal(t, newPlayStates[0].GroupPlayState, playstate.PlayStateSolo)
		assert.Equal(t, newPlayStates[1].GroupPlayState, playstate.PlayStatePlay)
	})

	t.Run("First UnSolo", func(t *testing.T) {
		playStates := []playstate.LineState{
			{GroupPlayState: playstate.PlayStateSolo},
			{GroupPlayState: playstate.PlayStatePlay},
		}
		newPlayStates := Solo(playStates, 0)
		assert.Equal(t, newPlayStates[0].GroupPlayState, playstate.PlayStatePlay)
		assert.Equal(t, newPlayStates[1].GroupPlayState, playstate.PlayStatePlay)
	})
}
