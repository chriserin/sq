// Package config provides configuration management for the sequencer application.
// It handles loading and processing Lua configuration files that define templates,
// instruments, and various sequencer settings including accents, control changes,
// gate lengths, and line actions.
package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aarzilli/golua/lua"
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/chriserin/sq/internal/grid"
	"github.com/chriserin/sq/internal/operation"
)

type Config struct {
	Accents     []Accent
	C1          int
	LineActions map[grid.Action]lineaction
}

//go:embed init.lua
var initialConfig string

type Accent uint8

var Accents = []Accent{
	0,
	120,
	105,
	90,
	75,
	60,
	45,
	30,
	15,
}

const C1 = 36

type lineaction struct {
	Shape string
	Color color.Color
}

var Lineactions = map[grid.Action]lineaction{
	grid.ActionNothing:     {" ", lipgloss.Color("#000000")},
	grid.ActionLineReset:   {"↔", lipgloss.Color("#cf142b")},
	grid.ActionLineReverse: {"←", lipgloss.Color("#f8730e")},
	// grid.ActionLineReverseAll:  {"←͚͒", lipgloss.Color("#f8730e")},
	grid.ActionLineSkipBeat:  {"⇒", lipgloss.Color("#a9e5bb")},
	grid.ActionLineResetAll:  {"⇚", lipgloss.Color("#fcf6b1")},
	grid.ActionLineBounce:    {"↨", lipgloss.Color("#fcf6b1")},
	grid.ActionLineBounceAll: {"↨͚͒", lipgloss.Color("#fcf6b1")},
	grid.ActionLineDelay:     {"ℤ", lipgloss.Color("#cc4bc2")},
	grid.ActionSpecificValue: {"V", lipgloss.Color("#cc4bc2")},
	grid.ActionTempoChange:   {"T", lipgloss.Color("#5cdffb")},
}

type ratchetDiacritical string

var Ratchets = []ratchetDiacritical{
	"",
	"\u0307",
	"\u030A",
	"\u030B",
	"\u030C",
	"\u0312",
	"\u0313",
	"\u0344",
}

type Gate struct {
	Shape string
	Value float32
}

var ShortGates = []Gate{
	{"", 20},
	{"\u032A", 0.125},
	{"\u032B", 0.250},
	{"\u032C", 0.375},
	{"\u032D", 0.5},
	{"\u032E", 0.625},
	{"\u032F", 0.750},
	{"\u0330", 0.875},
}

var LongGates = []Gate{}

type Wait float32

var WaitPercentages = []Wait{
	0,
	8,
	16,
	24,
	32,
	40,
	48,
	54,
}

type ControlChange struct {
	Value      uint8
	UpperLimit uint8
	Name       string
}

var StandardCCs = []ControlChange{
	{0, 127, "Bank Select"},
	{1, 127, "Modulation Wheel or Lever"},
	{2, 127, "Breath Controller"},
	{4, 127, "Foot Controller"},
	{5, 127, "Portamento Time"},
	{6, 127, "Data Entry MSB"},
	{7, 127, "Channel Volume"},
	{8, 127, "Balance"},
	{10, 127, "Pan"},
	{11, 127, "Expression Controller"},
	{12, 127, "Effect Control 1"},
	{13, 127, "Effect Control 2"},
	{16, 127, "General Purpose Controller 1"},
	{17, 127, "General Purpose Controller 2"},
	{18, 127, "General Purpose Controller 3"},
	{19, 127, "General Purpose Controller 4"},
}

func FindCC(value uint8, instrumentName string) (ControlChange, bool) {
	instrument := GetInstrument(instrumentName)
	if len(instrument.CCs) == 0 {
		for _, cc := range StandardCCs {
			if cc.Value == value {
				return cc, true
			}
		}
	} else {
		for _, cc := range instrument.CCs {
			if cc.Value == value {
				return cc, true
			}
		}
	}
	return ControlChange{}, false
}

func GetInstrumentForOutput(outputName string) (Instrument, bool) {
	for _, instrument := range instruments {
		if instrument.Output != "" && instrument.Output == outputName {
			return instrument, true
		}
	}
	return Instrument{}, false
}

func CCsForOutput(outputName string, fallbackInstrument string) []ControlChange {
	if inst, ok := GetInstrumentForOutput(outputName); ok {
		return inst.CCs
	}
	instrument := GetInstrument(fallbackInstrument)
	if len(instrument.CCs) > 0 {
		return instrument.CCs
	}
	return StandardCCs
}

func FindCCForOutput(value uint8, outputName string, fallbackInstrument string) (ControlChange, bool) {
	for _, cc := range CCsForOutput(outputName, fallbackInstrument) {
		if cc.Value == value {
			return cc, true
		}
	}
	return ControlChange{}, false
}

func NearestCCForOutput(value uint8, outputName string, fallbackInstrument string) (ControlChange, bool) {
	ccs := CCsForOutput(outputName, fallbackInstrument)
	if len(ccs) == 0 {
		return ControlChange{}, false
	}
	best := ccs[0]
	bestDiff := absDiff(value, best.Value)
	for _, cc := range ccs[1:] {
		if diff := absDiff(value, cc.Value); diff < bestDiff {
			best, bestDiff = cc, diff
		}
	}
	return best, true
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

type Template struct {
	Name          string
	Lines         []grid.LineDefinition
	UIStyle       string
	MaxGateLength int
	SequencerType operation.SequencerMode
}

func InitTemplate(
	name string,
	uIStyle string,
	maxGateLength int,
	sequencerType string,

) Template {
	var seqType operation.SequencerMode

	switch sequencerType {
	case "trigger":
		seqType = operation.SeqModeLine
	case "polyphony":
		seqType = operation.SeqModeChord
	}
	return Template{Name: name, UIStyle: uIStyle, MaxGateLength: maxGateLength, SequencerType: seqType}
}

func GetGateLengths(maxGateLength int) []Gate {
	gateMarkers := []float32{0.0, 0.125, 0.25, 0.375, 0.5, 0.625, 0.75, 0.875}
	chars := []string{"\u258F", "\u258E", "\u258D", "\u258C", "\u258B", "\u258A", "\u2589", "\u2588"}
	result := make([]Gate, 0, maxGateLength*8)

	for i := range maxGateLength {
		if i > 0 {
			for j, v := range gateMarkers {
				newGate := Gate{
					Shape: strings.Repeat("\u2588", i-1) + chars[j],
					Value: float32(i) + v,
				}
				result = append(result, newGate)
			}
		}
	}
	return result
}

var templates []Template

func GetTemplate(name string) (Template, bool) {
	for _, template := range templates {
		if template.Name == name {
			return template, true
		}
	}
	return Template{}, false
}

func GetTemplateNames() []string {
	names := make([]string, len(templates))
	for i, template := range templates {
		names[i] = template.Name
	}
	return names
}

func GetDefaultTemplate() Template {
	defaultTemplate := InitTemplate("DEFAULT", "plain", 32, "trigger")
	initNote := uint8(60)
	defaultTemplate.Lines = []grid.LineDefinition{
		{Channel: 10, MsgType: grid.MessageTypeNote, Note: initNote},
		{Channel: 10, MsgType: grid.MessageTypeNote, Note: initNote + 1},
		{Channel: 10, MsgType: grid.MessageTypeNote, Note: initNote + 2},
		{Channel: 10, MsgType: grid.MessageTypeNote, Note: initNote + 3},
		{Channel: 10, MsgType: grid.MessageTypeNote, Note: initNote + 4},
		{Channel: 10, MsgType: grid.MessageTypeNote, Note: initNote + 5},
		{Channel: 10, MsgType: grid.MessageTypeNote, Note: initNote + 6},
		{Channel: 10, MsgType: grid.MessageTypeNote, Note: initNote + 7},
	}
	return defaultTemplate
}

type Instrument struct {
	Name   string
	Output string // optional; if set, this CC set applies to lines whose MidiOutput matches this name instead of the sequence-wide Instrument
	CCs    []ControlChange
}

var instruments []Instrument

func GetInstrument(name string) Instrument {
	for _, instrument := range instruments {
		if instrument.Name == name {
			return instrument
		}
	}
	return Instrument{}
}

func GetInstrumentNames() []string {
	names := make([]string, len(instruments))
	for i, instrument := range instruments {
		names[i] = instrument.Name
	}
	return names
}

func Init() {
	configFilePath, exists := findConfigFile()

	if exists {
		ProcessConfig(configFilePath)
	}
}

func ProcessConfig(configFilePath string) {
	L := lua.NewState()
	defer L.Close()

	L.OpenPackage()
	L.OpenLibs()
	L.RegisterLibrary("sq", seqFunctions)

	// Set CONFIG_DIR global for Lua scripts
	configDir := filepath.Dir(configFilePath)
	L.PushString(configDir)
	L.SetGlobal("CONFIG_DIR")

	if fileExists(configFilePath) {
		err := L.DoFile(configFilePath)
		if err != nil {
			fmt.Println(err.Error())
			panic("Do File error!!")
		}
	}
}

func findConfigFile() (string, bool) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("Could not find home directory")
	}

	filename := "init.lua"
	xdgConfigDir := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigDir == "" {
		xdgConfigDir = homeDir + "/.config"
	}

	possibleDirs := []string{
		"./",
		"./config",
		homeDir + "/.sq",
		xdgConfigDir + "/sq",
	}

	for _, dir := range possibleDirs {
		filePath := dir + "/" + filename
		if fileExists(filePath) {
			return filePath, true
		}
	}

	writePath := xdgConfigDir + "/sq"
	if _, err := os.Stat(writePath); os.IsNotExist(err) {
		err := os.Mkdir(writePath, 0755)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Could not create config directory:", writePath)
			return "", false
		}
	}

	configFilePath := writePath + "/" + filename
	if !fileExists(configFilePath) {
		err := os.WriteFile(configFilePath, []byte(initialConfig), 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Could not write initial config:", writePath)
			return "", false
		}
		return configFilePath, true
	}

	return "", false
}

func fileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// Lua Function
func addInstrument(L *lua.State) int {
	if L.IsTable(1) {
		L.GetField(1, "name")
		name := L.ToString(2)
		L.Pop(1)
		L.GetField(1, "output")
		output := L.ToString(2)
		L.Pop(1)
		instrument := Instrument{Name: name, Output: output}

		L.GetField(1, "controlchanges")
		if L.IsTable(2) {

			for i := 1; true; i++ {
				L.PushInteger(int64(i))
				L.GetTable(2)
				if L.IsTable(3) {
					cc := ControlChange{}
					for i := range 3 {
						L.PushInteger(int64(i + 1))
						L.GetTable(3)
						switch i + 1 {
						case 1:
							value := L.ToNumber(4)
							cc.Value = uint8(value)
						case 2:
							upperLimit := L.ToNumber(4)
							cc.UpperLimit = uint8(upperLimit)
						case 3:
							name := L.ToString(4)
							cc.Name = name
						}
						L.Pop(1)
					}
					instrument.CCs = append(instrument.CCs, cc)
				} else {
					break
				}
				L.Pop(1)
			}
		}
		instruments = append(instruments, instrument)
	} else {
		panic("Instrument not formatted correctly")
		// Communicate Error
	}
	return 0
}

// Lua Function
func addTemplate(L *lua.State) int {
	if L.IsTable(1) {
		L.GetField(1, "name")
		name := L.ToString(2)
		L.Pop(1)
		L.GetField(1, "uistyle")
		uistyle := L.ToString(2)
		if uistyle == "" {
			uistyle = "plain"
		}
		L.Pop(1)
		L.GetField(1, "seqtype")
		seqtype := L.ToString(2)
		if seqtype == "" {
			seqtype = "trigger"
		}
		L.Pop(1)
		L.GetField(1, "maxgatelength")
		maxGateLength := L.ToInteger(2)
		if maxGateLength == 0 {
			maxGateLength = 1
		}
		L.Pop(1)

		template := InitTemplate(name, uistyle, maxGateLength, seqtype)

		L.GetField(1, "lines")
		if L.IsTable(2) {

			for i := 1; true; i++ {
				L.PushInteger(int64(i))
				L.GetTable(2)
				if L.IsTable(3) {
					ld := grid.LineDefinition{}
					for i := range 5 {
						L.PushInteger(int64(i + 1))
						L.GetTable(3)
						switch i + 1 {
						case 1:
							channel := L.ToNumber(4)
							ld.Channel = uint8(channel)
						case 2:
							messageType := L.ToString(4)
							switch messageType {
							case "NOTE":
								ld.MsgType = grid.MessageTypeNote
							case "CC":
								ld.MsgType = grid.MessageTypeCc
							case "PC":
								ld.MsgType = grid.MessageTypeCc
							}
						case 3:
							note := L.ToNumber(4)
							ld.Note = uint8(note)
						case 4:
							name := L.ToString(4)
							ld.Name = name
						case 5:
							ld.MidiOutput = L.ToString(4)
						}
						L.Pop(1)
					}
					template.Lines = append(template.Lines, ld)
				} else {
					break
				}
				L.Pop(1)
			}
		}
		templates = append(templates, template)
	} else {
		panic("Template not formatted correctly")
		// Communicate Error
	}
	return 0
}

var ClockGateDeviceName string

type ClockGateMapping struct {
	Subdivision uint8 // 1-8, how many times per beat this mapping fires
	Channel     uint8 // 1-indexed, same convention as grid.LineDefinition.Channel
	Note        uint8
}

var ClockGateMappings []ClockGateMapping

// Lua Function
func setClockGates(L *lua.State) int {
	if L.IsTable(1) {
		L.GetField(1, "device")
		ClockGateDeviceName = L.ToString(2)
		L.Pop(1)

		L.GetField(1, "gates")
		if L.IsTable(2) {
			for i := 1; true; i++ {
				L.PushInteger(int64(i))
				L.GetTable(2)
				if L.IsTable(3) {
					L.GetField(3, "subdivision")
					subdivision := uint8(L.ToNumber(4))
					L.Pop(1)
					L.GetField(3, "channel")
					channel := uint8(L.ToNumber(4))
					L.Pop(1)
					L.GetField(3, "note")
					note := uint8(L.ToNumber(4))
					L.Pop(1)

					if subdivision == 0 || subdivision > 8 {
						panic("Clock gate subdivision must be between 1 and 8")
					}
					ClockGateMappings = append(ClockGateMappings, ClockGateMapping{
						Subdivision: subdivision,
						Channel:     channel,
						Note:        note,
					})
				} else {
					break
				}
				L.Pop(1)
			}
		}
		L.Pop(1)
	} else {
		panic("Clock gates not formatted correctly")
	}
	return 0
}

var ResetDeviceName string

type ResetMapping struct {
	Channel uint8
	Note    uint8
}

var (
	SongResetMapping       ResetMapping
	PartStartResetMapping  ResetMapping
	PartLoopResetMapping   ResetMapping
	GroupStartResetMapping ResetMapping
	GroupLoopResetMapping  ResetMapping
)

// ResetRange describes a contiguous span of MIDI notes on one channel, used
// to assign individually-distinguishable resets to parts/groups (see
// resets-pool.md) rather than sharing a single note across every part or
// group.
type ResetRange struct {
	Channel   uint8
	StartNote uint8
	EndNote   uint8
}

var (
	StartResetRange ResetRange
	LoopResetRange  ResetRange
)

func readResetMapping(L *lua.State, field string) ResetMapping {
	L.GetField(1, field)
	var mapping ResetMapping
	if L.IsTable(2) {
		L.GetField(2, "channel")
		mapping.Channel = uint8(L.ToNumber(3))
		L.Pop(1)
		L.GetField(2, "note")
		mapping.Note = uint8(L.ToNumber(3))
		L.Pop(1)
	}
	L.Pop(1)
	return mapping
}

func readResetRange(L *lua.State, field string) ResetRange {
	L.GetField(1, field)
	var r ResetRange
	if L.IsTable(2) {
		L.GetField(2, "channel")
		r.Channel = uint8(L.ToNumber(3))
		L.Pop(1)
		L.GetField(2, "startnote")
		startNote := L.ToNumber(3)
		L.Pop(1)
		L.GetField(2, "endnote")
		endNote := L.ToNumber(3)
		L.Pop(1)

		// Checked against the raw Lua number, not uint8(startNote)/uint8(endNote):
		// a value like 300 would silently wrap around (300 % 256 = 44) if cast
		// before validating, passing as a bogus but "valid-looking" uint8.
		if startNote < 0 || startNote > 127 {
			panic(fmt.Sprintf("resets %q startnote must be between 0 and 127", field))
		}
		if endNote < 0 || endNote > 127 {
			panic(fmt.Sprintf("resets %q endnote must be between 0 and 127", field))
		}
		r.StartNote = uint8(startNote)
		r.EndNote = uint8(endNote)

		if r.EndNote < r.StartNote {
			panic(fmt.Sprintf("resets %q endnote must be >= startnote", field))
		}
	}
	L.Pop(1)
	return r
}

// Lua Function
func setResets(L *lua.State) int {
	if L.IsTable(1) {
		L.GetField(1, "device")
		ResetDeviceName = L.ToString(2)
		L.Pop(1)

		SongResetMapping = readResetMapping(L, "song")
		PartStartResetMapping = readResetMapping(L, "partStart")
		PartLoopResetMapping = readResetMapping(L, "partLoop")
		GroupStartResetMapping = readResetMapping(L, "groupStart")
		GroupLoopResetMapping = readResetMapping(L, "groupLoop")
		StartResetRange = readResetRange(L, "starts")
		LoopResetRange = readResetRange(L, "loops")
	} else {
		panic("Resets not formatted correctly")
	}
	return 0
}

// ValidateClockGateResetOverlap returns an error if any configured reset
// mapping shares a channel/note with a configured clock-gate mapping on the
// same device. SendClockGate and SendReset (internal/seqmidi) have no shared
// note-identity bookkeeping between them — each independently fires its own
// NoteOn and schedules its own NoteOff on its own timer — so a channel/note
// collision produces two uncoordinated NoteOn/NoteOff pairs racing on the
// same note instead of a well-defined signal (one can cut the other's pulse
// short, or the two overlap into an ambiguous double-trigger).
//
// Only checked when both are pointed at the same device name — if they're
// configured for different devices there's no actual collision, since they
// go out on different physical ports.
//
// Called once at startup, before any MIDI connection or the TUI exist, so a
// misconfiguration is reported as a plain CLI message and the process exits
// before ever reaching the TUI, rather than surfacing as a confusing runtime
// MIDI glitch during playback.
type noteKey struct {
	channel uint8
	note    uint8
}

func ValidateClockGateResetOverlap() error {
	if ClockGateDeviceName == "" || ResetDeviceName == "" || ClockGateDeviceName != ResetDeviceName {
		return nil
	}

	gateKeys := make(map[noteKey]uint8, len(ClockGateMappings)) // value: subdivision, for the error message
	for _, m := range ClockGateMappings {
		gateKeys[noteKey{m.Channel, m.Note}] = m.Subdivision
	}

	resetKinds := []struct {
		name    string
		mapping ResetMapping
	}{
		{"song", SongResetMapping},
		{"partStart", PartStartResetMapping},
		{"partLoop", PartLoopResetMapping},
		{"groupStart", GroupStartResetMapping},
		{"groupLoop", GroupLoopResetMapping},
	}

	for _, rk := range resetKinds {
		if rk.mapping.Channel == 0 {
			continue // unconfigured
		}
		if subdivision, collides := gateKeys[noteKey{rk.mapping.Channel, rk.mapping.Note}]; collides {
			return fmt.Errorf(
				"config error: clock gate (subdivision %d) and reset %q both target channel %d, note %d on device %q — "+
					"give them different notes so their NoteOn/NoteOff pairs don't collide",
				subdivision, rk.name, rk.mapping.Channel, rk.mapping.Note, ClockGateDeviceName,
			)
		}
	}

	if err := validateResetRangeOverlap("starts", StartResetRange, gateKeys, ClockGateDeviceName); err != nil {
		return err
	}
	if err := validateResetRangeOverlap("loops", LoopResetRange, gateKeys, ClockGateDeviceName); err != nil {
		return err
	}

	return nil
}

// validateResetRangeOverlap checks every note in r (if configured) against
// gateKeys, mirroring the scalar-mapping check above one note at a time.
func validateResetRangeOverlap(name string, r ResetRange, gateKeys map[noteKey]uint8, device string) error {
	if r.Channel == 0 {
		return nil
	}
	for note := r.StartNote; note <= r.EndNote; note++ {
		if subdivision, collides := gateKeys[noteKey{r.Channel, note}]; collides {
			return fmt.Errorf(
				"config error: clock gate (subdivision %d) and reset range %q both target channel %d, note %d on device %q — "+
					"give them different notes so their NoteOn/NoteOff pairs don't collide",
				subdivision, name, r.Channel, note, device,
			)
		}
	}
	return nil
}

type LuaFn = lua.LuaGoFunction

var seqFunctions = map[string]LuaFn{
	"addtemplate":   addTemplate,
	"addinstrument": addInstrument,
	"setclockgates": setClockGates,
	"setresets":     setResets,
}
