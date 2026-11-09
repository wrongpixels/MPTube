//go:build windows

package main

import "github.com/lxn/walk"

type iconKind int

const (
	iconNone iconKind = iota
	iconPaste
	iconFolder
	iconDownload
	iconCancel
	iconRetry
)

// drawIcon paints a mark in native pixels, mapping a 16×16 design into bounds.
func drawIcon(canvas *walk.Canvas, bounds walk.Rectangle, kind iconKind, color walk.Color) {
	pen, err := walk.NewCosmeticPen(walk.PenSolid, color)
	if err != nil {
		return
	}
	defer pen.Dispose()
	size := bounds.Width
	if bounds.Height < size {
		size = bounds.Height
	}
	if size < 8 {
		size = 8
	}
	x := bounds.X + (bounds.Width-size)/2
	y := bounds.Y + (bounds.Height-size)/2
	s := func(v int) int { return v * size / 16 }
	switch kind {
	case iconPaste:
		_ = canvas.DrawRectanglePixels(pen, walk.Rectangle{X: x + s(4), Y: y + s(3), Width: s(9), Height: s(11)})
		_ = canvas.DrawRectanglePixels(pen, walk.Rectangle{X: x + s(6), Y: y + s(1), Width: s(5), Height: s(3)})
		_ = canvas.DrawLinePixels(pen, walk.Point{X: x + s(6), Y: y + s(7)}, walk.Point{X: x + s(11), Y: y + s(7)})
		_ = canvas.DrawLinePixels(pen, walk.Point{X: x + s(6), Y: y + s(10)}, walk.Point{X: x + s(10), Y: y + s(10)})
	case iconFolder:
		_ = canvas.DrawRectanglePixels(pen, walk.Rectangle{X: x + s(1), Y: y + s(5), Width: s(14), Height: s(9)})
		_ = canvas.DrawRectanglePixels(pen, walk.Rectangle{X: x + s(1), Y: y + s(3), Width: s(6), Height: s(3)})
	case iconDownload:
		cx := x + size/2
		_ = canvas.DrawLinePixels(pen, walk.Point{X: cx, Y: y + s(2)}, walk.Point{X: cx, Y: y + s(11)})
		_ = canvas.DrawLinePixels(pen, walk.Point{X: cx - s(4), Y: y + s(8)}, walk.Point{X: cx, Y: y + s(12)})
		_ = canvas.DrawLinePixels(pen, walk.Point{X: cx + s(4), Y: y + s(8)}, walk.Point{X: cx, Y: y + s(12)})
		_ = canvas.DrawLinePixels(pen, walk.Point{X: x + s(3), Y: y + s(14)}, walk.Point{X: x + s(13), Y: y + s(14)})
	case iconCancel:
		_ = canvas.DrawLinePixels(pen, walk.Point{X: x + s(4), Y: y + s(4)}, walk.Point{X: x + s(12), Y: y + s(12)})
		_ = canvas.DrawLinePixels(pen, walk.Point{X: x + s(12), Y: y + s(4)}, walk.Point{X: x + s(4), Y: y + s(12)})
	case iconRetry:
		_ = canvas.DrawRectanglePixels(pen, walk.Rectangle{X: x + s(3), Y: y + s(3), Width: s(10), Height: s(10)})
		_ = canvas.DrawLinePixels(pen, walk.Point{X: x + s(13), Y: y + s(3)}, walk.Point{X: x + s(13), Y: y + s(7)})
		_ = canvas.DrawLinePixels(pen, walk.Point{X: x + s(10), Y: y + s(7)}, walk.Point{X: x + s(13), Y: y + s(7)})
	}
}
