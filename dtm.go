package epd7in5v2

import (
	"image"
	"image/draw"
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

func (s *seq) displayStartTransmission(buf ImageBuffer, img image.Image) {
	if s.err != nil {
		return
	}
	s.sendCommand(Command(buf))
	bwImage, ok := img.(*BlackAndWhiteImage)
	if !ok {
		bwImage = NewBlackAndWhiteImage(ScreenBounds)
		draw.Draw(bwImage, bwImage.Rect, ScreenBounds, img.Bounds().Min, draw.Src)
	}
	s.sendImageData(bwImage, ScreenBounds)
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
