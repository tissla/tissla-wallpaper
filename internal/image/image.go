// package image
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
	Data   []byte // ARGB8888 (little-endian: B,G,R,A bytes), premultiplied, row-major
}

// Load decodes the image at path and scales it to exactly width x height using
// mode and a bilinear resampler. The returned Data is always width*height*4
// bytes, so it matches a wl_buffer described with those dimensions and a
// width*4 stride.
//
// If width or height is non-positive (output size not yet known), Load returns
// the source at its native size with no scaling.
func Load(path string, width, height int, mode ScaleMode) (*Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoded, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	src := toBGRA(decoded)

	if width <= 0 || height <= 0 {
		return &Image{Width: src.w, Height: src.h, Data: src.pix}, nil
	}

	srcRect, dstRect := scaleRects(src.w, src.h, width, height, mode)
	data := resample(src, srcRect, width, height, dstRect)
	return &Image{Width: width, Height: height, Data: data}, nil
}

// bgra is a 0-based, premultiplied B,G,R,A pixel buffer. Converting the decoded
// image into this form once drops the image.Image bounds offset and gives the
// resampler tight, predictable indexing.
type bgra struct {
	w, h int
	pix  []byte
}

// toBGRA converts any image.Image to a premultiplied BGRA buffer. RGBA() already
// returns alpha-premultiplied 16-bit channels, which is what ARGB8888 wants;
// >>8 takes them to 8-bit.
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

// resample draws src[srcRect] into a dstW x dstH buffer at dstRect using
// bilinear interpolation, leaving any area outside dstRect (letterbox in fit
// mode) opaque black. Interpolating premultiplied channels directly is correct
// and avoids dark halos near transparent edges.
func resample(src *bgra, srcRect image.Rectangle, dstW, dstH int, dstRect image.Rectangle) []byte {
	dst := make([]byte, dstW*dstH*4)
	for i := 3; i < len(dst); i += 4 { // opaque black background
		dst[i] = 255
	}

	sw, sh := srcRect.Dx(), srcRect.Dy()
	dw, dh := dstRect.Dx(), dstRect.Dy()
	if sw == 0 || sh == 0 || dw == 0 || dh == 0 {
		return dst
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
	return dst
}

// scaleRects computes the source sub-rectangle to sample and the destination
// sub-rectangle to draw into, for a source of size (sw,sh) onto a destination
// of size (dw,dh) under mode. All rectangles are 0-based.
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
		if sw*dh > dw*sh { // source wider than destination -> width-constrained
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
		if sw*dh > dw*sh { // source wider -> crop width
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
