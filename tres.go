package epd7in5v2

import (
	"encoding/binary"
	"fmt"
)

// Resolution is a pixel count along one axis, as used by the controller's TRES
// (resolution setting) command.
type Resolution uint16

const (
	// MinHorizontalRes is the smallest horizontal resolution the controller
	// accepts.
	MinHorizontalRes Resolution = 1
	// MaxHorizontalRes is the largest horizontal resolution the controller
	// accepts.
	MaxHorizontalRes Resolution = 800
	// MinVerticalRes is the smallest vertical resolution the controller accepts.
	MinVerticalRes Resolution = 1
	// MaxVerticalRes is the largest vertical resolution the controller accepts.
	// It is greater than this panel's [ScreenHeight] because the same
	// controller also drives taller panels.
	MaxVerticalRes Resolution = 600
)

// ResolutionSettings configures the controller's TRES command with the pixel
// dimensions of the attached panel. For this panel that is [ScreenWidth] by
// [ScreenHeight].
type ResolutionSettings struct {
	// Horizontal is the panel width in pixels. Must be a multiple of 8.
	Horizontal Resolution
	// Vertical is the panel height in pixels.
	Vertical Resolution
}

// Flags encodes the settings into the parameter bytes for CommandTRES, as two
// big-endian 16-bit values.
//
// It returns an error if either axis falls outside the controller's limits, or
// if Horizontal is not a multiple of 8 - one byte of image data carries eight
// horizontal pixels, so the panel width has to land on a byte boundary.
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

// resolutionSetting sends CommandTRES with the encoded settings.
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
