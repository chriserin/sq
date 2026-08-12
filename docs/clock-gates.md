# Clock Gates

sq's timing engine can send MIDI gates (NoteOn/NoteOff pairs) that pulse in
sync with the sequencer's clock, independent of anything in your sequence.
Each gate fires at a chosen subdivision of the beat — once per beat, twice per
beat, and so on, up to eight times per beat.

This exists to drive external hardware from sq's clock — for instance a
Eurorack setup via a MIDI-to-CV/gate interface, where each configured
channel/note becomes its own gate output that can trigger or sync an LFO,
envelope, or clock divider.

## Configuration

Clock gates are configured in Lua with a single `sq.setclockgates` call:

```lua
sq.setclockgates({
	device = "MIDI-to-CV",
	gates = {
		{ subdivision = 1, channel = 1, note = 1 },
		{ subdivision = 2, channel = 1, note = 2 },
		{ subdivision = 3, channel = 1, note = 3 },
		{ subdivision = 4, channel = 1, note = 4 },
		{ subdivision = 5, channel = 1, note = 5 },
		{ subdivision = 6, channel = 1, note = 6 },
		{ subdivision = 7, channel = 1, note = 7 },
		{ subdivision = 8, channel = 1, note = 8 },
	},
})
```

- **`device`** — a (partial) name of the MIDI output port the gates should go
  to, matched the same way as the `--midiout` flag.
- **`gates`** — a list of mappings, each with:
  - **`subdivision`** — how many times per beat this gate fires, `1` through
    `8`.
  - **`channel`** — the MIDI channel to send on, `1` through `16`.
  - **`note`** — the MIDI note number the gate uses for NoteOn/NoteOff.

Only subdivisions listed in `gates` ever produce MIDI traffic — there's no
default set of outputs. Add one entry per clock rate you actually want wired
to a physical gate output; leave the rest out. The same subdivision can
appear more than once with different channel/note pairs if you want one
clock rate driving two separate outputs.

Add this to your `init.lua` (see [Configuration](../README.md#configuration)
for where sq looks for it).

## Requirement: `--midiout` must match your `device`

Clock gates only ever go out to a device you've selected twice — once in
config, and once on the command line:

```
sq --midiout MIDI-to-CV
```

The `--midiout` value and the `gates` `device` value both have to (partially)
match the same MIDI output port. If they don't — say you forgot the flag, or
`--midiout` points at a different port than `device` — no clock gates are
sent at all, silently. This is intentional: it stops a `setclockgates` call
sitting in your config from firing at whatever device happens to match its
name, unless you've deliberately pointed sq's output at that device for this
run.

## Behavior

- Each gate's NoteOff is scheduled automatically, half a beat-subdivision
  after its NoteOn (a fixed 50% duty cycle).
- Gates fire during pre-roll count-in as well as normal playback.
- Gates aren't tied to the sequence's own `Subdivisions` setting or the
  `transmitting` toggle — they fire purely off sq's master clock, any time
  it's playing.
