package epd7in5v2

// DualSPIModeSettings configures the controller's DUSPI command, which selects
// between the standard single-data-line SPI interface and the faster two-line
// variant.
//
// This driver runs 4-line SPI with dual mode off, matching the Driver HAT's
// Interface Config = 0 switch position. See [Epd.Init].
type DualSPIModeSettings struct {
	// MMInputPinEnabled enables the MM input pin (MM_EN).
	MMInputPinEnabled bool
	// DualSPIModeEnabled turns on dual SPI mode (DUSPI_EN), in which image data
	// is clocked over two data lines instead of one.
	DualSPIModeEnabled bool
}

// Flags encodes the settings into the parameter byte for CommandDUSPI.
func (s DualSPIModeSettings) Flags() []byte {
	return []byte{s.MM_EN()<<5 | s.DUSPI_EN()<<4}
}

// MM_EN reports the MM input pin enable bit, as named in the datasheet.
func (s DualSPIModeSettings) MM_EN() byte {
	return boolToByte(s.MMInputPinEnabled)
}

// DUSPI_EN reports the dual SPI mode enable bit, as named in the datasheet.
func (s DualSPIModeSettings) DUSPI_EN() byte {
	return boolToByte(s.DualSPIModeEnabled)
}

// dualSPIMode sends CommandDUSPI with the encoded settings.
func (s *seq) dualSPIMode(settings DualSPIModeSettings) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandDUSPI)
	s.sendData(settings.Flags())
}
