package appicon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
)

// Sizes written into the multi-resolution .ico (Windows 10+).
var Sizes = []int{16, 20, 24, 32, 48, 64, 256}

var (
	brandBlue = color.RGBA{R: 37, G: 99, B: 235, A: 255}
	brandInk  = color.RGBA{R: 255, G: 255, B: 255, A: 255}
)

// Render draws the app mark at size×size: a rounded blue tile with an original
// music note (not a YouTube logo).
func Render(size int) *image.RGBA {
	if size < 8 {
		size = 8
	}
	const mul = 4
	n := size * mul
	src := image.NewRGBA(image.Rect(0, 0, n, n))
	r := float64(n)
	fillRoundRect(src, 0, 0, r, r, r*0.22, brandBlue)

	// Note head.
	fillEllipse(src, 0.36*r, 0.70*r, 0.18*r, 0.13*r, brandInk)
	// Stem on the right of the head.
	fillRect(src, 0.50*r, 0.20*r, 0.07*r, 0.52*r, brandInk)
	// Flag.
	fillTriangle(src,
		0.57*r, 0.20*r,
		0.84*r, 0.30*r,
		0.57*r, 0.44*r,
		brandInk)
	return downscale(src, size)
}

// ICO returns a multi-size ICO with PNG frames.
func ICO() ([]byte, error) {
	pngs := make([][]byte, len(Sizes))
	for i, s := range Sizes {
		var buf bytes.Buffer
		if err := png.Encode(&buf, Render(s)); err != nil {
			return nil, err
		}
		pngs[i] = buf.Bytes()
	}
	var out bytes.Buffer
	if err := binary.Write(&out, binary.LittleEndian, uint16(0)); err != nil {
		return nil, err
	}
	if err := binary.Write(&out, binary.LittleEndian, uint16(1)); err != nil {
		return nil, err
	}
	if err := binary.Write(&out, binary.LittleEndian, uint16(len(Sizes))); err != nil {
		return nil, err
	}
	offset := 6 + 16*len(Sizes)
	for i, s := range Sizes {
		w, h := s, s
		if w >= 256 {
			w = 0
		}
		if h >= 256 {
			h = 0
		}
		if err := out.WriteByte(byte(w)); err != nil {
			return nil, err
		}
		if err := out.WriteByte(byte(h)); err != nil {
			return nil, err
		}
		if err := out.WriteByte(0); err != nil {
			return nil, err
		}
		if err := out.WriteByte(0); err != nil {
			return nil, err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint16(1)); err != nil {
			return nil, err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint16(32)); err != nil {
			return nil, err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint32(len(pngs[i]))); err != nil {
			return nil, err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint32(offset)); err != nil {
			return nil, err
		}
		offset += len(pngs[i])
	}
	for _, p := range pngs {
		if _, err := out.Write(p); err != nil {
			return nil, err
		}
	}
	if out.Len() < 64 {
		return nil, fmt.Errorf("ico too small")
	}
	return out.Bytes(), nil
}

func fillRoundRect(img *image.RGBA, x0, y0, x1, y1, rad float64, c color.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if insideRoundRect(float64(x)+0.5, float64(y)+0.5, x0, y0, x1, y1, rad) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func insideRoundRect(px, py, x0, y0, x1, y1, rad float64) bool {
	cx := (x0 + x1) / 2
	cy := (y0 + y1) / 2
	hw := (x1 - x0) / 2
	hh := (y1 - y0) / 2
	if rad < 0 {
		rad = 0
	}
	if rad > hw {
		rad = hw
	}
	if rad > hh {
		rad = hh
	}
	ax := math.Abs(px-cx) - hw + rad
	ay := math.Abs(py-cy) - hh + rad
	ox := math.Max(ax, 0)
	oy := math.Max(ay, 0)
	dist := math.Hypot(ox, oy) + math.Min(math.Max(ax, ay), 0) - rad
	return dist <= 0
}

func fillEllipse(img *image.RGBA, cx, cy, rx, ry float64, c color.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dx := (float64(x) + 0.5 - cx) / rx
			dy := (float64(y) + 0.5 - cy) / ry
			if dx*dx+dy*dy <= 1 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func fillRect(img *image.RGBA, x, y, w, h float64, c color.RGBA) {
	b := img.Bounds()
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := int(math.Ceil(x + w))
	y1 := int(math.Ceil(y + h))
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			if image.Pt(px, py).In(b) {
				img.SetRGBA(px, py, c)
			}
		}
	}
}

func fillTriangle(img *image.RGBA, x1, y1, x2, y2, x3, y3 float64, c color.RGBA) {
	b := img.Bounds()
	minX := int(math.Floor(math.Min(x1, math.Min(x2, x3))))
	maxX := int(math.Ceil(math.Max(x1, math.Max(x2, x3))))
	minY := int(math.Floor(math.Min(y1, math.Min(y2, y3))))
	maxY := int(math.Ceil(math.Max(y1, math.Max(y2, y3))))
	den := (y2-y3)*(x1-x3) + (x3-x2)*(y1-y3)
	if den == 0 {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if !image.Pt(x, y).In(b) {
				continue
			}
			px := float64(x) + 0.5
			py := float64(y) + 0.5
			a := ((y2-y3)*(px-x3) + (x3-x2)*(py-y3)) / den
			bb := ((y3-y1)*(px-x3) + (x1-x3)*(py-y3)) / den
			cc := 1 - a - bb
			if a >= 0 && bb >= 0 && cc >= 0 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func downscale(src *image.RGBA, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	factor := src.Bounds().Dx() / size
	if factor < 1 {
		factor = 1
	}
	n := uint32(factor * factor)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var r, g, b, a uint32
			for dy := 0; dy < factor; dy++ {
				for dx := 0; dx < factor; dx++ {
					c := src.RGBAAt(x*factor+dx, y*factor+dy)
					r += uint32(c.R)
					g += uint32(c.G)
					b += uint32(c.B)
					a += uint32(c.A)
				}
			}
			dst.SetRGBA(x, y, color.RGBA{
				R: uint8(r / n),
				G: uint8(g / n),
				B: uint8(b / n),
				A: uint8(a / n),
			})
		}
	}
	return dst
}
