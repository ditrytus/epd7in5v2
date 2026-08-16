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

// gateVoltageAllowedValuesMsgPart lists the legal gate voltages for use in
// error messages. It is built once by init from gateVoltageToFlagsMap.
var gateVoltageAllowedValuesMsgPart string

// Externality says whether a supply rail is generated inside the controller or
// fed in from outside.
type Externality byte

const (
	// External means the rail is supplied by circuitry outside the controller.
	External Externality = 0
	// Internal means the controller generates the rail itself.
	Internal Externality = 1
)

// Speed selects between the controller's slow and fast slew rates.
type Speed byte

const (
	// Slow is the gentler slew rate.
	Slow Speed = 0
	// Fast is the quicker slew rate.
	Fast Speed = 1
)

// Voltage is an electrical potential in volts.
//
// Build a value by multiplying [Volt], for example 15 * Volt. Negative values
// are meaningful: the drain low rail is negative.
type Voltage float32

// String formats the voltage with a trailing "V".
func (v Voltage) String() string {
	return fmt.Sprintf("%fV", v)
}

// Volt is the unit of [Voltage]. Multiply it to build a voltage.
const Volt Voltage = 1

const (
	// VoltageDrainHighMin is the lowest source drain voltage the controller can
	// be configured for.
	VoltageDrainHighMin = 2.4 * Volt
	// VoltageDrainHighMax is the highest source drain voltage the controller can
	// be configured for.
	VoltageDrainHighMax = 15 * Volt

	// VoltageDrainHighFlagBits is the width in bits of the drain voltage
	// register field. Together with the min and max above it fixes the step
	// size to 0.2 V, and drain voltages must land on that grid.
	VoltageDrainHighFlagBits = 6
)

// VoltageRange is an asymmetric pair of supply rails.
type VoltageRange struct {
	// High is the positive rail.
	High Voltage
	// Low is the negative rail, and is normally negative.
	Low Voltage
}

// SymmetricVoltageRange is a rail pair described by a single magnitude: the
// positive and negative rails are mirror images. Its sign is ignored, so
// SymmetricVoltageRange(20 * Volt) and SymmetricVoltageRange(-20 * Volt) mean
// the same thing.
type SymmetricVoltageRange Voltage

// High returns the positive rail, that is, the magnitude of the range.
func (sv SymmetricVoltageRange) High() Voltage {
	if sv < 0 {
		return Voltage(-sv)
	}
	return Voltage(sv)
}

// Low returns the negative rail, that is, the negated magnitude of the range.
func (sv SymmetricVoltageRange) Low() Voltage {
	if sv > 0 {
		return Voltage(-sv)
	}
	return Voltage(sv)
}

// PowerSettings configures the controller's PWR command: which supply rails the
// controller generates internally, and at what voltages.
//
// Start from [NewDefaultPowerSettings] rather than a zero value; the zero value
// asks for external rails and an invalid gate voltage.
type PowerSettings struct {
	// BorderLowDropoutEnabled turns on the border low-dropout regulator (BD_EN).
	BorderLowDropoutEnabled bool
	// SourceLowVoltagePower selects the origin of the source low voltage rail
	// (VSR_EN).
	SourceLowVoltagePower Externality
	// SourcePower selects the origin of the source rail (VS_EN).
	SourcePower Externality
	// GatePower selects the origin of the gate rail (VG_EN).
	GatePower Externality
	// OTPProgram selects the origin of the OTP programming voltage (VPP_EN).
	// Leave it [External] unless you intend to burn the panel's one-time
	// programmable memory.
	OTPProgram Externality
	// CommonVoltageSlew is the slew rate of the common (VCOM) rail.
	CommonVoltageSlew Speed
	// GateVoltage is the gate drive rail. Its magnitude must be one of 9, 10,
	// 11, 12, 17, 18, 19 or 20 volts.
	GateVoltage SymmetricVoltageRange
	// BlackWhiteVoltageDrain is the source drive rail pair for black and white
	// pixels. Each side's magnitude must lie between [VoltageDrainHighMin] and
	// [VoltageDrainHighMax] and be a multiple of 0.2 V.
	BlackWhiteVoltageDrain VoltageRange
}

// NewDefaultPowerSettings returns the controller's power-on defaults: all rails
// generated internally except the OTP programming voltage, a slow VCOM slew, a
// 20 V gate rail and a +/-14 V source drain rail.
//
// [Epd.Init] starts from these and raises the drain rail to +/-15 V.
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

// Flags encodes the settings into the four parameter bytes for CommandPWR.
//
// It returns an error if the gate voltage is not one of the values the
// controller supports, or if either drain voltage is out of range or off the
// 0.2 V grid.
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

// BD_EN reports the border low-dropout enable bit, as named in the datasheet.
func (ps PowerSettings) BD_EN() byte {
	return boolToByte(ps.BorderLowDropoutEnabled)
}

// VSR_EN reports the source low voltage rail origin bit, as named in the
// datasheet.
func (ps PowerSettings) VSR_EN() byte {
	return byte(ps.SourceLowVoltagePower)
}

// VS_EN reports the source rail origin bit, as named in the datasheet.
func (ps PowerSettings) VS_EN() byte {
	return byte(ps.SourcePower)
}

// VG_EN reports the gate rail origin bit, as named in the datasheet.
func (ps PowerSettings) VG_EN() byte {
	return byte(ps.GatePower)
}

// VPP_EN reports the OTP programming voltage origin bit, as named in the
// datasheet.
func (ps PowerSettings) VPP_EN() byte {
	return byte(ps.OTPProgram)
}

// VCOM_SLEW reports the common voltage slew rate bit, as named in the
// datasheet.
func (ps PowerSettings) VCOM_SLEW() byte {
	return byte(ps.CommonVoltageSlew)
}

// VG_LVL encodes [PowerSettings.GateVoltage] as the datasheet's VG_LVL field.
//
// It returns an error if the magnitude is not one of the eight supported gate
// voltages.
func (ps PowerSettings) VG_LVL() (byte, error) {
	return gateVoltageFlags(ps.GateVoltage.High())
}

// VDL_LVL encodes the negative side of
// [PowerSettings.BlackWhiteVoltageDrain] as the datasheet's VDL_LVL field.
//
// It returns an error if the magnitude is out of range or off the 0.2 V grid.
func (ps PowerSettings) VDL_LVL() (byte, error) {
	return voltageDrainFlag(-ps.BlackWhiteVoltageDrain.Low, VoltageDrainHighMin, VoltageDrainHighMax, VoltageDrainHighFlagBits)
}

// VDH_LVL encodes the positive side of
// [PowerSettings.BlackWhiteVoltageDrain] as the datasheet's VDH_LVL field.
//
// It returns an error if the voltage is out of range or off the 0.2 V grid.
func (ps PowerSettings) VDH_LVL() (byte, error) {
	return voltageDrainFlag(ps.BlackWhiteVoltageDrain.High, VoltageDrainHighMin, VoltageDrainHighMax, VoltageDrainHighFlagBits)
}

// gateVoltageFlags looks up the register value for a gate voltage magnitude,
// erroring with the full list of legal values when there is no match.
func gateVoltageFlags(v Voltage) (byte, error) {
	flags, ok := gateVoltageToFlagsMap[v]
	if !ok {
		return 0, fmt.Errorf("voltage must be one of: %s", gateVoltageAllowedValuesMsgPart)
	}
	return flags, nil
}

// gateVoltageToFlagsMap holds the controller's fixed set of gate voltages. The
// gap between 12 V and 17 V is the controller's, not a transcription slip.
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

// voltageDrainFlag converts a drain voltage into its register value, given the
// field's range and width.
func voltageDrainFlag(v Voltage, lo, hi Voltage, bits uint) (byte, error) {
	if v < lo {
		return 0, fmt.Errorf("voltage must not be lower than %s", lo)
	}
	if v > hi {
		return 0, fmt.Errorf("voltage must not be greater than %s", hi)
	}
	delta := hi - lo
	totalSteps := math.Pow(2, float64(bits)) - 1
	step := float64(delta) / totalSteps
	exact := float64(v-lo) / step
	flag := math.Round(exact)
	// Compare against the nearest step with tolerance: float32 voltages cannot
	// represent multiples of `step` exactly, but genuinely off-grid voltages are
	// at least ~0.5 steps away, so a small epsilon cleanly separates the two.
	if math.Abs(exact-flag) > 1e-3 {
		return 0, fmt.Errorf("voltage must be a multiple of %f", step)
	}
	return byte(flag), nil
}

// powerSetting sends CommandPWR with the encoded settings.
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
