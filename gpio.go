package epd7in5v2

import (
	"errors"
	"fmt"
	"strconv"

	"periph.io/x/conn/v3/driver/driverreg"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	// Registers the host GPIO/SPI drivers (e.g. bcm283x on the Raspberry Pi) so
	// driverreg.Init() can find them; without this no pins are registered and
	// gpioreg.ByName returns nil for every pin.
	_ "periph.io/x/host/v3"
)

// Pin assignments, as BCM GPIO numbers, matching the Waveshare e-Paper Driver
// HAT's standard Raspberry Pi wiring.
//
// Only [PinDC], [PinRST], [PinBUSY] and [PinPWR] are claimed as GPIO by
// [Epd.GPIOInit]. [PinCS], [PinMOSI] and [PinSCLK] belong to the SPI
// peripheral, which drives them itself, and are listed here for reference only
// - claiming them as GPIO would take them away from the SPI driver.
const (
	PinRST  = 17 // reset, active low
	PinDC   = 25 // data/command select: low for a command, high for its data
	PinCS   = 8  // chip select (SPI0 CE0), driven by the SPI peripheral
	PinPWR  = 18 // panel power enable
	PinBUSY = 24 // panel busy, low while a refresh is running
	PinMOSI = 10 // SPI data out, driven by the SPI peripheral
	PinSCLK = 11 // SPI clock, driven by the SPI peripheral
)

// GPIOInit claims the GPIO pins and opens the SPI port.
//
// It registers the periph.io host drivers, takes DC, RST, BUSY and PWR, and
// opens the first available SPI port at 10 MHz in mode 0, 8 bits per word.
// Chip select is left to the SPI driver, which frames it around each transfer.
//
// Call it once, after [NewEPD] and before any init method. On failure it
// releases whatever it had already taken, so the [Epd] can be discarded. On
// success the caller owns the pins and the port until [Epd.Close].
func (e *Epd) GPIOInit() error {
	if _, err := driverreg.Init(); err != nil {
		return fmt.Errorf("failed to initialize driver register: %w", err)
	}

	s := &seq{e: e}
	s.initPinOut(&e.rst, PinRST, gpio.Low)
	s.initPinOut(&e.dc, PinDC, gpio.Low)
	s.initPinOut(&e.pwr, PinPWR, gpio.High)
	s.initPinIn(&e.busy, PinBUSY, gpio.PullNoChange, gpio.RisingEdge)
	if s.err != nil {
		return fmt.Errorf("failed to initialize GPIO: %w", s.err)
	}

	p, err := spireg.Open("")
	if err != nil {
		return errors.Join(
			e.closeGPIO(),
			fmt.Errorf("failed to open SPI port: %w", err),
		)
	}
	e.spiCloser = p

	e.spi, err = p.Connect(10*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		return errors.Join(
			e.closeSPI(),
			fmt.Errorf("failed to initialize SPI: %w", err),
		)
	}
	return nil
}

// closeGPIO drives every output pin low. It joins the failures rather than
// stopping at the first, so one stuck pin does not leave the others high.
func (e *Epd) closeGPIO() error {
	if e.pwr == nil && e.dc == nil && e.rst == nil {
		return nil
	}
	return tryAll(
		"failed to close GPIO; some pins might have been left on HIGH",
		e.pwr.Out(gpio.Low),
		e.dc.Out(gpio.Low),
		e.rst.Out(gpio.Low),
	)
}

// setPin drives pin to level, recording the first failure in the sequence.
func (s *seq) setPin(pin gpio.PinOut, level gpio.Level) {
	if s.err != nil {
		return
	}
	if err := pin.Out(level); err != nil {
		s.err = fmt.Errorf("failed to set %s to %s", pin.Name(), gpio.High)
	}
}

// initPinOut looks up GPIO pinNum, stores it in pin as an output and drives it
// to init.
func (s *seq) initPinOut(pin *gpio.PinOut, pinNum int, init gpio.Level) {
	pinIO := gpioreg.ByName(strconv.Itoa(pinNum))
	if pinIO == nil {
		s.err = fmt.Errorf("failed to find GPIO%d", pinNum)
		return
	}
	var ok bool
	*pin, ok = pinIO.(gpio.PinOut)
	if !ok {
		s.err = fmt.Errorf("invalid pin type of GPIO%d, expected %T", pinNum, pin)
		return
	}
	s.setPin(*pin, init)
}

// initPinIn looks up GPIO pinNum and stores it in pin as an input configured
// with the given pull and edge detection.
func (s *seq) initPinIn(pin *gpio.PinIn, pinNum int, pull gpio.Pull, edge gpio.Edge) {
	pinIO := gpioreg.ByName(strconv.Itoa(pinNum))
	if pinIO == nil {
		s.err = fmt.Errorf("failed to find GPIO%d", pinNum)
		return
	}
	var ok bool
	*pin, ok = pinIO.(gpio.PinIn)
	if !ok {
		s.err = fmt.Errorf("invalid pin type of GPIO%d, expected %T", pinNum, pin)
		return
	}
	if err := (*pin).In(pull, edge); err != nil {
		s.err = fmt.Errorf("failed to initialize input pin GPIO%d: %w", pinNum, err)
		return
	}
}
