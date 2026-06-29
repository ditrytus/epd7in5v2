package epd7in5v2

import (
	"bytes"
	"testing"
)

func TestPowerSettingsFlagsInitValues(t *testing.T) {
	ps := NewDefaultPowerSettings()
	ps.BlackWhiteVoltageDrain = VoltageRange{High: 15 * Volt, Low: -15 * Volt}
	got, err := ps.Flags()
	if err != nil {
		t.Fatalf("Flags() error: %v", err)
	}
	if !bytes.Equal(got, []byte{0x07, 0x07, 0x3F, 0x3F}) {
		t.Errorf("Flags() = % X, want 07 07 3F 3F", got)
	}
}

func TestPowerSettingsFlagsDefault(t *testing.T) {
	got, err := NewDefaultPowerSettings().Flags()
	if err != nil {
		t.Fatalf("default Flags() error: %v", err)
	}
	// VDH/VDL of ±14V → flag (14-2.4)/0.2 = 58 = 0x3A
	if !bytes.Equal(got, []byte{0x07, 0x07, 0x3A, 0x3A}) {
		t.Errorf("default Flags() = % X, want 07 07 3A 3A", got)
	}
}

func TestPowerSettingsInvalidGateVoltage(t *testing.T) {
	ps := NewDefaultPowerSettings()
	ps.GateVoltage = SymmetricVoltageRange(13 * Volt) // not a valid step
	if _, err := ps.Flags(); err == nil {
		t.Error("expected error for invalid gate voltage")
	}
}

func TestPowerSettingsDrainOutOfRange(t *testing.T) {
	ps := NewDefaultPowerSettings()
	ps.BlackWhiteVoltageDrain = VoltageRange{High: 16 * Volt, Low: -15 * Volt} // >15V
	if _, err := ps.Flags(); err == nil {
		t.Error("expected error for VDH > 15V")
	}
}

func TestVoltageDrainFlag(t *testing.T) {
	tests := []struct {
		v       Voltage
		wantErr bool
		want    byte
	}{
		{2.4 * Volt, false, 0},
		{15 * Volt, false, 63},
		{16 * Volt, true, 0},  // too high
		{2.0 * Volt, true, 0}, // too low
		{2.5 * Volt, true, 0}, // not on a 0.2V grid step
	}
	for _, tt := range tests {
		got, err := voltageDrainFlag(tt.v, VoltageDrainHighMin, VoltageDrainHighMax, VoltageDrainHighFlagBits)
		if tt.wantErr {
			if err == nil {
				t.Errorf("voltageDrainFlag(%v): expected error", tt.v)
			}
			continue
		}
		if err != nil {
			t.Errorf("voltageDrainFlag(%v): unexpected error %v", tt.v, err)
			continue
		}
		if got != tt.want {
			t.Errorf("voltageDrainFlag(%v) = %d, want %d", tt.v, got, tt.want)
		}
	}
}

func TestGateVoltageFlags(t *testing.T) {
	if got, err := gateVoltageFlags(20 * Volt); err != nil || got != 0b111 {
		t.Errorf("gateVoltageFlags(20) = %#03b, %v; want 111, nil", got, err)
	}
	if got, err := gateVoltageFlags(9 * Volt); err != nil || got != 0b000 {
		t.Errorf("gateVoltageFlags(9) = %#03b, %v; want 000, nil", got, err)
	}
	if _, err := gateVoltageFlags(15 * Volt); err == nil {
		t.Error("expected error for unsupported gate voltage 15V")
	}
}

func TestSymmetricVoltageRange(t *testing.T) {
	sv := SymmetricVoltageRange(20 * Volt)
	if sv.High() != 20*Volt || sv.Low() != -20*Volt {
		t.Errorf("SymmetricVoltageRange(20).High/Low = %v/%v, want 20/-20", sv.High(), sv.Low())
	}
	neg := SymmetricVoltageRange(-20 * Volt)
	if neg.High() != 20*Volt || neg.Low() != -20*Volt {
		t.Errorf("SymmetricVoltageRange(-20).High/Low = %v/%v, want 20/-20", neg.High(), neg.Low())
	}
}
