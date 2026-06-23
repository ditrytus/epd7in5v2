package epd7in5v2

type LookupTableSource byte

const (
	LookupTableSource_OTP      LookupTableSource = 0
	LookupTableSource_Register LookupTableSource = 1
)

type ColorMode byte

const (
	ColorMode_BlackWhiteRed ColorMode = 0
	ColorMode_BlackWhite    ColorMode = 1
)

type VerticalDirection byte

const (
	VerticalDirection_Up   VerticalDirection = 0
	VerticalDirection_Down VerticalDirection = 1
)

type HorizontalDirection byte

const (
	HorizontalDirection_Left  HorizontalDirection = 0
	HorizontalDirection_Right HorizontalDirection = 1
)

type ResetBehavior byte

const (
	ResetBehavior_Reset    ResetBehavior = 0
	ResetBehavior_NoEffect ResetBehavior = 1
)

type PanelSettings struct {
	LookupTableSource    LookupTableSource
	ColorMode            ColorMode
	GateScanDirection    VerticalDirection
	SourceShiftDirection HorizontalDirection
	BoosterEnabled       bool
	SoftReset            ResetBehavior
}

func NewDefaultPanelSettings() PanelSettings {
	return PanelSettings{
		LookupTableSource:    LookupTableSource_OTP,
		ColorMode:            ColorMode_BlackWhiteRed,
		GateScanDirection:    VerticalDirection_Up,
		SourceShiftDirection: HorizontalDirection_Right,
		BoosterEnabled:       true,
		SoftReset:            ResetBehavior_NoEffect,
	}
}

func (s PanelSettings) Flags() byte {
	return byte(s.LookupTableSource)<<5 |
		byte(s.ColorMode)<<4 |
		byte(s.GateScanDirection)<<3 |
		byte(s.SourceShiftDirection)<<2 |
		boolToByte(s.BoosterEnabled)<<1 |
		byte(s.SoftReset)
}

func (s *seq) panelSetting(settings PanelSettings) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandPSR)
	s.sendData([]byte{settings.Flags()})
}
