package epd7in5v2

import (
	"fmt"
	"math"
	"strings"
)

func init() {
	var b strings.Builder
	first := true
	for voltage := range gateVoltageToFlagsMap {
		if !first {
			b.WriteString(", ")
		}
		b.WriteString(voltage.String())
		first = false
	}
	gateVoltageAllowedValuesMsgPart = b.String()
}

var gateVoltageAllowedValuesMsgPart string

type Externality byte

const (
	External Externality = 0
	Internal Externality = 1
)

type Speed byte

const (
	Slow Speed = 0
	Fast Speed = 1
)

type Voltage float32

func (v Voltage) String() string {
	return fmt.Sprintf("%fV", v)
}

const Volt Voltage = 1
const VoltageDrainHighMin = 2.4 * Volt
const VoltageDrainHighMax = 15 * Volt

const VoltageDrainHighFlagBits = 6
const VoltageDrainLowMin = -VoltageDrainHighMax
const VoltageDrainLowMax = -VoltageDrainHighMin

const VoltageDrainLowFlagBits = VoltageDrainHighFlagBits

type VoltageRange struct {
	High Voltage
	Low  Voltage
}

type SymmetricVoltageRange Voltage

func (sv SymmetricVoltageRange) High() Voltage {
	if sv < 0 {
		return Voltage(-sv)
	}
	return Voltage(sv)
}

func (sv SymmetricVoltageRange) Low() Voltage {
	if sv > 0 {
		return Voltage(-sv)
	}
	return Voltage(sv)
}

type PowerSettings struct {
	BorderLowDropoutEnabled bool
	SourceLowVoltagePower   Externality
	SourcePower             Externality
	GatePower               Externality
	OTPProgram              Externality
	CommonVoltageSlew       Speed
	GateVoltage             SymmetricVoltageRange
	BlackWhiteVoltageDrain  VoltageRange
}

func NewDefaultPowerSettings() PowerSettings {
	return PowerSettings{
		BorderLowDropoutEnabled: false,
		SourceLowVoltagePower:   Internal,
		SourcePower:             Internal,
		GatePower:               Internal,
		OTPProgram:              External,
		CommonVoltageSlew:       Slow,
		GateVoltage:             SymmetricVoltageRange(20 * Volt),
		BlackWhiteVoltageDrain: VoltageRange{
			High: 14 * Volt,
			Low:  -14 * Volt,
		},
	}
}

func (ps PowerSettings) Flags() ([]byte, error) {
	vg_lvl, err := ps.VG_LVL()
	if err != nil {
		return nil, fmt.Errorf("invalid gate voltage: %w", err)
	}
	vdh_lvl, err := ps.VDH_LVL()
	if err != nil {
		return nil, fmt.Errorf("invalid voltage drain high for black and white pixel: %w", err)
	}
	vdl_lvl, err := ps.VDL_LVL()
	if err != nil {
		return nil, fmt.Errorf("invalid voltage drain low for black and white pixel: %w", err)
	}
	return []byte{
		ps.BD_EN()<<4 | ps.VSR_EN()<<2 | ps.VS_EN()<<1 | ps.VG_EN(),
		ps.VPP_EN()<<7 | ps.VCOM_SLEW()<<4 | vg_lvl,
		vdh_lvl,
		vdl_lvl,
	}, nil
}

func (ps PowerSettings) BD_EN() byte {
	return boolToByte(ps.BorderLowDropoutEnabled)
}

func (ps PowerSettings) VSR_EN() byte {
	return byte(ps.SourceLowVoltagePower)
}

func (ps PowerSettings) VS_EN() byte {
	return byte(ps.SourcePower)
}

func (ps PowerSettings) VG_EN() byte {
	return byte(ps.GatePower)
}

func (ps PowerSettings) VPP_EN() byte {
	return byte(ps.OTPProgram)
}

func (ps PowerSettings) VCOM_SLEW() byte {
	return byte(ps.CommonVoltageSlew)
}

func (ps PowerSettings) VG_LVL() (byte, error) {
	return gateVoltageFlags(ps.GateVoltage.High())
}

func (ps PowerSettings) VDL_LVL() (byte, error) {
	return voltageDrainFlag(-ps.BlackWhiteVoltageDrain.Low, VoltageDrainLowMin, VoltageDrainLowMax, VoltageDrainLowFlagBits)
}

func (ps PowerSettings) VDH_LVL() (byte, error) {
	return voltageDrainFlag(ps.BlackWhiteVoltageDrain.High, VoltageDrainHighMin, VoltageDrainHighMax, VoltageDrainHighFlagBits)
}

func gateVoltageFlags(v Voltage) (byte, error) {
	flags, ok := gateVoltageToFlagsMap[v]
	if !ok {
		return 0, fmt.Errorf("voltage must be one of: %s", gateVoltageAllowedValuesMsgPart)
	}
	return flags, nil
}

var gateVoltageToFlagsMap = map[Voltage]byte{
	9 * Volt:  0b000,
	10 * Volt: 0b001,
	11 * Volt: 0b010,
	12 * Volt: 0b011,
	17 * Volt: 0b100,
	18 * Volt: 0b101,
	19 * Volt: 0b110,
	20 * Volt: 0b111,
}

func voltageDrainFlag(v Voltage, min, max Voltage, bits uint) (byte, error) {
	if v < min {
		return 0, fmt.Errorf("voltage must not be lower than %s", min)
	}
	if v > max {
		return 0, fmt.Errorf("voltage must not be greater than %s", max)
	}
	delta := max - min
	totalSteps := math.Pow(2, float64(bits)) - 1
	step := float64(delta) / totalSteps
	flag, fraction := math.Modf(float64(v-min) / step)
	if fraction != 0 {
		return 0, fmt.Errorf("voltage must be a multiple of %f", step)
	}
	return byte(flag), nil
}

func (s *seq) powerSetting(
	settings PowerSettings,
) {
	if s.err != nil {
		return
	}
	data, err := settings.Flags()
	if err != nil {
		s.err = err
		return
	}
	s.sendCommand(CommandPWR)
	s.sendData(data)
}
