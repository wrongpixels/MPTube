//go:build windows

package main

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"

	"youtubemp3/internal/uidim"
)

// Layout at 96 DPI (walk 1/96" units). Scale to pixels exactly once via uidim.Scale.
const (
	winMinW = 720
	winMinH = 520
	winW    = 760
	winH    = 560

	pagePadL = uidim.PagePadL
	pagePadT = 20
	pagePadR = 24
	pagePadB = 20
	pageGap  = 14

	headerH    = 36
	headerIcon = 32

	cardPad     = 12
	previewH    = 126
	thumbW      = uidim.ThumbWidth
	thumbH      = uidim.ThumbHeight
	labelColW   = uidim.LabelCol
	rowControlH = uidim.ButtonHeight
	urlCardH    = uidim.URLCardPad + uidim.ButtonHeight + uidim.URLCardPad

	btnH         = uidim.ButtonHeight
	btnPasteW    = uidim.PasteWidth
	btnBrowseW   = uidim.BrowseWidth
	btnCancelW   = uidim.CancelWidth
	btnDownloadW = uidim.DownloadWidth
	btnRetryW    = uidim.RetryWidth
	btnRevealW   = uidim.RevealWidth

	progressH      = uidim.ProgressHeight
	previewTextMin = uidim.PreviewTextMin
	titleSlotH     = 40
	durationSlotH  = 18
	footerLineH    = uidim.FooterLineHeight
	footerTextH    = uidim.FooterStatusHeight
)

func colorPage() walk.Color      { return walk.RGB(243, 244, 246) }
func colorCard() walk.Color      { return walk.RGB(255, 255, 255) }
func colorText() walk.Color      { return walk.RGB(31, 36, 46) }
func colorMuted() walk.Color     { return walk.RGB(107, 114, 128) }
func colorLabel() walk.Color     { return walk.RGB(75, 85, 99) }
func colorBorder() walk.Color    { return walk.RGB(209, 213, 219) } // #D1D5DB
func colorPrimary() walk.Color   { return walk.RGB(37, 99, 235) }
func colorPriHover() walk.Color  { return walk.RGB(29, 88, 226) }
func colorPriDown() walk.Color   { return walk.RGB(29, 78, 216) }
func colorPriOff() walk.Color    { return walk.RGB(186, 193, 206) }
func colorSec() walk.Color       { return walk.RGB(255, 255, 255) }
func colorSecHover() walk.Color  { return walk.RGB(243, 244, 246) }
func colorSecDown() walk.Color   { return walk.RGB(229, 231, 235) }
func colorSecOff() walk.Color    { return walk.RGB(243, 244, 246) }
func colorThumbWell() walk.Color { return walk.RGB(31, 41, 55) }
func colorTrack() walk.Color     { return walk.RGB(226, 232, 240) }
func colorWhite() walk.Color     { return walk.RGB(255, 255, 255) }
func colorFocus() walk.Color     { return walk.RGB(29, 78, 216) }

type btnStyle int

const (
	styleSecondary btnStyle = iota
	stylePrimary
)

type styledButton struct {
	a        *App
	widget   *walk.CustomWidget
	key      string
	icon     iconKind
	style    btnStyle
	backdrop walk.Color
	hover    bool
	pressed  bool
	click    func()
	canClick func() bool
}

func (a *App) makeButton(key string, icon iconKind, style btnStyle, click func()) *styledButton {
	return &styledButton{a: a, key: key, icon: icon, style: style, backdrop: colorPage(), click: click}
}

func (b *styledButton) enabled() bool {
	if b.canClick != nil {
		return b.canClick()
	}
	return b.widget == nil || b.widget.Enabled()
}

func fillRectPixels(canvas *walk.Canvas, bounds walk.Rectangle, c walk.Color) error {
	brush, err := walk.NewSolidColorBrush(c)
	if err != nil {
		return err
	}
	defer brush.Dispose()
	return canvas.FillRectanglePixels(brush, bounds)
}

func (b *styledButton) paintPixels(canvas *walk.Canvas, _ walk.Rectangle) error {
	if b.widget == nil {
		return nil
	}
	bounds := b.widget.ClientBoundsPixels()
	// Always paint the full HWND first. PaintBuffered bitmaps start black;
	// GDI RoundRect does not touch the four corners outside the curve.
	if err := fillRectPixels(canvas, bounds, b.backdrop); err != nil {
		return err
	}

	enabled := b.enabled()
	bg, fg, border := colorSec(), colorText(), colorBorder()
	if b.style == stylePrimary {
		fg = colorWhite()
		switch {
		case !enabled:
			bg, border = colorPriOff(), colorPriOff()
		case b.pressed:
			bg, border = colorPriDown(), colorPriDown()
		case b.hover:
			bg, border = colorPriHover(), colorPriHover()
		default:
			bg, border = colorPrimary(), colorPrimary()
		}
	} else {
		switch {
		case !enabled:
			fg = colorMuted()
			bg = colorSecOff()
			border = colorBorder()
		case b.pressed:
			bg = colorSecDown()
		case b.hover:
			bg = colorSecHover()
		default:
			bg = colorSec()
		}
	}

	dpi := b.widget.DPI()
	if dpi <= 0 {
		dpi = 96
	}
	inset := 0
	face := walk.Rectangle{
		X:      bounds.X + inset,
		Y:      bounds.Y + inset,
		Width:  bounds.Width - inset*2,
		Height: bounds.Height - inset*2,
	}
	if face.Width < 2 || face.Height < 2 {
		return nil
	}
	radius := uidim.Scale(uidim.ButtonRadius, dpi)
	rad := walk.Size{Width: radius, Height: radius}
	brush, err := walk.NewSolidColorBrush(bg)
	if err != nil {
		return err
	}
	defer brush.Dispose()
	_ = canvas.FillRoundedRectanglePixels(brush, face, rad)
	pen, err := walk.NewCosmeticPen(walk.PenSolid, border)
	if err == nil {
		_ = canvas.DrawRoundedRectanglePixels(pen, face, rad)
		pen.Dispose()
	}

	if b.widget.Focused() && enabled {
		focusInset := uidim.Scale(3, dpi)
		if focusInset < 2 {
			focusInset = 2
		}
		ring := walk.Rectangle{
			X:      face.X + focusInset,
			Y:      face.Y + focusInset,
			Width:  face.Width - focusInset*2,
			Height: face.Height - focusInset*2,
		}
		if ring.Width > 4 && ring.Height > 4 {
			if fp, err := walk.NewCosmeticPen(walk.PenSolid, colorFocus()); err == nil {
				_ = canvas.DrawRoundedRectanglePixels(fp, ring, walk.Size{Width: radius, Height: radius})
				fp.Dispose()
			}
		}
	}

	label := b.a.t(b.key)
	font := b.a.btnFont
	if b.style == stylePrimary && b.a.dlFont != nil {
		font = b.a.dlFont
	}
	slot := uidim.Scale(uidim.IconSlot, dpi)
	gap := uidim.Scale(uidim.IconTextGap, dpi)
	pad := uidim.Scale(uidim.ButtonInset, dpi)
	textW := uidim.Scale(72, dpi)
	if font != nil {
		if measured, _, mErr := canvas.MeasureTextPixels(label, font, face, walk.TextSingleLine|walk.TextCalcRect); mErr == nil {
			textW = measured.Width
		}
	}
	groupW := slot + gap + textW
	x := face.X + (face.Width-groupW)/2
	if x < face.X+pad {
		x = face.X + pad
	}
	iconRect := walk.Rectangle{
		X:      x,
		Y:      face.Y + (face.Height-slot)/2,
		Width:  slot,
		Height: slot,
	}
	drawIcon(canvas, iconRect, b.icon, fg)
	if font != nil {
		textRect := walk.Rectangle{
			X:      x + slot + gap,
			Y:      face.Y,
			Width:  face.Width - (x - face.X) - slot - gap - pad,
			Height: face.Height,
		}
		_ = canvas.DrawTextPixels(label, font, fg, textRect, walk.TextLeft|walk.TextVCenter|walk.TextSingleLine|walk.TextNoPrefix)
	}
	return nil
}

func (b *styledButton) mouseDown(x, y int, button walk.MouseButton) {
	if button != walk.LeftButton || !b.enabled() {
		return
	}
	if b.widget != nil {
		_ = b.widget.SetFocus()
	}
	b.pressed = true
	if b.widget != nil {
		b.widget.Invalidate()
	}
}

func (b *styledButton) mouseMove(x, y int, button walk.MouseButton) {
	if b.widget == nil {
		return
	}
	box := b.widget.ClientBoundsPixels()
	hover := x >= 0 && y >= 0 && x < box.Width && y < box.Height
	if hover != b.hover {
		b.hover = hover
		b.widget.Invalidate()
	}
}

func (b *styledButton) mouseUp(x, y int, button walk.MouseButton) {
	pressed := b.pressed
	b.pressed = false
	if b.widget != nil {
		b.widget.Invalidate()
	}
	if button != walk.LeftButton || !pressed || !b.enabled() {
		return
	}
	box := b.widget.ClientBoundsPixels()
	if x < 0 || y < 0 || x >= box.Width || y >= box.Height {
		return
	}
	if b.click != nil {
		b.click()
	}
}

func (b *styledButton) keyDown(key walk.Key) {
	if !b.enabled() {
		return
	}
	if key == walk.KeyReturn || key == walk.KeySpace {
		if b.click != nil {
			b.click()
		}
	}
}

func (b *styledButton) wire() {
	if b == nil || b.widget == nil {
		return
	}
	b.widget.FocusedChanged().Attach(func() {
		if b.widget != nil {
			b.widget.Invalidate()
		}
	})
}

func (b *styledButton) invalidate() {
	if b != nil && b.widget != nil {
		b.widget.Invalidate()
	}
}

func (a *App) paintProgressPixels(canvas *walk.Canvas, _ walk.Rectangle) error {
	if a.progress == nil {
		return nil
	}
	bounds := a.progress.ClientBoundsPixels()
	if err := fillRectPixels(canvas, bounds, colorPage()); err != nil {
		return err
	}
	dpi := a.progress.DPI()
	radN := uidim.Scale(progressH, dpi)
	rad := walk.Size{Width: radN, Height: radN}
	track, err := walk.NewSolidColorBrush(colorTrack())
	if err != nil {
		return err
	}
	defer track.Dispose()
	_ = canvas.FillRoundedRectanglePixels(track, bounds, rad)
	a.maybeLogPaint()
	fill, err := walk.NewSolidColorBrush(colorPrimary())
	if err != nil {
		return err
	}
	defer fill.Dispose()
	if a.progressDeterminate {
		if a.progressValue > 0 {
			w := 0
			if bounds.Width > 0 {
				w = bounds.Width * a.progressValue / 100
			}
			if w < 4 {
				w = 4
			}
			bar := bounds
			bar.Width = w
			_ = canvas.FillRoundedRectanglePixels(fill, bar, rad)
		}
		return nil
	}
	if !a.progressActive {
		return nil
	}
	span := bounds.Width * 28 / 100
	if span < uidim.Scale(24, dpi) {
		span = uidim.Scale(24, dpi)
	}
	if span > bounds.Width {
		span = bounds.Width
	}
	travel := bounds.Width - span
	x := bounds.X
	if travel > 0 {
		x += travel * a.progressPulse / 100
	}
	bar := walk.Rectangle{X: x, Y: bounds.Y, Width: span, Height: bounds.Height}
	_ = canvas.FillRoundedRectanglePixels(fill, bar, rad)
	return nil
}

func chromeBtn(b *styledButton, width int) CustomWidget {
	return CustomWidget{
		AssignTo:            &b.widget,
		MinSize:             Size{Width: width, Height: btnH},
		MaxSize:             Size{Width: width, Height: btnH},
		Alignment:           AlignHCenterVCenter,
		Style:               uint32(win.WS_TABSTOP),
		Accessibility:       Accessibility{Role: AccRolePushbutton, Name: b.a.t(b.key), DefaultAction: "Press"},
		OnKeyDown:           b.keyDown,
		ClearsBackground:    false,
		InvalidatesOnResize: true,
		PaintMode:           PaintBuffered,
		PaintPixels:         b.paintPixels,
		OnMouseDown:         b.mouseDown,
		OnMouseMove:         b.mouseMove,
		OnMouseUp:           b.mouseUp,
	}
}

type clipLabel struct {
	widget   *walk.CustomWidget
	text     string
	tip      string
	color    func() walk.Color
	backdrop func() walk.Color
	path     bool
	lines    int
}

func (c *clipLabel) setText(full, tip string) {
	if c == nil {
		return
	}
	c.text = full
	c.tip = tip
	if c.widget == nil {
		return
	}
	hint := tip
	if hint == "" {
		hint = full
	}
	_ = c.widget.SetToolTipText(hint)
	c.widget.Invalidate()
}

func (c *clipLabel) paint(canvas *walk.Canvas, _ walk.Rectangle) error {
	if c == nil || c.widget == nil {
		return nil
	}
	bounds := c.widget.ClientBoundsPixels()
	bg := colorPage()
	if c.backdrop != nil {
		bg = c.backdrop()
	}
	if err := fillRectPixels(canvas, bounds, bg); err != nil {
		return err
	}
	if c.text == "" {
		return nil
	}
	fg := colorMuted()
	if c.color != nil {
		fg = c.color()
	}
	flags := walk.TextLeft | walk.TextVCenter | walk.TextSingleLine | walk.TextNoPrefix | walk.TextEndEllipsis
	if c.path {
		flags = walk.TextLeft | walk.TextVCenter | walk.TextSingleLine | walk.TextNoPrefix | walk.TextPathEllipsis
	} else if c.lines >= 2 {
		flags = walk.TextLeft | walk.TextWordbreak | walk.TextEditControl | walk.TextEndEllipsis | walk.TextNoPrefix
	}
	font := c.widget.Font()
	return canvas.DrawTextPixels(c.text, font, fg, bounds, flags)
}

func (c *clipLabel) decl(minH int) CustomWidget {
	return CustomWidget{
		AssignTo:            &c.widget,
		MinSize:             Size{Width: 80, Height: minH},
		MaxSize:             Size{Height: minH},
		AlwaysConsumeSpace:  true,
		ClearsBackground:    false,
		InvalidatesOnResize: true,
		PaintMode:           PaintBuffered,
		PaintPixels:         c.paint,
	}
}

func (a *App) paintThumb(canvas *walk.Canvas, _ walk.Rectangle) error {
	if a.thumbWidget == nil {
		return nil
	}
	bounds := a.thumbWidget.ClientBoundsPixels()
	if err := fillRectPixels(canvas, bounds, colorThumbWell()); err != nil {
		return err
	}
	if a.thumbImg == nil {
		return nil
	}
	x, y, w, h := uidim.Contain(bounds.Width, bounds.Height, a.thumbPxW, a.thumbPxH)
	if w < 1 || h < 1 {
		return nil
	}
	dest := walk.Rectangle{
		X:      bounds.X + x,
		Y:      bounds.Y + y,
		Width:  w,
		Height: h,
	}
	return canvas.DrawImageStretchedPixels(a.thumbImg, dest)
}
