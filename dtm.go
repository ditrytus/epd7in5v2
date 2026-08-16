package epd7in5v2

import (
	"image"
)

// ScreenBounds is the full drawing area of the panel: image.Rect(0, 0,
// [ScreenWidth], [ScreenHeight]).
//
// It is the rectangle [Epd.DisplayImage] converts into, and every partial
// window passed to [Epd.DisplayPart] must lie inside it.
var ScreenBounds image.Rectangle

func init() {
	ScreenBounds = image.Rect(0, 0, int(ScreenWidth), int(ScreenHeight))
}

// ImageBuffer selects one of the controller's two image planes.
//
// The controller holds an "old" and a "new" plane and drives each pixel
// according to the transition between the two, which is what makes a
// differential refresh possible - see [DifferentialRefresh].
type ImageBuffer byte

const (
	// ImageBuffer_Old is the plane holding the previous frame, written with
	// CommandDTM1.
	ImageBuffer_Old ImageBuffer = CommandDTM1
	// ImageBuffer_New is the plane holding the incoming frame, written with
	// CommandDTM2.
	ImageBuffer_New ImageBuffer = CommandDTM2
)

// displayStartTransmission writes a whole image into one of the controller's
// planes.
func (s *seq) displayStartTransmission(buf ImageBuffer, img *BlackAndWhiteImage) {
	if s.err != nil {
		return
	}
	s.sendCommand(Command(buf))
	s.sendImageData(img, img.Bounds())
}

// sendImageData clocks out the packed rows of img that fall inside bounds,
// padding each row with zero bytes when bounds ends mid-byte.
func (s *seq) sendImageData(img *BlackAndWhiteImage, bounds image.Rectangle) {
	subImage := img.subImage(bounds)
	missingBytes := (subImage.Rect.Max.X - bounds.Max.X) / 8
	for y := subImage.Rect.Min.Y; y < subImage.Rect.Max.Y; y++ {
		from := subImage.PixOffset(subImage.Rect.Min.X, y)
		to := subImage.PixOffset(subImage.Rect.Max.X, y)
		data := subImage.Pix[from:to]
		s.sendData(data)
		if missingBytes > 0 {
			s.sendData(make([]byte, missingBytes))
		}
	}
}
