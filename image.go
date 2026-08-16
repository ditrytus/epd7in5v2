package epd7in5v2

import (
	"image"
	"image/color"
	"image/draw"
)

// BlackAndWhiteColorModel converts any colour to pure black or pure white by
// thresholding its luma.
//
// It implements [color.Model].
type BlackAndWhiteColorModel struct {
	// Cutoff is the luma threshold, from 0 (everything becomes white) to 1
	// (everything becomes black). Colours at or above it become white.
	// [NewBlackAndWhiteImage] uses 0.5.
	Cutoff float64
}

// Convert returns [color.White] if c's Rec. 601 luma is at or above
// [BlackAndWhiteColorModel.Cutoff], and [color.Black] otherwise.
func (m BlackAndWhiteColorModel) Convert(c color.Color) color.Color {
	r, g, b, _ := c.RGBA()
	luma := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 0xffff
	if luma >= m.Cutoff {
		return color.White
	}
	return color.Black
}

// BlackAndWhiteImage is a one-bit-per-pixel image laid out the way the panel
// controller expects: eight horizontal pixels per byte, most significant bit
// leftmost, and a set bit meaning white.
//
// It implements [image.Image] and [draw.Image], so [image/draw], font renderers
// and anything else in the standard library can draw straight into it. Callers
// that keep their own framebuffer can build one of these and hand it to
// [Epd.DisplayImage] to skip a conversion pass per frame.
type BlackAndWhiteImage struct {
	// Pix holds the packed pixel bits, row by row.
	Pix []byte

	// Stride is the number of bytes between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect image.Rectangle
	// model converts incoming colours to black or white. It is unset on images
	// returned by SubImage, which are read-only in practice.
	model color.Model
}

// NewBlackAndWhiteImage returns a new [BlackAndWhiteImage] with the given
// bounds, all pixels black, thresholding incoming colours at a luma of 0.5.
//
// Rows are padded to a whole number of bytes, so an image whose width is not a
// multiple of 8 has unused bits at the end of each row.
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

// BlackAndWhiteImageFromImage converts img into a new [BlackAndWhiteImage] with
// the given bounds, thresholding each pixel with
// [BlackAndWhiteColorModel].
//
// img is drawn from its own top-left corner, so it is cropped rather than
// scaled when it is larger than bounds. img is never modified.
func BlackAndWhiteImageFromImage(img image.Image, bounds image.Rectangle) *BlackAndWhiteImage {
	bwImage := NewBlackAndWhiteImage(bounds)
	draw.Draw(bwImage, bwImage.Rect, img, img.Bounds().Min, draw.Src)
	return bwImage
}

// ColorModel returns the image's [color.Model], a [BlackAndWhiteColorModel].
func (p *BlackAndWhiteImage) ColorModel() color.Model {
	return p.model

}

// Bounds returns the image's bounds.
func (p *BlackAndWhiteImage) Bounds() image.Rectangle {
	return p.Rect
}

// At returns the colour at (x, y) as [color.White] or [color.Black]. Points
// outside the image's bounds read as black.
func (p *BlackAndWhiteImage) At(x, y int) color.Color {
	if !(image.Point{X: x, Y: y}.In(p.Rect)) {
		return color.Black
	}
	if p.BitAt(x, y) {
		return color.White
	}
	return color.Black
}

// BitAt reports whether the pixel at (x, y) is white.
//
// It does not check bounds: a point outside the image reads whatever byte the
// offset lands on, or panics if that falls outside
// [BlackAndWhiteImage.Pix]. Use [BlackAndWhiteImage.At] for a bounds-checked
// read.
func (p *BlackAndWhiteImage) BitAt(x, y int) bool {
	i := p.PixOffset(x, y)
	mask := byte(0x80 >> (x % 8))
	return p.Pix[i]&mask != 0

}

// SetBit sets the pixel at (x, y) to white when on is true and black otherwise.
// Points outside the image's bounds are ignored.
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

// Set converts c through the image's colour model and stores it at (x, y).
// Points outside the image's bounds are ignored.
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

// PixOffset returns the index into [BlackAndWhiteImage.Pix] of the byte holding
// the pixel at (x, y). The pixel's bit within that byte is 0x80 >> (x % 8).
func (p *BlackAndWhiteImage) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)/8
}

// SubImage returns an image representing the portion of p visible through r,
// sharing pixel storage with p.
//
// The returned image's rows are still p's rows, so its stride spans the whole
// of p and its left edge is only byte-aligned when r.Min.X is a multiple of 8.
// An empty intersection yields an empty image.
//
// The result carries no colour model, so it is for reading only:
// [BlackAndWhiteImage.ColorModel] returns nil on it and
// [BlackAndWhiteImage.Set] panics. [BlackAndWhiteImage.SetBit] still works.
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

// Negative returns a new image with every pixel inverted. p is not modified.
//
// [Epd.DisplayImage] uses this to fill the controller's second plane, which is
// also why a caller's framebuffer survives a display call intact.
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
