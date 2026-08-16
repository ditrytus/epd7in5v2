package epd7in5v2

import (
	"errors"
	"image"
	"image/color"
	"testing"
	"time"

	"periph.io/x/conn/v3/gpio"
)

func TestNewEPD(t *testing.T) {
	e := NewEPD()
	if e == nil {
		t.Fatal("NewEPD returned nil")
	}
	if e.sleepFn == nil {
		t.Error("NewEPD did not set sleepFn")
	}
}

func TestInit(t *testing.T) {
	e, rec := newTestEPD()
	if err := e.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	assertOps(t, rec.ops(), []op{
		{0x01, []byte{0x07, 0x07, 0x3F, 0x3F}}, // PWR
		{0x06, []byte{0x17, 0x17, 0x28, 0x17}}, // BTST
		{0x04, []byte{}},                       // PON
		{0x00, []byte{0x1F}},                   // PSR
		{0x61, []byte{0x03, 0x20, 0x01, 0xE0}}, // TRES 800x480
		{0x15, []byte{0x00}},                   // DUSPI off
		{0x50, []byte{0x10, 0x07}},             // CDI
		{0x60, []byte{0x22}},                   // TCON
	})
}

func TestInitFast(t *testing.T) {
	e, rec := newTestEPD()
	if err := e.InitFast(); err != nil {
		t.Fatalf("InitFast: %v", err)
	}
	assertOps(t, rec.ops(), []op{
		{0x00, []byte{0x1F}},
		{0x50, []byte{0x10, 0x07}},
		{0x04, []byte{}},
		{0x06, []byte{0x27, 0x27, 0x18, 0x17}},
		{0xE0, []byte{0x02}},
		{0xE5, []byte{0x5A}}, // 90 °C
	})
}

func TestInitPart(t *testing.T) {
	e, rec := newTestEPD()
	if err := e.InitPart(); err != nil {
		t.Fatalf("InitPart: %v", err)
	}
	assertOps(t, rec.ops(), []op{
		{0x00, []byte{0x1F}},
		{0x04, []byte{}},
		{0xE0, []byte{0x02}},
		{0xE5, []byte{0x6E}}, // 110 °C
	})
}

func TestReset(t *testing.T) {
	e, _ := newTestEPD()
	if err := e.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	rst := e.rst.(*fakePin)
	want := []gpio.Level{gpio.High, gpio.Low, gpio.High}
	if len(rst.outs) != len(want) {
		t.Fatalf("rst writes = %v, want %v", rst.outs, want)
	}
	for i := range want {
		if rst.outs[i] != want[i] {
			t.Errorf("rst write[%d] = %v, want %v", i, rst.outs[i], want[i])
		}
	}
}

func TestClearToWhite(t *testing.T) {
	e, rec := newTestEPD()
	if err := e.ClearToWhite(); err != nil {
		t.Fatalf("ClearToWhite: %v", err)
	}
	ops := rec.ops()
	want := []op{
		{0x10, repeatByte(0xFF, 48000)}, // OLD plane: white = 1s
		{0x13, repeatByte(0x00, 48000)}, // NEW plane: inverted
		{0x12, []byte{}},                // refresh
	}
	assertOps(t, ops, want)
}

func TestClearToBlack(t *testing.T) {
	e, rec := newTestEPD()
	if err := e.ClearToBlack(); err != nil {
		t.Fatalf("ClearToBlack: %v", err)
	}
	assertOps(t, rec.ops(), []op{
		{0x10, repeatByte(0x00, 48000)},
		{0x13, repeatByte(0xFF, 48000)},
		{0x12, []byte{}},
	})
}

func TestDisplayImageInvertsSecondPlane(t *testing.T) {
	e, rec := newTestEPD()
	// A non-uniform image: left half black, right half white.
	img := image.NewGray(ScreenBounds)
	for y := range int(ScreenHeight) {
		for x := range int(ScreenWidth) {
			if x >= int(ScreenWidth)/2 {
				img.Set(x, y, color.White)
			}
		}
	}
	if err := e.DisplayImage(img); err != nil {
		t.Fatalf("DisplayImage: %v", err)
	}
	ops := rec.ops()
	if len(ops) != 3 {
		t.Fatalf("op count = %d, want 3", len(ops))
	}
	if ops[0].cmd != 0x10 || ops[1].cmd != 0x13 || ops[2].cmd != 0x12 {
		t.Fatalf("cmds = %#02x %#02x %#02x, want 10 13 12", ops[0].cmd, ops[1].cmd, ops[2].cmd)
	}
	if len(ops[0].params) != 48000 || len(ops[1].params) != 48000 {
		t.Fatalf("plane lengths = %d/%d, want 48000/48000", len(ops[0].params), len(ops[1].params))
	}
	// NEW plane must be the exact bitwise complement of the OLD plane.
	for i := range ops[0].params {
		if ops[1].params[i] != ^ops[0].params[i] {
			t.Fatalf("plane mismatch at byte %d: old=%#02x new=%#02x", i, ops[0].params[i], ops[1].params[i])
		}
	}
}

func TestDisplayPart(t *testing.T) {
	e, rec := newTestEPD()
	rect := image.Rect(0, 0, 16, 8)
	if err := e.DisplayPart(image.White, rect); err != nil {
		t.Fatalf("DisplayPart: %v", err)
	}
	assertOps(t, rec.ops(), []op{
		{0x50, []byte{0x89, 0x07}}, // CDI: float border, N2OCP=1, DDX=01
		{0x91, []byte{}},           // PTIN
		{0x90, []byte{0x00, 0x00, 0x00, 0x0F, 0x00, 0x00, 0x00, 0x07, 0x01}}, // window
		{0x13, repeatByte(0xFF, 16)},                                         // NEW plane (DTM2), uninverted white
		{0x12, []byte{}},                                                     // refresh
	})
}

func TestDisplayPartOffsetWindow(t *testing.T) {
	e, rec := newTestEPD()
	rect := image.Rect(8, 16, 24, 32) // 16x16, x-aligned to 8
	if err := e.DisplayPart(image.Black, rect); err != nil {
		t.Fatalf("DisplayPart: %v", err)
	}
	ops := rec.ops()
	// window bytes: xs=8->0x0008, xe-1=23->0x0017, ys=16->0x0010, ye-1=31->0x001F
	wantWindow := []byte{0x00, 0x08, 0x00, 0x17, 0x00, 0x10, 0x00, 0x1F, 0x01}
	if ops[2].cmd != 0x90 {
		t.Fatalf("third op cmd = %#02x, want 0x90", ops[2].cmd)
	}
	if got := ops[2].params; !bytesEq(got, wantWindow) {
		t.Errorf("window = % X, want % X", got, wantWindow)
	}
	// image plane: 16x16 black = stride 2 * 16 = 32 bytes of 0x00
	if len(ops[3].params) != 32 {
		t.Errorf("plane length = %d, want 32", len(ops[3].params))
	}
}

func TestSleep(t *testing.T) {
	e, rec := newTestEPD()
	if err := e.Sleep(); err != nil {
		t.Fatalf("Sleep: %v", err)
	}
	assertOps(t, rec.ops(), []op{
		{0x50, []byte{0x83, 0x07}}, // CDI: border float, FullRefresh, DDX=11
		{0x02, []byte{}},           // POF
		{0x07, []byte{0xA5}},       // DSLP + check code
	})
}

func TestTurnOnAndDisplayRefresh(t *testing.T) {
	e, rec := newTestEPD()
	if err := e.TurnOn(); err != nil {
		t.Fatalf("TurnOn: %v", err)
	}
	assertOps(t, rec.ops(), []op{{0x12, []byte{}}})

	e2, rec2 := newTestEPD()
	if err := e2.DisplayRefresh(); err != nil {
		t.Fatalf("DisplayRefresh: %v", err)
	}
	assertOps(t, rec2.ops(), []op{{0x12, []byte{}}})
}

func TestInitPropagatesSPIError(t *testing.T) {
	e, _ := newTestEPD()
	e.spi.(*fakeSPI).txErr = errors.New("boom")
	if err := e.Init(); err == nil {
		t.Fatal("expected error from Init when SPI fails")
	}
}

func TestInitTimesOutWhenBusyStuck(t *testing.T) {
	e, _ := newTestEPD()
	busy := e.busy.(*fakePin)
	busy.level = gpio.Low                                     // permanently busy
	busy.waitHook = func(time.Duration) bool { return false } // edge never arrives
	if err := e.Init(); err == nil {
		t.Fatal("expected timeout error from Init when BUSY stuck low")
	}
}

// bytesEq is a tiny local equality helper to avoid importing bytes everywhere.
func bytesEq(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
