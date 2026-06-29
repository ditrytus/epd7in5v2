package epd7in5v2

import (
	"bytes"
	"testing"
)

func TestDualSPIModeFlags(t *testing.T) {
	tests := []struct {
		name string
		s    DualSPIModeSettings
		want byte
	}{
		{"both off", DualSPIModeSettings{}, 0x00},
		{"MM only", DualSPIModeSettings{MMInputPinEnabled: true}, 0x20},
		{"DUSPI only", DualSPIModeSettings{DualSPIModeEnabled: true}, 0x10},
		{"both on", DualSPIModeSettings{MMInputPinEnabled: true, DualSPIModeEnabled: true}, 0x30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Flags(); !bytes.Equal(got, []byte{tt.want}) {
				t.Errorf("Flags() = % X, want %#02x", got, tt.want)
			}
		})
	}
}
