package epd7in5v2

type TemperatureSource byte

const (
	TemperatureSource_Sensor   = 0
	TemperatureSource_Register = 1
)

type CCSETSettings struct {
	TemperatureSource  TemperatureSource
	OutputClockAtCLPin bool
}

func (s CCSETSettings) Flags() []byte {
	return []byte{s.TSFIX()<<1 | s.CCEN()}
}

func (s CCSETSettings) TSFIX() byte {
	return byte(s.TemperatureSource)
}

func (s CCSETSettings) CCEN() byte {
	return boolToByte(s.OutputClockAtCLPin)
}

func (s *seq) cascadeSetting(settings CCSETSettings) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandCCSET)
	s.sendData(settings.Flags())
}
