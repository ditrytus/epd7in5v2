package epd7in5v2

import (
	"bytes"
	"testing"
)

func TestPhaseSettingsFlags(t *testing.T) {
	// strength in bits [5:3], off-time in [2:0]
	p := PhaseSettings{DrivingStrength: Strength_3, MinGDROffDuration: OffDuration_6_58us}
	if got := p.Flags(); got != 0x17 { // 010<<3 | 111 = 0001 0111
		t.Errorf("PhaseSettings.Flags = %#02x, want 0x17", got)
	}
}

func TestPhaseABFlags(t *testing.T) {
	// period in bits [7:6]
	p := PhaseABSettings{
		SoftStartPeriod: StartPeriod_40ms, // 11<<6 = 0xC0
		PhaseSettings:   PhaseSettings{DrivingStrength: Strength_1, MinGDROffDuration: OffDuration_0_27us},
	}
	if got := p.Flags(); got != 0xC0 {
		t.Errorf("PhaseABSettings.Flags = %#02x, want 0xC0", got)
	}
}

func TestPhaseC2FlagsEnableBit(t *testing.T) {
	enabled := PhaseC2Settings{Enabled: true, PhaseCSettings: PhaseSettings{DrivingStrength: Strength_1, MinGDROffDuration: OffDuration_0_27us}}
	if got := enabled.Flags(); got != 0x80 {
		t.Errorf("enabled PhaseC2.Flags = %#02x, want 0x80", got)
	}
	disabled := PhaseC2Settings{Enabled: false, PhaseCSettings: PhaseSettings{DrivingStrength: Strength_3, MinGDROffDuration: OffDuration_6_58us}}
	if got := disabled.Flags(); got != 0x17 {
		t.Errorf("disabled PhaseC2.Flags = %#02x, want 0x17", got)
	}
}

func TestBoosterSoftStartFlags(t *testing.T) {
	normal := BoosterSoftStartSettings{
		PhaseA:  PhaseABSettings{SoftStartPeriod: StartPeriod_10ms, PhaseSettings: PhaseSettings{DrivingStrength: Strength_3, MinGDROffDuration: OffDuration_6_58us}},
		PhaseB:  PhaseABSettings{SoftStartPeriod: StartPeriod_10ms, PhaseSettings: PhaseSettings{DrivingStrength: Strength_3, MinGDROffDuration: OffDuration_6_58us}},
		PhaseC1: PhaseCSettings{DrivingStrength: Strength_6, MinGDROffDuration: OffDuration_0_27us},
		PhaseC2: PhaseC2Settings{Enabled: false, PhaseCSettings: PhaseSettings{DrivingStrength: Strength_3, MinGDROffDuration: OffDuration_6_58us}},
	}
	if got := normal.Flags(); !bytes.Equal(got, []byte{0x17, 0x17, 0x28, 0x17}) {
		t.Errorf("normal booster = % X, want 17 17 28 17", got)
	}

	fast := BoosterSoftStartSettings{
		PhaseA:  PhaseABSettings{SoftStartPeriod: StartPeriod_10ms, PhaseSettings: PhaseSettings{DrivingStrength: Strength_5, MinGDROffDuration: OffDuration_6_58us}},
		PhaseB:  PhaseABSettings{SoftStartPeriod: StartPeriod_10ms, PhaseSettings: PhaseSettings{DrivingStrength: Strength_5, MinGDROffDuration: OffDuration_6_58us}},
		PhaseC1: PhaseCSettings{DrivingStrength: Strength_4, MinGDROffDuration: OffDuration_0_27us},
		PhaseC2: PhaseC2Settings{Enabled: false, PhaseCSettings: PhaseSettings{DrivingStrength: Strength_3, MinGDROffDuration: OffDuration_6_58us}},
	}
	if got := fast.Flags(); !bytes.Equal(got, []byte{0x27, 0x27, 0x18, 0x17}) {
		t.Errorf("fast booster = % X, want 27 27 18 17", got)
	}
}
