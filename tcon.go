package epd7in5v2

import (
	"fmt"
	"time"
)

const MinSourceGateNonOverlapPeriod = 4 * TCONPeriod
const MaxSourceGateNonOverlapPeriod = 64 * TCONPeriod

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
	return tconPeriodFlag(s.SourceToGate, "source to gate")
}

func (s TCONSettings) G2S() (byte, error) {
	return tconPeriodFlag(s.GateToSource, "gate to source")
}

func tconPeriodFlag(period time.Duration, prop string) (byte, error) {
	if period < MinSourceGateNonOverlapPeriod {
		return 0, fmt.Errorf("%s must not be smaller than %s", prop, MinSourceGateNonOverlapPeriod)
	}
	if period > MaxSourceGateNonOverlapPeriod {
		return 0, fmt.Errorf("%s must not be greater than %s", prop, MaxSourceGateNonOverlapPeriod)
	}
	if period%TCONPeriodConfigStep > 0 {
		return 0, fmt.Errorf("%s must be a multiple of %s", TCONPeriodConfigStep)
	}
	return byte((period - MinSourceGateNonOverlapPeriod) / TCONPeriodConfigStep), nil
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
