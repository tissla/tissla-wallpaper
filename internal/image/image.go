// Package image decodes and scales images into the premultiplied BGRA byte
// layout Wayland's ARGB8888 buffers expect.
package image

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

type Image struct {
	Width  int
	Height int
	Data   []byte // ARGB8888 (little-endian: B,G,R,A bytes), premultiplied, row-major
}

// Load decodes and scales the image at path to width x height
func Load(path string, width, height int, mode ScaleMode) (*Image, error) {
	src, err := decode(path)
	if err != nil {
		return nil, err
	}
	if width <= 0 || height <= 0 {
		b := src.Bounds()
		width, height = b.Dx(), b.Dy()
		mode = ScaleStretch
	}
	dst := make([]byte, width*height*4)
	renderInto(dst, src, width, height, mode)
	return &Image{Width: width, Height: height, Data: dst}, nil
}

// LoadInto decodes the image at path and renders it directly into dst, which
// must be exactly width*height*4 bytes (e.g. an shm mapping). Unlike Load it
// allocates no result buffer and makes no extra copy, so the only full-frame
// buffers alive are the decoded source and its BGRA form.
func LoadInto(dst []byte, path string, width, height int, mode ScaleMode) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("LoadInto: output size unknown (%dx%d)", width, height)
	}
	if len(dst) != width*height*4 {
		return fmt.Errorf("LoadInto: dst has %d bytes, want %d", len(dst), width*height*4)
	}
	src, err := decode(path)
	if err != nil {
		return err
	}
	renderInto(dst, src, width, height, mode)
	return nil
}

func decode(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

func renderInto(dst []byte, src image.Image, width, height int, mode ScaleMode) {
	sb := toBGRA(src)
	srcRect, dstRect := scaleRects(sb.w, sb.h, width, height, mode)
	resampleInto(dst, sb, srcRect, width, height, dstRect)
}

// bgra is a 0-based, premultiplied B,G,R,A pixel buffer.
type bgra struct {
	w, h int
	pix  []byte
}

// toBGRA converts any image.Image to a premultiplied BGRA buffer. RGBA() already
// returns alpha-premultiplied 16-bit channels; >>8 takes them to 8-bit.
func toBGRA(img image.Image) *bgra {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	pix := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, a := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			o := (y*w + x) * 4
			pix[o+0] = uint8(bl >> 8)
			pix[o+1] = uint8(g >> 8)
			pix[o+2] = uint8(r >> 8)
			pix[o+3] = uint8(a >> 8)
		}
	}
	return &bgra{w: w, h: h, pix: pix}
}

// resampleInto draws src[srcRect] into dst (a dstW x dstH BGRA buffer) at
// dstRect using bilinear interpolation. Areas outside dstRect (letterbox in fit
// mode) are filled opaque black. dst is written in place; no allocation.
func resampleInto(dst []byte, src *bgra, srcRect image.Rectangle, dstW, dstH int, dstRect image.Rectangle) {
	// Background only matters where the image won't cover the surface, i.e. the
	// letterbox in fit mode. For fill/stretch the image spans the whole buffer,
	// so skip the redundant fill.
	covers := dstRect.Min.X == 0 && dstRect.Min.Y == 0 && dstRect.Dx() == dstW && dstRect.Dy() == dstH
	if !covers {
		for i := 0; i+3 < len(dst); i += 4 {
			dst[i+0], dst[i+1], dst[i+2], dst[i+3] = 0, 0, 0, 255 // opaque black
		}
	}

	sw, sh := srcRect.Dx(), srcRect.Dy()
	dw, dh := dstRect.Dx(), dstRect.Dy()
	if sw == 0 || sh == 0 || dw == 0 || dh == 0 {
		return
	}

	for ly := 0; ly < dh; ly++ {
		sy := (float64(ly)+0.5)*float64(sh)/float64(dh) - 0.5
		if sy < 0 {
			sy = 0
		} else if sy > float64(sh-1) {
			sy = float64(sh - 1)
		}
		y0 := int(sy)
		y1 := y0 + 1
		if y1 > sh-1 {
			y1 = sh - 1
		}
		wy := sy - float64(y0)
		gy0 := srcRect.Min.Y + y0
		gy1 := srcRect.Min.Y + y1

		for lx := 0; lx < dw; lx++ {
			sx := (float64(lx)+0.5)*float64(sw)/float64(dw) - 0.5
			if sx < 0 {
				sx = 0
			} else if sx > float64(sw-1) {
				sx = float64(sw - 1)
			}
			x0 := int(sx)
			x1 := x0 + 1
			if x1 > sw-1 {
				x1 = sw - 1
			}
			wx := sx - float64(x0)
			gx0 := srcRect.Min.X + x0
			gx1 := srcRect.Min.X + x1

			i00 := (gy0*src.w + gx0) * 4
			i10 := (gy0*src.w + gx1) * 4
			i01 := (gy1*src.w + gx0) * 4
			i11 := (gy1*src.w + gx1) * 4
			o := ((dstRect.Min.Y+ly)*dstW + dstRect.Min.X + lx) * 4

			for c := 0; c < 4; c++ {
				top := float64(src.pix[i00+c])*(1-wx) + float64(src.pix[i10+c])*wx
				bot := float64(src.pix[i01+c])*(1-wx) + float64(src.pix[i11+c])*wx
				dst[o+c] = uint8(top*(1-wy) + bot*wy + 0.5)
			}
		}
	}
}

// scaleRects computes the source sub-rectangle to sample and the destination
// sub-rectangle to draw into. All rectangles are 0-based.
//
//	fill    cover:   crop source to the destination aspect, draw to the full destination
//	fit     contain: draw the whole source into a centered aspect-correct box, letterbox the rest
//	stretch:         whole source to whole destination, aspect ignored
func scaleRects(sw, sh, dw, dh int, mode ScaleMode) (src, dst image.Rectangle) {
	switch mode {
	case ScaleStretch:
		return image.Rect(0, 0, sw, sh), image.Rect(0, 0, dw, dh)

	case ScaleFit:
		src = image.Rect(0, 0, sw, sh)
		if sw*dh > dw*sh {
			fitH := dw * sh / sw
			y0 := (dh - fitH) / 2
			dst = image.Rect(0, y0, dw, y0+fitH)
		} else {
			fitW := dh * sw / sh
			x0 := (dw - fitW) / 2
			dst = image.Rect(x0, 0, x0+fitW, dh)
		}
		return src, dst

	default: // ScaleFill
		dst = image.Rect(0, 0, dw, dh)
		if sw*dh > dw*sh {
			cropW := sh * dw / dh
			x0 := (sw - cropW) / 2
			src = image.Rect(x0, 0, x0+cropW, sh)
		} else {
			cropH := sw * dh / dw
			y0 := (sh - cropH) / 2
			src = image.Rect(0, y0, sw, y0+cropH)
		}
		return src, dst
	}
}
