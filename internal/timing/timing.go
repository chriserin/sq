package timing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	tea "charm.land/bubbletea/v2"
	"github.com/chriserin/sq/internal/beats"
	"github.com/chriserin/sq/internal/playstate"
	"github.com/chriserin/sq/internal/seqmidi"
	midi "gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
)

// NOTE: PPQN is Pulses Per Quarter Note
// NOTE: 840 is the common multiple of 2, 3, 5, 7, 8. The allowable subdivisions.
var PPQN = 840

func (t *Timing) BeatInterval() time.Duration {
	tickInterval := t.TickInterval()
	adjuster := time.Since(t.playTime) - t.trackTime
	t.trackTime = t.trackTime + tickInterval
	next := tickInterval - adjuster
	return next
}

func (t Timing) TickInterval() time.Duration {
	return time.Minute / time.Duration(t.tempo*t.subdivisions)
}

func (t Tick) ReceiverBeatInterval(subdivisions int) time.Duration {
	return time.Minute / time.Duration(t.tempo*subdivisions)
}

func (t Timing) PulseInterval() time.Duration {
	return time.Minute / time.Duration(t.tempo*PPQN)
}

type Tick struct {
	tempo int
}

type Timing struct {
	playTime     time.Time
	trackTime    time.Duration
	tempo        int
	subdivisions int
	started      bool
	pulseCount   int
	pulseLimit   int
	preRollBeats uint8
	transmitting bool
	beatsLooper  beats.BeatsLooper
	ctx          context.Context
}

type MidiLoopMode uint8

const (
	MlmStandAlone MidiLoopMode = iota
	MlmTransmitter
	MlmReceiver
)

var timingChannel chan TimingMsg

func init() {
	timingChannel = make(chan TimingMsg)
}

func GetTimingChannel() chan TimingMsg {
	return timingChannel
}

func Loop(mode MidiLoopMode, lockReceiverChannel, unlockReceiverChannel chan bool, ctx context.Context, beatsLooper beats.BeatsLooper, sendFn func(tea.Msg), midiConnection *seqmidi.MidiConnection) error {
	timing := Timing{beatsLooper: beatsLooper, ctx: ctx}
	switch mode {
	case MlmStandAlone:
		timing.StandAloneLoop(sendFn)
	case MlmTransmitter:
		err := timing.TransmitterLoop(sendFn, midiConnection)
		if err != nil {
			return fault.Wrap(err, fmsg.With("cannot start transmitter loop"))
		}
	case MlmReceiver:
		err := timing.ReceiverLoop(lockReceiverChannel, unlockReceiverChannel, sendFn, midiConnection)
		timing.StandAloneLoop(sendFn)
		if err != nil {
			// NOTE: In case the receiver loop was not setup correctly, swallow the lock/unlock messages
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-lockReceiverChannel:
					case <-unlockReceiverChannel:
					}
				}
			}()
			return fault.Wrap(err, fmsg.With("cannot start receiver loop"))
		}
	}
	return nil
}

type Transmitter struct {
	out drivers.Out
}

func (tmtr Transmitter) Start(loopMode playstate.LoopMode) error {
	message := midi.SPP(uint16(loopMode))
	err := tmtr.out.Send(message)
	if err != nil {
		return fault.Wrap(err, fmsg.With("cannot send midi spp pre-start"))
	}
	message = midi.Start()
	err = tmtr.out.Send(message)
	if err != nil {
		return fault.Wrap(err, fmsg.With("cannot send midi start"))
	}
	return nil
}

func (tmtr Transmitter) Stop() error {
	err := tmtr.out.Send(midi.Stop())
	if err != nil {
		return fault.Wrap(err, fmsg.With("cannot send midi stop"))
	}
	return nil
}

func (tmtr Transmitter) Pulse() error {
	err := tmtr.out.Send(midi.TimingClock())
	if err != nil {
		return fault.Wrap(err, fmsg.With("cannot send midi clock"))
	}
	return nil
}

func (tmtr Transmitter) ActiveSense() error {
	err := tmtr.out.Send(midi.Activesense())
	if err != nil {
		return fault.Wrap(err, fmsg.With("cannot send midi active sense"))
	}
	return nil
}

func (t *Timing) TransmitterLoop(sendFn func(tea.Msg), midiConnection *seqmidi.MidiConnection) error {
	var beatChannel = t.beatsLooper.BeatChannel
	var clockChannel = t.beatsLooper.ClockChannel
	out, err := midiConnection.TransmitterOut()
	if err != nil {
		return fault.Wrap(err, fmsg.With("cannot open transmitter out"))
	}
	transmitter := Transmitter{out}
	err = transmitter.ActiveSense()
	if err != nil {
		return fault.Wrap(err)
	}

	tickChannel := make(chan Tick)
	activeSenseChannel := make(chan bool)
	var command TimingMsg

	var tickTimer *time.Timer
	pulse := func(adjustedInterval time.Duration) {
		tickTimer = time.AfterFunc(adjustedInterval, func() {
			tickChannel <- Tick{}
		})
	}

	activesense := func() {
		time.AfterFunc(300*time.Millisecond, func() {
			activeSenseChannel <- true
		})
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Recovered in timing transmitter loop from panic: %v\n", r)
				debug.PrintStack()
			}
		}()
		for {
			select {
			case <-t.ctx.Done():
				return
			case command = <-timingChannel:
				switch command := command.(type) {
				case StartMsg:
					t.started = true
					t.playTime = time.Now()
					t.tempo = command.Tempo
					t.subdivisions = command.Subdivisions
					t.trackTime = time.Duration(0)
					t.pulseCount = 0
					t.pulseLimit = 0
					t.preRollBeats = command.Prerollbeats
					t.transmitting = command.Transmitting
					pulse(0)
					err := transmitter.Start(command.LoopMode)
					if err != nil {
						sendFn(ErrorMsg{err})
					}
				case StopMsg:
					t.started = false
					tickTimer.Stop()
					err := transmitter.Stop()
					if err != nil {
						sendFn(ErrorMsg{err})
					}
					// m.playing should be false now.
				case AnticipatoryStopMsg:
					//NOTE: A receiver must not receive a Pulse message and a Stop message in immediate succession.
					//This will result in a race condition on the receiver end.  Instead, we anticipate stopping
					//and set a limit on the pulses that will be accumulated, preventing the final pulse.
					if t.pulseLimit == 0 {
						t.pulseLimit = t.pulseCount + ((PPQN / t.subdivisions) - 1)
					}
				case TempoMsg:
					t.tempo = command.Tempo
					t.subdivisions = command.Subdivisions
				}
			case <-tickChannel:
				if t.started {
					if t.pulseCount%(PPQN/24) == 0 {
						clockChannel <- beats.ClockMsg{}
					}
					if t.preRollBeats == 0 {
						if t.pulseLimit == 0 || t.pulseCount < t.pulseLimit {
							if t.transmitting {
								err := transmitter.Pulse()
								if err != nil {
									wrappedErr := fault.Wrap(err)
									sendFn(ErrorMsg{wrappedErr})
								}
							}
						}
						if t.pulseCount%(PPQN/t.subdivisions) == 0 {
							tickInterval := t.TickInterval()
							beatChannel <- beats.BeatMsg{Interval: tickInterval}
						}
					} else {
						if t.pulseCount%(PPQN/t.subdivisions) == 0 {
							t.preRollBeats--
							if t.preRollBeats == 0 {
								t.pulseCount = -1
							}
						}
					}
					pulseInterval := t.PulseInterval()

					adjuster := time.Since(t.playTime) - t.trackTime
					t.trackTime = t.trackTime + pulseInterval
					next := pulseInterval - adjuster
					pulse(next)
					t.pulseCount++
				}
			case <-activeSenseChannel:
				if !t.started {
					err := transmitter.ActiveSense()
					if err != nil {
						wrappedErr := fault.Wrap(err, fmsg.With("activesense interrupted"))
						sendFn(ErrorMsg{wrappedErr})
					}
				}
				activesense()
			}
		}
	}()
	// Start active sense loop
	activesense()
	return nil
}

func (t *Timing) ReceiverLoop(lockReceiverChannel, unlockReceiverChannel chan bool, sendFn func(tea.Msg), midiConnection *seqmidi.MidiConnection) (receiverError error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Recovered in timing receiver loop from panic: %v\n", r)
				debug.PrintStack()
			}
		}()
		// only store last 3 intervals then average them for tempo calculation
		var intervals []time.Duration
		for {
			var beatChannel = t.beatsLooper.BeatChannel
			receiverChannel := make(chan TimingMsg)
			tickChannel := make(chan Tick)
			activeSenseChannel := make(chan bool)
			var loopMode playstate.LoopMode
			var timingClockTime time.Time
			var ReceiverFunc seqmidi.ReceiverFunc = func(msg []byte, milliseconds int32) {
				midiMessage := midi.Message(msg)
				switch midiMessage.Type() {
				case midi.SPPMsg:
					var ref uint16
					midiMessage.GetSPP(&ref)
					loopMode = playstate.LoopMode(ref)
				case midi.StartMsg:
					timingClockTime = time.Time{}
					receiverChannel <- StartMsg{LoopMode: loopMode}
				case midi.StopMsg:
					receiverChannel <- StopMsg{}
				case midi.TimingClockMsg:
					if timingClockTime.IsZero() {
						timingClockTime = time.Now()
						//NOTE: We don't have enough information to determine the tempo at this point, send a reasonable guess tempo
						//TODO: Figure out how to communicate tempo between sender and receiver or how to play ratchets based on another heuristic
						tickChannel <- Tick{tempo: 120}
					} else {
						intervals = append(intervals, time.Since(timingClockTime))
						if len(intervals) > 3 {
							intervals = intervals[1:]
						}
						var total time.Duration
						for _, inter := range intervals {
							total += inter
						}
						averageInterval := total / time.Duration(len(intervals))
						division := averageInterval * time.Duration(PPQN)
						tempo := (1 * time.Minute) / division
						tickChannel <- Tick{tempo: int(tempo)}
						timingClockTime = time.Now()
					}
				case midi.ActiveSenseMsg:
					activeSenseChannel <- true
				default:
					println("receiving unknown msg")
					println(midiMessage.Type().String())
				}
			}
			err := midiConnection.ListenToTransmitter(ReceiverFunc)
			if err != nil {
				sendFn(ErrorMsg{errors.New("error in setting up midi listener for transmitter")})
			}
			activeSenseTimer := time.AfterFunc(330*time.Millisecond, func() {
				sendFn(TransmitterNotConnectedMsg{})
				activeSenseChannel <- false
			})
			var command TimingMsg
		inner:
			for {
				select {
				case <-t.ctx.Done():
					return
				case <-lockReceiverChannel:
					midiConnection.StopReceivingFromTransmitter()
					midiConnection.DoNotListen = true
					activeSenseTimer.Stop()
					<-unlockReceiverChannel
					midiConnection.DoNotListen = false
					break inner
				case command = <-receiverChannel:
					switch command := command.(type) {
					case StartMsg:
						t.started = true
						t.playTime = time.Now()
						t.trackTime = time.Duration(0)
						t.pulseCount = 0
						sendFn(UIStartMsg{LoopMode: command.LoopMode})
					case StopMsg:
						t.started = false
						sendFn(UIStopMsg{})
						// m.playing should be false now.
					}
				case command = <-timingChannel:
					switch command := command.(type) {
					case TempoMsg:
						t.tempo = command.Tempo
						t.subdivisions = command.Subdivisions
					case QuitMsg:
						activeSenseTimer.Stop()
						midiConnection.StopFn()
					}
				case pulseTiming := <-tickChannel:
					activeSenseTimer.Reset(330 * time.Millisecond)
					if t.started {
						if t.pulseCount%(PPQN/t.subdivisions) == 0 {
							beatChannel <- beats.BeatMsg{Interval: pulseTiming.ReceiverBeatInterval(t.subdivisions)}
						}
						t.pulseCount++
					}
				case isGood := <-activeSenseChannel:
					activeSenseTimer.Reset(330 * time.Millisecond)
					if isGood {
						sendFn(TransmitterConnectedMsg{})
					} else {
						sendFn(TransmitterNotConnectedMsg{})
					}
				}
			}
		}
	}()
	return nil
}

func (t *Timing) StandAloneLoop(sendFn func(tea.Msg)) {
	var beatChannel = t.beatsLooper.BeatChannel
	tickChannel := make(chan Tick)
	var command TimingMsg

	var tickTimer *time.Timer
	tick := func(adjustedInterval time.Duration) {
		tickTimer = time.AfterFunc(adjustedInterval, func() {
			tickChannel <- Tick{}
		})
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Recovered in timing standalone loop from panic: %v\n", r)
				debug.PrintStack()
			}
		}()
		for {
			select {
			case <-t.ctx.Done():
				return
			case command = <-timingChannel:
				switch command := command.(type) {
				case StartMsg:
					t.started = true
					t.playTime = time.Now()
					t.tempo = command.Tempo
					t.subdivisions = command.Subdivisions
					t.trackTime = time.Duration(0)
					t.preRollBeats = command.Prerollbeats
					tick(0)
				case StopMsg:
					t.started = false
					if tickTimer != nil {

						tickTimer.Stop()
					}
				case TempoMsg:
					t.tempo = command.Tempo
					t.subdivisions = command.Subdivisions
				}
			case <-tickChannel:
				if t.started {
					adjustedInterval := t.BeatInterval()
					tick(adjustedInterval)
					if t.preRollBeats == 0 {
						beatChannel <- beats.BeatMsg{Interval: adjustedInterval}
					} else {
						t.preRollBeats--
					}
				}
			}
		}
	}()
}

type TimingMsg = any
type StartMsg struct {
	Transmitting bool
	LoopMode     playstate.LoopMode
	Prerollbeats uint8
	Tempo        int
	Subdivisions int
}

type StopMsg struct{}
type AnticipatoryStopMsg struct{}
type QuitMsg struct{}

type TempoMsg struct {
	Tempo        int
	Subdivisions int
}

type ErrorMsg struct {
	error error
}

type UIStopMsg struct{}
type UIStartMsg struct{ LoopMode playstate.LoopMode }

type TransmitterConnectedMsg struct{}
type TransmitterNotConnectedMsg struct{}
