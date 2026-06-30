package image

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestScaleRectsStretch(t *testing.T) {
	src, dst := scaleRects(800, 600, 1920, 1080, ScaleStretch)
	if src != image.Rect(0, 0, 800, 600) {
		t.Errorf("stretch src = %v, want full source", src)
	}
	if dst != image.Rect(0, 0, 1920, 1080) {
		t.Errorf("stretch dst = %v, want full destination", dst)
	}
}

func TestScaleRectsFit(t *testing.T) {
	// Source wider than destination (3:1 into 1:1) -> width fills, letterbox top/bottom.
	src, dst := scaleRects(3000, 1000, 1000, 1000, ScaleFit)
	if src != image.Rect(0, 0, 3000, 1000) {
		t.Errorf("fit src = %v, want whole source", src)
	}
	if dst.Dx() != 1000 {
		t.Errorf("fit wide: dst width = %d, want 1000 (full)", dst.Dx())
	}
	if dst.Min.X != 0 || dst.Max.X != 1000 {
		t.Errorf("fit wide: dst should span full width, got x[%d,%d]", dst.Min.X, dst.Max.X)
	}
	if dst.Min.Y <= 0 || dst.Max.Y >= 1000 {
		t.Errorf("fit wide: expected vertical letterbox, got y[%d,%d]", dst.Min.Y, dst.Max.Y)
	}

	// Source taller than destination -> height fills, letterbox left/right.
	_, dst = scaleRects(1000, 3000, 1000, 1000, ScaleFit)
	if dst.Min.Y != 0 || dst.Max.Y != 1000 {
		t.Errorf("fit tall: dst should span full height, got y[%d,%d]", dst.Min.Y, dst.Max.Y)
	}
	if dst.Min.X <= 0 || dst.Max.X >= 1000 {
		t.Errorf("fit tall: expected horizontal letterbox, got x[%d,%d]", dst.Min.X, dst.Max.X)
	}
}

func TestScaleRectsFill(t *testing.T) {
	// Source wider than destination -> crop width, draw to full dst.
	src, dst := scaleRects(3000, 1000, 1000, 1000, ScaleFill)
	if dst != image.Rect(0, 0, 1000, 1000) {
		t.Errorf("fill dst = %v, want full destination", dst)
	}
	if src.Dy() != 1000 {
		t.Errorf("fill wide: src height = %d, want 1000 (uncropped)", src.Dy())
	}
	if src.Min.X <= 0 || src.Max.X >= 3000 {
		t.Errorf("fill wide: expected horizontal crop, got x[%d,%d]", src.Min.X, src.Max.X)
	}

	// Source taller -> crop height.
	src, _ = scaleRects(1000, 3000, 1000, 1000, ScaleFill)
	if src.Dx() != 1000 {
		t.Errorf("fill tall: src width = %d, want 1000 (uncropped)", src.Dx())
	}
	if src.Min.Y <= 0 || src.Max.Y >= 3000 {
		t.Errorf("fill tall: expected vertical crop, got y[%d,%d]", src.Min.Y, src.Max.Y)
	}
}

// writeTestPNG creates a solid-color PNG of size w x h and returns its path.
func writeTestPNG(t *testing.T, w, h int) string {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			im.Set(x, y, color.RGBA{R: 10, G: 120, B: 200, A: 255})
		}
	}
	path := filepath.Join(t.TempDir(), "test.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, im); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestLoadProducesExactSize is the load-bearing test: whatever the source
// dimensions and mode, Load must return exactly width*height*4 bytes, or the
// compositor reads past the shm pool.
func TestLoadProducesExactSize(t *testing.T) {
	cases := []struct {
		name       string
		srcW, srcH int
		dstW, dstH int
		mode       ScaleMode
	}{
		{"upscale_fill", 100, 100, 1920, 1080, ScaleFill},
		{"downscale_fit", 4000, 3000, 1280, 720, ScaleFit},
		{"stretch_oddsize", 333, 777, 640, 480, ScaleStretch},
		{"exact_match", 800, 600, 800, 600, ScaleFill},
		{"wide_src_fit", 3000, 500, 1000, 1000, ScaleFit},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := writeTestPNG(t, c.srcW, c.srcH)
			out, err := Load(path, c.dstW, c.dstH, c.mode)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if out.Width != c.dstW || out.Height != c.dstH {
				t.Errorf("dims = %dx%d, want %dx%d", out.Width, out.Height, c.dstW, c.dstH)
			}
			want := c.dstW * c.dstH * 4
			if len(out.Data) != want {
				t.Errorf("len(Data) = %d, want %d", len(out.Data), want)
			}
		})
	}
}

// TestLoadZeroDimsFallsBack covers the not-yet-configured output case: dims <= 0
// should fall back to the image's native size rather than panicking.
func TestLoadZeroDimsFallsBack(t *testing.T) {
	path := writeTestPNG(t, 320, 240)
	out, err := Load(path, 0, 0, ScaleFill)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Width != 320 || out.Height != 240 {
		t.Errorf("fallback dims = %dx%d, want 320x240", out.Width, out.Height)
	}
	if len(out.Data) != 320*240*4 {
		t.Errorf("fallback len = %d, want %d", len(out.Data), 320*240*4)
	}
}

// writeSolidPNG writes a w x h PNG filled with one opaque color.
func writeSolidPNG(t *testing.T, w, h int, c color.RGBA) string {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			im.Set(x, y, c)
		}
	}
	path := filepath.Join(t.TempDir(), "solid.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, im); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestResamplePreservesSolidColor: bilinear of a uniform source is that same
// color everywhere. Catches channel swaps, bad weights, and indexing errors
// that the size invariant would miss. Output is premultiplied B,G,R,A.
func TestResamplePreservesSolidColor(t *testing.T) {
	src := color.RGBA{R: 200, G: 50, B: 100, A: 255}
	path := writeSolidPNG(t, 64, 64, src)

	out, err := Load(path, 500, 300, ScaleFill) // upscale, non-square
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	wantB, wantG, wantR, wantA := byte(100), byte(50), byte(200), byte(255)
	for i := 0; i+3 < len(out.Data); i += 4 {
		if out.Data[i] != wantB || out.Data[i+1] != wantG || out.Data[i+2] != wantR || out.Data[i+3] != wantA {
			t.Fatalf("pixel %d = B%d G%d R%d A%d, want B%d G%d R%d A%d",
				i/4, out.Data[i], out.Data[i+1], out.Data[i+2], out.Data[i+3], wantB, wantG, wantR, wantA)
		}
	}
}

// TestFitLetterboxIsBlack: a 4:1 source into a square destination under fit must
// leave the top and bottom bands opaque black, with the source color in the
// middle band.
func TestFitLetterboxIsBlack(t *testing.T) {
	path := writeSolidPNG(t, 400, 100, color.RGBA{R: 0, G: 255, B: 0, A: 255}) // 4:1
	w, h := 200, 200
	out, err := Load(path, w, h, ScaleFit)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// scaled source height = 200 * 100/400 = 50, centered -> rows [75,125)
	pixel := func(x, y int) (b, g, r, a byte) {
		o := (y*w + x) * 4
		return out.Data[o], out.Data[o+1], out.Data[o+2], out.Data[o+3]
	}
	// top band (row 10) should be opaque black
	if b, g, r, a := pixel(100, 10); b != 0 || g != 0 || r != 0 || a != 255 {
		t.Errorf("top letterbox = B%d G%d R%d A%d, want opaque black", b, g, r, a)
	}
	// center (row 100) should be green
	if _, g, _, a := pixel(100, 100); g != 255 || a != 255 {
		t.Errorf("center = green? G%d A%d, want G255 A255", g, a)
	}
	// bottom band (row 190) opaque black
	if b, g, r, a := pixel(100, 190); b != 0 || g != 0 || r != 0 || a != 255 {
		t.Errorf("bottom letterbox = B%d G%d R%d A%d, want opaque black", b, g, r, a)
	}
}

// TestLoadIntoMatchesLoad: LoadInto writing into a caller buffer must produce
// byte-identical output to Load allocating its own.
func TestLoadIntoMatchesLoad(t *testing.T) {
	path := writeSolidPNG(t, 300, 200, color.RGBA{R: 30, G: 90, B: 150, A: 255})
	const w, h = 256, 256

	want, err := Load(path, w, h, ScaleFit)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := make([]byte, w*h*4)
	if err := LoadInto(got, path, w, h, ScaleFit); err != nil {
		t.Fatalf("LoadInto: %v", err)
	}
	for i := range got {
		if got[i] != want.Data[i] {
			t.Fatalf("byte %d differs: LoadInto=%d Load=%d", i, got[i], want.Data[i])
		}
	}
}

func TestLoadIntoRejectsWrongSize(t *testing.T) {
	path := writeSolidPNG(t, 64, 64, color.RGBA{A: 255})
	small := make([]byte, 10)
	if err := LoadInto(small, path, 100, 100, ScaleFill); err == nil {
		t.Error("expected error for undersized dst")
	}
}

func TestLoadIntoRejectsZeroDims(t *testing.T) {
	path := writeSolidPNG(t, 64, 64, color.RGBA{A: 255})
	if err := LoadInto(nil, path, 0, 0, ScaleFill); err == nil {
		t.Error("expected error for zero dims")
	}
}
