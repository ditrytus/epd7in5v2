package epd7in5v2

type DualSPIModeSettings struct {
	MMInputPinEnabled  bool
	DualSPIModeEnabled bool
}

func (s DualSPIModeSettings) Flags() []byte {
	return []byte{s.MM_EN()<<5 | s.DUSPI_EN()<<4}
}

func (s DualSPIModeSettings) MM_EN() byte {
	return boolToByte(s.MMInputPinEnabled)
}

func (s DualSPIModeSettings) DUSPI_EN() byte {
	return boolToByte(s.DualSPIModeEnabled)
}

func (s *seq) dualSPIMode(settings DualSPIModeSettings) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandDUSPI)
	s.sendData(settings.Flags())
}
