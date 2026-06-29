package epd7in5v2

import (
	"bytes"
	"testing"
)

func TestResolutionSettingsFlags(t *testing.T) {
	got, err := ResolutionSettings{Horizontal: 800, Vertical: 480}.Flags()
	if err != nil {
		t.Fatalf("Flags() error: %v", err)
	}
	if !bytes.Equal(got, []byte{0x03, 0x20, 0x01, 0xE0}) {
		t.Errorf("Flags() = % X, want 03 20 01 E0", got)
	}
}

func TestResolutionSettingsErrors(t *testing.T) {
	tests := []struct {
		name string
		rs   ResolutionSettings
	}{
		{"horizontal too small", ResolutionSettings{Horizontal: 0, Vertical: 480}},
		{"horizontal too large", ResolutionSettings{Horizontal: 808, Vertical: 480}},
		{"vertical too small", ResolutionSettings{Horizontal: 800, Vertical: 0}},
		{"vertical too large", ResolutionSettings{Horizontal: 800, Vertical: 601}},
		{"horizontal not divisible by 8", ResolutionSettings{Horizontal: 100, Vertical: 480}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.rs.Flags(); err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}
