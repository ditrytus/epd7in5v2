package epd7in5v2

import (
	"image"
	"image/color"
	"image/draw"
)

type BlackAndWhiteColorModel struct {
	Cutoff float64
}

func (m BlackAndWhiteColorModel) Convert(c color.Color) color.Color {
	r, g, b, _ := c.RGBA()
	luma := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 0xffff
	if luma >= m.Cutoff {
		return color.White
	}
	return color.Black
}

type BlackAndWhiteImage struct {
	Pix []byte

	Stride int
	Rect   image.Rectangle
	model  color.Model
}

func NewBlackAndWhiteImage(r image.Rectangle) *BlackAndWhiteImage {
	w := r.Dx()
	h := r.Dy()
	stride := (w + 7) / 8 // bytes per row
	return &BlackAndWhiteImage{
		Pix:    make([]byte, stride*h),
		Stride: stride,
		Rect:   r,
		model: BlackAndWhiteColorModel{
			Cutoff: 0.5,
		},
	}
}

func BlackAndWhiteImageFromImage(img image.Image, bounds image.Rectangle) *BlackAndWhiteImage {
	bwImage := NewBlackAndWhiteImage(bounds)
	draw.Draw(bwImage, bwImage.Rect, bounds, img.Bounds().Min, draw.Src)
	return bwImage
}

func (p *BlackAndWhiteImage) ColorModel() color.Model {
	return p.model

}

func (p *BlackAndWhiteImage) Bounds() image.Rectangle {
	return p.Rect
}

func (p *BlackAndWhiteImage) At(x, y int) color.Color {
	if !(image.Point{X: x, Y: y}.In(p.Rect)) {
		return color.Black
	}
	if p.BitAt(x, y) {
		return color.White
	}
	return color.Black
}

func (p *BlackAndWhiteImage) BitAt(x, y int) bool {
	i := p.PixOffset(x, y)
	mask := byte(0x80 >> (x % 8))
	return p.Pix[i]&mask != 0

}

func (p *BlackAndWhiteImage) SetBit(x, y int, on bool) {
	if !(image.Point{X: x, Y: y}.In(p.Rect)) {
		return
	}
	i := p.PixOffset(x, y)
	mask := byte(0x80 >> (x % 8))
	if on {
		p.Pix[i] |= mask
	} else {
		p.Pix[i] &^= mask
	}
}

func (p *BlackAndWhiteImage) Set(x, y int, c color.Color) {
	if !(image.Point{X: x, Y: y}.In(p.Rect)) {
		return
	}
	converted := p.model.Convert(c)
	if converted == color.White {
		p.SetBit(x, y, true)
	} else {
		p.SetBit(x, y, false)
	}
}

func (p *BlackAndWhiteImage) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)/8
}

func (p *BlackAndWhiteImage) SubImage(r image.Rectangle) image.Image {
	r = r.Intersect(p.Rect)
	if r.Empty() {
		return &BlackAndWhiteImage{}
	}
	i := p.PixOffset(r.Min.X, r.Min.Y)
	return &BlackAndWhiteImage{
		Pix:    p.Pix[i:],
		Stride: p.Stride,
		Rect:   r,
	}
}

func (p *BlackAndWhiteImage) Negative() *BlackAndWhiteImage {
	pix := make([]byte, len(p.Pix))
	copy(pix, p.Pix)
	for i, b := range pix {
		pix[i] = ^b
	}
	return &BlackAndWhiteImage{
		Pix:    pix,
		Stride: p.Stride,
		Rect:   p.Rect,
		model:  p.model,
	}
}
