package epd7in5v2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"time"

	"periph.io/x/conn/v3/gpio"
)

// seq is a sticky-error accumulator for a sequence of controller operations.
//
// It holds the first error and turns every later step into a no-op, so a panel
// init sequence reads like the programming guide instead of like error
// plumbing. The public API unwraps it back into a plain error. The standard
// library uses the same pattern in bufio.Writer.
//
// Short-circuiting suits setup and protocol sequences, where nothing after a
// failure can succeed. Cleanup paths use errors.Join instead, via tryAll, so
// that every pin is still lowered even when one of them fails.
type seq struct {
	e   *Epd
	err error
}

// boolToByte maps false to 0 and true to 1, for packing flag bits.
func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// tryAll joins errs and, if any are non-nil, wraps them with errMsg. It returns
// nil when every error is nil.
func tryAll(errMsg string, errs ...error) error {
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	return nil
}

// sendCommand drops DC low and clocks out a single opcode byte.
//
// DC must be set before the transfer rather than during it: spi.Conn.Tx holds
// chip select for the whole call, so DC cannot change mid-transaction. That is
// also why a command and its data are never sent in one Tx.
func (s *seq) sendCommand(cmd Command) {
	s.setPin(s.e.dc, gpio.Low)
	if s.err != nil {
		return
	}
	if err := s.e.spi.Tx([]byte{byte(cmd)}, nil); err != nil {
		s.err = fmt.Errorf("failed to send command %d over SPI: %w", cmd, err)
	}
}

// sendData raises DC and clocks out parameter or image bytes. A whole frame in
// one call is fine, since DC stays high throughout.
func (s *seq) sendData(data []byte) {
	s.setPin(s.e.dc, gpio.High)
	if s.err != nil {
		return
	}
	if err := s.e.spi.Tx(data, nil); err != nil {
		s.err = fmt.Errorf("failed to send data (%d bytes) over SPI: %w", len(data), err)
	}
}

// sleep blocks for dur, using the Epd's overridable sleep function so that
// tests can run the sequences without real delays.
func (s *seq) sleep(dur time.Duration) {
	if s.err != nil {
		return
	}
	if s.e.sleepFn != nil {
		s.e.sleepFn(dur)
		return
	}
	time.Sleep(dur)
}

// wait blocks until the panel pulls BUSY high, meaning it has finished the
// current operation.
//
// It waits on a pin edge rather than spinning, which matters on a
// battery-powered host, and gives up after 10 seconds so that a disconnected or
// unpowered panel surfaces as an error instead of hanging.
func (s *seq) wait() {
	if s.err != nil {
		return
	}
	for s.e.busy.Read() == gpio.Low {
		if !s.e.busy.WaitForEdge(time.Second * 10) {
			s.err = fmt.Errorf("waiting for %s input pin timed out", s.e.busy.Name())
			return
		}
	}
}

// enterPartialMode puts the controller into partial refresh mode.
func (s *seq) enterPartialMode() {
	s.sendCommand(CommandPTIN)
}

// GateScan selects which gates the controller drives during a partial refresh.
type GateScan byte

const (
	// GateScan_Inside drives only the gates inside the partial window.
	GateScan_Inside GateScan = 0
	// GateScan_InsideAndOutside also drives the gates outside the window, which
	// is what [Epd.DisplayPart] uses.
	GateScan_InsideAndOutside GateScan = 1
)

// setPartialWindow tells the controller which rectangle the next transmission
// covers.
//
// The end coordinates are sent as rect.Max-1, since the controller's window is
// inclusive while an [image.Rectangle] is half-open. Getting that wrong is the
// vendor driver's borrow bug, which corrupts any window ending at 256, 512 or
// 768 - coordinates an 800-wide screen hits constantly.
//
// It fails if rect leaves [ScreenBounds], or if its horizontal edges are not
// multiples of 8: one byte of image data carries eight horizontal pixels, so a
// window cannot start or end mid-byte.
func (s *seq) setPartialWindow(rect image.Rectangle, scan GateScan) {
	if s.err != nil {
		return
	}
	if !rect.In(ScreenBounds) {
		s.err = fmt.Errorf("partial window is not within screen bounds")
		return
	}
	if rect.Min.X%8 != 0 || rect.Max.X%8 != 0 {
		s.err = fmt.Errorf("partial window's horizontal coordinates must be a multiple of 8")
		return
	}
	s.sendCommand(CommandPTL)
	// The conversions below are guarded by the rect.In(ScreenBounds) check
	// above: every coordinate is within 0..800, so none can overflow uint16.
	data := make([]byte, 0, 9)
	data = binary.BigEndian.AppendUint16(data, uint16(rect.Min.X))   //nolint:gosec // G115: bounded by ScreenBounds
	data = binary.BigEndian.AppendUint16(data, uint16(rect.Max.X-1)) //nolint:gosec // G115: bounded by ScreenBounds
	data = binary.BigEndian.AppendUint16(data, uint16(rect.Min.Y))   //nolint:gosec // G115: bounded by ScreenBounds
	data = binary.BigEndian.AppendUint16(data, uint16(rect.Max.Y-1)) //nolint:gosec // G115: bounded by ScreenBounds
	data = append(data, byte(scan))
	s.sendData(data)
}

// DeepSleepCheckCode is the magic parameter the deep sleep command requires.
// The controller ignores the command without it, so a stray DSLP opcode cannot
// put the panel to sleep by accident.
const DeepSleepCheckCode byte = 0xA5

// deepSleep puts the controller into deep sleep. Only a reset wakes it, which
// is why every init method starts by toggling RST.
func (s *seq) deepSleep() {
	if s.err != nil {
		return
	}
	s.sendCommand(CommandDSLP)
	s.sendData([]byte{DeepSleepCheckCode})
}
