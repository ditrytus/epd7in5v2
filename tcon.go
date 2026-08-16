package epd7in5v2

import (
	"fmt"
	"time"
)

const (
	// MinSourceGateNonOverlapPeriod is the shortest non-overlap period the
	// controller accepts.
	MinSourceGateNonOverlapPeriod = 4 * TCONPeriod
	// MaxSourceGateNonOverlapPeriod is the longest non-overlap period the
	// controller accepts.
	MaxSourceGateNonOverlapPeriod = 64 * TCONPeriod
)

const (
	// TCONPeriod is the controller's TCON time base, one tick of the internal
	// timing generator.
	TCONPeriod = 667 * time.Nanosecond
	// TCONPeriodConfigStep is the granularity of [TCONSettings]: non-overlap
	// periods are encoded in units of four [TCONPeriod] ticks, so any value in
	// between cannot be expressed.
	TCONPeriodConfigStep = 4 * TCONPeriod
)

// TCONSettings configures the controller's TCON command - the non-overlap
// period between the source and gate driver signals.
//
// Both durations must lie between [MinSourceGateNonOverlapPeriod] and
// [MaxSourceGateNonOverlapPeriod] and be a whole multiple of
// [TCONPeriodConfigStep].
type TCONSettings struct {
	// SourceToGate is the delay from the source signal to the gate signal.
	SourceToGate time.Duration
	// GateToSource is the delay from the gate signal to the source signal.
	GateToSource time.Duration
}

// Flags encodes the settings into the parameter byte for CommandTCON, packing
// [TCONSettings.S2G] into the high nibble and [TCONSettings.G2S] into the low
// one.
//
// It returns an error if either duration is out of range or is not a multiple
// of [TCONPeriodConfigStep].
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

// S2G encodes [TCONSettings.SourceToGate] as the datasheet's S2G nibble.
func (s TCONSettings) S2G() (byte, error) {
	return tconPeriodFlag(s.SourceToGate, "source to gate")
}

// G2S encodes [TCONSettings.GateToSource] as the datasheet's G2S nibble.
func (s TCONSettings) G2S() (byte, error) {
	return tconPeriodFlag(s.GateToSource, "gate to source")
}

// tconPeriodFlag converts a non-overlap duration into its register nibble,
// naming prop in any error it returns.
func tconPeriodFlag(period time.Duration, prop string) (byte, error) {
	if period < MinSourceGateNonOverlapPeriod {
		return 0, fmt.Errorf("%s must not be smaller than %s", prop, MinSourceGateNonOverlapPeriod)
	}
	if period > MaxSourceGateNonOverlapPeriod {
		return 0, fmt.Errorf("%s must not be greater than %s", prop, MaxSourceGateNonOverlapPeriod)
	}
	if period%TCONPeriodConfigStep > 0 {
		return 0, fmt.Errorf("%s must be a multiple of %s", prop, TCONPeriodConfigStep)
	}
	return byte((period - MinSourceGateNonOverlapPeriod) / TCONPeriodConfigStep), nil
}

// setGateSourceNonOverlapPeriod sends CommandTCON with the encoded settings.
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
