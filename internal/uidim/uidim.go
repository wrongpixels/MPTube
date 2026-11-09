// Package uidim holds 96-DPI layout numbers shared by the Windows UI and tests.
// Walk declarative sizes are expressed in these 1/96" units and must be scaled
// to native pixels exactly once (value * dpi / 96).
package uidim

const (
	ButtonHeight = 32
	ButtonRadius = 6
	ButtonInset  = 10
	IconSlot     = 16
	IconTextGap  = 8

	PasteWidth    = 112
	BrowseWidth   = 124
	CancelWidth   = 124
	DownloadWidth = 176
	RetryWidth    = 124
	RevealWidth   = 160

	LabelCol   = 88
	PagePadL   = 24
	RowGap     = 10
	URLCardPad = 8

	ThumbWidth     = 160
	ThumbHeight    = 90
	PreviewGap     = 14
	CardPad        = 12
	PreviewTextMin = 240
	ProgressHeight = 8

	FooterLineHeight   = 18
	FooterStatusHeight = 40
)

// Scale converts a 96-DPI unit to native pixels for the given DPI.
func Scale(value96, dpi int) int {
	if dpi <= 0 {
		dpi = 96
	}
	return value96 * dpi / 96
}

// ButtonPixels is the on-screen size of a chrome button at dpi.
func ButtonPixels(width96, dpi int) (w, h int) {
	return Scale(width96, dpi), Scale(ButtonHeight, dpi)
}

// FieldOriginX is where filename and folder fields should start (same column).
func FieldOriginX(dpi int) int {
	return Scale(PagePadL+LabelCol+RowGap, dpi)
}

// URLCardHeight is the fixed URL row height: padding + compact button + padding.
func URLCardHeight(dpi int) int {
	return Scale(URLCardPad+ButtonHeight+URLCardPad, dpi)
}

func RowHeight(dpi int) int {
	return Scale(ButtonHeight, dpi)
}

// PreviewColumns is the card-local geometry of the preview row.
// Origins do not depend on title text, language, loading/loaded state, or retry.
func PreviewColumns(cardWidthPx, dpi int) (thumbX, thumbW, textX, textW int) {
	pad := Scale(CardPad, dpi)
	thumbW = Scale(ThumbWidth, dpi)
	gap := Scale(PreviewGap, dpi)
	thumbX = pad
	textX = pad + thumbW + gap
	textW = cardWidthPx - textX - pad
	if textW < 0 {
		textW = 0
	}
	return
}

func ProgressSlotHeight(dpi int) int {
	return Scale(ProgressHeight, dpi)
}
