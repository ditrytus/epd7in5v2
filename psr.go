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
	VerticalDirection_Down VerticalDirection = 0
	VerticalDirection_Up   VerticalDirection = 1
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
	return s.REG()<<5 | s.KW_R()<<4 | s.UD()<<3 | s.SHL()<<2 | s.SHD_N()<<1 | s.RST_N()
}

func (s PanelSettings) REG() byte {
	return byte(s.LookupTableSource)
}

func (s PanelSettings) KW_R() byte {
	return byte(s.ColorMode)
}

func (s PanelSettings) UD() byte {
	return byte(s.GateScanDirection)
}

func (s PanelSettings) SHL() byte {
	return byte(s.SourceShiftDirection)
}

func (s PanelSettings) SHD_N() byte {
	return boolToByte(s.BoosterEnabled)
}

func (s PanelSettings) RST_N() byte {
	return byte(s.SoftReset)
}

func (s *seq) panelSetting(settings PanelSettings) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandPSR)
	s.sendData([]byte{settings.Flags()})
}
