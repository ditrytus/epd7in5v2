package epd7in5v2

import (
	"image"
)

var ScreenBounds image.Rectangle

func init() {
	ScreenBounds = image.Rect(0, 0, int(ScreenWidth), int(ScreenHeight))
}

type ImageBuffer byte

const (
	ImageBuffer_New ImageBuffer = CommandDTM1
	ImageBuffer_Old ImageBuffer = CommandDTM2
)

func (s *seq) displayStartTransmission(buf ImageBuffer, img *BlackAndWhiteImage) {
	if s.err != nil {
		return
	}
	s.sendCommand(Command(buf))
	s.sendImageData(img, ScreenBounds)
}

func (s *seq) sendImageData(image *BlackAndWhiteImage, bounds image.Rectangle) {
	subImage := image.SubImage(bounds).(*BlackAndWhiteImage)
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
