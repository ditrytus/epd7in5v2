package epd7in5v2

import (
	"errors"
	"fmt"
	"strconv"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/pin"
)

const (
	PinRST  = 17
	PinDC   = 25
	PicCS   = 8
	PinPWR  = 18
	PinBUSY = 24
	PinMOSI = 10
	PinSCLK = 11
)

type Epd struct {
	rst  gpio.PinOut
	dc   gpio.PinOut
	cs   gpio.PinOut
	pwr  gpio.PinOut
	busy gpio.PinIn
	mosi gpio.PinOut
	sclk gpio.PinOut
}

func NewEPD() *Epd {
	return &Epd{}
}

func (e *Epd) Init() error {
	return all(
		"failed to initialize GPIO pins",
		initPin(&e.rst, 17),
		initPin(&e.dc, 25),
		initPin(&e.cs, 8),
		initPin(&e.pwr, 18),
		initPin(&e.busy, 24),
		initPin(&e.mosi, 10),
		initPin(&e.sclk, 11),
		e.cs.Out(gpio.High),
		e.pwr.Out(gpio.High),
	)
}

func (e *Epd) Close() error {
	return all(
		"failed to uninitialize GPIO; some pins might have been left on HIGH",
		e.cs.Out(gpio.Low),
		e.pwr.Out(gpio.Low),
		e.dc.Out(gpio.Low),
		e.rst.Out(gpio.Low),
	)
}

func all(errMsg string, errs ...error) error {
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	return nil
}

func initPin[T pin.Pin](pin *T, pinNum int) error {
	pinIO := gpioreg.ByName(strconv.Itoa(pinNum))
	if pinIO == nil {
		return fmt.Errorf("failed to find GPIO%d", pinNum)
	}
	var ok bool
	*pin, ok = pinIO.(T)
	if !ok {
		return fmt.Errorf("invalid pin type of GPIO%d", pinNum)
	}
	return nil
}
