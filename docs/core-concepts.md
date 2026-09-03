# Core Concepts

## Pattern Creation

**Pattern Mode** enables rapid beat creation:

- **Numbers (1-9)**: Add/remove notes every N beats
- **Pattern Mode - Value**: `na` (accent), `nw` (wait), `ng` (gate), `nr` (ratchet)
- **Shift+Numbers**: Increase values, Numbers alone decrease values

> **Detailed Guide**: [Beat Creation](docs/beat-creation.md)

## Note Attributes

Each note has four modifiable properties:

- **Accent** (`A`/`a`): MIDI velocity (8 levels)
- **Gate** (`G`/`g`): Note duration (8 levels + long gates with `E`/`e`)
- **Ratchet** (`R`/`r`): Multiple hits per beat (8 levels)
- **Wait** (`W`/`w`): Swing timing delay (8 levels)

> **Detailed Guide**: [Note Alteration](docs/note-alteration.md)

## Key Cycles and Arrangements

**Key Cycles** track sequence iterations and drive both arrangements and overlays:

- Key line (marked with `K`) determines when cycles advance
- Arrangements use key cycles to switch between sections
- Overlays use key cycles to determine when variations apply

**Arrangements** structure your songs:

- **Sections**: Containers that reference parts with playback attributes
- **Parts**: The actual musical content
- **Groups**: Collections of sections that can repeat

> **Detailed Guides**: [Key Cycles](docs/key-cycles.md) | [Arrangement System](docs/arrangement.md)

## Overlay System

**Overlays** add mathematical variations to patterns:

- `1/1`: Root overlay
- `2/1`: Every 2nd cycle
- `3/1`: Every 3rd cycle
- `1/4`, `2/4`, `3/4`, `4/4`: First, second, third, fourth of every 4 cycles

Complex overlay keys support width, start delays, and stacking behaviors.

> **Detailed Guide**: [Overlay System](docs/overlay-key.md)

## Actions

**Actions** manipulate playback cursors:

- **Line Reset** (`ss`): Reset cursor to first beat
- **Line Bounce** (`sb`): Bounce between first beat and action
- **Line Skip** (`sk`): Skip current beat
- **Line Reverse** (`sr`): Reverse playback direction
- **Line Delay** (`sz`): Repeat current beat

Some actions have "All" variants (`sS`, `sB`, `sK`) that affect all playback cursors.

A separate pair of **value actions** carry a number instead of manipulating
the cursor: **Specific Value** (`bv`) sends an exact CC/PC value, and
**Tempo Change** (`bT`) changes the playing tempo when reached.

> **Detailed Guide**: [Actions](docs/actions.md)
