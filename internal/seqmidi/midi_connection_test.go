package seqmidi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	midi "gitlab.com/gomidi/midi/v2"
)

// fakeOut is a minimal, fully-controlled drivers.Out so tests can assert
// exactly which device received which bytes, without depending on testdrv's
// Listen/Reader plumbing (which panics on Send if nothing ever called Listen).
type fakeOut struct {
	name   string
	isOpen bool
	sent   [][]byte
}

func (f *fakeOut) String() string          { return f.name }
func (f *fakeOut) Number() int             { return -1 }
func (f *fakeOut) IsOpen() bool            { return f.isOpen }
func (f *fakeOut) Underlying() interface{} { return nil }
func (f *fakeOut) Close() error            { f.isOpen = false; return nil }
func (f *fakeOut) Open() error             { f.isOpen = true; return nil }
func (f *fakeOut) Send(data []byte) error {
	f.sent = append(f.sent, append([]byte(nil), data...))
	return nil
}

func newDevice(name string, isOpen, selected bool) (*OutDeviceInfo, *fakeOut) {
	out := &fakeOut{name: name, isOpen: isOpen}
	return &OutDeviceInfo{Out: out, Name: name, IsOpen: isOpen, Selected: selected}, out
}

func TestSendMidi_NamedRouting(t *testing.T) {
	deviceA, outA := newDevice("A", false, false)
	deviceB, outB := newDevice("B", false, false)
	mc := MidiConnection{outDevices: map[string]*OutDeviceInfo{"A": deviceA, "B": deviceB}}

	msg := midi.NoteOn(0, 60, 100)
	err := mc.SendMidi(msg, "B")

	assert.NoError(t, err)
	assert.Empty(t, outA.sent, "device not named in the route should receive nothing")
	assert.Len(t, outB.sent, 1, "named device should receive exactly one message")
	assert.True(t, deviceB.IsOpen, "SendMidi should open the named device on demand")
	assert.False(t, deviceA.IsOpen, "unrelated device should be left untouched")
}

func TestSendMidi_NamedDeviceNotFound(t *testing.T) {
	deviceA, outA := newDevice("A", true, true)
	mc := MidiConnection{outDevices: map[string]*OutDeviceInfo{"A": deviceA}}

	err := mc.SendMidi(midi.NoteOn(0, 60, 100), "does-not-exist")

	assert.NoError(t, err, "a disconnected/unknown named device should silently no-op, not error")
	assert.Empty(t, outA.sent, "no device should receive anything")
}

func TestSendMidi_DefaultRoutesToSelectedDevice(t *testing.T) {
	deviceA, outA := newDevice("A", true, false)
	deviceB, outB := newDevice("B", true, true) // Selected
	mc := MidiConnection{outDevices: map[string]*OutDeviceInfo{"A": deviceA, "B": deviceB}}

	err := mc.SendMidi(midi.NoteOn(0, 60, 100), "")

	assert.NoError(t, err)
	assert.Empty(t, outA.sent, "non-selected device should not receive the default-routed message")
	assert.Len(t, outB.sent, 1, "the Selected device should receive the default-routed message")
}

func TestSendMidi_DefaultWithNoSelectedDeviceIsSilentNoOp(t *testing.T) {
	deviceA, outA := newDevice("A", true, false)
	mc := MidiConnection{outDevices: map[string]*OutDeviceInfo{"A": deviceA}}

	err := mc.SendMidi(midi.NoteOn(0, 60, 100), "")

	assert.NoError(t, err)
	assert.Empty(t, outA.sent)
}

func TestPanic_SendsToEveryOpenDeviceNotJustSelected(t *testing.T) {
	deviceA, outA := newDevice("A", true, true)   // open + selected (default)
	deviceB, outB := newDevice("B", true, false)  // open, not selected (a line routed here directly)
	deviceC, outC := newDevice("C", false, false) // not open, should be skipped
	mc := MidiConnection{outDevices: map[string]*OutDeviceInfo{"A": deviceA, "B": deviceB, "C": deviceC}}

	err := mc.Panic([]uint8{1})

	assert.NoError(t, err)
	assert.Len(t, outA.sent, 127, "open selected device should get a note-off for every note")
	assert.Len(t, outB.sent, 127, "open non-selected device should also get note-offs — a stuck note could be on any device a line used")
	assert.Empty(t, outC.sent, "closed device should not receive anything")
}

func TestEnsureConnection_PreferencePriority(t *testing.T) {
	t.Run("no-op when a connection already exists", func(t *testing.T) {
		device, _ := newDevice("A", true, true)
		mc := MidiConnection{outDevices: map[string]*OutDeviceInfo{"A": device}}
		mc.EnsureConnection()
		assert.True(t, device.Selected)
	})

	t.Run("prefers a real device over the virtual port", func(t *testing.T) {
		real, _ := newDevice("A", false, false)
		virtual, _ := newDevice(OutputName, false, false)
		virtual.IsVirtual = true
		mc := MidiConnection{
			outDevices:       map[string]*OutDeviceInfo{"A": real, OutputName: virtual},
			virtualOutDevice: virtual,
		}
		mc.EnsureConnection()
		assert.True(t, real.Selected, "real device should become the default")
		assert.True(t, real.IsOpen)
		assert.False(t, virtual.Selected, "virtual port should not be selected while a real device is available")
	})

	t.Run("falls back to the virtual port when no real device exists", func(t *testing.T) {
		virtual, _ := newDevice(OutputName, true, false)
		virtual.IsVirtual = true
		mc := MidiConnection{
			outDevices:       map[string]*OutDeviceInfo{OutputName: virtual},
			virtualOutDevice: virtual,
		}
		mc.EnsureConnection()
		assert.True(t, virtual.Selected, "virtual port should become the default when nothing else is available")
	})
}

func TestDefaultOutputName(t *testing.T) {
	t.Run("returns the selected+open device's name", func(t *testing.T) {
		deviceA, _ := newDevice("A", true, false)
		deviceB, _ := newDevice("B", true, true)
		mc := MidiConnection{outDevices: map[string]*OutDeviceInfo{"A": deviceA, "B": deviceB}}
		assert.Equal(t, "B", mc.DefaultOutputName())
	})

	t.Run("returns empty string when nothing is selected", func(t *testing.T) {
		deviceA, _ := newDevice("A", true, false)
		mc := MidiConnection{outDevices: map[string]*OutDeviceInfo{"A": deviceA}}
		assert.Equal(t, "", mc.DefaultOutputName())
	})
}

func TestOutDeviceNames_SortedAndComplete(t *testing.T) {
	deviceA, _ := newDevice("Zeta", true, false)
	deviceB, _ := newDevice("Alpha", true, false)
	mc := MidiConnection{outDevices: map[string]*OutDeviceInfo{"Zeta": deviceA, "Alpha": deviceB}}

	assert.Equal(t, []string{"Alpha", "Zeta"}, mc.OutDeviceNames())
}

func TestHasOutDevice(t *testing.T) {
	deviceA, _ := newDevice("A", true, false)
	mc := MidiConnection{outDevices: map[string]*OutDeviceInfo{"A": deviceA}}

	assert.True(t, mc.HasOutDevice("A"))
	assert.False(t, mc.HasOutDevice("B"))
}
