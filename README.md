[![Tests](https://github.com/chriserin/sq/actions/workflows/test.yml/badge.svg)](https://github.com/chriserin/sq/actions/workflows/test.yml)

# sq

![Beta](beta.png)

A powerful MIDI sequencer designed for the command line

## Overview

sq is a terminal-based MIDI sequencer that brings midi sequencer capabilities to your CLI. Built with Go and designed for efficiency, sq offers rapid beat creation, complex arrangements, and advanced pattern manipulation through an intuitive keyboard-driven interface.

## Key Features

- **MIDI Integration**: Control hardware or software instruments via MIDI
- **Rapid Beat Creation**: Create drum patterns in just a few keystrokes using pattern mode
- **Euclidean Rhythms**: Generate mathematically-distributed note patterns for complex polyrhythms
- **Advanced Overlays**: Add mathematical variations to sequences with the overlay system
- **Flexible Arrangements**: Structure songs with sections, parts and groups
- **Real-time Manipulation**: Modify patterns, accents, gates and timing while playing
- **Vim-inspired**: Familiar key bindings for efficient workflow

## Install from package

Pre-built packages for macOS and Linux are found on the [Releases](https://github.com/chriserin/sq/releases) page.

## Install with mise

```
mise install ubi:chriserin/sq
```

## Quick Start

1. **Launch sq**: `sq`
2. **Create a beat**: Move cursor with `hjkl`, press `1` for notes on every beat
3. **Play**: Press `Space` to play/stop
4. **Save**: `Ctrl+s` to save your sequence

### Connecting to hardware or software

Midi inports and outports can be confusing. One software's inport is another
software's outport and vice versa. When passing the `--outport` flag to `sq`
you create an outport for sq that will be an inport for the other software.
Both Logic and GarageBand scan the inports and automatically listen to all
inports. Likewise, Both Logic and GarageBand create an inport that sq will
automatically connect to if it is the first outport listed by `sq list`. When
the outport is not listed first, you can connect to that outport with the `--midiout
<partial name>` flag. `--midiout Logic` will connect to Logic's outport.

### Basic Beat Creation Example

Create a basic beat with just 6 keystrokes:

- Bass drum: cursor on BD line, press `8` (note every 8 beats)
- Snare: `j` (down), `4` then `8` (note every 8 beats starting at beat 5)
- Hi-hat: `j` (down), `1` (note every beat)

The resulting sequence will look like this:

```
BDK┤▧       ▧       ▧       ▧
SN │    ▧       ▧       ▧       ▧
H1 │▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧▧
```

## Documentation

- [Core Concepts](docs/core-concepts.md) - A summary of the important concepts
- **[Beat Creation](docs/beat-creation.md)** — Pattern mode and rapid beat creation techniques
- **[Note Alteration](docs/note-alteration.md)** — Accent, gate, ratchet and wait controls
- **[Key Cycles](docs/key-cycles.md)** — Understanding sequence iteration and timing
- **[Arrangement System](docs/arrangement.md)** — Sections, parts, groups and song structure
- **[Overlay System](docs/overlay-key.md)** — Mathematical variations and overlay keys
- **[Actions](docs/actions.md)** — Playback cursor manipulation and sequencer actions
- **[Key Mappings](docs/key-mappings.md)** — Complete keyboard shortcut reference
- **[Clock Gates and Resets](docs/clock-gates-and-resets.md)** — Send per-subdivision MIDI gates and song/part/group reset triggers to drive Eurorack clocks and dividers

## Configuration

sq uses Lua for configuration. Configuration files can be located in 4 different locations:

- `./`
- `./config/`
- `~/.sq/`
- `~/.config/sq/`

If a configuration file is not found, an initial configuration is written to `~/.confing/init.lua`.

## Development

### Build from Source

#### Dependencies MACOS

```
# Lua 5.4
brew install lua@5.4-dev
```

#### Dependencies Linux

```
sudo apt-get install liblua5.4-dev
sudo apt-get install libasound2-dev
```

```bash
# Clone the repository
git clone https://github.com/chriserin/sq.git
cd sq

# Build with Lua support
go build -tags lua54 -o sq

# Run
./sq
```

#### Building

```bash

# All commands must use -tags lua54
go build -tags lua54 -o sq
go test -tags lua54 ./...
go fmt ./...

# Consider exporting a GOFLAGS variable to avoid including `-tags lua54` for every command
export GOFLAGS="-tags=lua54"
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes following Go conventions
4. Add tests for new features
5. Submit a pull request

### Code Style

- Use Go 1.25+ features
- Follow standard Go naming conventions
- Use `stretchr/testify/assert` for tests
- Run `go fmt` before committing

## License

MIT License in `LICENSE`

## Support

For issues and feature requests, please use the GitHub issue tracker.
