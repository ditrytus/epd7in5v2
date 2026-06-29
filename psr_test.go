package epd7in5v2

import "testing"

func TestPanelSettingsFlags(t *testing.T) {
	def := NewDefaultPanelSettings()
	if got := def.Flags(); got != 0x0F {
		t.Errorf("default PSR = %#02x, want 0x0F", got)
	}

	bw := NewDefaultPanelSettings()
	bw.ColorMode = ColorMode_BlackWhite
	if got := bw.Flags(); got != 0x1F {
		t.Errorf("B/W PSR = %#02x, want 0x1F", got)
	}
}

func TestPanelSettingsFields(t *testing.T) {
	s := PanelSettings{
		LookupTableSource:    LookupTableSource_Register, // REG=1 → bit5
		ColorMode:            ColorMode_BlackWhiteRed,    // 0
		GateScanDirection:    VerticalDirection_Down,     // 0
		SourceShiftDirection: HorizontalDirection_Left,   // 0
		BoosterEnabled:       false,                      // 0
		SoftReset:            ResetBehavior_Reset,        // 0
	}
	if got := s.Flags(); got != 0x20 {
		t.Errorf("PSR = %#02x, want 0x20", got)
	}
}
