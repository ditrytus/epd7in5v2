package epd7in5v2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"periph.io/x/conn/v3/driver/driverreg"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
)

var gateVoltageAllowedValuesMsgPart string

func init() {
	var b strings.Builder
	first := true
	for voltage := range gateVoltageToFlagsMap {
		if !first {
			b.WriteString(", ")
		}
		b.WriteString(voltage.String())
		first = false
	}
	gateVoltageAllowedValuesMsgPart = b.String()
}

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

const (
	ScreenWidth  Resolution = 800
	ScreenHeight Resolution = 480
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
	ps := NewDefaultPowerSettings()
	ps.BlackWhiteVoltageDrain = VoltageRange{
		High: 20 * Volt,
		Low:  20 * Volt,
	}
	s.powerSetting(ps)
	s.boosterSoftStart(BoosterSoftStartSettings{
		PhaseA: PhaseABSettings{
			SoftStartPeriod: StartPeriod_10ms,
			PhaseSettings: PhaseSettings{
				DrivingStrength:   Strength_3,
				MinGDROffDuration: OffDuration_6_58us,
			},
		},
		PhaseB: PhaseABSettings{
			SoftStartPeriod: StartPeriod_10ms,
			PhaseSettings: PhaseSettings{
				DrivingStrength:   Strength_3,
				MinGDROffDuration: OffDuration_6_58us,
			},
		},
		PhaseC1: PhaseCSettings{
			DrivingStrength:   Strength_6,
			MinGDROffDuration: OffDuration_0_27us,
		},
		PhaseC2: PhaseC2Settings{
			Enabled: false,
			PhaseCSettings: PhaseSettings{
				DrivingStrength:   Strength_3,
				MinGDROffDuration: OffDuration_6_58us,
			},
		},
	})
	s.powerON()
	panelSettings := NewDefaultPanelSettings()
	panelSettings.ColorMode = ColorMode_BlackWhite
	s.panelSetting(panelSettings)
	s.resolutionSetting(ResolutionSettings{
		Horizontal: ScreenWidth,
		Vertical:   ScreenHeight,
	})
	s.commonVoltageAndDataIntervalSetting(CDISettings{
		ColorMode: BlackWhiteSettings{
			Refresh: DifferentialRefresh{
				CopyNewToOld: false,
			},
			Border: Driven[BlackWhiteBorder]{
				LookupTable: BlackWhiteBorder_BlackToWhite,
			},
		},
		BlackWhitePolarity:        BlackWhitePolarity_ZeroIsWhite,
		CommonVoltageDataInterval: 10 * HSync,
	})
	return s.err
}

type HSyncInterval int

const HSync HSyncInterval = 1

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

type PowerSettings struct {
	BorderLowDropoutEnabled bool
	SourceLowVoltagePower   Externality
	SourcePower             Externality
	GatePower               Externality
	OTPProgram              Externality
	CommonVoltageSlew       Speed
	GateVoltage             SymmetricVoltageRange
	BlackWhiteVoltageDrain  VoltageRange
}

func (ps PowerSettings) Flags() ([]byte, error) {
	vg, err := gateVoltageFlags(ps.GateVoltage.High())
	if err != nil {
		return nil, fmt.Errorf("invalid gate voltage: %w", err)
	}
	vdh, err := voltageDrainFlag(ps.BlackWhiteVoltageDrain.High, VoltageDrainHighMin, VoltageDrainHighMax, VoltageDrainHighFlagBits)
	if err != nil {
		return nil, fmt.Errorf("invalid voltage drain high for black and white pixel: %w", err)
	}
	vdl, err := voltageDrainFlag(ps.BlackWhiteVoltageDrain.Low, VoltageDrainLowMin, VoltageDrainLowMax, VoltageDrainLowFlagBits)
	if err != nil {
		return nil, fmt.Errorf("invalid voltage drain low for black and white pixel: %w", err)
	}
	return []byte{
		boolToByte(ps.BorderLowDropoutEnabled)<<4 |
			byte(ps.SourceLowVoltagePower)<<2 |
			byte(ps.SourcePower)<<1 |
			byte(ps.GatePower),
		byte(ps.OTPProgram)<<7 | byte(ps.CommonVoltageSlew)<<4 | vg,
		vdh,
		vdl,
	}, nil
}

func NewDefaultPowerSettings() PowerSettings {
	return PowerSettings{
		BorderLowDropoutEnabled: false,
		SourceLowVoltagePower:   Internal,
		SourcePower:             Internal,
		GatePower:               Internal,
		OTPProgram:              External,
		CommonVoltageSlew:       Slow,
		GateVoltage:             SymmetricVoltageRange(20 * Volt),
		BlackWhiteVoltageDrain: VoltageRange{
			High: 14 * Volt,
			Low:  14 * Volt,
		},
	}
}

type Externality byte

const (
	External Externality = 0
	Internal Externality = 1
)

type Speed byte

const (
	Slow Speed = 0
	Fast Speed = 1
)

type Voltage float32

func (v Voltage) String() string {
	return fmt.Sprintf("%fV", v)
}

const Volt Voltage = 1

type VoltageRange struct {
	High Voltage
	Low  Voltage
}

type SymmetricVoltageRange Voltage

func (sv SymmetricVoltageRange) High() Voltage {
	if sv < 0 {
		return Voltage(-sv)
	}
	return Voltage(sv)
}

func (sv SymmetricVoltageRange) Low() Voltage {
	if sv > 0 {
		return Voltage(-sv)
	}
	return Voltage(sv)
}

var gateVoltageToFlagsMap = map[Voltage]byte{
	9 * Volt:  0b000,
	10 * Volt: 0b001,
	11 * Volt: 0b010,
	12 * Volt: 0b011,
	17 * Volt: 0b100,
	18 * Volt: 0b101,
	19 * Volt: 0b110,
	20 * Volt: 0b111,
}

func gateVoltageFlags(v Voltage) (byte, error) {
	flags, ok := gateVoltageToFlagsMap[v]
	if !ok {
		return 0, fmt.Errorf("voltage must be one of: %s", gateVoltageAllowedValuesMsgPart)
	}
	return flags, nil
}

const VoltageDrainHighMin = 2.4 * Volt
const VoltageDrainHighMax = 15 * Volt
const VoltageDrainHighFlagBits = 6

const VoltageDrainLowMin = -VoltageDrainHighMax
const VoltageDrainLowMax = -VoltageDrainHighMin
const VoltageDrainLowFlagBits = VoltageDrainHighFlagBits

func voltageDrainFlag(v Voltage, min, max Voltage, bits uint) (byte, error) {
	if v < min {
		return 0, fmt.Errorf("voltage must not be lower than %s", min)
	}
	if v > max {
		return 0, fmt.Errorf("voltage must not be greater than %s", max)
	}
	delta := max - min
	totalSteps := math.Pow(2, float64(bits)) - 1
	step := float64(delta) / totalSteps
	flag, fraction := math.Modf(math.Mod(float64(v-min), step))
	if fraction != 0 {
		return 0, fmt.Errorf("voltage must be a multiple of %f", step)
	}
	return byte(flag), nil
}

func (s *seq) powerSetting(
	settings PowerSettings,
) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandPWR)
	data, err := settings.Flags()
	if err != nil {
		s.err = err
		return
	}
	s.sendData(data)
}

func (s *seq) boosterSoftStart(settings BoosterSoftStartSettings) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandBTST)
	s.sendData(settings.Flags())
}

func (s *seq) powerON() {
	s.sendCommand(CommandPON)
	s.sleep(100 * time.Millisecond)
	s.wait()
}

type LookupTableSource byte

const (
	LookupTableSource_OTP      LookupTableSource = 0
	LookupTableSource_Register LookupTableSource = 1
)

type ColorMode byte

const (
	ColorMode_BlackWhiteRed ColorMode = 0
	ColorMode_BlackWhite    ColorMode = 1
)

type VerticalDirection byte

const (
	Up   VerticalDirection = 0
	Down VerticalDirection = 1
)

type HorizontalDirection byte

const (
	Left  HorizontalDirection = 0
	Right HorizontalDirection = 1
)

type ResetBehavior byte

const (
	Reset    ResetBehavior = 0
	NoEffect ResetBehavior = 1
)

type PanelSettings struct {
	LookupTableSource    LookupTableSource
	ColorMode            ColorMode
	GateScanDirection    VerticalDirection
	SourceShiftDirection HorizontalDirection
	BoosterEnabled       bool
	SoftReset            ResetBehavior
}

func NewDefaultPanelSettings() PanelSettings {
	return PanelSettings{
		LookupTableSource:    LookupTableSource_OTP,
		ColorMode:            ColorMode_BlackWhiteRed,
		GateScanDirection:    Up,
		SourceShiftDirection: Right,
		BoosterEnabled:       true,
		SoftReset:            NoEffect,
	}
}

func (s PanelSettings) Flags() byte {
	return byte(s.LookupTableSource)<<5 |
		byte(s.ColorMode)<<4 |
		byte(s.GateScanDirection)<<3 |
		byte(s.SourceShiftDirection)<<2 |
		boolToByte(s.BoosterEnabled)<<1 |
		byte(s.SoftReset)
}

func (s *seq) panelSetting(settings PanelSettings) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandPSR)
	s.sendData([]byte{settings.Flags()})
}

type Resolution uint16

const (
	MinHorizontalRes Resolution = 1
	MaxHorizontalRes Resolution = 800
	MinVerticalRes   Resolution = 1
	MaxVerticalRes   Resolution = 600
)

type ResolutionSettings struct {
	Horizontal Resolution
	Vertical   Resolution
}

func (rs ResolutionSettings) Flags() ([]byte, error) {
	if rs.Horizontal < MinHorizontalRes {
		return nil, fmt.Errorf("horizontal resolution must not be lower than %d", MinHorizontalRes)
	}
	if rs.Horizontal > MaxHorizontalRes {
		return nil, fmt.Errorf("horizontal resolution must not be greater than %d", MaxHorizontalRes)
	}
	if rs.Vertical < MinVerticalRes {
		return nil, fmt.Errorf("vertical resolution must not be lower than %d", MinVerticalRes)
	}
	if rs.Horizontal > MaxHorizontalRes {
		return nil, fmt.Errorf("vertical resolution must not be greater than %d", MaxVerticalRes)
	}
	if rs.Horizontal%8 != 0 {
		return nil, fmt.Errorf("horizontal resolution must be divisible by 8")
	}
	flags := make([]byte, 0, 4)
	binary.BigEndian.AppendUint16(flags, uint16(rs.Horizontal))
	binary.BigEndian.AppendUint16(flags, uint16(rs.Vertical))
	return flags, nil
}

func (s *seq) resolutionSetting(settings ResolutionSettings) {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandTRES)
	data, err := settings.Flags()
	if s.err == nil && err != nil {
		s.err = err
		return
	}
	s.sendData(data)

}

type ColorModeSettings interface {
	isColorModeSettings()
}

type BlackWhiteRedSettings struct {
	RedPolarity RedPolarity
	Border      Border[BlackWhiteRedBorder]
}

func (s BlackWhiteRedSettings) isColorModeSettings() {}

type Refresh interface {
	isRefresh()
}

type FullRefresh struct{}

func (f FullRefresh) isRefresh() {}

type DifferentialRefresh struct {
	CopyNewToOld bool
}

func (d DifferentialRefresh) isRefresh() {}

type BlackWhiteSettings struct {
	Refresh Refresh
	Border  Border[BlackWhiteBorder]
}

func (s BlackWhiteSettings) isColorModeSettings() {}

type BlackWhitePolarity byte

const (
	BlackWhitePolarity_ZeroIsBlack BlackWhitePolarity = iota
	BlackWhitePolarity_ZeroIsWhite
)

type RedPolarity byte

type ColorModeBorder interface {
	isColorModeBorder()
}

type BlackWhiteRedBorder byte

func (b BlackWhiteRedBorder) isColorModeBorder() {}

const (
	BlackWhiteRedBorder_Border BlackWhiteRedBorder = iota
	BlackWhiteRedBorder_Red
	BlackWhiteRedBorder_White
	BlackWhiteRedBorder_Black
)

type BlackWhiteBorder byte

func (b BlackWhiteBorder) isColorModeBorder() {}

const (
	BlackWhiteBorder_Border BlackWhiteBorder = iota
	BlackWhiteBorder_BlackToWhite
	BlackWhiteBorder_WhiteToBlack
	BlackWhiteBorder_BlackToBlack
)

type Border[T ColorModeBorder] interface {
	isBorder()
}

type Float struct{}

func (f Float) isBorder() {}

type Driven[T ColorModeBorder] struct {
	LookupTable T
}

func (f Driven[T]) isBorder() {}

const (
	ZeroIsRed RedPolarity = iota
	OneIsRed
)

type CommonVoltageDataInterval byte

type CDISettings struct {
	ColorMode                 ColorModeSettings
	BlackWhitePolarity        BlackWhitePolarity
	CommonVoltageDataInterval HSyncInterval
}

func (s *seq) commonVoltageAndDataIntervalSetting(settings CDISettings) {

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
