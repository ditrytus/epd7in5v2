package epd7in5v2

import (
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
)

// recTx is a single recorded SPI transfer, tagged with whether it carried a
// command (DC low) or data (DC high).
type recTx struct {
	command bool
	data    []byte
}

// busRecorder captures the ordered command/data stream that the driver emits
// over SPI. Whether a Tx is a command or data is determined by the level of the
// DC pin at the moment of the transfer (DC low = command, DC high = data),
// exactly as the controller interprets it.
type busRecorder struct {
	dcLevel gpio.Level
	txs     []recTx
}

func (r *busRecorder) record(w []byte) {
	b := append([]byte(nil), w...)
	r.txs = append(r.txs, recTx{command: r.dcLevel == gpio.Low, data: b})
}

// op is a logical controller operation: a command byte plus all of the data
// bytes that followed it before the next command.
type op struct {
	cmd    byte
	params []byte
}

func (r *busRecorder) ops() []op {
	var ops []op
	for _, tx := range r.txs {
		if tx.command {
			// A command transfer is always a single opcode byte.
			ops = append(ops, op{cmd: tx.data[0], params: []byte{}})
			continue
		}
		if len(ops) == 0 {
			// Data before any command; shouldn't happen, record defensively.
			ops = append(ops, op{params: append([]byte{}, tx.data...)})
			continue
		}
		last := &ops[len(ops)-1]
		last.params = append(last.params, tx.data...)
	}
	return ops
}

// fakePin implements gpio.PinIO so it can be used as a PinIn, a PinOut, or
// registered with gpioreg.
type fakePin struct {
	name string
	num  int

	level  gpio.Level
	outErr error
	inErr  error

	outs []gpio.Level // recorded Out() calls

	// Hooks (optional) for fine-grained behaviour.
	outHook  func(gpio.Level)         // invoked on every Out()
	readHook func() gpio.Level        // overrides Read()
	waitHook func(time.Duration) bool // overrides WaitForEdge()

	inCalls int // number of In() calls
}

// pin.Pin / conn.Resource
func (p *fakePin) String() string   { return p.name }
func (p *fakePin) Halt() error      { return nil }
func (p *fakePin) Name() string     { return p.name }
func (p *fakePin) Number() int      { return p.num }
func (p *fakePin) Function() string { return "" }

// gpio.PinOut
func (p *fakePin) Out(l gpio.Level) error {
	p.outs = append(p.outs, l)
	if p.outHook != nil {
		p.outHook(l)
	}
	p.level = l
	return p.outErr
}
func (p *fakePin) PWM(gpio.Duty, physic.Frequency) error { return nil }

// gpio.PinIn
func (p *fakePin) In(pull gpio.Pull, edge gpio.Edge) error {
	p.inCalls++
	return p.inErr
}
func (p *fakePin) Read() gpio.Level {
	if p.readHook != nil {
		return p.readHook()
	}
	return p.level
}
func (p *fakePin) WaitForEdge(t time.Duration) bool {
	if p.waitHook != nil {
		return p.waitHook(t)
	}
	return true
}
func (p *fakePin) Pull() gpio.Pull        { return gpio.PullNoChange }
func (p *fakePin) DefaultPull() gpio.Pull { return gpio.PullNoChange }

// fakeSPI implements spi.Conn and records transfers into a busRecorder.
type fakeSPI struct {
	rec   *busRecorder
	txErr error
}

func (f *fakeSPI) String() string      { return "fakeSPI" }
func (f *fakeSPI) Duplex() conn.Duplex { return conn.Full }
func (f *fakeSPI) Tx(w, r []byte) error {
	if f.txErr != nil {
		return f.txErr
	}
	f.rec.record(w)
	return nil
}
func (f *fakeSPI) TxPackets(p []spi.Packet) error { return nil }

// fakeCloser implements io.Closer.
type fakeCloser struct {
	err    error
	closed bool
}

func (c *fakeCloser) Close() error {
	c.closed = true
	return c.err
}

// newTestEPD wires an *Epd to fake pins and a fake SPI bus. By default BUSY
// reads High (ready) so wait() returns immediately, and sleeps are no-ops.
func newTestEPD() (*Epd, *busRecorder) {
	rec := &busRecorder{dcLevel: gpio.High}
	dc := &fakePin{name: "DC"}
	dc.outHook = func(l gpio.Level) { rec.dcLevel = l }
	e := &Epd{
		rst:     &fakePin{name: "RST"},
		dc:      dc,
		pwr:     &fakePin{name: "PWR"},
		busy:    &fakePin{name: "BUSY", level: gpio.High},
		spi:     &fakeSPI{rec: rec},
		sleepFn: func(time.Duration) {},
	}
	return e, rec
}
