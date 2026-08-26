package seqmidi

import (
	"fmt"
	"os"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/chriserin/sq/internal/config"
	"gitlab.com/gomidi/midi/v2/drivers"
)

type OutDeviceInfo struct {
	IsOpen      bool
	Selected    bool
	IsDaw       bool
	IsClockGate bool
	IsReset     bool
	IsVirtual   bool // true only for the sq-managed virtual out
	Out         drivers.Out
	Name        string
	Type        string
}

type InDeviceInfo struct {
	IsOpen        bool
	IsTransmitter bool
	In            drivers.In
	Name          string
	Type          string
}

func (di OutDeviceInfo) Matches(name string) bool {
	return strings.Contains(di.Name, name)
}

func (di *OutDeviceInfo) Open() {
	if !di.IsOpen {
		err := di.Out.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening MIDI output %v: %v\n", di.Name, err)
			return
		}
		di.IsOpen = true
	}
}

func (di InDeviceInfo) Matches(name string) bool {
	return strings.Contains(di.Name, name)
}

func (di *InDeviceInfo) Open() {
	if !di.IsOpen {
		err := di.In.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening MIDI input %v: %v\n", di.Name, err)
			return
		}
		di.IsOpen = true
	}
}

type ReceiverFunc func(msg []byte, milliseconds int32)

func (mc *MidiConnection) ListenToTransmitter(recFunc ReceiverFunc) error {
	mc.ReceiverFunc = recFunc
	for _, device := range mc.inDevices {
		if device.Matches(TransmitterName) {
			if device.In != nil && device.In.IsOpen() {
				if mc.StopFn != nil {
					mc.StopFn()
					mc.StopFn = nil
				}
				stopFn, err := device.In.Listen(recFunc, drivers.ListenConfig{TimeCode: true, ActiveSense: true})
				if err != nil {
					return fault.Wrap(err, fmsg.With("cannot listen to transmitter"))
				}
				mc.StopFn = stopFn
			}
		}
	}
	return nil
}

func (mc *MidiConnection) UpdateOutDeviceList(driver drivers.Driver) error {
	newDevices := make(map[string]*OutDeviceInfo)

	outs, err := driver.Outs()
	if err != nil {
		return fmt.Errorf("failed to get output ports: %v", err)
	}

	for _, out := range outs {
		name := out.String()
		if name == "" {
			// "" is LineDefinition.MidiOutput's pre-backfill zero value —
			// never let a real device claim it, however unlikely that is in
			// practice.
			continue
		}
		if existing, ok := mc.outDevices[name]; ok {
			existing.Out = out
			existing.IsOpen = false
			if mc.outportName != "" && existing.Matches(mc.outportName) {
				existing.Open()
				if config.ClockGateDeviceName != "" && existing.Matches(config.ClockGateDeviceName) {
					existing.IsClockGate = true
				}
				if config.ResetDeviceName != "" && existing.Matches(config.ResetDeviceName) {
					existing.IsReset = true
				}
			} else if existing.IsDaw {
				existing.Open()
			}
			newDevices[name] = existing
			continue
		}

		newDevice := &OutDeviceInfo{Out: out, Name: name}
		if mc.outportName != "" && newDevice.Matches(mc.outportName) {
			newDevice.Open()
			newDevice.Selected = true
			if config.ClockGateDeviceName != "" && newDevice.Matches(config.ClockGateDeviceName) {
				newDevice.IsClockGate = true
			}
			if config.ResetDeviceName != "" && newDevice.Matches(config.ResetDeviceName) {
				newDevice.IsReset = true
			}
		}
		for _, dawName := range dawOutports {
			if strings.Contains(newDevice.Name, dawName) {
				newDevice.Open()
				newDevice.IsDaw = true
			}
		}
		newDevices[name] = newDevice
	}

	if mc.virtualOutDevice != nil {
		if mc.outportName != "" && mc.virtualOutDevice.Matches(mc.outportName) {
			mc.virtualOutDevice.Selected = true
		}
		newDevices[mc.virtualOutDevice.Name] = mc.virtualOutDevice
	}

	mc.outDevices = newDevices
	mc.EnsureConnection()

	return nil
}

func SeqOut() (drivers.Out, error) {
	return OpenVirtualOut(OutputName)
}

func (mc *MidiConnection) TransmitterOut() (drivers.Out, error) {
	out, err := OpenVirtualOut(TransmitterName)
	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("cannot open virtual out"))
	}
	mc.IsTransmitter = true
	return out, nil
}
