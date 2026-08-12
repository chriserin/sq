package seqmidi

import (
	"testing"

	"github.com/chriserin/sq/internal/config"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/drivers/testdrv"
)

type fakeDriver struct {
	outs []drivers.Out
}

func (d *fakeDriver) Ins() ([]drivers.In, error)   { return nil, nil }
func (d *fakeDriver) Outs() ([]drivers.Out, error) { return d.outs, nil }
func (d *fakeDriver) String() string               { return "fake" }
func (d *fakeDriver) Close() error                 { return nil }

// namedOut returns a real drivers.Out backed by testdrv, named "<name>-out".
func namedOut(name string) drivers.Out {
	outs, _ := testdrv.New(name).Outs()
	return outs[0]
}

func resetClockGateConfig() {
	config.ClockGateDeviceName = ""
}

func TestUpdateOutDeviceList_ClockGateArming(t *testing.T) {
	t.Run("armed when --midiout and clock device both match the same device", func(t *testing.T) {
		defer resetClockGateConfig()
		config.ClockGateDeviceName = "widget"
		mc := &MidiConnection{outportName: "widget"}
		driver := &fakeDriver{outs: []drivers.Out{namedOut("widget")}}

		assert.NoError(t, mc.UpdateOutDeviceList(driver))
		assert.True(t, mc.outDevices[0].IsClockGate)
	})

	t.Run("not armed when only --midiout matches", func(t *testing.T) {
		defer resetClockGateConfig()
		mc := &MidiConnection{outportName: "widget"}
		driver := &fakeDriver{outs: []drivers.Out{namedOut("widget")}}

		assert.NoError(t, mc.UpdateOutDeviceList(driver))
		assert.False(t, mc.outDevices[0].IsClockGate)
	})

	t.Run("not armed when only the clock device config matches", func(t *testing.T) {
		defer resetClockGateConfig()
		config.ClockGateDeviceName = "widget"
		mc := &MidiConnection{}
		driver := &fakeDriver{outs: []drivers.Out{namedOut("widget")}}

		assert.NoError(t, mc.UpdateOutDeviceList(driver))
		assert.False(t, mc.outDevices[0].IsClockGate)
	})

	t.Run("not armed when both are empty", func(t *testing.T) {
		defer resetClockGateConfig()
		mc := &MidiConnection{}
		driver := &fakeDriver{outs: []drivers.Out{namedOut("widget")}}

		assert.NoError(t, mc.UpdateOutDeviceList(driver))
		assert.False(t, mc.outDevices[0].IsClockGate)
	})

	t.Run("not armed when --midiout and clock device name two different devices", func(t *testing.T) {
		defer resetClockGateConfig()
		config.ClockGateDeviceName = "other"
		mc := &MidiConnection{outportName: "widget"}
		driver := &fakeDriver{outs: []drivers.Out{namedOut("widget"), namedOut("other")}}

		assert.NoError(t, mc.UpdateOutDeviceList(driver))
		for _, d := range mc.outDevices {
			assert.False(t, d.IsClockGate, "device %q should not be armed", d.Name)
		}
	})

	t.Run("stays armed across a refresh (already-known device branch)", func(t *testing.T) {
		defer resetClockGateConfig()
		config.ClockGateDeviceName = "widget"
		mc := &MidiConnection{outportName: "widget"}
		driver := &fakeDriver{outs: []drivers.Out{namedOut("widget")}}

		assert.NoError(t, mc.UpdateOutDeviceList(driver))
		assert.True(t, mc.outDevices[0].IsClockGate)

		// Second scan re-enumerates the same device name, exercising the
		// "already-known device" branch instead of "new device".
		assert.NoError(t, mc.UpdateOutDeviceList(driver))
		assert.True(t, mc.outDevices[0].IsClockGate)
	})
}
