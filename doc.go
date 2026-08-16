// Package epd7in5v2 drives the Waveshare 7.5 inch e-Paper display V2, an
// 800x480 black and white panel built around a GD7965/UC8179-class controller.
//
// The panel is reached over 4-line SPI plus four GPIO pins, using periph.io for
// hardware access. Importing this package registers the periph.io host drivers,
// so calling code does not need a blank import of its own.
//
// # Usage
//
// The lifecycle is [NewEPD], [Epd.GPIOInit], one of the init methods, drawing,
// [Epd.Sleep] and finally [Epd.Close]:
//
//	screen := epd7in5v2.NewEPD()
//	defer screen.Close()
//
//	if err := screen.GPIOInit(); err != nil {
//		return err
//	}
//	if err := screen.Init(); err != nil {
//		return err
//	}
//	if err := screen.DisplayImage(img); err != nil {
//		return err
//	}
//	return screen.Sleep()
//
// [Epd.Sleep] is not optional. An e-paper panel holds its image with no power at
// all, and leaving the controller powered and idle shortens the panel's life.
// Calling any init method again wakes the panel from deep sleep.
//
// # Two layers
//
// [Epd] is the high-level layer. Its methods are whole operations - [Epd.Init],
// [Epd.DisplayImage], [Epd.DisplayPart] - each of which sends a fixed sequence
// of controller commands and blocks until the panel reports itself idle.
//
// Underneath, every controller command that takes parameters has its own
// settings type carrying real units rather than raw register values:
// [PanelSettings], [PowerSettings], [BoosterSoftStartSettings], [CDISettings],
// [TCONSettings], [ResolutionSettings], [DualSPIModeSettings] and
// [CCSETSettings]. Each has a Flags method that encodes it into the bytes the
// controller expects and rejects out-of-range values before anything reaches
// the wire. These types are exported so that the init sequences can be read
// against the panel datasheet; displaying an image does not require them.
//
// Field and method names in that lower layer deliberately keep the datasheet's
// own abbreviations - PSR, BTST, CDI, VDH_LVL, N2OCP and so on - so that the
// code can be compared line by line against the programming guide.
//
// # Images
//
// [Epd.DisplayImage] accepts any [image.Image] and thresholds it to one bit per
// pixel over [ScreenBounds]. The image passed in is never modified.
//
// Callers that maintain their own framebuffer can build a [BlackAndWhiteImage]
// directly. It matches the controller's memory layout, and it implements
// [draw.Image], so [image/draw], font renderers and the rest of the standard
// library can draw straight into it.
//
// # Errors
//
// Every exported operation returns an error. Internally the long command
// sequences use a sticky-error accumulator: the first failure is retained and
// every later step becomes a no-op, so an init sequence reads like the
// programming guide rather than like error plumbing.
package epd7in5v2
