# Clock Gates and Resets

sq's timing engine can send two kinds of MIDI gates (NoteOn/NoteOff pairs)
that are independent of anything in your sequence, both aimed at driving
external hardware — most notably a Eurorack setup via a MIDI-to-CV/gate
interface:

- **[Clock gates](#clock-gates)** — a running pulse at a chosen rate, to
  drive an LFO or clock divider continuously.
- **[Resets](#resets)** — a one-off trigger fired whenever a song, part, or
  group starts playing, to snap that LFO or divider back to a known phase so
  every replay of the same arrangement looks and sounds the same.

The two are meant to work together: a clock divider free-running off a clock
gate will drift out of phase with the music after the first loop, since
nothing else re-syncs it — that's what resets are for. Because both can be
aimed at the same Eurorack module, and the module's behavior can depend on
which one it sees first when they coincide, sq guarantees **resets always
reach the device before any clock gate due at the same instant**.

## Clock Gates

sq's timing engine can send MIDI gates that pulse in sync with the
sequencer's clock. Each gate fires at a chosen subdivision of the beat — once
per beat, twice per beat, and so on, up to eight times per beat.

### Configuration

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

Add this to your `init.lua` (see [Configuration](../README.md#configuration)).

### Behavior

- Each gate's NoteOff is scheduled automatically, half a beat-subdivision
  after its NoteOn (a fixed 50% duty cycle).
- Gates fire during pre-roll count-in as well as normal playback.
- Gates aren't tied to the sequence's own `Subdivisions` setting or the
  `transmitting` toggle — they fire purely off sq's master clock, any time
  it's playing.

## Resets

A reset is a short MIDI trigger (NoteOn/NoteOff, a fixed 10ms hold) sent on
its own configurable channel/note whenever playback starts a **song**, a
**part**, or a **group**. Point it at the same reset input of a clock divider or
LFO, and every replay of your arrangement starts from the
same known position.

### Configuration

Resets are configured in Lua with a single `sq.setresets` call:

```lua
sq.setresets({
	device = "MIDI-to-CV",
	song       = { channel = 1, note = 9 },
	partStart  = { channel = 1, note = 10 },
	partLoop   = { channel = 1, note = 11 },
	groupStart = { channel = 1, note = 12 },
	groupLoop  = { channel = 1, note = 13 },
})
```

- **`device`** — matched against `--midiout` exactly like clock gates, and
  the same "must match twice" requirement applies (see above) — nothing
  fires unless both agree on the same device.
- **`song`** — fires every time the whole song starts, including the very first
  Play press and every loop-around when looping the whole sequence.
- **`partStart`** — fires whenever a part is being freshly entered. This happens
  when a group or sequence is looped as well.
- **`partLoop`** — the broader signal: fires every time `partStart` fires
  (including the very first Play press), _plus_ on a beat where the part
  repeats itself in place via its own Cycles setting with nothing above it
  (its group, or the sequence) also starting or looping. `partStart` never
  fires without `partLoop` firing alongside it.
- **`groupStart`** — fires whenever a group is being freshly entered. This happens
  when a group above it or sequence is looped as well.
- **`groupLoop`** — the same relationship one level up: fires every time
  `groupStart` fires, _plus_ on a beat where the group repeats itself in
  place via its own Cycles setting with nothing above it (its parent group,
  or the sequence) also starting or looping.

Add any combination of the above to your configuration.

### Individually-distinguishable part/group resets

The five mappings above are shared — every part fires the same `partStart`
note, every group fires the same `groupStart` note.

`sq` can also send start and loop events for individual parts and groups by
establishing a pool of notes, from which a group or part can be assigned a
note.

```lua
sq.setresets({
	device = "MIDI-to-CV",
	-- ... song/partStart/partLoop/groupStart/groupLoop as above ...
	starts = { channel = 1, startnote = 20, endnote = 27 },
	loops  = { channel = 1, startnote = 30, endnote = 37 },
})
```

- **`starts`** — fires an additional note, specific to the part or group,
  alongside every `partStart`/`groupStart` event.
- **`loops`** — the same, alongside every `partLoop`/`groupLoop` event.
- Both are a single contiguous range of notes on one channel (`channel`,
  `startnote`, `endnote`), and are **shared between parts and groups** —
  not a separate range for each.
- Assignment is sticky and deterministic: the first time a part or group is
  encountered at start time, it claims the next note in the range and
  keeps it for the rest of the session. The same named part reused in multiple
  places in the arrangement always claims the same note, since it's really the
  same part playing again — a group node, since groups don't have a reusable
  identity the way named parts do, is scoped to that specific place in the
  arrangement.
- If there are more distinct parts/groups than notes in the range, the
  range wraps around — a part and an unrelated group can end up sharing an
  output once the range is exhausted. This is expected, not an error: it
  just means that one output now represents "a start happened here",
  shared by whichever entities wrapped onto it.
- When two or more nested groups start on the same beat, each one still
  gets its own individual `starts`/`loops` note — but the shared
  `groupStart`/`groupLoop` mapping above only ever fires once per beat,
  regardless of how many levels started together.

### Behavior

- Resets are guaranteed to be sent before any clock gate
- Playing a single part or overlay in isolation (`ctrl+space`, `'` `space`)
  doesn't fire a song started reset event, since those don't move the cursor to
  the top of the song.
- Resets and Clocks configured to the same device+channel+note is disallowed at startup
- The same collision check applies to `starts`/`loops`: any note in either
  range that matches a configured clock gate on the same device is
  disallowed at startup, the same as the scalar mappings above.

## Requirement: `--midiout` must match your `device`

Clock gates and Resets only ever go out to a device you've selected twice — once in
config, and once at the command line:

```
sq --midiout MIDI-to-CV
```

The `--midiout` value and the `gates` `device` value both have to (partially)
match the same MIDI output port. If they don't — say you forgot the flag, or
`--midiout` points at a different port than `device` — no clock gates are
sent at all.
