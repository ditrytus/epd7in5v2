package epd7in5v2

import (
	"testing"
	"time"
)

// Each seq wrapper that encodes settings must surface an encoder error into
// seq.err (and emit nothing).
func TestSeqWrappersPropagateEncoderErrors(t *testing.T) {
	t.Run("resolutionSetting", func(t *testing.T) {
		e, rec := newTestEPD()
		s := &seq{e: e}
		s.resolutionSetting(ResolutionSettings{Horizontal: 100, Vertical: 480}) // not /8
		if s.err == nil {
			t.Error("expected error")
		}
		if len(rec.ops()) != 0 {
			t.Error("must not emit on encode error")
		}
	})
	t.Run("setGateSourceNonOverlapPeriod", func(t *testing.T) {
		e, _ := newTestEPD()
		s := &seq{e: e}
		s.setGateSourceNonOverlapPeriod(TCONSettings{SourceToGate: 2 * TCONPeriod, GateToSource: 12 * TCONPeriod})
		if s.err == nil {
			t.Error("expected error")
		}
	})
	t.Run("commonVoltageAndDataIntervalSetting", func(t *testing.T) {
		e, _ := newTestEPD()
		s := &seq{e: e}
		s.commonVoltageAndDataIntervalSetting(CDISettings{
			ColorMode:                 BlackWhiteSettings{Refresh: DifferentialRefresh{}, Border: FloatBehavior[BlackWhiteBorder]{}},
			CommonVoltageDataInterval: 99 * HSync, // out of range
		})
		if s.err == nil {
			t.Error("expected error")
		}
	})
	t.Run("powerSetting", func(t *testing.T) {
		e, _ := newTestEPD()
		s := &seq{e: e}
		ps := NewDefaultPowerSettings()
		ps.GateVoltage = SymmetricVoltageRange(13 * Volt) // invalid step
		s.powerSetting(ps)
		if s.err == nil {
			t.Error("expected error")
		}
	})
}

func TestPowerSettingsVDLError(t *testing.T) {
	ps := NewDefaultPowerSettings()
	ps.BlackWhiteVoltageDrain.Low = 5 * Volt // positive low rail -> -5V is out of range
	if _, err := ps.Flags(); err == nil {
		t.Error("expected VDL encoding error for positive low rail")
	}
}

func TestSleepFallbackToTimeSleep(t *testing.T) {
	e, _ := newTestEPD()
	e.sleepFn = nil // force the time.Sleep fallback
	s := &seq{e: e}
	s.sleep(0) // zero duration returns immediately
}

func TestSleepFnReceivesDuration(t *testing.T) {
	e, _ := newTestEPD()
	var d time.Duration
	e.sleepFn = func(x time.Duration) { d = x }
	s := &seq{e: e}
	s.sleep(7 * time.Millisecond)
	if d != 7*time.Millisecond {
		t.Errorf("sleepFn duration = %v, want 7ms", d)
	}
}
