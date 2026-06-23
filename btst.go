package epd7in5v2

type BoosterSoftStartSettings struct {
	PhaseA  PhaseABSettings
	PhaseB  PhaseABSettings
	PhaseC1 PhaseCSettings
	PhaseC2 PhaseC2Settings
}

func (s BoosterSoftStartSettings) Flags() []byte {
	return []byte{
		s.BT_PHA(),
		s.BT_PHB(),
		s.BT_PHC1(),
		s.BT_PHC2(),
	}
}

func (s BoosterSoftStartSettings) BT_PHA() byte {
	return s.PhaseA.Flags()
}

func (s BoosterSoftStartSettings) BT_PHB() byte {
	return s.PhaseB.Flags()
}

func (s BoosterSoftStartSettings) BT_PHC1() byte {
	return s.PhaseC1.Flags()
}

func (s BoosterSoftStartSettings) BT_PHC2() byte {
	return s.PhaseC2.Flags()
}

type StartPeriod byte

const (
	StartPeriod_10ms StartPeriod = 0b00
	StartPeriod_20ms StartPeriod = 0b01
	StartPeriod_30ms StartPeriod = 0b10
	StartPeriod_40ms StartPeriod = 0b11
)

type Strength byte

const (
	Strength_1 Strength = 0b000
	Strength_2 Strength = 0b001
	Strength_3 Strength = 0b010
	Strength_4 Strength = 0b011
	Strength_5 Strength = 0b100
	Strength_6 Strength = 0b101
	Strength_7 Strength = 0b110
	Strength_8 Strength = 0b111
)

type OffDuration byte

const (
	OffDuration_0_27us OffDuration = 0b000
	OffDuration_0_34us OffDuration = 0b001
	OffDuration_0_40us OffDuration = 0b010
	OffDuration_0_54us OffDuration = 0b011
	OffDuration_0_80us OffDuration = 0b100
	OffDuration_1_54us OffDuration = 0b101
	OffDuration_3_34us OffDuration = 0b110
	OffDuration_6_58us OffDuration = 0b111
)

type PhaseSettings struct {
	DrivingStrength   Strength
	MinGDROffDuration OffDuration
}

func (s PhaseSettings) Flags() byte {
	return byte(s.DrivingStrength)<<3 | byte(s.MinGDROffDuration)
}

type PhaseABSettings struct {
	SoftStartPeriod StartPeriod
	PhaseSettings
}

func (s PhaseABSettings) Flags() byte {
	return byte(s.SoftStartPeriod)<<6 | s.PhaseSettings.Flags()
}

type PhaseCSettings = PhaseSettings

type PhaseC2Settings struct {
	Enabled bool
	PhaseCSettings
}

func (s PhaseC2Settings) Flags() byte {
	return boolToByte(s.Enabled)<<7 | s.PhaseCSettings.Flags()
}

func (s *seq) boosterSoftStart(settings BoosterSoftStartSettings) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandBTST)
	s.sendData(settings.Flags())
}
