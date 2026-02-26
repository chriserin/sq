package overlaykey

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/chriserin/sq/internal/themes"

	"charm.land/bubbles/v2/key"
)

type keymap struct {
	FocusWidth    key.Binding
	FocusInterval key.Binding
	FocusShift    key.Binding
	FocusStart    key.Binding
	RemoveStart   key.Binding
	Increase      key.Binding
	Decrease      key.Binding
}

var keys = keymap{
	FocusWidth:    Key("Focus Width", ":"),
	FocusInterval: Key("Focus Interval", "/"),
	FocusShift:    Key("Focus Shift", "^"),
	FocusStart:    Key("Focus Start", "S"),
	RemoveStart:   Key("Remove Start", "s"),
	Increase:      Key("Increase", "+"),
	Decrease:      Key("Decrease", "-"),
}

func Key(help string, keyboardKey ...string) key.Binding {
	return key.NewBinding(key.WithKeys(keyboardKey...), key.WithHelp(keyboardKey[0], help))
}

type Model struct {
	overlayKey        OverlayPeriodicity
	focus             focus
	firstDigitApplied bool
}

func InitModel() Model {
	return Model{ROOT, FocusNothing, false}
}

func (m *Model) SetOverlayKey(op OverlayPeriodicity) {
	m.overlayKey = op
}

func (m Model) GetKey() OverlayPeriodicity {
	return m.overlayKey
}

func (m *Model) Focus(shouldFocus bool) {
	if shouldFocus {
		m.focus = FocusShift
	} else {
		m.focus = FocusNothing
		m.firstDigitApplied = false
	}
}

func (m *Model) Escape(key OverlayPeriodicity) {
	m.focus = FocusNothing
	m.firstDigitApplied = false
	m.overlayKey = key
}

type focus int

const (
	FocusNothing focus = iota
	FocusShift
	FocusWidth
	FocusInterval
	FocusStart
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case msg.String() >= "0" && msg.String() <= "9":
			numberString := msg.String()
			newDigit, _ := strconv.Atoi(numberString)
			m.ApplyDigit(newDigit)
			m.firstDigitApplied = true
			if m.focus == FocusStart && m.overlayKey.StartCycle == 0 {
				m.focus = FocusShift
			}
			if m.focus == FocusWidth && m.overlayKey.Width == 0 {
				m.focus = FocusShift
			}
		case key.Matches(msg, keys.FocusWidth):
			m.focus = FocusWidth
			if m.overlayKey.Width == 0 {
				m.overlayKey.Width = 1
			}
			m.firstDigitApplied = false
		case key.Matches(msg, keys.FocusInterval):
			m.focus = FocusInterval
			m.firstDigitApplied = false
		case key.Matches(msg, keys.FocusShift):
			m.focus = FocusShift
			m.firstDigitApplied = false
		case key.Matches(msg, keys.FocusStart):
			m.focus = FocusStart
			if m.overlayKey.StartCycle == 0 {
				m.overlayKey.StartCycle = 1
			}
			m.firstDigitApplied = false
		case key.Matches(msg, keys.RemoveStart):
			m.focus = FocusShift
			m.overlayKey.StartCycle = 0
			m.firstDigitApplied = false
		case key.Matches(msg, keys.Increase):
			switch m.focus {
			case FocusShift:
				m.overlayKey.IncrementShift()
			case FocusInterval:
				m.overlayKey.IncrementInterval()
			case FocusWidth:
				m.overlayKey.IncrementWidth()
			case FocusStart:
				m.overlayKey.IncrementStartCycle()
			}
		case key.Matches(msg, keys.Decrease):
			switch m.focus {
			case FocusShift:
				m.overlayKey.DecrementShift()
			case FocusInterval:
				m.overlayKey.DecrementInterval()
			case FocusWidth:
				m.overlayKey.DecrementWidth()
			case FocusStart:
				m.overlayKey.DecrementStartCycle()
			}
		}
	}
	return m, Updated(m.overlayKey, true)
}

func (m *Model) ApplyDigit(newDigit int) {
	switch m.focus {
	case FocusShift:
		m.overlayKey.Shift = m.UnshiftDigit(m.overlayKey.Shift, newDigit)
		if m.overlayKey.Shift == 0 {
			m.overlayKey.Shift = 1
		}
	case FocusInterval:
		m.overlayKey.Interval = m.UnshiftDigit(m.overlayKey.Interval, newDigit)
		if m.overlayKey.Interval == 0 {
			m.overlayKey.Interval = 1
		}
	case FocusWidth:
		m.overlayKey.Width = m.UnshiftDigit(m.overlayKey.Width, newDigit)
	case FocusStart:
		m.overlayKey.StartCycle = m.UnshiftDigit(m.overlayKey.StartCycle, newDigit)
	}
}

func (m Model) UnshiftDigit(digits uint8, newDigit int) uint8 {
	if m.firstDigitApplied {
		return uint8((int(digits)%10)*10 + newDigit)
	} else {
		return uint8(newDigit)
	}
}

type UpdatedOverlayKey struct {
	OverlayKey OverlayPeriodicity
	HasFocus   bool
}

func Updated(overlayKey OverlayPeriodicity, maintainsFocus bool) tea.Cmd {
	return func() tea.Msg {
		return UpdatedOverlayKey{
			OverlayKey: overlayKey,
			HasFocus:   maintainsFocus,
		}
	}
}

func View(ok OverlayPeriodicity) string {
	var shift, interval, width, start string
	var buf strings.Builder

	shift = NormalColor(ok.Shift)
	interval = NormalColor(ok.Interval)
	width = NormalColor(ok.Width)
	start = NormalColor(ok.StartCycle)

	buf.WriteString(shift)
	if ok.Width > 1 {
		buf.WriteString(":")
		buf.WriteString(width)
	}
	buf.WriteString("/")
	buf.WriteString(interval)
	if ok.StartCycle > 0 {
		buf.WriteString("S")
		buf.WriteString(start)
	}
	return buf.String()
}

func (m Model) ViewOverlay() string {
	var shift, interval, width, start string
	var buf strings.Builder

	shift = NumberColor(m.overlayKey.Shift)
	interval = NumberColor(m.overlayKey.Interval)
	width = NumberColor(m.overlayKey.Width)
	start = NumberColor(m.overlayKey.StartCycle)

	switch m.focus {
	case FocusShift:
		shift = SelectedColor(m.overlayKey.Shift)
	case FocusWidth:
		width = SelectedColor(m.overlayKey.Width)
	case FocusInterval:
		interval = SelectedColor(m.overlayKey.Interval)
	case FocusStart:
		start = SelectedColor(m.overlayKey.StartCycle)
	}

	buf.WriteString(shift)
	if m.overlayKey.Width > 1 || m.focus == FocusWidth {
		buf.WriteString(":")
		buf.WriteString(width)
	}
	buf.WriteString("/")
	buf.WriteString(interval)
	if m.overlayKey.StartCycle > 0 {
		buf.WriteString("S")
		buf.WriteString(start)
	}
	return buf.String()
}

func NumberColor(number uint8) string {
	return themes.NumberStyle.Render(strconv.Itoa(int(number)))
}

func SelectedColor(number uint8) string {
	return themes.SelectedStyle.Render(strconv.Itoa(int(number)))
}

func NormalColor(number uint8) string {
	return strconv.Itoa(int(number))
}
