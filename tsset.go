package epd7in5v2

// Temperature is a temperature in whole degrees Celsius, as understood by the
// controller's TSSET (force temperature) command.
//
// Build a value by multiplying [Celsius], for example 90 * Celsius.
type Temperature byte

// Celsius is the unit of [Temperature]. Multiply it to build a temperature.
const Celsius Temperature = 1

// forceTemperature overrides the panel's temperature reading with a fixed
// value, which selects a shorter refresh waveform. It only takes effect when
// [CCSETSettings.TemperatureSource] is [TemperatureSource_Register].
func (s *seq) forceTemperature(temperature Temperature) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandTSSET)
	s.sendData([]byte{byte(temperature)})
}
