package timing

import (
	"time"

	"github.com/chriserin/sq/internal/config"
	"github.com/chriserin/sq/internal/seqmidi"
	midi "gitlab.com/gomidi/midi/v2"
)

// emitClockGates fires a MIDI gate for every configured clock-gate mapping
// whose subdivision boundary falls on this pulse. Subdivisions with no
// mapping in config.ClockGateMappings are never checked, so they never fire.
func emitClockGates(mc *seqmidi.MidiConnection, pulseCount int, pulseInterval time.Duration) {
	for _, mapping := range config.ClockGateMappings {
		divisor := PPQN / int(mapping.Subdivision)
		if pulseCount%divisor == 0 {
			gateInterval := pulseInterval * time.Duration(divisor)
			sendClockGatePulse(mc, mapping.Channel-1, mapping.Note, gateInterval/2)
		}
	}
}

func sendClockGatePulse(mc *seqmidi.MidiConnection, channel, note uint8, holdDuration time.Duration) {
	if err := mc.SendClockGate(midi.NoteOn(channel, note, 100)); err != nil {
		return
	}
	time.AfterFunc(holdDuration, func() {
		mc.SendClockGate(midi.NoteOff(channel, note))
	})
}
