package uidim

import "testing"

func TestSlotStaysSixteenByNine(t *testing.T) {
	if ThumbWidth != 160 || ThumbHeight != 90 {
		t.Fatalf("slot %dx%d", ThumbWidth, ThumbHeight)
	}
	if ThumbWidth*9 != ThumbHeight*16 {
		t.Fatalf("slot is not 16:9: %d x %d", ThumbWidth, ThumbHeight)
	}
}

func TestContainKeepsAspectInsideSlot(t *testing.T) {
	cases := []struct {
		name           string
		bw, bh, iw, ih int
		x, y, w, h     int
	}{
		{name: "16:9 fills 160x90", bw: 160, bh: 90, iw: 1280, ih: 720, x: 0, y: 0, w: 160, h: 90},
		{name: "16:9 fills 144dpi slot", bw: 240, bh: 135, iw: 1280, ih: 720, x: 0, y: 0, w: 240, h: 135},
		{name: "square side margins", bw: 160, bh: 90, iw: 720, ih: 720, x: 35, y: 0, w: 90, h: 90},
		{name: "square at 144dpi", bw: 240, bh: 135, iw: 720, ih: 720, x: 52, y: 0, w: 135, h: 135},
		{name: "portrait side margins", bw: 160, bh: 90, iw: 360, ih: 640, x: 55, y: 0, w: 50, h: 90},
		{name: "4:3 side margins", bw: 160, bh: 90, iw: 480, ih: 360, x: 20, y: 0, w: 120, h: 90},
		{name: "wider than 16:9 vertical margins", bw: 160, bh: 90, iw: 210, ih: 90, x: 0, y: 11, w: 160, h: 68},
		{name: "empty", bw: 160, bh: 90, iw: 0, ih: 10, x: 0, y: 0, w: 0, h: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x, y, w, h := Contain(tc.bw, tc.bh, tc.iw, tc.ih)
			if x != tc.x || y != tc.y || w != tc.w || h != tc.h {
				t.Fatalf("got %d,%d %dx%d", x, y, w, h)
			}
			if w == 0 && h == 0 {
				return
			}
			if x < 0 || y < 0 || x+w > tc.bw || y+h > tc.bh {
				t.Fatalf("dest escapes the slot")
			}
			if x != (tc.bw-w)/2 || y != (tc.bh-h)/2 {
				t.Fatalf("not centered")
			}
			// Uniform scale, truncated toward zero, so the painted box stays inside
			// the source aspect by less than a pixel on each axis.
			if diff := abs(w*tc.ih - h*tc.iw); diff > tc.iw+tc.ih {
				t.Fatalf("aspect drifted by %d", diff)
			}
		})
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
