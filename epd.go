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

type PwrArgs byte

type BdEn PwrArgs // Border LDO enable

const (
	BdEnDisable BdEn = 0      // Border LDO disable
	BdEnEnable  BdEn = 1 << 4 // Border LDO enable
	BdEnDefault      = BdEnDisable
)

type VsrEn PwrArgs // Source LV power selection

const (
	VsrEnExternal VsrEn = 0      // External source power from VDHR pins
	VsrEnInternal VsrEn = 1 << 2 // Internal DC/DC function for generating VDHR.
	VsrEnDefault        = VsrEnInternal
)

type VsEn PwrArgs // Source power selection

const (
	VsEnExternal VsEn = 0      // External source power from VDH/VDL pins
	VsEnInternal VsEn = 1 << 1 // Internal DC/DC function for generating VDH/VDL.
	VsEnDefault       = VsEnInternal
)

type VgEn PwrArgs // Gate power selection

const (
	VgEnExternal = 0      // External gate power from VGH/VGL pins
	VgEnInternal = 1 << 0 // Internal DC/DC function for generating VGH/VGL.
	VgEnDefault  = VgEnInternal
)

type VppEn PwrArgs // OTP program power selection

const (
	VppEnExternalOTP VppEn = 0      // External OTP program power from VPP pin
	VppEnInternalOTP VppEn = 1 << 7 // OTP program power from internal power circuit.
)

type VCOMSlew PwrArgs // VCOM slew rate selection for voltage transition

const (
	VCOMSlewSlow VCOMSlew = 0 //Slow slew rate
	VCOMSlewFast VCOMSlew = 1 << 4
)

type VgLvl PwrArgs // VGH / VGL Voltage Level selection.

const (
	VgLvl9V  VgLvl = 0b000
	VgLvl10V VgLvl = 0b001
	VgLvl11V VgLvl = 0b010
	VgLvl12V VgLvl = 0b011
	VgLvl17V VgLvl = 0b100
	VgLvl18V VgLvl = 0b101
	VgLvl19V VgLvl = 0b110
	VgLvl20V VgLvl = 0b111
)

type VoltageLvl PwrArgs // Internal VDH power selection for K/W pixel.

const (
	VoltageLvl2_4V VoltageLvl = iota
	VoltageLvl2_6V
	VoltageLvl2_8V
	VoltageLvl3_0V
	VoltageLvl3_2V
	VoltageLvl3_4V
	VoltageLvl3_6V
	VoltageLvl3_8V
	VoltageLvl4_0V
	VoltageLvl4_2V
	VoltageLvl4_4V
	VoltageLvl4_6V
	VoltageLvl4_8V
	VoltageLvl5_0V
	VoltageLvl5_2V
	VoltageLvl5_4V
	VoltageLvl5_6V
	VoltageLvl5_8V
	VoltageLvl6_0V
	VoltageLvl6_2V
	VoltageLvl6_4V
	VoltageLvl6_6V
	VoltageLvl6_8V
	VoltageLvl7_0V
	VoltageLvl7_2V
	VoltageLvl7_4V
	VoltageLvl7_6V
	VoltageLvl7_8V
	VoltageLvl8_0V
	VoltageLvl8_2V
	VoltageLvl8_4V
	VoltageLvl8_6V
	VoltageLvl8_8V
	VoltageLvl9_0V
	VoltageLvl9_2V
	VoltageLvl9_4V
	VoltageLvl9_6V
	VoltageLvl9_8V
	VoltageLvl10_0V
	VoltageLvl10_2V
	VoltageLvl10_4V
	VoltageLvl10_6V
	VoltageLvl10_8V
	VoltageLvl11_0V
	VoltageLvl11_2V
	VoltageLvl11_4V
	VoltageLvl11_6V
	VoltageLvl11_8V
	VoltageLvl12_0V
	VoltageLvl12_2V
	VoltageLvl12_4V
	VoltageLvl12_6V
	VoltageLvl12_8V
	VoltageLvl13_0V
	VoltageLvl13_2V
	VoltageLvl13_4V
	VoltageLvl13_6V
	VoltageLvl13_8V
	VoltageLvl14_0V
	VoltageLvl14_2V
	VoltageLvl14_4V
	VoltageLvl14_6V
	VoltageLvl14_8V
	VoltageLvl15_0V
)

type VdhLvl VoltageLvl //  Internal VDH power selection for K/W pixel.

const VdhLvlDefault = VdhLvl(VoltageLvl14_0V)

type VdlLvl VoltageLvl // Internal VDL power selection for K/W pixel.

const VdlLvlDefault = VdlLvl(VoltageLvl14_0V)

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

func (e *Epd) Reset() error {
	s := &seq{e: e}
	s.reset()
	return s.err
}

func (s *seq) reset() {
	s.setPin(s.e.rst, gpio.High)
	s.sleep(20 * time.Millisecond)
	s.setPin(s.e.rst, gpio.Low)
	s.sleep(2 * time.Millisecond)
	s.setPin(s.e.rst, gpio.High)
	s.sleep(20 * time.Millisecond)
}

func (e *Epd) SendCommand(cmd Command) error {
	s := &seq{e: e}
	s.sendCommand(cmd)
	return s.err
}

func (e *Epd) SendData(data ...byte) error {
	s := &seq{e: e}
	s.sendData(data)
	return s.err
}

func (e *Epd) TurnOn() error {
	s := &seq{e: e}
	s.displayRefresh()
	s.sleep(10 * time.Millisecond)
	s.wait()
	return s.err
}

func (e *Epd) InitRegister() error {
	s := &seq{e: e}
	s.reset()
	s.powerSetting(
		BdEnDefault,
		VsrEnDefault,
		VsEnDefault,
		VgEnDefault,
		VppEnExternalOTP,
		VCOMSlewSlow,
		VgLvl20V,
		VdhLvl(VoltageLvl15_0V),
		VdlLvl(VoltageLvl15_0V),
	)
	s.boosterSoftStart()
	//TODO: Continue HERE!
}

func (e *Epd) DisplayRefresh() error {
	s := &seq{e: e}
	s.displayRefresh()
	return s.err
}

func (e *Epd) Close() error {
	return tryAll(
		"failed to close device",
		e.closeGPIO(),
		e.closeSPI(),
	)
}

func (e *Epd) closeGPIO() error {
	return tryAll(
		"failed to close GPIO; some pins might have been left on HIGH",
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

type seq struct {
	e   *Epd
	err error
}

func (s *seq) setPin(pin gpio.PinOut, level gpio.Level) {
	if s.err != nil {
		return
	}
	if err := pin.Out(level); err != nil {
		err = fmt.Errorf("failed to set %s to %s", pin.Name(), gpio.High)
	}
}

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

func (s *seq) sleep(dur time.Duration) {
	if s.err != nil {
		return
	}
	time.Sleep(dur)
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

func (s *seq) wait() {
	if s.err != nil {
		return
	}
	if !s.e.busy.WaitForEdge(time.Second * 10) {
		s.err = fmt.Errorf("waiting for %s input pin timed out", s.e.busy.Name())
	}
}

func (s *seq) displayRefresh() {
	s.sendCommand(CommandDRF)
}

func (s *seq) powerSetting(
	borderLDO BdEn,
	sourceLVPower VsrEn,
	sourcePower VsEn,
	gatePower VgEn,
	otpProgram VppEn,
	vcomSlew VCOMSlew,
	vgVoltage VgLvl,
	vdhKWVoltage VdhLvl,
	vdlKWVoltage VdlLvl,
) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandPWR)
	s.sendData([]byte{
		byte(borderLDO) | byte(sourceLVPower) | byte(sourcePower) | byte(gatePower),
		byte(otpProgram) | byte(vcomSlew) | byte(vgVoltage),
		byte(vdhKWVoltage),
		byte(vdlKWVoltage),
	})
}

func (s *seq) boosterSoftStart(settings BoosterSoftStartSettings) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandBTST)
	s.sendData(settings.Flags())
}

type BoosterSoftStartSettings struct {
	PhaseA  PhaseABSettings
	PhaseB  PhaseABSettings
	PhaseC1 PhaseCSettings
	PhaseC2 PhaseC2Settings
}

func (s BoosterSoftStartSettings) Flags() []byte {
	return []byte{
		s.PhaseA.Flags(),
		s.PhaseB.Flags(),
		s.PhaseC1.Flags(),
		s.PhaseC2.Flags(),
	}
}

type StartPeriod byte

const (
	StartPeriod_10ms StartPeriod = 0b00
	StartPeriod_20ms StartPeriod = 0b01
	StartPeriod_30ms StartPeriod = 0b10
	StartPeriod_40ms StartPeriod = 0b11
)

type Strength byte

const (
	Strength_1 Strength = 0b000
	Strength_2 Strength = 0b001
	Strength_3 Strength = 0b010
	Strength_4 Strength = 0b011
	Strength_5 Strength = 0b100
	Strength_6 Strength = 0b101
	Strength_7 Strength = 0b110
	Strength_8 Strength = 0b111
)

type OffDuration byte

const (
	OffDuration_0_27us OffDuration = 0b000
	OffDuration_0_34us OffDuration = 0b001
	OffDuration_0_40us OffDuration = 0b010
	OffDuration_0_54us OffDuration = 0b011
	OffDuration_0_80us OffDuration = 0b100
	OffDuration_1_54us OffDuration = 0b101
	OffDuration_3_34us OffDuration = 0b110
	OffDuration_6_58us OffDuration = 0b111
)

type PhaseSettings struct {
	DrivingStrength   Strength
	MinGDROffDuration OffDuration
}

func (s PhaseSettings) Flags() byte {
	return byte(s.DrivingStrength)<<3 | byte(s.MinGDROffDuration)
}

type PhaseABSettings struct {
	SoftStartPeriod StartPeriod
	PhaseSettings
}

func (s PhaseABSettings) Flags() byte {
	return byte(s.SoftStartPeriod)<<6 | s.PhaseSettings.Flags()
}

type PhaseCSettings = PhaseSettings

type PhaseC2Settings struct {
	Enabled bool
	PhaseCSettings
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func (s PhaseC2Settings) Flags() byte {
	return boolToByte(s.Enabled)<<7 | s.PhaseCSettings.Flags()
}
