package epd7in5v2

import (
	"errors"
	"fmt"
	"time"

	"periph.io/x/conn/v3/gpio"
)

const MinSourceToGateNonOverlapPeriod = 4 * TCONPeriod
const MaxSourceToGateNonOverlapPeriod = 32 * TCONPeriod
const MinGateToSourceNonOverlapPeriod = 36 * TCONPeriod
const MaxGateToSourceNonOverlapPeriod = 64 * TCONPeriod

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
		err = fmt.Errorf("failed to send command %s over SPI: %w", cmd, err)
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
	if !s.e.busy.WaitForEdge(time.Second * 10) {
		s.err = fmt.Errorf("waiting for %s input pin timed out", s.e.busy.Name())
	}
}

const TCONPeriod = 667 * time.Nanosecond

const TCONPeriodConfigStep = 4 * TCONPeriod

type TCONSettings struct {
	SourceToGate time.Duration
	GateToSource time.Duration
}

func (s TCONSettings) Flags() ([]byte, error) {
	s2g, err := s.S2G()
	if err != nil {
		return nil, err
	}
	g2s, err := s.G2S()
	if err != nil {
		return nil, err
	}
	return []byte{s2g<<4 | g2s}, nil
}

func (s TCONSettings) S2G() (byte, error) {
	if s.SourceToGate < MinSourceToGateNonOverlapPeriod {
		return 0, fmt.Errorf("source to gate must not be smaller than %s", MinSourceToGateNonOverlapPeriod)
	}
	if s.SourceToGate > MaxSourceToGateNonOverlapPeriod {
		return 0, fmt.Errorf("source to gate must not be greater than %s", MaxSourceToGateNonOverlapPeriod)
	}
	if s.SourceToGate%TCONPeriodConfigStep > 0 {
		return 0, fmt.Errorf("source to gate must be a multiple of %s", TCONPeriodConfigStep)
	}
	return byte((s.SourceToGate - MinSourceToGateNonOverlapPeriod) / TCONPeriodConfigStep), nil
}

func (s TCONSettings) G2S() (byte, error) {
	if s.GateToSource < MinGateToSourceNonOverlapPeriod {
		return 0, fmt.Errorf("gate to source must not be smaller than %s", MinGateToSourceNonOverlapPeriod)
	}
	if s.GateToSource > MaxGateToSourceNonOverlapPeriod {
		return 0, fmt.Errorf("gate to source must not be greater than %s", MaxGateToSourceNonOverlapPeriod)
	}
	if s.GateToSource%TCONPeriodConfigStep > 0 {
		return 0, fmt.Errorf("gate to source must be a multiple of %s", TCONPeriodConfigStep)
	}
	return byte((s.GateToSource - MinGateToSourceNonOverlapPeriod) / TCONPeriodConfigStep), nil
}

func (s *seq) setGateSourceNonOverlapPeriod(settings TCONSettings) {
	if s.err != nil {
		return
	}
	data, err := settings.Flags()
	if err != nil {
		s.err = err
		return
	}
	s.sendCommand(CommandTCON)
	s.sendData(data)
}
