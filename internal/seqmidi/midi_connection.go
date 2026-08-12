// Package seqmidi provides MIDI connection management and message sending
// functionality for the sequencer. It handles virtual and physical MIDI
// port connections, thread-safe message transmission, and DAW integration
// features including record triggering for external recording systems.
package seqmidi

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/chriserin/sq/internal/notereg"
	midi "gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
)

const TransmitterName string = "sq-transmitter"

type MidiConnection struct {
	IsTransmitter bool
	DoNotListen   bool
	outportName   string
	seqOutport    drivers.Out
	midiChannel   chan Message
	outDevices    []*OutDeviceInfo
	inDevices     []*InDeviceInfo
	TestQueue     *[]Message
	Test          bool
	StopFn        func()
	ReceiverFunc  ReceiverFunc
}

func (mc *MidiConnection) HasTransmitter() bool {
	for _, device := range mc.inDevices {
		if device.IsTransmitter {
			return true
		}
	}
	return false
}

func (mc *MidiConnection) StopReceivingFromTransmitter() {
	if mc.StopFn != nil {
		mc.StopFn()
		mc.StopFn = nil
	}
}

func (mc *MidiConnection) HasOutport() bool {
	return mc.seqOutport != nil
}

func (mc *MidiConnection) HasDevices() bool {
	return len(mc.outDevices) > 0
}

func (mc *MidiConnection) EnsureConnection() {
	if !mc.HasConnection() {
		if len(mc.outDevices) > 0 {
			mc.outDevices[0].Open()
			mc.outDevices[0].Selected = true
		}
	}
}

func (mc *MidiConnection) HasConnection() bool {
	hasConnection := false
	for _, device := range mc.outDevices {
		if device.IsOpen && device.Selected {
			hasConnection = true
			break
		}
	}
	return hasConnection
}

type Message struct {
	Delay time.Duration
	Msg   midi.Message
}

func InitMidiConnection(createOut bool, outportName string, ctx context.Context) *MidiConnection {
	var midiConn MidiConnection
	if createOut {
		midiConn = MidiConnection{midiChannel: make(chan Message)}
	} else {
		midiConn = MidiConnection{outportName: outportName, midiChannel: make(chan Message)}
	}

	return &midiConn
}

var playMutex = sync.Mutex{}

func (mc *MidiConnection) LoopMidi(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Recovered in MIDI send loop from panic: %v\n", r)
				debug.PrintStack()
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-mc.midiChannel:
				if msg.Delay == 0 {
					key := notereg.GetKey(msg.Msg)
					if msg.Msg.Type().Is(midi.NoteOnMsg) && !notereg.HasKey(key) {
						notereg.AddKey(key)
					}
					playMutex.Lock()
					err := mc.SendMidi(msg.Msg)
					playMutex.Unlock()
					if err != nil {
						panic(err)
					}
				} else {
					key := notereg.GetKey(msg.Msg)
					timer := time.AfterFunc(msg.Delay, func() {
						if msg.Msg.Type().Is(midi.NoteOffMsg) && !notereg.HasKey(key) {
							return
						}
						if msg.Msg.Type().Is(midi.NoteOnMsg) && !notereg.HasKey(key) {
							notereg.AddKey(key)
						}
						playMutex.Lock()
						err := mc.SendMidi(msg.Msg)
						playMutex.Unlock()
						if msg.Msg.Type().Is(midi.NoteOffMsg) {

							notereg.RemoveKey(key)
						}
						if err != nil {
							panic(err)
						}
					})
					notereg.AddTimer(key, timer)
				}
			}
		}
	}()
}

func (mc *MidiConnection) Send(msg Message) {
	if mc.Test {
		*mc.TestQueue = append(*mc.TestQueue, msg)
	} else {
		mc.midiChannel <- msg
	}
}

func (mc MidiConnection) SendMidi(msg midi.Message) error {
	if mc.seqOutport != nil {
		if mc.seqOutport.IsOpen() {
			err := mc.seqOutport.Send(msg)
			if err != nil {
				return err
			}
		}
	} else {
		// Send to all selected devices
		for _, device := range mc.outDevices {
			if device.Selected {
				if device.Out.IsOpen() {
					err := device.Out.Send(msg)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (mc *MidiConnection) Panic(channels []uint8) error {
	// NOTE: No connection means nothing to panic about
	for _, channel := range channels {
		for i := range 127 {
			err := mc.SendMidi(midi.NoteOff(channel-1, uint8(i)))
			if err != nil {
				return fault.Wrap(err, fmsg.With("cannot send panic note off"))
			}
		}
	}

	return nil
}

func (mc *MidiConnection) Close() {
	for _, device := range mc.outDevices {
		if device.IsOpen {
			_ = device.Out.Close()
			device.IsOpen = false
		}
	}
}

var dawOutports = []string{"Logic Pro Virtual In", "TESTDAW"}

func (mc MidiConnection) SendPlayMessage() error {
	var selectedOutport = mc.GetDawOutport()

	if selectedOutport == nil {
		return fault.New("could not find daw outport", fmsg.WithDesc("could not find daw outport", "Could not find a daw outport, please ensure the DAW record destination is open and then restart sq"))
	}

	// NOTE: The record message must be configured in Logic.
	// Logic Pro -> Control Surfaces -> Controller Assignments...
	// Create a new zone (sq), a new Mode (commands), a new control
	// Within the new control press "Learn" and then in sq use the PlayRecord mapping (`:<Space>`).
	// Then Chose `Class: Key Command` `Command: Global Commands` and then `Record` in the unlabeled parameter list
	err := selectedOutport.Send(midi.ControlChange(16, 127, 126))
	if err != nil {
		return fault.Wrap(err, fmsg.With("could not send record message"))
	}
	return nil
}

func (mc MidiConnection) SendRecordMessage() error {
	var selectedOutport = mc.GetDawOutport()

	if selectedOutport == nil {
		return fault.New("could not find daw outport", fmsg.WithDesc("could not find daw outport", "Could not find a daw outport, please ensure the DAW record destination is open and then restart sq"))
	}

	// NOTE: The record message must be configured in Logic.
	// Logic Pro -> Control Surfaces -> Controller Assignments...
	// Create a new zone (sq), a new Mode (commands), a new control
	// Within the new control press "Learn" and then in sq use the PlayRecord mapping (`:<Space>`).
	// Then Chose `Class: Key Command` `Command: Global Commands` and then `Record` in the unlabeled parameter list
	err := selectedOutport.Send(midi.ControlChange(16, 127, 127))
	if err != nil {
		return fault.Wrap(err, fmsg.With("could not send record message"))
	}
	return nil
}

func (mc *MidiConnection) GetDawOutport() drivers.Out {
	for _, device := range mc.outDevices {
		if device.IsDaw && device.IsOpen {
			return device.Out
		}
	}
	return nil
}

func (mc *MidiConnection) GetClockGateOutport() drivers.Out {
	for _, device := range mc.outDevices {
		if device.IsClockGate && device.IsOpen {
			return device.Out
		}
	}
	return nil
}

// SendClockGate sends directly to the armed clock-gate device, bypassing the
// notereg-backed delayed-message queue (LoopMidi) so a clock-gate NoteOff
// can't collide with notereg dedup state for a real sequence note sharing
// the same channel/note. It still takes playMutex because the armed device
// is always the same drivers.Out that LoopMidi writes to under that lock.
func (mc *MidiConnection) SendClockGate(msg midi.Message) error {
	out := mc.GetClockGateOutport()
	if out == nil {
		return nil
	}
	playMutex.Lock()
	defer playMutex.Unlock()
	return out.Send(msg)
}

func (mc *MidiConnection) SendStopMessage() error {
	var selectedOutport = mc.GetDawOutport()

	if selectedOutport == nil {
		return nil
	}

	err := selectedOutport.Send(midi.ControlChange(16, 127, 125))
	if err != nil {
		return fault.Wrap(err, fmsg.With("could not send record message"))
	}
	return nil
}

func FindTransmitterPort() (drivers.In, error) {
	return FindInPort(TransmitterName)
}

func FindInPort(inPortName string) (drivers.In, error) {
	ins, err := GetIns()
	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("cannot get midi ins"))
	}
	for _, in := range ins {
		if strings.Contains(in.String(), inPortName) {
			return in, nil
		}
	}
	return nil, fault.New("cannot find transmitter port", fmsg.With("cannot find transmitter port"))
}

var OutputName string = "sq-cli-out"

func (mc *MidiConnection) CreateOutport() error {
	outport, err := OpenVirtualOut(OutputName)
	if err != nil {
		return fault.Wrap(err, fmsg.With("cannot create virtual outport"))
	}
	mc.seqOutport = outport
	return nil
}
