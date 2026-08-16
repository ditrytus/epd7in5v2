package epd7in5v2

// LookupTableSource selects where the controller reads its refresh waveform
// lookup tables from.
type LookupTableSource byte

const (
	// LookupTableSource_OTP uses the waveforms burned into the panel's one-time
	// programmable memory. This driver always uses these.
	LookupTableSource_OTP LookupTableSource = 0
	// LookupTableSource_Register uses waveforms written into the controller's
	// registers by the host.
	LookupTableSource_Register LookupTableSource = 1
)

// ColorMode selects which pixel colours the panel drives.
type ColorMode byte

const (
	// ColorMode_BlackWhiteRed drives a three-colour panel, using both image
	// planes as black/white and red channels.
	ColorMode_BlackWhiteRed ColorMode = 0
	// ColorMode_BlackWhite drives a two-colour panel. This is the mode for the
	// 7.5 inch V2 panel, and the one every init sequence here selects.
	ColorMode_BlackWhite ColorMode = 1
)

// VerticalDirection is the order in which the gate driver scans rows.
type VerticalDirection byte

const (
	// VerticalDirection_Down scans from the last row towards the first.
	VerticalDirection_Down VerticalDirection = 0
	// VerticalDirection_Up scans from the first row towards the last.
	VerticalDirection_Up VerticalDirection = 1
)

// HorizontalDirection is the order in which the source driver shifts pixels
// along a row.
type HorizontalDirection byte

const (
	// HorizontalDirection_Left shifts from the last column towards the first.
	HorizontalDirection_Left HorizontalDirection = 0
	// HorizontalDirection_Right shifts from the first column towards the last.
	HorizontalDirection_Right HorizontalDirection = 1
)

// ResetBehavior says whether the panel setting command also performs a soft
// reset.
type ResetBehavior byte

const (
	// ResetBehavior_Reset soft-resets the controller as the command is applied.
	ResetBehavior_Reset ResetBehavior = 0
	// ResetBehavior_NoEffect leaves the controller state alone.
	ResetBehavior_NoEffect ResetBehavior = 1
)

// PanelSettings configures the controller's PSR command, which describes the
// attached panel: how many colours it has, which way it scans, and where its
// waveforms come from.
//
// Start from [NewDefaultPanelSettings] rather than a zero value; the zero value
// selects a soft reset and a disabled booster.
type PanelSettings struct {
	// LookupTableSource picks OTP or register waveforms.
	LookupTableSource LookupTableSource
	// ColorMode picks two- or three-colour operation.
	ColorMode ColorMode
	// GateScanDirection is the row scan order.
	GateScanDirection VerticalDirection
	// SourceShiftDirection is the column shift order.
	SourceShiftDirection HorizontalDirection
	// BoosterEnabled turns on the internal DC-DC booster that generates the
	// panel's drive voltages. It must be on for the panel to display anything.
	BoosterEnabled bool
	// SoftReset says whether applying this command also resets the controller.
	SoftReset ResetBehavior
}

// NewDefaultPanelSettings returns the controller's power-on defaults: OTP
// waveforms, three-colour mode, upward gate scan, rightward source shift,
// booster on and no soft reset.
//
// Callers normally override [PanelSettings.ColorMode] to
// [ColorMode_BlackWhite], which is what every init sequence in this package
// does.
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

// Flags encodes the settings into the single parameter byte for CommandPSR.
func (s PanelSettings) Flags() byte {
	return s.REG()<<5 | s.KW_R()<<4 | s.UD()<<3 | s.SHL()<<2 | s.SHD_N()<<1 | s.RST_N()
}

// REG reports the lookup table source bit, as named in the datasheet.
func (s PanelSettings) REG() byte {
	return byte(s.LookupTableSource)
}

// KW_R reports the colour mode bit, as named in the datasheet.
func (s PanelSettings) KW_R() byte {
	return byte(s.ColorMode)
}

// UD reports the gate scan direction bit, as named in the datasheet.
func (s PanelSettings) UD() byte {
	return byte(s.GateScanDirection)
}

// SHL reports the source shift direction bit, as named in the datasheet.
func (s PanelSettings) SHL() byte {
	return byte(s.SourceShiftDirection)
}

// SHD_N reports the booster enable bit, as named in the datasheet.
func (s PanelSettings) SHD_N() byte {
	return boolToByte(s.BoosterEnabled)
}

// RST_N reports the soft reset bit, as named in the datasheet.
func (s PanelSettings) RST_N() byte {
	return byte(s.SoftReset)
}

// panelSetting sends CommandPSR with the encoded settings.
func (s *seq) panelSetting(settings PanelSettings) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandPSR)
	s.sendData([]byte{settings.Flags()})
}
