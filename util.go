package epd7in5v2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"time"

	"periph.io/x/conn/v3/gpio"
)

type seq struct {
	e   *Epd
	err error
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func tryAll(errMsg string, errs ...error) error {
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	return nil
}

func (s *seq) sendCommand(cmd Command) {
	s.setPin(s.e.dc, gpio.Low)
	if s.err != nil {
		return
	}
	if err := s.e.spi.Tx([]byte{byte(cmd)}, nil); err != nil {
		s.err = fmt.Errorf("failed to send command %d over SPI: %w", cmd, err)
	}
}

func (s *seq) sendData(data []byte) {
	s.setPin(s.e.dc, gpio.High)
	if s.err != nil {
		return
	}
	if err := s.e.spi.Tx(data, nil); err != nil {
		s.err = fmt.Errorf("failed to send data (%d bytes) over SPI: %w", len(data), err)
	}
}

func (s *seq) sleep(dur time.Duration) {
	if s.err != nil {
		return
	}
	time.Sleep(dur)
}

func (s *seq) wait() {
	if s.err != nil {
		return
	}
	for s.e.busy.Read() == gpio.Low {
		if !s.e.busy.WaitForEdge(time.Second * 10) {
			s.err = fmt.Errorf("waiting for %s input pin timed out", s.e.busy.Name())
			return
		}
	}
}

func (s *seq) enterPartialMode() {
	s.sendCommand(CommandPTIN)
}

type GateScan byte

const (
	GateScan_Inside           GateScan = 0
	GateScan_InsideAndOutside GateScan = 1
)

func (s *seq) setPartialWindow(rect image.Rectangle, scan GateScan) {
	if s.err != nil {
		return
	}
	if !rect.In(ScreenBounds) {
		s.err = fmt.Errorf("partial window is not within screen bounds")
		return
	}
	if rect.Min.X%8 != 0 || rect.Max.X%8 != 0 {
		s.err = fmt.Errorf("partial window's horizontal coordinates must be a multiple of 8")
		return
	}
	s.sendCommand(CommandPTL)
	data := make([]byte, 0, 9)
	data = binary.BigEndian.AppendUint16(data, uint16(rect.Min.X))
	data = binary.BigEndian.AppendUint16(data, uint16(rect.Max.X-1))
	data = binary.BigEndian.AppendUint16(data, uint16(rect.Min.Y))
	data = binary.BigEndian.AppendUint16(data, uint16(rect.Max.Y-1))
	data = append(data, byte(scan))
	s.sendData(data)
}

const DeepSleepCheckCode byte = 0xA5

func (s *seq) deepSleep() {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandDSLP)
	s.sendData([]byte{DeepSleepCheckCode})
}
