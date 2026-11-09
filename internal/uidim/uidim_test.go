package uidim

import "testing"

func TestScaleOnceAtCommonDPIs(t *testing.T) {
	cases := []struct {
		dpi     int
		in, out int
	}{
		{96, 32, 32},
		{120, 32, 40},
		{144, 32, 48},
		{96, 112, 112},
		{120, 112, 140},
		{144, 112, 168},
		{96, 124, 124},
		{144, 176, 264},
	}
	for _, tc := range cases {
		if got := Scale(tc.in, tc.dpi); got != tc.out {
			t.Fatalf("Scale(%d,%d)=%d want %d", tc.in, tc.dpi, got, tc.out)
		}
	}
}

func TestButtonHeightStaysCompact(t *testing.T) {
	if ButtonHeight < 30 || ButtonHeight > 34 {
		t.Fatalf("ButtonHeight %d not in 30-34", ButtonHeight)
	}
	for _, dpi := range []int{96, 120, 144} {
		_, h := ButtonPixels(PasteWidth, dpi)
		if h != Scale(ButtonHeight, dpi) {
			t.Fatalf("dpi %d button height %d", dpi, h)
		}
	}
}

func TestFilenameAndFolderFieldsShareOrigin(t *testing.T) {
	for _, dpi := range []int{96, 120, 144} {
		if FieldOriginX(dpi) != Scale(PagePadL, dpi)+Scale(LabelCol, dpi)+Scale(RowGap, dpi) {
			t.Fatalf("dpi %d origin %d", dpi, FieldOriginX(dpi))
		}
	}
	if FieldOriginX(96) != 122 {
		t.Fatalf("96-dpi origin %d", FieldOriginX(96))
	}
}

func TestURLCardFitsCompactButton(t *testing.T) {
	if URLCardHeight(96) != 48 {
		t.Fatalf("url card 96dpi=%d", URLCardHeight(96))
	}
	if URLCardHeight(144) != 72 {
		t.Fatalf("url card 144dpi=%d", URLCardHeight(144))
	}
}

func TestPreviewColumnsStableAcrossStatesAndWidths(t *testing.T) {
	states := []string{
		"Pega un enlace de YouTube",
		"Cargando…",
		"Loading…",
		"Couldn't load this video.",
		"A very long multilingual title that would previously stretch the label and shove the thumbnail",
	}
	widths := []int{500, 700, 900, 1100}
	for _, dpi := range []int{96, 120, 144} {
		for _, cardW := range widths {
			var firstThumbX, firstTextX, firstThumbW int
			for i, title := range states {
				// Layout ignores the title string on purpose: columns are fixed.
				_ = title
				thumbX, thumbW, textX, textW := PreviewColumns(cardW, dpi)
				if i == 0 {
					firstThumbX, firstTextX, firstThumbW = thumbX, textX, thumbW
				}
				if thumbX != firstThumbX || textX != firstTextX || thumbW != firstThumbW {
					t.Fatalf("dpi %d card %d state %q moved columns", dpi, cardW, title)
				}
				if thumbX != Scale(CardPad, dpi) {
					t.Fatalf("thumb origin dpi %d: %d", dpi, thumbX)
				}
				if thumbW != Scale(ThumbWidth, dpi) {
					t.Fatalf("thumb width dpi %d: %d", dpi, thumbW)
				}
				if textX != thumbX+thumbW+Scale(PreviewGap, dpi) {
					t.Fatalf("text origin dpi %d: %d", dpi, textX)
				}
				if textW != cardW-textX-Scale(CardPad, dpi) {
					t.Fatalf("text width dpi %d card %d: %d", dpi, cardW, textW)
				}
			}
		}
	}
	thumbX, thumbW, textX, textW := PreviewColumns(700, 96)
	if thumbX != 12 || thumbW != 160 || textX != 186 || textW != 502 {
		t.Fatalf("96dpi/700: %d %d %d %d", thumbX, thumbW, textX, textW)
	}
	_, tw144, tx144, _ := PreviewColumns(1050, 144)
	if tw144 != 240 || tx144 != Scale(CardPad, 144)+240+Scale(PreviewGap, 144) {
		t.Fatalf("144dpi: thumbW %d textX %d", tw144, tx144)
	}
}

func TestProgressSlotIsAllocated(t *testing.T) {
	if ProgressHeight < 6 || ProgressHeight > 12 {
		t.Fatalf("progress height %d", ProgressHeight)
	}
	if ProgressSlotHeight(96) != 8 || ProgressSlotHeight(144) != 12 {
		t.Fatalf("scaled progress %d %d", ProgressSlotHeight(96), ProgressSlotHeight(144))
	}
}
