package image

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

type Image struct {
	Width  int
	Height int
	Data   []byte // ARGB8888, row-major
}

func Load(path string, width, height int, mode ScaleMode) (*Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	return toARGB(img, width, height), nil
}

// TODO: add scaling using width and height
func toARGB(img image.Image, width, height int) *Image {
	bounds := img.Bounds()
	w := bounds.Max.X - bounds.Min.X
	h := bounds.Max.Y - bounds.Min.Y

	data := make([]byte, w*h*4)

	for y := range h {
		for x := range w {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			offset := (y*w + x) * 4
			data[offset+0] = uint8(b >> 8)
			data[offset+1] = uint8(g >> 8)
			data[offset+2] = uint8(r >> 8)
			data[offset+3] = uint8(a >> 8)
		}
	}

	return &Image{
		Width:  w,
		Height: h,
		Data:   data,
	}
}
