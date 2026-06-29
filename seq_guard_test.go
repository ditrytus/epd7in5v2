package epd7in5v2

import (
	"errors"
	"image"
	"testing"
)

// When a seq already carries an error, every step must be a no-op: nothing is
// sent on the bus and the original error is preserved.
func TestSeqMethodsShortCircuitOnError(t *testing.T) {
	prior := errors.New("prior failure")
	e, rec := newTestEPD()
	s := &seq{e: e, err: prior}

	bw := BlackAndWhiteImageFromImage(image.White, ScreenBounds)

	s.panelSetting(NewDefaultPanelSettings())
	s.powerSetting(NewDefaultPowerSettings())
	s.boosterSoftStart(BoosterSoftStartSettings{})
	s.resolutionSetting(ResolutionSettings{Horizontal: 800, Vertical: 480})
	s.dualSPIMode(DualSPIModeSettings{})
	s.commonVoltageAndDataIntervalSetting(CDISettings{
		ColorMode:                 BlackWhiteSettings{Refresh: DifferentialRefresh{}, Border: FloatBehavior[BlackWhiteBorder]{}},
		CommonVoltageDataInterval: 10 * HSync,
	})
	s.setGateSourceNonOverlapPeriod(TCONSettings{SourceToGate: 12 * TCONPeriod, GateToSource: 12 * TCONPeriod})
	s.cascadeSetting(CCSETSettings{})
	s.forceTemperature(90 * Celsius)
	s.deepSleep()
	s.enterPartialMode()
	s.displayStartTransmission(ImageBuffer_New, bw)
	s.setPartialWindow(image.Rect(0, 0, 16, 8), GateScan_Inside)
	s.powerON()
	s.powerOFF()
	s.displayRefresh()
	s.reset()

	if len(rec.ops()) != 0 {
		t.Errorf("expected no bus activity, got %d ops", len(rec.ops()))
	}
	if !errors.Is(s.err, prior) {
		t.Errorf("err = %v, want the original prior error preserved", s.err)
	}
}

// Interface marker methods carry no behaviour but must exist; invoke them so
// they are exercised.
func TestMarkerMethods(t *testing.T) {
	BlackWhiteSettings{}.isColorModeSettings()
	BlackWhiteRedSettings{}.isColorModeSettings()
	FullRefresh{}.isRefresh()
	DifferentialRefresh{}.isRefresh()
	BlackWhiteBorder_Border.isColorModeBorder()
	BlackWhiteRedBorder_Border.isColorModeBorder()
	FloatBehavior[BlackWhiteBorder]{}.isBorderBehavior()
	DrivenBehavior[BlackWhiteBorder]{}.isBorderBehavior()
}
