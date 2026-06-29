package epd7in5v2

import (
	"bytes"
	"testing"
)

func TestTCONSettingsFlags(t *testing.T) {
	got, err := TCONSettings{SourceToGate: 12 * TCONPeriod, GateToSource: 12 * TCONPeriod}.Flags()
	if err != nil {
		t.Fatalf("Flags() error: %v", err)
	}
	if !bytes.Equal(got, []byte{0x22}) {
		t.Errorf("Flags() = % X, want 22", got)
	}
}

func TestTCONSettingsAsymmetric(t *testing.T) {
	// S2G=4 periods → flag 0; G2S=64 periods → flag 15 → 0x0F
	got, err := TCONSettings{SourceToGate: 4 * TCONPeriod, GateToSource: 64 * TCONPeriod}.Flags()
	if err != nil {
		t.Fatalf("Flags() error: %v", err)
	}
	if !bytes.Equal(got, []byte{0x0F}) {
		t.Errorf("Flags() = % X, want 0F", got)
	}
}

func TestTCONPeriodFlagErrors(t *testing.T) {
	tests := []struct {
		name string
		s    TCONSettings
	}{
		{name: "too small", s: TCONSettings{SourceToGate: 2 * TCONPeriod, GateToSource: 12 * TCONPeriod}},
		{name: "too large", s: TCONSettings{SourceToGate: 12 * TCONPeriod, GateToSource: 68 * TCONPeriod}},
		{name: "not a multiple", s: TCONSettings{SourceToGate: 5 * TCONPeriod, GateToSource: 12 * TCONPeriod}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.s.Flags(); err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}
