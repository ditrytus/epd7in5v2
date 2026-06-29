package epd7in5v2

import (
	"bytes"
	"testing"
)

func TestCDIFlags(t *testing.T) {
	tests := []struct {
		name string
		s    CDISettings
		want []byte
	}{
		{
			name: "normal full BW (init)",
			s: CDISettings{
				ColorMode: BlackWhiteSettings{
					Refresh: DifferentialRefresh{CopyNewToOld: false},
					Border:  DrivenBehavior[BlackWhiteBorder]{LookupTable: BlackWhiteBorder_BlackToWhite},
				},
				BlackWhitePolarity:        BlackWhitePolarity_ZeroIsWhite,
				CommonVoltageDataInterval: 10 * HSync,
			},
			want: []byte{0x10, 0x07},
		},
		{
			name: "partial BW (float border, copy new->old)",
			s: CDISettings{
				ColorMode: BlackWhiteSettings{
					Refresh: DifferentialRefresh{CopyNewToOld: true},
					Border:  FloatBehavior[BlackWhiteBorder]{},
				},
				BlackWhitePolarity:        BlackWhitePolarity_ZeroIsBlack,
				CommonVoltageDataInterval: 10 * HSync,
			},
			want: []byte{0x89, 0x07},
		},
		{
			name: "sleep BW (float border, full refresh)",
			s: CDISettings{
				ColorMode: BlackWhiteSettings{
					Refresh: FullRefresh{},
					Border:  FloatBehavior[BlackWhiteBorder]{},
				},
				BlackWhitePolarity:        BlackWhitePolarity_ZeroIsBlack,
				CommonVoltageDataInterval: 10 * HSync,
			},
			want: []byte{0x83, 0x07},
		},
		{
			name: "KWR driven red border",
			s: CDISettings{
				ColorMode: BlackWhiteRedSettings{
					RedPolarity: RedPolarity_OneIsRed,
					Border:      DrivenBehavior[BlackWhiteRedBorder]{LookupTable: BlackWhiteRedBorder_Red},
				},
				BlackWhitePolarity:        BlackWhitePolarity_ZeroIsBlack,
				CommonVoltageDataInterval: 10 * HSync,
			},
			want: []byte{0x21, 0x07}, // BDV=10 (red, ZeroIsBlack), DDX=01
		},
		{
			name: "min interval encodes to 0x0F",
			s: CDISettings{
				ColorMode:                 BlackWhiteSettings{Refresh: DifferentialRefresh{}, Border: FloatBehavior[BlackWhiteBorder]{}},
				BlackWhitePolarity:        BlackWhitePolarity_ZeroIsWhite,
				CommonVoltageDataInterval: 2 * HSync,
			},
			want: []byte{0x80, 0x0F},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.Flags()
			if err != nil {
				t.Fatalf("Flags() error: %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("Flags() = % X, want % X", got, tt.want)
			}
		})
	}
}

func TestCDIIntervalOutOfRange(t *testing.T) {
	base := func(interval HSyncInterval) CDISettings {
		return CDISettings{
			ColorMode:                 BlackWhiteSettings{Refresh: DifferentialRefresh{}, Border: FloatBehavior[BlackWhiteBorder]{}},
			BlackWhitePolarity:        BlackWhitePolarity_ZeroIsWhite,
			CommonVoltageDataInterval: interval,
		}
	}
	if _, err := base(18 * HSync).Flags(); err == nil {
		t.Error("expected error for interval > 17")
	}
	if _, err := base(1 * HSync).Flags(); err == nil {
		t.Error("expected error for interval < 2")
	}
}

func TestCDIDDXBitOrder(t *testing.T) {
	// DDX[0] = B/W polarity (low bit), DDX[1] = mode bit (high bit).
	tests := []struct {
		pol  BlackWhitePolarity
		mode Refresh
		want byte
	}{
		{BlackWhitePolarity_ZeroIsWhite, DifferentialRefresh{}, 0b00},
		{BlackWhitePolarity_ZeroIsBlack, DifferentialRefresh{}, 0b01},
		{BlackWhitePolarity_ZeroIsWhite, FullRefresh{}, 0b10},
		{BlackWhitePolarity_ZeroIsBlack, FullRefresh{}, 0b11},
	}
	for _, tt := range tests {
		s := CDISettings{
			ColorMode:          BlackWhiteSettings{Refresh: tt.mode, Border: FloatBehavior[BlackWhiteBorder]{}},
			BlackWhitePolarity: tt.pol,
		}
		if got := s.DDX(); got != tt.want {
			t.Errorf("DDX(pol=%d mode=%T) = %#02b, want %#02b", tt.pol, tt.mode, got, tt.want)
		}
	}
}

func TestRefreshAndBorderBits(t *testing.T) {
	if (FullRefresh{}).DDX1() != 1 || (FullRefresh{}).N2OCP() != 0 {
		t.Error("FullRefresh bits wrong")
	}
	if (DifferentialRefresh{CopyNewToOld: true}).N2OCP() != 1 {
		t.Error("DifferentialRefresh{true}.N2OCP should be 1")
	}
	if (DifferentialRefresh{CopyNewToOld: false}).N2OCP() != 0 {
		t.Error("DifferentialRefresh{false}.N2OCP should be 0")
	}
	if (DifferentialRefresh{}).DDX1() != 0 {
		t.Error("DifferentialRefresh.DDX1 should be 0")
	}
	if (FloatBehavior[BlackWhiteBorder]{}).BDZ() != 1 {
		t.Error("Float border BDZ should be 1")
	}
	if (DrivenBehavior[BlackWhiteBorder]{}).BDZ() != 0 {
		t.Error("Driven border BDZ should be 0")
	}
}

func TestCDIBDVTables(t *testing.T) {
	// Spot-check both polarity columns against the spec (p.37).
	kw := BlackWhiteSettings{Border: DrivenBehavior[BlackWhiteBorder]{LookupTable: BlackWhiteBorder_Border}, Refresh: DifferentialRefresh{}}
	if got := kw.BDV(BlackWhitePolarity_ZeroIsWhite); got != 0b00 {
		t.Errorf("KW border LUTBD @ZeroIsWhite = %#02b, want 00", got)
	}
	if got := kw.BDV(BlackWhitePolarity_ZeroIsBlack); got != 0b11 {
		t.Errorf("KW border LUTBD @ZeroIsBlack = %#02b, want 11", got)
	}
	kwr := BlackWhiteRedSettings{Border: DrivenBehavior[BlackWhiteRedBorder]{LookupTable: BlackWhiteRedBorder_Black}}
	if got := kwr.BDV(BlackWhitePolarity_ZeroIsBlack); got != 0b00 {
		t.Errorf("KWR LUTK @ZeroIsBlack = %#02b, want 00", got)
	}
	if got := kwr.BDV(BlackWhitePolarity_ZeroIsWhite); got != 0b11 {
		t.Errorf("KWR LUTK @ZeroIsWhite = %#02b, want 11", got)
	}
}
