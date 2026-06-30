package image

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func benchPNG(b *testing.B, w, h int) string {
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			im.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}
	p := filepath.Join(b.TempDir(), "bench.png")
	f, _ := os.Create(p)
	png.Encode(f, im)
	f.Close()
	return p
}

// 4K source -> 4K output, the realistic worst case.
const benchW, benchH = 3840, 2160

// Old path: Load allocates an Image, caller copies into the shm buffer.
func BenchmarkLoadThenCopy(b *testing.B) {
	path := benchPNG(b, 2000, 1500)
	shm := make([]byte, benchW*benchH*4) // stand-in for the mmap
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		img, err := Load(path, benchW, benchH, ScaleFill)
		if err != nil {
			b.Fatal(err)
		}
		copy(shm, img.Data)
	}
}

// New path: LoadInto renders straight into the shm buffer.
func BenchmarkLoadInto(b *testing.B) {
	path := benchPNG(b, 2000, 1500)
	shm := make([]byte, benchW*benchH*4)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := LoadInto(shm, path, benchW, benchH, ScaleFill); err != nil {
			b.Fatal(err)
		}
	}
}
