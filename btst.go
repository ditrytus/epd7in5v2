package epd7in5v2

// BoosterSoftStartSettings configures the controller's BTST command, which
// controls how the internal DC-DC booster ramps up its output.
//
// The booster runs through phases A, B and C. Phases A and B each have their
// own soft-start period; phase C is split into C1 and C2, where C2 is optional
// and only used with a temperature boundary.
//
// Every init sequence in this package sets these explicitly - see [Epd.Init].
type BoosterSoftStartSettings struct {
	// PhaseA is the first soft-start phase.
	PhaseA PhaseABSettings
	// PhaseB is the second soft-start phase.
	PhaseB PhaseABSettings
	// PhaseC1 is the steady-state drive phase.
	PhaseC1 PhaseCSettings
	// PhaseC2 is the optional second steady-state phase.
	PhaseC2 PhaseC2Settings
}

// Flags encodes the settings into the four parameter bytes for CommandBTST,
// one per phase.
func (s BoosterSoftStartSettings) Flags() []byte {
	return []byte{
		s.BT_PHA(),
		s.BT_PHB(),
		s.BT_PHC1(),
		s.BT_PHC2(),
	}
}

// BT_PHA reports the phase A parameter byte, as named in the datasheet.
func (s BoosterSoftStartSettings) BT_PHA() byte {
	return s.PhaseA.Flags()
}

// BT_PHB reports the phase B parameter byte, as named in the datasheet.
func (s BoosterSoftStartSettings) BT_PHB() byte {
	return s.PhaseB.Flags()
}

// BT_PHC1 reports the phase C1 parameter byte, as named in the datasheet.
func (s BoosterSoftStartSettings) BT_PHC1() byte {
	return s.PhaseC1.Flags()
}

// BT_PHC2 reports the phase C2 parameter byte, as named in the datasheet.
func (s BoosterSoftStartSettings) BT_PHC2() byte {
	return s.PhaseC2.Flags()
}

// StartPeriod is how long a booster soft-start phase lasts.
type StartPeriod byte

// The soft-start periods the controller supports.
const (
	StartPeriod_10ms StartPeriod = 0b00 // 10 ms soft-start period
	StartPeriod_20ms StartPeriod = 0b01 // 20 ms soft-start period
	StartPeriod_30ms StartPeriod = 0b10 // 30 ms soft-start period
	StartPeriod_40ms StartPeriod = 0b11 // 40 ms soft-start period
)

// Strength is the booster's driving strength. Higher values drive harder, which
// ramps the voltage faster at the cost of a larger current spike.
type Strength byte

// The eight driving strengths, weakest to strongest.
const (
	Strength_1 Strength = 0b000 // weakest driving strength
	Strength_2 Strength = 0b001 // driving strength 2 of 8
	Strength_3 Strength = 0b010 // driving strength 3 of 8
	Strength_4 Strength = 0b011 // driving strength 4 of 8
	Strength_5 Strength = 0b100 // driving strength 5 of 8
	Strength_6 Strength = 0b101 // driving strength 6 of 8
	Strength_7 Strength = 0b110 // driving strength 7 of 8
	Strength_8 Strength = 0b111 // strongest driving strength
)

// OffDuration is the minimum off time of the booster's gate driver (GDR)
// within a switching cycle. A longer off duration means a gentler ramp.
type OffDuration byte

// The gate driver off durations, shortest to longest.
const (
	OffDuration_0_27us OffDuration = 0b000 // 0.27 us
	OffDuration_0_34us OffDuration = 0b001 // 0.34 us
	OffDuration_0_40us OffDuration = 0b010 // 0.40 us
	OffDuration_0_54us OffDuration = 0b011 // 0.54 us
	OffDuration_0_80us OffDuration = 0b100 // 0.80 us
	OffDuration_1_54us OffDuration = 0b101 // 1.54 us
	OffDuration_3_34us OffDuration = 0b110 // 3.34 us
	OffDuration_6_58us OffDuration = 0b111 // 6.58 us
)

// PhaseSettings is the drive configuration shared by every booster phase.
type PhaseSettings struct {
	// DrivingStrength is how hard the booster drives during this phase.
	DrivingStrength Strength
	// MinGDROffDuration is the minimum gate driver off time in a switching
	// cycle.
	MinGDROffDuration OffDuration
}

// Flags encodes the phase into its parameter byte, without any phase-specific
// high bits.
func (s PhaseSettings) Flags() byte {
	return byte(s.DrivingStrength)<<3 | byte(s.MinGDROffDuration)
}

// PhaseABSettings configures booster phase A or B, which add a soft-start
// period on top of the shared [PhaseSettings].
type PhaseABSettings struct {
	// SoftStartPeriod is how long this phase ramps for.
	SoftStartPeriod StartPeriod
	PhaseSettings
}

// Flags encodes the phase into its parameter byte, with the soft-start period
// in the top two bits.
func (s PhaseABSettings) Flags() byte {
	return byte(s.SoftStartPeriod)<<6 | s.PhaseSettings.Flags()
}

// PhaseCSettings configures booster phase C1. Phase C has no soft-start period,
// so it is exactly [PhaseSettings].
type PhaseCSettings = PhaseSettings

// PhaseC2Settings configures the optional booster phase C2.
type PhaseC2Settings struct {
	// Enabled turns phase C2 on. When false the remaining fields are still
	// encoded but the controller ignores them.
	Enabled bool
	PhaseCSettings
}

// Flags encodes the phase into its parameter byte, with the enable bit in the
// top bit.
func (s PhaseC2Settings) Flags() byte {
	return boolToByte(s.Enabled)<<7 | s.PhaseCSettings.Flags()
}

// boosterSoftStart sends CommandBTST with the encoded settings.
func (s *seq) boosterSoftStart(settings BoosterSoftStartSettings) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandBTST)
	s.sendData(settings.Flags())
}
