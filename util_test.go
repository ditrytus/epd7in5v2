package epd7in5v2

import (
	"errors"
	"image"
	"testing"
	"time"

	"periph.io/x/conn/v3/gpio"
)

func TestBoolToByte(t *testing.T) {
	if boolToByte(true) != 1 {
		t.Error("boolToByte(true) should be 1")
	}
	if boolToByte(false) != 0 {
		t.Error("boolToByte(false) should be 0")
	}
}

func TestTryAll(t *testing.T) {
	if err := tryAll("ctx"); err != nil {
		t.Errorf("tryAll() with no errors = %v, want nil", err)
	}
	if err := tryAll("ctx", nil, nil); err != nil {
		t.Errorf("tryAll(nil,nil) = %v, want nil", err)
	}
	err := tryAll("context", errors.New("a"), nil, errors.New("b"))
	if err == nil {
		t.Fatal("tryAll with errors should return an error")
	}
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}

func TestSendCommandErrorPropagation(t *testing.T) {
	e, _ := newTestEPD()
	e.spi.(*fakeSPI).txErr = errors.New("spi down")
	s := &seq{e: e}
	s.sendCommand(0x12)
	if s.err == nil {
		t.Error("sendCommand should set err when SPI fails")
	}
}

func TestSendDataErrorPropagation(t *testing.T) {
	e, _ := newTestEPD()
	e.spi.(*fakeSPI).txErr = errors.New("spi down")
	s := &seq{e: e}
	s.sendData([]byte{0x00})
	if s.err == nil {
		t.Error("sendData should set err when SPI fails")
	}
}

func TestSleepUsesSleepFn(t *testing.T) {
	e, _ := newTestEPD()
	var got time.Duration
	e.sleepFn = func(d time.Duration) { got = d }
	s := &seq{e: e}
	s.sleep(42 * time.Millisecond)
	if got != 42*time.Millisecond {
		t.Errorf("sleepFn called with %v, want 42ms", got)
	}
}

func TestSleepShortCircuitsOnError(t *testing.T) {
	e, _ := newTestEPD()
	called := false
	e.sleepFn = func(time.Duration) { called = true }
	s := &seq{e: e, err: errors.New("prior")}
	s.sleep(time.Second)
	if called {
		t.Error("sleep should not run when err is already set")
	}
}

func TestWaitReadyImmediately(t *testing.T) {
	e, _ := newTestEPD() // busy defaults High
	s := &seq{e: e}
	s.wait()
	if s.err != nil {
		t.Errorf("wait() on ready bus = %v, want nil", s.err)
	}
}

func TestWaitBusyThenReleased(t *testing.T) {
	e, _ := newTestEPD()
	busy := e.busy.(*fakePin)
	calls := 0
	busy.readHook = func() gpio.Level {
		calls++
		if calls == 1 {
			return gpio.Low // busy on first poll
		}
		return gpio.High // released afterwards
	}
	busy.waitHook = func(time.Duration) bool { return true }
	s := &seq{e: e}
	s.wait()
	if s.err != nil {
		t.Errorf("wait() = %v, want nil", s.err)
	}
}

func TestWaitTimeout(t *testing.T) {
	e, _ := newTestEPD()
	busy := e.busy.(*fakePin)
	busy.readHook = func() gpio.Level { return gpio.Low } // never ready
	busy.waitHook = func(time.Duration) bool { return false }
	s := &seq{e: e}
	s.wait()
	if s.err == nil {
		t.Error("wait() should time out when BUSY stays low and no edge arrives")
	}
}

func TestEnterPartialMode(t *testing.T) {
	e, rec := newTestEPD()
	s := &seq{e: e}
	s.enterPartialMode()
	assertOps(t, rec.ops(), []op{{0x91, []byte{}}})
}

func TestDeepSleep(t *testing.T) {
	e, rec := newTestEPD()
	s := &seq{e: e}
	s.deepSleep()
	assertOps(t, rec.ops(), []op{{0x07, []byte{0xA5}}})
}

func TestSetPartialWindow(t *testing.T) {
	e, rec := newTestEPD()
	s := &seq{e: e}
	s.setPartialWindow(image.Rect(0, 0, 16, 8), GateScan_InsideAndOutside)
	if s.err != nil {
		t.Fatalf("setPartialWindow err: %v", s.err)
	}
	assertOps(t, rec.ops(), []op{
		{0x90, []byte{0x00, 0x00, 0x00, 0x0F, 0x00, 0x00, 0x00, 0x07, 0x01}},
	})
}

func TestSetPartialWindowErrors(t *testing.T) {
	t.Run("out of bounds", func(t *testing.T) {
		e, _ := newTestEPD()
		s := &seq{e: e}
		s.setPartialWindow(image.Rect(0, 0, 808, 480), GateScan_Inside)
		if s.err == nil {
			t.Error("expected error for window outside screen bounds")
		}
	})
	t.Run("x not multiple of 8", func(t *testing.T) {
		e, _ := newTestEPD()
		s := &seq{e: e}
		s.setPartialWindow(image.Rect(1, 0, 16, 8), GateScan_Inside)
		if s.err == nil {
			t.Error("expected error for non 8-aligned x")
		}
	})
}
