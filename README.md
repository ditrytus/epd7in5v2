# epd7in5v2

[![CI](https://github.com/ditrytus/epd7in5v2/actions/workflows/ci.yml/badge.svg)](https://github.com/ditrytus/epd7in5v2/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ditrytus/epd7in5v2.svg)](https://pkg.go.dev/github.com/ditrytus/epd7in5v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/ditrytus/epd7in5v2)](https://goreportcard.com/report/github.com/ditrytus/epd7in5v2)
[![Go](https://img.shields.io/github/go-mod/go-version/ditrytus/epd7in5v2)](go.mod)
[![License](https://img.shields.io/github/license/ditrytus/epd7in5v2)](LICENSE)

A pure-Go driver for the **Waveshare 7.5" e-Paper display V2** (800×480, black & white),
built on [periph.io](https://periph.io) and talking straight to the panel's
GD7965/UC8179-class controller over SPI.

This is not a wrapper around Waveshare's C driver — it is a rewrite. The controller
command set is modelled as typed Go values, the command sequences are transcribed from
the panel programming guide, and several known bugs in the vendor code are fixed by
construction.

```go
screen := epd7in5v2.NewEPD()
defer screen.Close()

screen.GPIOInit()
screen.Init()
screen.DisplayImage(img)
screen.Sleep()
```

---

## Contents

- [Why this driver](#why-this-driver)
- [Hardware](#hardware)
- [Raspberry Pi setup](#raspberry-pi-setup)
- [Installation](#installation)
- [Quick start](#quick-start)
- [Example program](#example-program)
- [API](#api)
  - [Lifecycle](#lifecycle)
  - [Init modes](#init-modes)
  - [Drawing](#drawing)
  - [Partial refresh](#partial-refresh)
  - [Images](#images)
  - [Low-level command settings](#low-level-command-settings)
- [Design notes](#design-notes)
- [Testing](#testing)
- [Project layout](#project-layout)
- [Troubleshooting](#troubleshooting)
- [Status](#status)
- [License](#license)

---

## Why this driver

| | Waveshare C driver | epd7in5v2 |
|---|---|---|
| Language | C (+ Python bindings) | Go, no cgo |
| Hardware access | Custom `DEV_Config` layer per platform | `periph.io/x/conn/v3` GPIO + SPI |
| Command parameters | Magic byte literals | Typed structs with validation |
| Chip select | Manually bit-banged | Owned by `spi.Conn.Tx` |
| Input buffer | `Display()` mutates the caller's buffer | Caller's image is never touched |
| Partial windows | Off-by-one at x/y ends of 256, 512, 768 | Correct `(end-1)` encoding |
| Tests | — | ~96 % statement coverage, no hardware needed |

Four concrete bugs in the vendor code that this driver does not inherit:

1. **`Display()` mutates its input buffer.** It writes the bitwise complement back into
   the caller's framebuffer. Anything that keeps a persistent framebuffer and diffs
   against it — a terminal, for instance — silently corrupts itself. Here,
   `DisplayImage` converts into a fresh buffer and inverts into a copy.
2. **`Display_Part` borrow bug.** Window end coordinates of 256, 512 or 768 encode
   incorrectly. On an 800-wide screen those boundaries come up constantly. The correct
   encoding is `(end-1)/256, (end-1)%256`, which is what `setPartialWindow` does.
3. **8-pixel horizontal alignment.** Partial windows must start and end on multiples of
   8. This driver rejects a misaligned rectangle with an error instead of drawing
   garbage.
4. **`Display_Part` leaves CDI clobbered and never sends Partial Out.** Documented
   below so callers can structure their refresh policy around it.

## Hardware

- **Panel:** Waveshare 7.5inch e-Paper (V2), 800×480, black & white
  ([wiki](https://www.waveshare.com/wiki/7.5inch_e-Paper_HAT))
- **Driver board:** e-Paper Driver HAT Rev2.3
- **Host:** Raspberry Pi (developed against a Pi Zero 2 W); anything periph.io supports
  and that exposes the pins below should work

### Driver HAT switches

Set these with the power **off**:

| Switch | Setting | Meaning |
|---|---|---|
| Interface Config | `0` | 4-line SPI — separate DC pin. Required. |
| Display Config | `B` | 0.47 Ω booster resistor, required for the 7.5" panel. |

### Wiring

| Signal | BCM pin | Header pin | Driven by |
|---|---|---|---|
| MOSI (DIN) | GPIO10 | 19 | SPI peripheral |
| SCLK | GPIO11 | 23 | SPI peripheral |
| CS | GPIO8 (CE0) | 24 | SPI peripheral (`Tx` frames it) |
| DC | GPIO25 | 22 | GPIO output |
| RST | GPIO17 | 11 | GPIO output |
| BUSY | GPIO24 | 18 | GPIO input |
| PWR | GPIO18 | 12 | GPIO output |

The pin numbers live in `gpio.go` as `PinMOSI`, `PinSCLK`, `PinCS`, `PinDC`, `PinRST`,
`PinBUSY` and `PinPWR`. Only DC, RST, BUSY and PWR are claimed as GPIO; MOSI, SCLK and CS
belong to the SPI peripheral and their constants are documentation only.

## Raspberry Pi setup

Enable the SPI bus so the kernel exposes `/dev/spidev0.0`:

```bash
sudo raspi-config nonint do_spi 0     # or: add dtparam=spi=on to /boot/firmware/config.txt
sudo reboot
ls /dev/spidev0.0                     # should exist after reboot
```

Access to `/dev/spidev*` and `/dev/gpiomem` normally requires membership in the `spi` and
`gpio` groups:

```bash
sudo usermod -aG spi,gpio "$USER"     # log out and back in
```

## Installation

```bash
go get github.com/ditrytus/epd7in5v2
```

Requires Go 1.25 or newer — the floor is set by `periph.io/x/conn/v3`, not by this
package. It is pure Go — no cgo, no vendor C library — so it cross-compiles from any
host:

```bash
GOOS=linux GOARCH=arm   GOARM=7 go build ./example/displayimage   # Pi Zero 2 W (32-bit OS)
GOOS=linux GOARCH=arm64         go build ./example/displayimage   # Pi Zero 2 W (64-bit OS)
```

Importing the package registers the periph.io host drivers (`periph.io/x/host/v3`), so
you do not need a blank import of your own.

## Quick start

```go
package main

import (
	"image"
	"image/color"
	"log"

	"github.com/ditrytus/epd7in5v2"
)

func main() {
	screen := epd7in5v2.NewEPD()
	defer screen.Close()

	// Claim GPIO pins and open the SPI port.
	if err := screen.GPIOInit(); err != nil {
		log.Fatal(err)
	}

	// Power the panel up and load the full-refresh waveform.
	if err := screen.Init(); err != nil {
		log.Fatal(err)
	}

	// Any image.Image works; it is thresholded to 1 bit per pixel and
	// scaled/cropped to the 800×480 screen bounds.
	img := image.NewRGBA(epd7in5v2.ScreenBounds)
	for y := 0; y < 480; y++ {
		for x := 0; x < 800; x++ {
			if (x/40+y/40)%2 == 0 {
				img.Set(x, y, color.Black)
			} else {
				img.Set(x, y, color.White)
			}
		}
	}
	if err := screen.DisplayImage(img); err != nil {
		log.Fatal(err)
	}

	// E-paper holds its image without power. Always sleep when done —
	// leaving the panel powered and idle degrades it.
	if err := screen.Sleep(); err != nil {
		log.Fatal(err)
	}
}
```

## Example program

`example/displayimage` shows a PNG on the panel, waits for `SIGINT`/`SIGTERM`, then
clears the screen to white and sleeps the panel before exiting.

```bash
GOOS=linux GOARCH=arm64 go build -o displayimage ./example/displayimage
scp displayimage example/beach.png pi@raspberrypi:~
ssh pi@raspberrypi './displayimage --image beach.png'
# ...then Ctrl-C to clear the screen and exit cleanly
```

## API

### Lifecycle

| Method | What it does |
|---|---|
| `NewEPD() *Epd` | Allocates the driver. Touches no hardware. |
| `GPIOInit() error` | Registers periph.io drivers, claims DC/RST/BUSY/PWR, opens SPI at 10 MHz, mode 0, 8 bits. |
| `Init() / InitFast() / InitPart() error` | Reset the controller and load a waveform — see below. |
| `Reset() error` | Toggles RST with the datasheet's timing. Called for you by every `Init*`. |
| `DisplayRefresh() error` | Sends the refresh command and blocks on BUSY. |
| `TurnOn() error` | Refresh plus an extra settle-and-wait. |
| `Sleep() error` | Powers the panel off and enters deep sleep. |
| `Close() error` | Lowers PWR/DC/RST and closes the SPI port. |

The intended order is `NewEPD` → `GPIOInit` → `Init*` → drawing → `Sleep` → `Close`.
`Close` is safe to `defer` immediately after `NewEPD`; it no-ops on anything that was
never opened.

**Sleep is not optional.** Once the panel is asleep the image stays on the glass with no
power at all, and leaving the controller powered and idle shortens the panel's life. Wake
from deep sleep by calling an `Init*` method again.

### Init modes

The three init sequences look very different, but the thing that actually makes the last
two fast is a single command — a forced temperature that selects a shorter waveform.

| Method | Forced temperature | Typical refresh | Use for |
|---|---|---|---|
| `Init()` | none (real sensor) | ~4 s | Full-quality full-screen redraws. |
| `InitFast()` | 90 °C | ~1.5 s | Full-screen redraws where speed beats contrast. |
| `InitPart()` | 110 °C | ~0.4 s | Partial updates via `DisplayPart`. |

PLL is never sent, so the frame rate stays at the panel default, and the waveform lookup
tables come from OTP. The driver makes no promise about how long a refresh takes: it
sends the command and blocks on BUSY, with a 10-second timeout as a backstop.

### Drawing

```go
func (e *Epd) DisplayImage(img image.Image) error
func (e *Epd) ClearToWhite() error
func (e *Epd) ClearToBlack() error
```

`DisplayImage` converts `img` to 1 bit per pixel over `ScreenBounds`, writes it into both
controller planes (the "old" plane as given, the "new" plane inverted), and refreshes.
The image you pass in is never modified. `ClearToWhite` and `ClearToBlack` are
`DisplayImage(image.White)` and `DisplayImage(image.Black)`.

### Partial refresh

```go
func (e *Epd) DisplayPart(img image.Image, rect image.Rectangle) error
```

Redraws only `rect`. Requires `InitPart()` first.

Two rules the controller enforces, and this driver checks:

- `rect` must lie inside `ScreenBounds` (`image.Rect(0, 0, 800, 480)`).
- `rect.Min.X` and `rect.Max.X` must both be multiples of 8 — one byte carries eight
  horizontal pixels. Snap your dirty box down at the start and up at the end. If your
  cell width is a multiple of 8 this is free.

Both violations return an error rather than sending a malformed window.

**Mode discipline.** `DisplayPart` sets partial-mode CDI and sends Partial In, but does
not restore CDI or send Partial Out. That is deliberate and matches the vendor's
behaviour: stay in partial mode for as long as you are doing partial updates, then call
`Init()` (or `InitFast()`) again for the next full refresh. Partial updates also
accumulate ghosting, so a periodic full refresh is worth building into your refresh
policy regardless.

```go
screen.InitPart()
for _, box := range dirtyBoxes {
	screen.DisplayPart(frame, snapTo8(box))
}
// later, to clear ghosting:
screen.Init()
screen.DisplayImage(frame)
```

### Images

`BlackAndWhiteImage` is a 1-bit-per-pixel `image.Image` implementation matching the
controller's memory layout — MSB-first, one byte per eight horizontal pixels, set bit
means white.

```go
type BlackAndWhiteImage struct {
	Pix    []byte
	Stride int
	Rect   image.Rectangle
}

func NewBlackAndWhiteImage(r image.Rectangle) *BlackAndWhiteImage
func BlackAndWhiteImageFromImage(img image.Image, bounds image.Rectangle) *BlackAndWhiteImage

func (p *BlackAndWhiteImage) BitAt(x, y int) bool
func (p *BlackAndWhiteImage) SetBit(x, y int, on bool)
func (p *BlackAndWhiteImage) SubImage(r image.Rectangle) image.Image
func (p *BlackAndWhiteImage) Negative() *BlackAndWhiteImage
```

It implements `image.Image` and `draw.Image`, so `image/draw`, font renderers and
anything else in the standard library can draw straight into it. Conversion from colour
goes through `BlackAndWhiteColorModel`, which thresholds Rec. 601 luma at a configurable
`Cutoff` (0.5 by default).

Building the framebuffer yourself and handing it to `DisplayImage` avoids a conversion
pass per frame:

```go
buf := epd7in5v2.NewBlackAndWhiteImage(epd7in5v2.ScreenBounds)
draw.Draw(buf, buf.Bounds(), image.White, image.Point{}, draw.Src)
// ...draw text, shapes, whatever...
screen.DisplayImage(buf)
```

### Low-level command settings

Every controller command that takes parameters has its own file, its own settings struct
and its own `Flags()` encoder, validated before anything reaches the wire. You do not
need these to display an image — they are what the `Init*` sequences are written in, and
they are exported so you can read the code against the datasheet.

| File | Command | Struct |
|---|---|---|
| `psr.go` | PSR — panel setting | `PanelSettings` |
| `pwr.go` | PWR — power setting | `PowerSettings`, `VoltageRange` |
| `btst.go` | BTST — booster soft start | `BoosterSoftStartSettings` |
| `cdi.go` | CDI — VCOM and data interval | `CDISettings`, `BlackWhiteSettings` |
| `tcon.go` | TCON — gate/source non-overlap | `TCONSettings` |
| `tres.go` | TRES — resolution | `ResolutionSettings` |
| `duspi.go` | DUSPI — dual SPI mode | `DualSPIModeSettings` |
| `ccset.go` | CCSET — cascade setting | `CCSETSettings` |
| `tsset.go` | TSSET — force temperature | `Temperature` |
| `dtm.go` | DTM1/DTM2 — image transmission | `ImageBuffer` |
| `commands.go` | every opcode | `Command` |

Parameters carry real units rather than raw byte values, and the encoders reject
out-of-range values with a message naming the legal range:

```go
ps := epd7in5v2.NewDefaultPowerSettings()
ps.GateVoltage = epd7in5v2.SymmetricVoltageRange(20 * epd7in5v2.Volt)
ps.BlackWhiteVoltageDrain = epd7in5v2.VoltageRange{
	High: 15 * epd7in5v2.Volt,
	Low:  -15 * epd7in5v2.Volt,
}
```

Ranges the encoders enforce:

- **Gate voltage** — one of 9, 10, 11, 12, 17, 18, 19, 20 V.
- **Drain voltage** — 2.4 V to 15 V, in steps of 0.2 V.
- **VCOM data interval** — 2 to 17 HSync periods.
- **Gate/source non-overlap** — 4 to 64 TCON periods (667 ns each), in steps of 4.
- **Resolution** — up to 800×600, horizontal a multiple of 8.

## Design notes

**Sticky error accumulator.** A panel init is thirty-odd operations that all fail the same
way. Instead of an `if err != nil` after each one, the internal `seq` type holds the first
error and turns every subsequent call into a no-op, so an init sequence reads like the
programming guide rather than like error plumbing. The public API still returns plain
errors — this is a library, and callers get to handle failure. `bufio.Writer` uses the
same pattern. Short-circuiting is for setup and protocol sequences only; cleanup paths use
`errors.Join` instead, so that every pin is still lowered even if one of them fails.

**SPI chip select belongs to `Tx`.** `spi.Conn.Tx` asserts CS before the bytes and
deasserts after, which makes the vendor's manual CS bit-banging dead code here. Three
consequences: don't claim BCM 8 as a GPIO, always set DC *before* calling `Tx`, and never
let a single `Tx` span a command byte and its data — CS is held for the whole call, so DC
cannot change mid-transaction. A whole 48 000-byte frame in one `Tx` is fine, because DC
is constant throughout.

**BUSY is edge-waited, not polled.** `wait()` blocks on `WaitForEdge` with a 10-second
timeout rather than spinning, which matters on a battery-powered host.

## Testing

The whole driver is testable without a panel. `testsupport_test.go` provides a fake SPI
connection and fake GPIO pins; the fake bus records every transfer and tags it as command
or data according to the DC pin level at the moment of transfer, exactly as the controller
interprets it. Tests then assert on the decoded stream of `(command, params)` operations —
so an init test checks the actual byte sequence the panel would see.

```bash
make check        # everything CI runs: build, vet, lint, test
make test         # go test ./...
make test-cover   # go test -cover ./...   → ~96 % of statements
make vet          # go vet ./...
make lint         # golangci-lint run ./...
make lint-fix     # golangci-lint run --fix ./...
make build        # go build ./...
make tools        # install the pinned golangci-lint
```

Lint configuration lives in [`.golangci.yml`](.golangci.yml) and is tuned for correctness
over style — every enabled linter reports zero issues on a clean tree, so a finding means
something is genuinely wrong. Notably `var-naming` is off: the lower layer deliberately
mirrors the datasheet's identifiers (`BT_PHA`, `VDH_LVL`, `ColorMode_BlackWhite`) so the
code can be read line by line against the programming guide.

## Project layout

```
epd.go          Epd type, public API, the three init sequences
gpio.go         pin constants, GPIO/SPI setup and teardown
util.go         seq (sticky-error accumulator), command/data transfer, BUSY wait,
                partial window, deep sleep
image.go        BlackAndWhiteImage and its colour model
commands.go     controller opcodes
dtm.go          image transmission and ScreenBounds
psr.go pwr.go btst.go cdi.go tcon.go tres.go duspi.go ccset.go tsset.go
                one command's settings, encoding and validation per file
*_test.go       unit tests, plus testsupport_test.go with the fakes
example/displayimage/
                CLI that puts a PNG on the panel
```

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| `failed to find GPIO17` | periph.io found no host driver. You are not on a supported host, or the process cannot reach `/dev/gpiomem`. |
| `failed to open SPI port` | SPI is not enabled (`dtparam=spi=on`), or the user is not in the `spi` group. |
| `waiting for BUSY input pin timed out` | BUSY not wired, wrong pin, or the panel is not powered. Check the Display Config switch is on `B`. |
| Screen stays blank | Interface Config switch must be `0` (4-line SPI). Also check the ribbon cable seating. |
| Garbled or shifted partial update | Window not aligned to 8 pixels horizontally, or a full refresh is needed after `Init()`. |
| Ghosting builds up | Expected with partial refresh. Run a periodic full `Init()` + `DisplayImage`. |
| Image fades after hours | The panel was left powered instead of asleep. Call `Sleep()` when done. |

## Status

Working and covered by tests, but young — the API is not frozen and may change.

## License

Released under the MIT License — see [LICENSE](LICENSE).

This is an independent implementation written against the panel programming guide. It is
not a translation of Waveshare's C driver and contains no vendor code; the command
opcodes and register layouts it uses are facts about the controller hardware. Datasheets
and the driver HAT schematic come from the
[Waveshare 7.5inch e-Paper HAT wiki](https://www.waveshare.com/wiki/7.5inch_e-Paper_HAT).

Hardware access is provided by [periph.io](https://periph.io), which is licensed under
Apache 2.0.
