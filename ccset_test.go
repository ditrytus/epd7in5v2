package epd7in5v2

import (
	"bytes"
	"testing"
)

func TestCCSETFlags(t *testing.T) {
	tests := []struct {
		name string
		s    CCSETSettings
		want byte
	}{
		{"register, clock off", CCSETSettings{TemperatureSource: TemperatureSource_Register, OutputClockAtCLPin: false}, 0x02},
		{"sensor, clock off", CCSETSettings{TemperatureSource: TemperatureSource_Sensor, OutputClockAtCLPin: false}, 0x00},
		{"register, clock on", CCSETSettings{TemperatureSource: TemperatureSource_Register, OutputClockAtCLPin: true}, 0x03},
		{"sensor, clock on", CCSETSettings{TemperatureSource: TemperatureSource_Sensor, OutputClockAtCLPin: true}, 0x01},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Flags(); !bytes.Equal(got, []byte{tt.want}) {
				t.Errorf("Flags() = % X, want %#02x", got, tt.want)
			}
		})
	}
}
