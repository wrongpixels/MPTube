package appicon

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"testing"
)

func TestRenderSizeAndBrandColor(t *testing.T) {
	img := Render(32)
	if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 32 {
		t.Fatalf("size %v", img.Bounds())
	}
	c := img.RGBAAt(2, 16) // left edge inside the rounded tile
	if c.A < 200 || c.B < 180 || int(c.B) <= int(c.R)+40 {
		t.Fatalf("expected blue tile, got %+v", c)
	}
	// Corners of a rounded square should be transparent.
	corner := img.RGBAAt(0, 0)
	if corner.A > 40 {
		t.Fatalf("corner should be transparent, got %+v", corner)
	}
}

func TestICOContainsExpectedPNGFrames(t *testing.T) {
	data, err := ICO()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 100 {
		t.Fatalf("too small: %d", len(data))
	}
	if binary.LittleEndian.Uint16(data[0:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		t.Fatal("not an ico")
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count != len(Sizes) {
		t.Fatalf("count %d want %d", count, len(Sizes))
	}
	found256 := false
	found16 := false
	for i := 0; i < count; i++ {
		entry := data[6+i*16 : 6+i*16+16]
		w := entry[0]
		size := binary.LittleEndian.Uint32(entry[8:12])
		off := binary.LittleEndian.Uint32(entry[12:16])
		if int(off)+int(size) > len(data) {
			t.Fatalf("frame %d out of range", i)
		}
		blob := data[off : off+size]
		if !bytes.HasPrefix(blob, []byte{0x89, 'P', 'N', 'G'}) {
			t.Fatalf("frame %d is not png", i)
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(blob))
		if err != nil {
			t.Fatal(err)
		}
		if w == 0 {
			found256 = cfg.Width == 256
		}
		if w == 16 {
			found16 = cfg.Width == 16
		}
	}
	if !found16 || !found256 {
		t.Fatalf("missing sizes 16/256 (16=%v 256=%v)", found16, found256)
	}
}
