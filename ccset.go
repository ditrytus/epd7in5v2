package epd7in5v2

// TemperatureSource selects where the controller reads the temperature that
// picks the refresh waveform.
type TemperatureSource byte

const (
	// TemperatureSource_Sensor reads the panel's own temperature sensor, which
	// gives the correct waveform for the ambient conditions.
	TemperatureSource_Sensor = 0
	// TemperatureSource_Register uses the value written by the TSSET command
	// instead of the sensor. This is how [Epd.InitFast] and [Epd.InitPart] force
	// a short waveform and so a quick refresh.
	TemperatureSource_Register = 1
)

// CCSETSettings configures the controller's CCSET (cascade setting) command.
type CCSETSettings struct {
	// TemperatureSource picks the sensor or the forced register value.
	TemperatureSource TemperatureSource
	// OutputClockAtCLPin drives the internal clock out on the CL pin (CCEN),
	// which is only needed when cascading several controllers. This driver
	// leaves it off.
	OutputClockAtCLPin bool
}

// Flags encodes the settings into the parameter byte for CommandCCSET.
func (s CCSETSettings) Flags() []byte {
	return []byte{s.TSFIX()<<1 | s.CCEN()}
}

// TSFIX reports the temperature source bit, as named in the datasheet.
func (s CCSETSettings) TSFIX() byte {
	return byte(s.TemperatureSource)
}

// CCEN reports the cascade clock output bit, as named in the datasheet.
func (s CCSETSettings) CCEN() byte {
	return boolToByte(s.OutputClockAtCLPin)
}

// cascadeSetting sends CommandCCSET with the encoded settings.
func (s *seq) cascadeSetting(settings CCSETSettings) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandCCSET)
	s.sendData(settings.Flags())
}
