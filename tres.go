package epd7in5v2

import (
	"encoding/binary"
	"fmt"
)

type Resolution uint16

const (
	MinHorizontalRes Resolution = 1
	MaxHorizontalRes Resolution = 800
	MinVerticalRes   Resolution = 1
	MaxVerticalRes   Resolution = 600
)

type ResolutionSettings struct {
	Horizontal Resolution
	Vertical   Resolution
}

func (rs ResolutionSettings) Flags() ([]byte, error) {
	if rs.Horizontal < MinHorizontalRes {
		return nil, fmt.Errorf("horizontal resolution must not be lower than %d", MinHorizontalRes)
	}
	if rs.Horizontal > MaxHorizontalRes {
		return nil, fmt.Errorf("horizontal resolution must not be greater than %d", MaxHorizontalRes)
	}
	if rs.Vertical < MinVerticalRes {
		return nil, fmt.Errorf("vertical resolution must not be lower than %d", MinVerticalRes)
	}
	if rs.Vertical > MaxVerticalRes {
		return nil, fmt.Errorf("vertical resolution must not be greater than %d", MaxVerticalRes)
	}
	if rs.Horizontal%8 != 0 {
		return nil, fmt.Errorf("horizontal resolution must be divisible by 8")
	}
	flags := make([]byte, 0, 4)
	flags = binary.BigEndian.AppendUint16(flags, uint16(rs.Horizontal))
	flags = binary.BigEndian.AppendUint16(flags, uint16(rs.Vertical))
	return flags, nil
}

func (s *seq) resolutionSetting(settings ResolutionSettings) {
	if s.err != nil {
		return
	}
	data, err := settings.Flags()
	if s.err == nil && err != nil {
		s.err = err
		return
	}
	s.sendCommand(CommandTRES)
	s.sendData(data)

}
