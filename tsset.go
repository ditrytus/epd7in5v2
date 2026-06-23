package epd7in5v2

type Temperature byte

const Celsius Temperature = 1

func (s *seq) forceTemperature(temperature Temperature) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandTSSET)
	s.sendData([]byte{byte(temperature)})
}
