package epd7in5v2

import "fmt"

// HSyncInterval is a duration counted in horizontal sync periods, the unit the
// controller uses for the VCOM-to-data interval.
//
// Build a value by multiplying [HSync], for example 10 * HSync.
type HSyncInterval int

// HSync is the unit of [HSyncInterval]. Multiply it to build an interval.
const HSync HSyncInterval = 1

// BlackWhitePolarity says which bit value means which colour in the image data
// sent to the controller.
//
// It has to agree with how the frame was packed. [BlackAndWhiteImage] sets a
// bit for white, which is why the full-refresh sequences pair it with
// [BlackWhitePolarity_ZeroIsWhite] and an inverted second plane.
type BlackWhitePolarity byte

const (
	// BlackWhitePolarity_ZeroIsWhite treats a clear bit as white.
	BlackWhitePolarity_ZeroIsWhite BlackWhitePolarity = 0
	// BlackWhitePolarity_ZeroIsBlack treats a clear bit as black.
	BlackWhitePolarity_ZeroIsBlack BlackWhitePolarity = 1
)

// RedPolarity says which bit value means red on a three-colour panel.
type RedPolarity byte

const (
	// RedPolarity_OneIsRed treats a set bit as red.
	RedPolarity_OneIsRed RedPolarity = 0
	// RedPolarity_ZeroIsRed treats a clear bit as red.
	RedPolarity_ZeroIsRed RedPolarity = 1
)

// ColorModeSettings is the colour-mode-dependent half of [CDISettings].
//
// It is a closed interface with two implementations: [BlackWhiteSettings] for
// two-colour panels and [BlackWhiteRedSettings] for three-colour ones. The
// methods carry the datasheet's own field names.
type ColorModeSettings interface {
	isColorModeSettings()
	// N2OCP reports the "copy new to old" bit.
	N2OCP() byte
	// DDX1 reports the data polarity bit for the second plane.
	DDX1() byte
	// BDV reports the border data value field for the given polarity.
	BDV(polarity BlackWhitePolarity) byte
	// BDZ reports the border floating bit.
	BDZ() byte
}

// BlackWhiteRedSettings is the [ColorModeSettings] for a three-colour panel.
//
// The 7.5 inch V2 panel is two-colour, so this driver's own sequences use
// [BlackWhiteSettings] instead; this type is here for completeness of the
// command encoding.
type BlackWhiteRedSettings struct {
	// RedPolarity says which bit value means red.
	RedPolarity RedPolarity
	// Border says how the screen border behaves during a refresh.
	Border BorderBehavior[BlackWhiteRedBorder]
}

// BDZ reports the border floating bit.
func (s BlackWhiteRedSettings) BDZ() byte {
	return s.Border.BDZ()
}

// BDV reports the border data value field. The encoding depends on the
// black/white polarity, so the same border colour maps to different bits under
// different polarities.
//
// It panics if the border and polarity combination has no encoding, which
// cannot happen for the exported [BlackWhiteRedBorder] values.
func (s BlackWhiteRedSettings) BDV(polarity BlackWhitePolarity) byte {
	return s.Border.BDV(func(border BlackWhiteRedBorder) byte {
		bdv, ok := cdi_bdv_kwr[polarity][border]
		if !ok {
			panic("invalid polarity border combination")
		}
		return bdv
	})
}

// cdi_bdv_kwr maps a border colour to its BDV field, per polarity, for
// three-colour panels.
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

// N2OCP reports the "copy new to old" bit, which is always zero in three-colour
// mode.
func (s BlackWhiteRedSettings) N2OCP() byte {
	return 0
}

// DDX1 reports the red data polarity bit.
func (s BlackWhiteRedSettings) DDX1() byte {
	return byte(s.RedPolarity)
}

func (s BlackWhiteRedSettings) isColorModeSettings() {}

var _ ColorModeSettings = BlackWhiteRedSettings{}

// BlackWhiteSettings is the [ColorModeSettings] for a two-colour panel, and the
// one this driver uses.
type BlackWhiteSettings struct {
	// Refresh picks a full or differential refresh.
	Refresh Refresh
	// Border says how the screen border behaves during a refresh.
	Border BorderBehavior[BlackWhiteBorder]
}

// BDZ reports the border floating bit.
func (s BlackWhiteSettings) BDZ() byte {
	return s.Border.BDZ()
}

// BDV reports the border data value field. The encoding depends on the
// black/white polarity, so the same border transition maps to different bits
// under different polarities.
//
// It panics if the border and polarity combination has no encoding, which
// cannot happen for the exported [BlackWhiteBorder] values.
func (s BlackWhiteSettings) BDV(polarity BlackWhitePolarity) byte {
	return s.Border.BDV(func(border BlackWhiteBorder) byte {
		bdv, ok := cdi_bdy_kw[polarity][border]
		if !ok {
			panic("invalid polarity border combination")
		}
		return bdv
	})
}

// cdi_bdy_kw maps a border transition to its BDV field, per polarity, for
// two-colour panels.
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

// N2OCP reports the "copy new to old" bit, taken from the refresh mode.
func (s BlackWhiteSettings) N2OCP() byte {
	return s.Refresh.N2OCP()
}

// DDX1 reports the second-plane data polarity bit, taken from the refresh mode.
func (s BlackWhiteSettings) DDX1() byte {
	return s.Refresh.DDX1()
}

func (s BlackWhiteSettings) isColorModeSettings() {}

var _ ColorModeSettings = BlackWhiteSettings{}

// Refresh selects how the controller drives pixels on the next refresh.
//
// It is a closed interface with two implementations: [FullRefresh] and
// [DifferentialRefresh].
type Refresh interface {
	isRefresh()
	// DDX1 reports the second-plane data polarity bit.
	DDX1() byte
	// N2OCP reports the "copy new to old" bit.
	N2OCP() byte
}

// FullRefresh drives every pixel through the complete waveform, regardless of
// what it previously showed. It is slower but leaves no ghosting.
type FullRefresh struct{}

func (f FullRefresh) isRefresh() {}

// DDX1 reports the second-plane data polarity bit.
func (f FullRefresh) DDX1() byte {
	return 1
}

// N2OCP reports the "copy new to old" bit.
func (f FullRefresh) N2OCP() byte {
	return 0
}

var _ Refresh = FullRefresh{}

// DifferentialRefresh drives only the pixels whose value changed between the
// old and new planes. It is much faster than [FullRefresh] but accumulates
// ghosting, so a periodic full refresh is still needed.
type DifferentialRefresh struct {
	// CopyNewToOld makes the controller copy the new plane over the old one
	// after the refresh, so the next differential refresh compares against what
	// is actually on the glass. [Epd.DisplayPart] sets this.
	CopyNewToOld bool
}

func (d DifferentialRefresh) isRefresh() {}

// N2OCP reports the "copy new to old" bit.
func (d DifferentialRefresh) N2OCP() byte {
	return boolToByte(d.CopyNewToOld)
}

// DDX1 reports the second-plane data polarity bit.
func (d DifferentialRefresh) DDX1() byte {
	return 0
}

var _ Refresh = DifferentialRefresh{}

// ColorModeBorder constrains the border colour types to those that match a
// panel's colour mode: [BlackWhiteBorder] or [BlackWhiteRedBorder].
type ColorModeBorder interface {
	isColorModeBorder()
}

// BlackWhiteRedBorder is what the screen border shows during a refresh on a
// three-colour panel.
type BlackWhiteRedBorder byte

func (b BlackWhiteRedBorder) isColorModeBorder() {}

const (
	// BlackWhiteRedBorder_Border keeps the border at its existing value.
	BlackWhiteRedBorder_Border BlackWhiteRedBorder = iota
	// BlackWhiteRedBorder_Red drives the border red.
	BlackWhiteRedBorder_Red
	// BlackWhiteRedBorder_White drives the border white.
	BlackWhiteRedBorder_White
	// BlackWhiteRedBorder_Black drives the border black.
	BlackWhiteRedBorder_Black
)

// BlackWhiteBorder is the transition the screen border is driven through during
// a refresh on a two-colour panel.
type BlackWhiteBorder byte

func (b BlackWhiteBorder) isColorModeBorder() {}

const (
	// BlackWhiteBorder_Border keeps the border at its existing value.
	BlackWhiteBorder_Border BlackWhiteBorder = iota
	// BlackWhiteBorder_BlackToWhite drives the border from black to white.
	BlackWhiteBorder_BlackToWhite
	// BlackWhiteBorder_WhiteToBlack drives the border from white to black.
	BlackWhiteBorder_WhiteToBlack
	// BlackWhiteBorder_BlackToBlack holds the border black.
	BlackWhiteBorder_BlackToBlack
)

// BorderBehavior says whether the screen border is actively driven during a
// refresh or left floating.
//
// It is a closed interface with two implementations: [DrivenBehavior] and
// [FloatBehavior].
type BorderBehavior[T ColorModeBorder] interface {
	isBorderBehavior()
	// BDV reports the border data value field, using encode to map the chosen
	// border to its bits.
	BDV(encode func(border T) byte) byte
	// BDZ reports the border floating bit.
	BDZ() byte
}

// FloatBehavior leaves the screen border floating during a refresh, so the
// border is not driven at all. [Epd.DisplayPart] and [Epd.Sleep] use it.
type FloatBehavior[T ColorModeBorder] struct{}

// BDZ reports the border floating bit, which is set for a floating border.
func (f FloatBehavior[T]) BDZ() byte {
	return 1
}

// BDV reports the border data value field. A floating border has no value, so
// encode is never called and the result is zero.
func (f FloatBehavior[T]) BDV(func(border T) byte) byte {
	return 0
}

func (f FloatBehavior[T]) isBorderBehavior() {}

var _ BorderBehavior[ColorModeBorder] = FloatBehavior[ColorModeBorder]{}

// DrivenBehavior actively drives the screen border through a chosen waveform
// during a refresh.
type DrivenBehavior[T ColorModeBorder] struct {
	// LookupTable is the border colour or transition to drive.
	LookupTable T
}

// BDZ reports the border floating bit, which is clear for a driven border.
func (d DrivenBehavior[T]) BDZ() byte {
	return 0
}

// BDV reports the border data value field for the chosen border, using f to map
// it to its bits.
func (d DrivenBehavior[T]) BDV(f func(border T) byte) byte {
	return f(d.LookupTable)
}

func (d DrivenBehavior[T]) isBorderBehavior() {}

var _ BorderBehavior[ColorModeBorder] = DrivenBehavior[ColorModeBorder]{}

// CDISettings configures the controller's CDI command, which covers the VCOM
// and data interval: how image bits map to colours, how the border behaves, and
// how long the controller waits between VCOM and data.
type CDISettings struct {
	// ColorMode is the colour-mode-dependent half of the command: either
	// [BlackWhiteSettings] or [BlackWhiteRedSettings].
	ColorMode ColorModeSettings
	// BlackWhitePolarity says which bit value means which colour.
	BlackWhitePolarity BlackWhitePolarity
	// CommonVoltageDataInterval is the gap between the VCOM and data phases. It
	// must lie between [MinCommonDataInterval] and [MaxCommonDataInterval].
	CommonVoltageDataInterval HSyncInterval
}

// Flags encodes the settings into the two parameter bytes for CommandCDI.
//
// It returns an error if [CDISettings.CommonVoltageDataInterval] is out of
// range.
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

// BDZ reports the border floating bit, as named in the datasheet.
func (s CDISettings) BDZ() byte {
	return s.ColorMode.BDZ()
}

// BDV reports the border data value field, as named in the datasheet.
func (s CDISettings) BDV() byte {
	return s.ColorMode.BDV(s.BlackWhitePolarity)
}

// N2OCP reports the "copy new to old" bit, as named in the datasheet.
func (s CDISettings) N2OCP() byte {
	return s.ColorMode.N2OCP()
}

// DDX reports the two-bit data polarity field, as named in the datasheet: the
// colour mode's second-plane polarity above the black/white polarity.
func (s CDISettings) DDX() byte {
	return s.ColorMode.DDX1()<<1 | byte(s.BlackWhitePolarity)
}

const (
	// MaxCommonDataInterval is the longest VCOM-to-data interval the controller
	// accepts.
	MaxCommonDataInterval = 17 * HSync
	// MinCommonDataInterval is the shortest VCOM-to-data interval the controller
	// accepts.
	MinCommonDataInterval = 2 * HSync
)

// CDI encodes [CDISettings.CommonVoltageDataInterval] as the datasheet's CDI
// field, which counts down from [MaxCommonDataInterval] rather than up.
//
// It returns an error if the interval is out of range.
func (s CDISettings) CDI() (byte, error) {
	if s.CommonVoltageDataInterval > MaxCommonDataInterval {
		return 0, fmt.Errorf("common voltage data interval must not be greater than %d", MaxCommonDataInterval)
	}
	if s.CommonVoltageDataInterval < MinCommonDataInterval {
		return 0, fmt.Errorf("common voltage data interval must not be smaller than %d", MinCommonDataInterval)
	}
	return byte(MaxCommonDataInterval - s.CommonVoltageDataInterval), nil
}

// commonVoltageAndDataIntervalSetting sends CommandCDI with the encoded
// settings.
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
