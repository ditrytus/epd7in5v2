package epd7in5v2

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"periph.io/x/conn/v3/driver/driverreg"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
)

const (
	PinRST  = 17
	PinDC   = 25
	PinCS   = 8
	PinPWR  = 18
	PinBUSY = 24
	PinMOSI = 10
	PinSCLK = 11
)

type Command byte

const (
	CommandPSR    = 0x00 // Panel Setting
	CommandPWR    = 0x01 // Power Setting
	CommandPOF    = 0x02 // Power OFF
	CommandPFS    = 0x03 // Power OFF Sequence Setting
	CommandPON    = 0x04 // Power ON
	CommandPMES   = 0x05 // Power ON Measure
	CommandBTST   = 0x06 // Booster Soft Start
	CommandDSLP   = 0x07 // Deep sleep
	CommandDTM1   = 0x10 // Display Start Transmission 1
	CommandDSP    = 0x11 // Data Stop
	CommandDRF    = 0x12 // Display Refresh
	CommandDTM2   = 0x13 // Display Start Transmission 2
	CommandDUSPI  = 0x15 // Dual SPI
	CommandAUTO   = 0x17 // Auto Sequence
	CommandPLL    = 0x30 // PLL control
	CommandTSC    = 0x40 // Temperature Sensor Calibration
	CommandTSE    = 0x41 // Temperature Sensor Selection
	CommandTSW    = 0x42 // Temperature Sensor Write
	CommandTSR    = 0x43 // Temperature Sensor Read
	CommandPBC    = 0x44 // Panel Break Check
	CommandCDI    = 0x50 // VCOM and data interval setting
	CommandLPD    = 0x51 // Lower Power Detection
	CommandEVS    = 0x52 // End Voltage Settings
	CommandTCON   = 0x60 // TCON setting
	CommandTRES   = 0x61 // Resolution setting
	CommandGSST   = 0x65 // Gate/Source Start setting
	CommandREV    = 0x70 // Revision
	CommandFLG    = 0x71 // Get Status
	CommandAMV    = 0x80 // Auto Measurement VCOM
	CommandVV     = 0x81 // Read VCOM Value
	CommandVDSC   = 0x82 // VCOM_DC Setting
	CommandPTL    = 0x90 // Partial Window
	CommandPTIN   = 0x91 // Partial In
	CommandPTOUT  = 0x92 // Partial Out
	CommandCCSET  = 0xE0 // Program Mode
	CommandPWS    = 0xE3 // Power Saving
	CommandLVSEL  = 0xE4 // LVD Voltage Select
	CommandTSSET  = 0xE5 // Force Temperature
	CommandTSDBRY = 0xE7 // Temperature Boundary Phase-C2
)

type Epd struct {
	rst       gpio.PinOut
	dc        gpio.PinOut
	pwr       gpio.PinOut
	busy      gpio.PinIn
	spi       spi.Conn
	spiCloser io.Closer
}

func NewEPD() *Epd {
	return &Epd{}
}

func (e *Epd) Init() error {
	if _, err := driverreg.Init(); err != nil {
		return fmt.Errorf("failed to initialize driver register: %w", err)
	}

	if err := tryAll(
		"failed to initialize GPIO pins",
		initPinOut(&e.rst, PinRST, gpio.Low),
		initPinOut(&e.dc, PinDC, gpio.Low),
		initPinOut(&e.pwr, PinPWR, gpio.High),
		initPinIn(&e.busy, PinBUSY, gpio.PullNoChange, gpio.RisingEdge),
		e.pwr.Out(gpio.High),
	); err != nil {
		return err
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

func (e *Epd) Reset() error {
	if err := setPin(e.rst, gpio.High); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	if err := setPin(e.rst, gpio.Low); err != nil {
		return err
	}
	time.Sleep(2 * time.Millisecond)
	if err := setPin(e.rst, gpio.High); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	return nil
}

func (e *Epd) SendCommand(cmd Command) error {
	if err := setPin(e.dc, gpio.Low); err != nil {
		return err
	}
	if err := e.spi.Tx([]byte{byte(cmd)}, nil); err != nil {
		return fmt.Errorf("failed to send command %s over SPI: %w", cmd, err)
	}
	return nil
}

func (e *Epd) SendData(data ...byte) error {
	if err := setPin(e.dc, gpio.High); err != nil {
		return err
	}
	if err := e.spi.Tx(data, nil); err != nil {
		return fmt.Errorf("failed to send data (%d bytes) over SPI: %w", len(data), err)
	}
	return nil
}

func (e *Epd) Wait() {
	e.busy.WaitForEdge(time.Second * 10)
}

func (e *Epd) Close() error {
	return tryAll(
		"failed to close device",
		e.closeGPIO(),
		e.spiCloser.Close(),
	)
}

func (e *Epd) closeGPIO() error {
	return tryAll(
		"failed to close GPIO; some pins might have been left on HIGH",
		//e.cs.Out(gpio.Low),
		e.pwr.Out(gpio.Low),
		e.dc.Out(gpio.Low),
		e.rst.Out(gpio.Low),
	)
}

func (e *Epd) closeSPI() error {
	if err := e.spiCloser.Close(); err != nil {
		return fmt.Errorf("failed to close SPI port: %w", err)
	}
	return nil
}

func tryAll(errMsg string, errs ...error) error {
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	return nil
}

func setPin(pin gpio.PinOut, level gpio.Level) error {
	if err := pin.Out(level); err != nil {
		return fmt.Errorf("failed to set %s to %s", pin.Name(), gpio.High)
	}
	return nil
}

func initPinOut(pin *gpio.PinOut, pinNum int, init gpio.Level) error {
	pinIO := gpioreg.ByName(strconv.Itoa(pinNum))
	if pinIO == nil {
		return fmt.Errorf("failed to find GPIO%d", pinNum)
	}
	var ok bool
	*pin, ok = pinIO.(gpio.PinOut)
	if !ok {
		return fmt.Errorf("invalid pin type of GPIO%d, expected %T", pinNum, pin)
	}
	if err := setPin(*pin, init); err != nil {
		return fmt.Errorf("failed to initialize output pin GPIO%d: %w", pinNum, err)
	}
	return nil
}

func initPinIn(pin *gpio.PinIn, pinNum int, pull gpio.Pull, edge gpio.Edge) error {
	pinIO := gpioreg.ByName(strconv.Itoa(pinNum))
	if pinIO == nil {
		return fmt.Errorf("failed to find GPIO%d", pinNum)
	}
	var ok bool
	*pin, ok = pinIO.(gpio.PinIn)
	if !ok {
		return fmt.Errorf("invalid pin type of GPIO%d, expected %T", pinNum, pin)
	}
	if err := (*pin).In(pull, edge); err != nil {
		return fmt.Errorf("failed to initialize input pin GPIO%d: %w", pinNum, err)
	}
	return nil
}
