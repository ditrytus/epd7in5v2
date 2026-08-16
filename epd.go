package epd7in5v2

import (
	"fmt"
	"image"
	"io"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/spi"
)

const (
	// ScreenWidth is the panel's width in pixels.
	ScreenWidth Resolution = 800
	// ScreenHeight is the panel's height in pixels.
	ScreenHeight Resolution = 480
)

// Epd is a connection to the panel.
//
// The zero value is not usable; get one from [NewEPD] and then call
// [Epd.GPIOInit] to claim the hardware. An Epd is not safe for concurrent use:
// its methods drive a shared bus and pins, so serialise access from one
// goroutine or guard it with your own lock.
type Epd struct {
	rst       gpio.PinOut
	dc        gpio.PinOut
	pwr       gpio.PinOut
	busy      gpio.PinIn
	spi       spi.Conn
	spiCloser io.Closer
	// sleepFn performs blocking delays. It is overridable for testing; when nil
	// it falls back to time.Sleep.
	sleepFn func(time.Duration)
}

// NewEPD returns an [Epd] that is not yet connected to anything. Call
// [Epd.GPIOInit] next to claim the pins and open the SPI port.
func NewEPD() *Epd {
	return &Epd{sleepFn: time.Sleep}
}

// Reset toggles the panel's reset line with the timing the datasheet requires.
//
// Every init method does this for you; call it directly only to recover a
// controller that has stopped responding.
func (e *Epd) Reset() error {
	s := &seq{e: e}
	s.reset()
	return s.err
}

// reset pulses RST low and back, holding each level long enough for the
// controller to notice.
func (s *seq) reset() {
	s.setPin(s.e.rst, gpio.High)
	s.sleep(20 * time.Millisecond)
	s.setPin(s.e.rst, gpio.Low)
	s.sleep(2 * time.Millisecond)
	s.setPin(s.e.rst, gpio.High)
	s.sleep(20 * time.Millisecond)
}

// TurnOn refreshes the display and waits for the panel to settle.
//
// It is [Epd.DisplayRefresh] with an extra settling delay, for callers that
// have written the image planes themselves.
func (e *Epd) TurnOn() error {
	s := &seq{e: e}
	s.displayRefresh()
	s.sleep(100 * time.Millisecond)
	s.wait()
	return s.err
}

// Init resets the panel and prepares it for full-quality full-screen refreshes,
// which take roughly four seconds.
//
// It configures power, booster, panel, resolution, SPI mode, VCOM interval and
// TCON timing, and leaves the temperature reading to the panel's own sensor, so
// the waveform matches the ambient conditions. Use [Epd.InitFast] to trade
// contrast for speed, or [Epd.InitPart] for partial updates.
//
// Init also wakes the panel from the deep sleep that [Epd.Sleep] puts it in.
func (e *Epd) Init() error {
	s := &seq{e: e}
	s.reset()
	ps := NewDefaultPowerSettings()
	ps.BlackWhiteVoltageDrain = VoltageRange{
		High: 15 * Volt,
		Low:  -15 * Volt,
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
	s.dualSPIMode(DualSPIModeSettings{
		MMInputPinEnabled:  false,
		DualSPIModeEnabled: false,
	})
	s.commonVoltageAndDataIntervalSetting(CDISettings{
		ColorMode: BlackWhiteSettings{
			Refresh: DifferentialRefresh{
				CopyNewToOld: false,
			},
			Border: DrivenBehavior[BlackWhiteBorder]{
				LookupTable: BlackWhiteBorder_BlackToWhite,
			},
		},
		BlackWhitePolarity:        BlackWhitePolarity_ZeroIsWhite,
		CommonVoltageDataInterval: 10 * HSync,
	})
	s.setGateSourceNonOverlapPeriod(TCONSettings{
		SourceToGate: 12 * TCONPeriod,
		GateToSource: 12 * TCONPeriod,
	})
	return s.err
}

// InitFast resets the panel and prepares it for quick full-screen refreshes of
// roughly a second and a half.
//
// The speed comes from one command: the temperature is forced to 90 °C, which
// selects a shorter waveform than the real ambient reading would. The trade is
// slightly weaker contrast. Everything else is the same power and panel setup
// as [Epd.Init].
func (e *Epd) InitFast() error {
	s := &seq{e: e}
	s.reset()
	panelSettings := NewDefaultPanelSettings()
	panelSettings.ColorMode = ColorMode_BlackWhite
	s.panelSetting(panelSettings)
	s.commonVoltageAndDataIntervalSetting(CDISettings{
		ColorMode: BlackWhiteSettings{
			Refresh: DifferentialRefresh{
				CopyNewToOld: false,
			},
			Border: DrivenBehavior[BlackWhiteBorder]{
				LookupTable: BlackWhiteBorder_BlackToWhite,
			},
		},
		BlackWhitePolarity:        BlackWhitePolarity_ZeroIsWhite,
		CommonVoltageDataInterval: 10 * HSync,
	})
	s.powerON()
	s.boosterSoftStart(BoosterSoftStartSettings{
		PhaseA: PhaseABSettings{
			SoftStartPeriod: StartPeriod_10ms,
			PhaseSettings: PhaseSettings{
				DrivingStrength:   Strength_5,
				MinGDROffDuration: OffDuration_6_58us,
			},
		},
		PhaseB: PhaseABSettings{
			SoftStartPeriod: StartPeriod_10ms,
			PhaseSettings: PhaseSettings{
				DrivingStrength:   Strength_5,
				MinGDROffDuration: OffDuration_6_58us,
			},
		},
		PhaseC1: PhaseCSettings{
			DrivingStrength:   Strength_4,
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
	s.cascadeSetting(CCSETSettings{
		TemperatureSource:  TemperatureSource_Register,
		OutputClockAtCLPin: false,
	})
	s.forceTemperature(90 * Celsius)
	return s.err
}

// InitPart resets the panel and prepares it for partial refreshes of roughly
// four tenths of a second, by forcing the temperature to 110 °C.
//
// Call it before [Epd.DisplayPart], and stay in partial mode for as long as you
// are doing partial updates. Partial refreshes accumulate ghosting, so run a
// full [Epd.Init] and [Epd.DisplayImage] periodically to clear it.
func (e *Epd) InitPart() error {
	s := &seq{e: e}
	s.reset()
	panelSettings := NewDefaultPanelSettings()
	panelSettings.ColorMode = ColorMode_BlackWhite
	s.panelSetting(panelSettings)
	s.powerON()
	s.cascadeSetting(CCSETSettings{
		TemperatureSource:  TemperatureSource_Register,
		OutputClockAtCLPin: false,
	})
	s.forceTemperature(110 * Celsius)
	return s.err
}

// ClearToWhite paints the whole screen white.
func (e *Epd) ClearToWhite() error {
	return e.DisplayImage(image.White)
}

// ClearToBlack paints the whole screen black.
func (e *Epd) ClearToBlack() error {
	return e.DisplayImage(image.Black)
}

// DisplayImage shows img on the whole screen and blocks until the refresh
// finishes.
//
// img may be any [image.Image]; it is thresholded to one bit per pixel over
// [ScreenBounds], drawn from its own top-left corner, so a larger image is
// cropped rather than scaled. Uniform images such as [image.White] fill the
// screen.
//
// img is never modified. That matters for callers keeping a persistent
// framebuffer, and is one place this driver differs from the vendor's C
// implementation, whose Display() writes the inverted frame back over its
// input.
//
// Requires [Epd.Init] or [Epd.InitFast] first.
func (e *Epd) DisplayImage(img image.Image) error {
	s := &seq{e: e}
	bwImage := BlackAndWhiteImageFromImage(img, ScreenBounds)
	s.displayStartTransmission(ImageBuffer_Old, bwImage)
	s.displayStartTransmission(ImageBuffer_New, bwImage.Negative())
	s.displayRefresh()
	return s.err
}

// DisplayPart redraws only rect, taking the pixels from img, and blocks until
// the refresh finishes.
//
// Requires [Epd.InitPart] first. rect must lie inside [ScreenBounds], and its
// left and right edges must both be multiples of 8, because one byte of image
// data carries eight horizontal pixels; snap a dirty rectangle outwards to the
// nearest multiple of 8 before passing it in. Either violation returns an error
// rather than sending a malformed window.
//
// Like the vendor driver, this leaves the controller in partial mode with the
// partial-mode VCOM settings still applied: it sends Partial In but no Partial
// Out, and does not restore the previous CDI. Keep calling DisplayPart while
// you are doing partial updates, then call [Epd.Init] again for the next full
// refresh.
func (e *Epd) DisplayPart(img image.Image, rect image.Rectangle) error {
	s := &seq{e: e}
	s.commonVoltageAndDataIntervalSetting(CDISettings{
		ColorMode: BlackWhiteSettings{
			Refresh: DifferentialRefresh{CopyNewToOld: true},
			Border:  FloatBehavior[BlackWhiteBorder]{},
		},
		BlackWhitePolarity:        BlackWhitePolarity_ZeroIsBlack,
		CommonVoltageDataInterval: 10 * HSync,
	})
	s.enterPartialMode()
	s.setPartialWindow(rect, GateScan_InsideAndOutside)
	bwImage := BlackAndWhiteImageFromImage(img, rect)
	s.displayStartTransmission(ImageBuffer_New, bwImage)
	s.displayRefresh()
	return s.err
}

// DisplayRefresh redraws the screen from whatever is already in the
// controller's image planes, and blocks until the refresh finishes.
func (e *Epd) DisplayRefresh() error {
	s := &seq{e: e}
	s.displayRefresh()
	return s.err
}

// Sleep powers the panel down and puts the controller into deep sleep.
//
// This is not optional: the image stays on the glass with no power at all,
// while leaving the controller powered and idle shortens the panel's life. Call
// it whenever you are done drawing, even between updates minutes apart. Any
// init method wakes the panel again.
func (e *Epd) Sleep() error {
	s := &seq{e: e}
	s.commonVoltageAndDataIntervalSetting(CDISettings{
		ColorMode: BlackWhiteSettings{
			Refresh: FullRefresh{},
			Border:  FloatBehavior[BlackWhiteBorder]{},
		},
		BlackWhitePolarity:        BlackWhitePolarity_ZeroIsBlack,
		CommonVoltageDataInterval: 10 * HSync,
	})
	s.powerOFF()
	s.deepSleep()
	return s.err
}

// Close releases the hardware: it drives the output pins low and closes the SPI
// port.
//
// It is safe to defer straight after [NewEPD], since it does nothing to
// anything that was never opened. Close does not put the panel to sleep - call
// [Epd.Sleep] first. Failures on the pins and on the port are reported
// together, so one of them cannot hide the other.
func (e *Epd) Close() error {
	return tryAll(
		"failed to close device",
		e.closeGPIO(),
		e.closeSPI(),
	)
}

// closeSPI closes the SPI port if one was opened.
func (e *Epd) closeSPI() error {
	if e.spiCloser == nil {
		return nil
	}
	if err := e.spiCloser.Close(); err != nil {
		return fmt.Errorf("failed to close SPI port: %w", err)
	}
	return nil
}

// displayRefresh triggers a refresh and blocks until the panel reports itself
// idle. Nothing here predicts how long that takes; the panel decides.
func (s *seq) displayRefresh() {
	s.sendCommand(CommandDRF)
	s.sleep(100 * time.Millisecond)
	s.wait()
}

// powerON turns the panel's supply rails on and waits for them to stabilise.
func (s *seq) powerON() {
	s.sendCommand(CommandPON)
	s.sleep(100 * time.Millisecond)
	s.wait()
}

// powerOFF turns the panel's supply rails off and waits for the controller to
// finish.
func (s *seq) powerOFF() {
	s.sendCommand(CommandPOF)
	s.wait()
}
