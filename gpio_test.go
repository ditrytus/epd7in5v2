package epd7in5v2

import (
	"errors"
	"strconv"
	"testing"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
)

func TestSetPin(t *testing.T) {
	e, _ := newTestEPD()
	pin := &fakePin{name: "X"}
	s := &seq{e: e}
	s.setPin(pin, gpio.High)
	if s.err != nil {
		t.Fatalf("setPin err: %v", s.err)
	}
	if len(pin.outs) != 1 || pin.outs[0] != gpio.High {
		t.Errorf("pin.outs = %v, want [High]", pin.outs)
	}
}

func TestSetPinError(t *testing.T) {
	e, _ := newTestEPD()
	pin := &fakePin{name: "X", outErr: errors.New("io")}
	s := &seq{e: e}
	s.setPin(pin, gpio.High)
	if s.err == nil {
		t.Error("setPin should propagate Out error")
	}
}

func TestSetPinShortCircuit(t *testing.T) {
	e, _ := newTestEPD()
	pin := &fakePin{name: "X"}
	s := &seq{e: e, err: errors.New("prior")}
	s.setPin(pin, gpio.High)
	if len(pin.outs) != 0 {
		t.Error("setPin must not write when err is already set")
	}
}

func TestInitPinOutNotFound(t *testing.T) {
	e, _ := newTestEPD()
	s := &seq{e: e}
	var p gpio.PinOut
	s.initPinOut(&p, 999999, gpio.Low)
	if s.err == nil {
		t.Error("initPinOut should error for an unknown pin")
	}
}

func TestInitPinInNotFound(t *testing.T) {
	e, _ := newTestEPD()
	s := &seq{e: e}
	var p gpio.PinIn
	s.initPinIn(&p, 999998, gpio.PullNoChange, gpio.RisingEdge)
	if s.err == nil {
		t.Error("initPinIn should error for an unknown pin")
	}
}

func TestInitPinOutSuccess(t *testing.T) {
	num := 49001
	fp := &fakePin{name: strconv.Itoa(num), num: num}
	if err := gpioreg.Register(fp); err != nil {
		t.Fatalf("register: %v", err)
	}
	e, _ := newTestEPD()
	s := &seq{e: e}
	var p gpio.PinOut
	s.initPinOut(&p, num, gpio.High)
	if s.err != nil {
		t.Fatalf("initPinOut err: %v", s.err)
	}
	if len(fp.outs) != 1 || fp.outs[0] != gpio.High {
		t.Errorf("init level not applied: outs=%v", fp.outs)
	}
}

func TestInitPinInSuccess(t *testing.T) {
	num := 49002
	fp := &fakePin{name: strconv.Itoa(num), num: num}
	if err := gpioreg.Register(fp); err != nil {
		t.Fatalf("register: %v", err)
	}
	e, _ := newTestEPD()
	s := &seq{e: e}
	var p gpio.PinIn
	s.initPinIn(&p, num, gpio.PullNoChange, gpio.RisingEdge)
	if s.err != nil {
		t.Fatalf("initPinIn err: %v", s.err)
	}
	if fp.inCalls != 1 {
		t.Errorf("In() calls = %d, want 1", fp.inCalls)
	}
}

func TestInitPinInError(t *testing.T) {
	num := 49003
	fp := &fakePin{name: strconv.Itoa(num), num: num, inErr: errors.New("in failed")}
	if err := gpioreg.Register(fp); err != nil {
		t.Fatalf("register: %v", err)
	}
	e, _ := newTestEPD()
	s := &seq{e: e}
	var p gpio.PinIn
	s.initPinIn(&p, num, gpio.PullNoChange, gpio.RisingEdge)
	if s.err == nil {
		t.Error("initPinIn should propagate In() error")
	}
}

func TestCloseGPIO(t *testing.T) {
	e, _ := newTestEPD()
	if err := e.closeGPIO(); err != nil {
		t.Errorf("closeGPIO = %v, want nil", err)
	}
	// All three output pins should have been driven Low.
	for _, name := range []string{"PWR", "DC", "RST"} {
		var fp *fakePin
		switch name {
		case "PWR":
			fp = e.pwr.(*fakePin)
		case "DC":
			fp = e.dc.(*fakePin)
		case "RST":
			fp = e.rst.(*fakePin)
		}
		if len(fp.outs) == 0 || fp.outs[len(fp.outs)-1] != gpio.Low {
			t.Errorf("%s last write = %v, want Low", name, fp.outs)
		}
	}
}

func TestCloseGPIOError(t *testing.T) {
	e, _ := newTestEPD()
	e.pwr.(*fakePin).outErr = errors.New("stuck")
	if err := e.closeGPIO(); err == nil {
		t.Error("closeGPIO should report a failing pin")
	}
}

// GPIOInit requires the e-paper HAT (real GPIO + SPI). In a host environment
// without it, the GPIO pins aren't registered, so it must fail rather than
// panic, and must not leave a half-open SPI connection.
func TestGPIOInitWithoutHardware(t *testing.T) {
	e := NewEPD()
	if err := e.GPIOInit(); err == nil {
		t.Skip("GPIOInit unexpectedly succeeded; real hardware present")
	}
	if e.spi != nil {
		t.Error("GPIOInit failed but left a non-nil SPI connection")
	}
}

func TestCloseSPIAndClose(t *testing.T) {
	e, _ := newTestEPD()
	closer := &fakeCloser{}
	e.spiCloser = closer
	if err := e.Close(); err != nil {
		t.Errorf("Close = %v, want nil", err)
	}
	if !closer.closed {
		t.Error("Close should close the SPI port")
	}

	e2, _ := newTestEPD()
	e2.spiCloser = &fakeCloser{err: errors.New("close failed")}
	if err := e2.Close(); err == nil {
		t.Error("Close should report SPI close failure")
	}
}
