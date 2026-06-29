package epd7in5v2

import (
	"image"
	"image/color"
	"testing"
)

func TestNewBlackAndWhiteImage(t *testing.T) {
	img := NewBlackAndWhiteImage(image.Rect(0, 0, 800, 480))
	if img.Stride != 100 {
		t.Errorf("stride = %d, want 100", img.Stride)
	}
	if len(img.Pix) != 100*480 {
		t.Errorf("len(Pix) = %d, want 48000", len(img.Pix))
	}
	// Non-byte-aligned width rounds up.
	odd := NewBlackAndWhiteImage(image.Rect(0, 0, 17, 2))
	if odd.Stride != 3 {
		t.Errorf("stride(17px) = %d, want 3", odd.Stride)
	}
}

func TestBitAtSetBit(t *testing.T) {
	img := NewBlackAndWhiteImage(image.Rect(0, 0, 16, 2))
	if img.BitAt(0, 0) {
		t.Error("fresh bit should be 0")
	}
	img.SetBit(0, 0, true)
	if !img.BitAt(0, 0) {
		t.Error("bit (0,0) should be set")
	}
	if img.Pix[0] != 0x80 { // MSB = leftmost pixel
		t.Errorf("Pix[0] = %#02x, want 0x80", img.Pix[0])
	}
	img.SetBit(0, 0, false)
	if img.BitAt(0, 0) {
		t.Error("bit (0,0) should be cleared")
	}
	// Out of bounds SetBit is a no-op (must not panic).
	img.SetBit(-1, -1, true)
	img.SetBit(100, 100, true)
}

func TestPixOffset(t *testing.T) {
	img := NewBlackAndWhiteImage(image.Rect(0, 0, 800, 480))
	if got := img.PixOffset(0, 0); got != 0 {
		t.Errorf("PixOffset(0,0) = %d, want 0", got)
	}
	if got := img.PixOffset(8, 0); got != 1 {
		t.Errorf("PixOffset(8,0) = %d, want 1", got)
	}
	if got := img.PixOffset(0, 1); got != 100 {
		t.Errorf("PixOffset(0,1) = %d, want 100", got)
	}
	if got := img.PixOffset(799, 479); got != 47999 {
		t.Errorf("PixOffset(799,479) = %d, want 47999", got)
	}
}

func TestAt(t *testing.T) {
	img := NewBlackAndWhiteImage(image.Rect(0, 0, 8, 8))
	img.SetBit(1, 1, true)
	if img.At(1, 1) != color.White {
		t.Error("set bit should read White")
	}
	if img.At(2, 2) != color.Black {
		t.Error("unset bit should read Black")
	}
	if img.At(100, 100) != color.Black {
		t.Error("out-of-bounds At should be Black")
	}
}

func TestSet(t *testing.T) {
	img := NewBlackAndWhiteImage(image.Rect(0, 0, 8, 8))
	img.Set(0, 0, color.White)
	if !img.BitAt(0, 0) {
		t.Error("Set White should set the bit")
	}
	img.Set(0, 0, color.Black)
	if img.BitAt(0, 0) {
		t.Error("Set Black should clear the bit")
	}
	// out of bounds is a no-op
	img.Set(50, 50, color.White)
}

func TestColorModelConvert(t *testing.T) {
	m := BlackAndWhiteColorModel{Cutoff: 0.5}
	if m.Convert(color.White) != color.White {
		t.Error("white should convert to white")
	}
	if m.Convert(color.Black) != color.Black {
		t.Error("black should convert to black")
	}
	// Luma just below cutoff → black.
	dark := color.Gray{Y: 100}
	if m.Convert(dark) != color.Black {
		t.Error("dark gray should convert to black")
	}
	// Luma above cutoff → white.
	light := color.Gray{Y: 200}
	if m.Convert(light) != color.White {
		t.Error("light gray should convert to white")
	}
}

func TestColorModelMethod(t *testing.T) {
	img := NewBlackAndWhiteImage(image.Rect(0, 0, 8, 8))
	if _, ok := img.ColorModel().(BlackAndWhiteColorModel); !ok {
		t.Error("ColorModel() should be BlackAndWhiteColorModel")
	}
}

func TestNegative(t *testing.T) {
	img := NewBlackAndWhiteImage(image.Rect(0, 0, 16, 2))
	img.SetBit(0, 0, true)
	neg := img.Negative()
	for i := range img.Pix {
		if neg.Pix[i] != ^img.Pix[i] {
			t.Fatalf("Negative byte %d = %#02x, want %#02x", i, neg.Pix[i], ^img.Pix[i])
		}
	}
	// Original must be untouched.
	if !img.BitAt(0, 0) {
		t.Error("Negative must not mutate the original")
	}
}

func TestSubImage(t *testing.T) {
	img := NewBlackAndWhiteImage(image.Rect(0, 0, 800, 480))
	sub := img.SubImage(image.Rect(8, 1, 16, 2)).(*BlackAndWhiteImage)
	if sub.Rect != image.Rect(8, 1, 16, 2) {
		t.Errorf("sub.Rect = %v, want (8,1)-(16,2)", sub.Rect)
	}
	// Empty intersection returns an empty image.
	empty := img.SubImage(image.Rect(1000, 1000, 1001, 1001)).(*BlackAndWhiteImage)
	if !empty.Rect.Empty() {
		t.Error("disjoint SubImage should be empty")
	}
}

func TestBlackAndWhiteImageFromImage(t *testing.T) {
	white := BlackAndWhiteImageFromImage(image.White, ScreenBounds)
	for i, b := range white.Pix {
		if b != 0xFF {
			t.Fatalf("white image byte %d = %#02x, want 0xFF", i, b)
		}
	}
	black := BlackAndWhiteImageFromImage(image.Black, ScreenBounds)
	for i, b := range black.Pix {
		if b != 0x00 {
			t.Fatalf("black image byte %d = %#02x, want 0x00", i, b)
		}
	}
}

func TestBounds(t *testing.T) {
	r := image.Rect(0, 0, 8, 8)
	img := NewBlackAndWhiteImage(r)
	if img.Bounds() != r {
		t.Errorf("Bounds() = %v, want %v", img.Bounds(), r)
	}
}
