package epd7in5v2

import "testing"

func TestForceTemperature(t *testing.T) {
	tests := []struct {
		temp Temperature
		want byte
	}{
		{90 * Celsius, 0x5A},
		{110 * Celsius, 0x6E},
		{0 * Celsius, 0x00},
		{255 * Celsius, 0xFF},
	}
	for _, tt := range tests {
		e, rec := newTestEPD()
		s := &seq{e: e}
		s.forceTemperature(tt.temp)
		assertOps(t, rec.ops(), []op{{0xE5, []byte{tt.want}}})
	}
}
