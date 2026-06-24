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

func (e *Epd) TurnOn() error {
	s := &seq{e: e}
	s.displayRefresh()
	s.sleep(10 * time.Millisecond)
	s.wait()
	return s.err
}

func (e *Epd) Init() error {
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

func (e *Epd) InitFast() error {
	s := &seq{e: e}
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

func (e *Epd) ClearToWhite() error {
	s := &seq{e: e}
	s.displayStartTransmission(ImageBuffer_New, image.White)
	s.displayStartTransmission(ImageBuffer_Old, image.Black)
	s.displayRefresh()
	return s.err
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

func (e *Epd) closeSPI() error {
	if err := e.spiCloser.Close(); err != nil {
		return fmt.Errorf("failed to close SPI port: %w", err)
	}
	return nil
}

func (s *seq) displayRefresh() {
	s.sendCommand(CommandDRF)
}

func (s *seq) powerON() {
	s.sendCommand(CommandPON)
	s.sleep(100 * time.Millisecond)
	s.wait()
}
