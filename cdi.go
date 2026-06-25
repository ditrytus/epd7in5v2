package epd7in5v2

import "fmt"

type HSyncInterval int

const HSync HSyncInterval = 1

type BlackWhitePolarity byte

const (
	BlackWhitePolarity_ZeroIsWhite BlackWhitePolarity = 0
	BlackWhitePolarity_ZeroIsBlack BlackWhitePolarity = 1
)

type RedPolarity byte

const (
	RedPolarity_OneIsRed  RedPolarity = 0
	RedPolarity_ZeroIsRed RedPolarity = 1
)

type ColorModeSettings interface {
	isColorModeSettings()
	N2OCP() byte
	DDX1() byte
	BDV(polarity BlackWhitePolarity) byte
	BDZ() byte
}

type BlackWhiteRedSettings struct {
	RedPolarity RedPolarity
	Border      BorderBehavior[BlackWhiteRedBorder]
}

func (s BlackWhiteRedSettings) BDZ() byte {
	return s.Border.BDZ()
}

func (s BlackWhiteRedSettings) BDV(polarity BlackWhitePolarity) byte {
	return s.Border.BDV(func(border BlackWhiteRedBorder) byte {
		bdv, ok := cdi_bdv_kwr[polarity][border]
		if !ok {
			panic("invalid polarity border combination")
		}
		return bdv
	})
}

var cdi_bdv_kwr = map[BlackWhitePolarity]map[BlackWhiteRedBorder]byte{
	BlackWhitePolarity_ZeroIsBlack: {
		BlackWhiteRedBorder_Black:  0b00,
		BlackWhiteRedBorder_White:  0b01,
		BlackWhiteRedBorder_Red:    0b10,
		BlackWhiteRedBorder_Border: 0b11,
	},
	BlackWhitePolarity_ZeroIsWhite: {
		BlackWhiteRedBorder_Border: 0b00,
		BlackWhiteRedBorder_Red:    0b01,
		BlackWhiteRedBorder_White:  0b10,
		BlackWhiteRedBorder_Black:  0b11,
	},
}

func (s BlackWhiteRedSettings) N2OCP() byte {
	return 0
}

func (s BlackWhiteRedSettings) DDX1() byte {
	return byte(s.RedPolarity)
}

func (s BlackWhiteRedSettings) isColorModeSettings() {}

var _ ColorModeSettings = BlackWhiteRedSettings{}

type BlackWhiteSettings struct {
	Refresh Refresh
	Border  BorderBehavior[BlackWhiteBorder]
}

func (s BlackWhiteSettings) BDZ() byte {
	return s.Border.BDZ()
}

func (s BlackWhiteSettings) BDV(polarity BlackWhitePolarity) byte {
	return s.Border.BDV(func(border BlackWhiteBorder) byte {
		bdv, ok := cdi_bdy_kw[polarity][border]
		if !ok {
			panic("invalid polarity border combination")
		}
		return bdv
	})
}

var cdi_bdy_kw = map[BlackWhitePolarity]map[BlackWhiteBorder]byte{
	BlackWhitePolarity_ZeroIsBlack: {
		BlackWhiteBorder_BlackToBlack: 0b00,
		BlackWhiteBorder_WhiteToBlack: 0b01,
		BlackWhiteBorder_BlackToWhite: 0b10,
		BlackWhiteBorder_Border:       0b11,
	},
	BlackWhitePolarity_ZeroIsWhite: {
		BlackWhiteBorder_Border:       0b00,
		BlackWhiteBorder_BlackToWhite: 0b01,
		BlackWhiteBorder_WhiteToBlack: 0b10,
		BlackWhiteBorder_BlackToBlack: 0b11,
	},
}

func (s BlackWhiteSettings) N2OCP() byte {
	return s.Refresh.N2OCP()
}

func (s BlackWhiteSettings) DDX1() byte {
	return s.Refresh.DDX1()
}

func (s BlackWhiteSettings) isColorModeSettings() {}

var _ ColorModeSettings = BlackWhiteSettings{}

type Refresh interface {
	isRefresh()
	DDX1() byte
	N2OCP() byte
}

type FullRefresh struct{}

func (f FullRefresh) isRefresh() {}

func (f FullRefresh) DDX1() byte {
	return 1
}

func (f FullRefresh) N2OCP() byte {
	return 0
}

var _ Refresh = FullRefresh{}

type DifferentialRefresh struct {
	CopyNewToOld bool
}

func (d DifferentialRefresh) isRefresh() {}

func (d DifferentialRefresh) N2OCP() byte {
	return boolToByte(d.CopyNewToOld)
}

func (d DifferentialRefresh) DDX1() byte {
	return 0
}

var _ Refresh = DifferentialRefresh{}

type ColorModeBorder interface {
	isColorModeBorder()
}

type BlackWhiteRedBorder byte

func (b BlackWhiteRedBorder) isColorModeBorder() {}

const (
	BlackWhiteRedBorder_Border BlackWhiteRedBorder = iota
	BlackWhiteRedBorder_Red
	BlackWhiteRedBorder_White
	BlackWhiteRedBorder_Black
)

type BlackWhiteBorder byte

func (b BlackWhiteBorder) isColorModeBorder() {}

const (
	BlackWhiteBorder_Border BlackWhiteBorder = iota
	BlackWhiteBorder_BlackToWhite
	BlackWhiteBorder_WhiteToBlack
	BlackWhiteBorder_BlackToBlack
)

type BorderBehavior[T ColorModeBorder] interface {
	isBorderBehavior()
	BDV(func(border T) byte) byte
	BDZ() byte
}

type FloatBehavior[T ColorModeBorder] struct{}

func (f FloatBehavior[T]) BDZ() byte {
	return 1
}

func (f FloatBehavior[T]) BDV(func(border T) byte) byte {
	return 0
}

func (f FloatBehavior[T]) isBorderBehavior() {}

var _ BorderBehavior[ColorModeBorder] = FloatBehavior[ColorModeBorder]{}

type DrivenBehavior[T ColorModeBorder] struct {
	LookupTable T
}

func (d DrivenBehavior[T]) BDZ() byte {
	return 0
}

func (d DrivenBehavior[T]) BDV(f func(border T) byte) byte {
	return f(d.LookupTable)
}

func (d DrivenBehavior[T]) isBorderBehavior() {}

var _ BorderBehavior[ColorModeBorder] = DrivenBehavior[ColorModeBorder]{}

type CDISettings struct {
	ColorMode                 ColorModeSettings
	BlackWhitePolarity        BlackWhitePolarity
	CommonVoltageDataInterval HSyncInterval
}

func (s CDISettings) Flags() ([]byte, error) {
	cdi, err := s.CDI()
	if err != nil {
		return nil, err
	}
	return []byte{
		s.BDZ()<<7 | s.BDV()<<4 | s.N2OCP()<<3 | s.DDX(),
		cdi,
	}, nil
}

func (s CDISettings) BDZ() byte {
	return s.ColorMode.BDZ()
}

func (s CDISettings) BDV() byte {
	return s.ColorMode.BDV(s.BlackWhitePolarity)
}

func (s CDISettings) N2OCP() byte {
	return s.ColorMode.N2OCP()
}

func (s CDISettings) DDX() byte {
	return byte(s.BlackWhitePolarity)<<1 | s.ColorMode.DDX1()
}

const MaxCommonDataInterval = 17 * HSync
const MinCommonDataInterval = 2 * HSync

func (s CDISettings) CDI() (byte, error) {
	if s.CommonVoltageDataInterval > MaxCommonDataInterval {
		return 0, fmt.Errorf("common voltage data interval must not be greater than %d", MaxCommonDataInterval)
	}
	if s.CommonVoltageDataInterval < MinCommonDataInterval {
		return 0, fmt.Errorf("common voltage data interval must not be smaller than %d", MinCommonDataInterval)
	}
	return byte(MaxCommonDataInterval - s.CommonVoltageDataInterval), nil
}

func (s *seq) commonVoltageAndDataIntervalSetting(settings CDISettings) {
	if s.err != nil {
		return
	}
	data, err := settings.Flags()
	if err != nil {
		s.err = err
		return
	}
	s.sendCommand(CommandCDI)
	s.sendData(data)
}
