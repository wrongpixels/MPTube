//go:build windows

package main

import (
	"context"
	"fmt"
	"image"
	"strings"
	"sync"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"youtubemp3/internal/appicon"
	"youtubemp3/internal/apppaths"
	"youtubemp3/internal/clip"
	"youtubemp3/internal/debounce"
	"youtubemp3/internal/diaglog"
	"youtubemp3/internal/download"
	"youtubemp3/internal/folderpick"
	"youtubemp3/internal/folderui"
	"youtubemp3/internal/i18n"
	"youtubemp3/internal/infocache"
	"youtubemp3/internal/metadata"
	"youtubemp3/internal/preview"
	"youtubemp3/internal/reveal"
	"youtubemp3/internal/sanitize"
	"youtubemp3/internal/settings"
	"youtubemp3/internal/tools"
	"youtubemp3/internal/uistate"
	"youtubemp3/internal/yturl"
)

const (
	appName    = "YouTube MP3 Saver"
	appVersion = "13.0.0"
)

type appMode int

const (
	modeIdle appMode = iota
	modeFetch
	modeSave
)

type previewKind int

const (
	previewHint previewKind = iota
	previewLoading
	previewError
	previewNotVideo
	previewReady
)

type App struct {
	mw            *walk.MainWindow
	headingIcon   *walk.ImageView
	headingLabel  *walk.TextLabel
	urlEdit       *walk.LineEdit
	thumbWidget   *walk.CustomWidget
	titleClip     clipLabel
	durationLabel *walk.TextLabel
	fileLabel     *walk.TextLabel
	fileEdit      *walk.LineEdit
	folderLabel   *walk.TextLabel
	folderCombo   *walk.ComboBox
	langCombo     *walk.ComboBox
	progress      *walk.CustomWidget
	statusClip    clipLabel
	pathClip      clipLabel

	pasteSB    *styledButton
	browseSB   *styledButton
	retrySB    *styledButton
	downloadSB *styledButton
	revealSB   *styledButton

	footer          uistate.Footer
	lastPaintLog    time.Time
	lastPaintDet    bool
	lastPaintPct    int
	lastPaintActive bool

	store         settings.Store
	current       settings.Settings
	lang          string
	updatingCombo bool
	updatingURL   bool
	updatingLang  bool

	gate         metadata.Gate
	thumbImg     walk.Image
	thumbPxW     int
	thumbPxH     int
	headerBmp    walk.Image
	previewState previewKind
	lastDuration string
	debouncer    *debounce.Debouncer

	btnFont             *walk.Font
	dlFont              *walk.Font
	headingFont         *walk.Font
	progressValue       int
	progressActive      bool
	progressDeterminate bool
	progressPulse       int
	pulseStop           chan struct{}
	lastSavedPath       string

	statusKey  string
	statusArgs []any

	mu     sync.Mutex
	cancel context.CancelFunc
	mode   appMode
	workID uint64
}

func main() {
	app := &App{
		debouncer: debounce.New(400 * time.Millisecond),
		statusKey: "status.hint",
	}
	if err := app.initStore(); err != nil {
		walk.MsgBox(nil, appName, err.Error(), walk.MsgBoxIconError)
		return
	}
	if err := app.buildUI(); err != nil {
		walk.MsgBox(nil, appName, err.Error(), walk.MsgBoxIconError)
		return
	}
}

func (a *App) t(key string) string { return i18n.T(a.lang, key) }

func (a *App) initStore() error {
	path, err := apppaths.SettingsPath()
	if err != nil {
		return err
	}
	a.store = settings.Store{Path: path}
	st, err := a.store.Load()
	if err != nil {
		return err
	}
	a.current = st
	a.lang = i18n.Resolve(st.Language, i18n.SystemLocale())
	a.current.Language = a.lang
	return nil
}

func (a *App) buildUI() error {
	a.pasteSB = a.makeButton("paste", iconPaste, styleSecondary, a.onPaste)
	a.browseSB = a.makeButton("folder.browse", iconFolder, styleSecondary, a.onBrowse)
	a.retrySB = a.makeButton("retry", iconRetry, styleSecondary, a.onRetry)
	a.downloadSB = a.makeButton("download", iconDownload, stylePrimary, a.onDownload)
	a.downloadSB.canClick = a.downloadEnabled
	a.revealSB = a.makeButton("reveal", iconFolder, styleSecondary, a.onReveal)
	a.titleClip = clipLabel{color: colorText, backdrop: colorCard, lines: 2}
	a.statusClip = clipLabel{color: colorMuted, backdrop: colorPage}
	a.pathClip = clipLabel{color: colorMuted, backdrop: colorPage, path: true}

	bg := SolidColorBrush{Color: colorPage()}
	card := SolidColorBrush{Color: colorCard()}
	well := SolidColorBrush{Color: colorThumbWell()}
	if err := (MainWindow{
		AssignTo:   &a.mw,
		Title:      a.t("app.title"),
		MinSize:    Size{Width: winMinW, Height: winMinH},
		Size:       Size{Width: winW, Height: winH},
		Font:       Font{Family: "Segoe UI", PointSize: 10},
		Background: bg,
		Layout:     VBox{Margins: Margins{Left: pagePadL, Top: pagePadT, Right: pagePadR, Bottom: pagePadB}, Spacing: pageGap},
		Children: []Widget{
			Composite{
				Layout:     HBox{MarginsZero: true, Spacing: 10},
				Background: bg,
				MinSize:    Size{Height: headerH},
				Children: []Widget{
					ImageView{
						AssignTo: &a.headingIcon,
						MinSize:  Size{Width: headerIcon, Height: headerIcon},
						MaxSize:  Size{Width: headerIcon, Height: headerIcon},
						Mode:     ImageViewModeZoom,
					},
					TextLabel{
						AssignTo:  &a.headingLabel,
						Text:      a.t("app.title"),
						TextColor: colorText(),
					},
					HSpacer{},
					ComboBox{
						AssignTo:              &a.langCombo,
						Model:                 []string{"EN", "ES"},
						MaxSize:               Size{Width: 64, Height: 32},
						MinSize:               Size{Width: 56, Height: 32},
						OnCurrentIndexChanged: a.onLanguageChanged,
					},
				},
			},
			Composite{
				Background: card,
				Layout:     HBox{Margins: Margins{Left: cardPad, Top: 8, Right: cardPad, Bottom: 8}, Spacing: 10},
				MinSize:    Size{Height: urlCardH},
				MaxSize:    Size{Height: urlCardH},
				Children: []Widget{
					LineEdit{
						AssignTo:      &a.urlEdit,
						CueBanner:     a.t("url.cue"),
						StretchFactor: 1,
						Alignment:     AlignHNearVCenter,
						OnTextChanged: a.onURLChanged,
					},
					chromeBtn(a.pasteSB, btnPasteW),
				},
			},
			Composite{
				Background: card,
				MinSize:    Size{Height: previewH},
				MaxSize:    Size{Height: previewH},
				Layout:     Grid{Columns: 2, Spacing: 14, Alignment: AlignHNearVNear, Margins: Margins{Left: cardPad, Top: cardPad, Right: cardPad, Bottom: cardPad}},
				Children: []Widget{
					Composite{
						Column:     0,
						Background: well,
						MinSize:    Size{Width: thumbW, Height: thumbH},
						MaxSize:    Size{Width: thumbW, Height: thumbH},
						Layout:     VBox{MarginsZero: true, Spacing: 0},
						Children: []Widget{
							CustomWidget{
								AssignTo:            &a.thumbWidget,
								MinSize:             Size{Width: thumbW, Height: thumbH},
								MaxSize:             Size{Width: thumbW, Height: thumbH},
								AlwaysConsumeSpace:  true,
								ClearsBackground:    false,
								InvalidatesOnResize: true,
								PaintMode:           PaintBuffered,
								PaintPixels:         a.paintThumb,
							},
						},
					},
					Composite{
						Column:        1,
						Layout:        VBox{MarginsZero: true, Spacing: 6},
						StretchFactor: 1,
						Children: []Widget{
							Composite{
								MinSize:            Size{Height: titleSlotH},
								MaxSize:            Size{Height: titleSlotH},
								AlwaysConsumeSpace: true,
								Layout:             VBox{MarginsZero: true, Spacing: 0},
								Children: []Widget{
									a.titleClip.decl(titleSlotH),
								},
							},
							TextLabel{
								AssignTo:           &a.durationLabel,
								Text:               "",
								TextColor:          colorMuted(),
								MinSize:            Size{Height: durationSlotH},
								MaxSize:            Size{Height: durationSlotH},
								AlwaysConsumeSpace: true,
							},
							Composite{
								Layout:             HBox{MarginsZero: true},
								MinSize:            Size{Height: btnH},
								MaxSize:            Size{Height: btnH},
								AlwaysConsumeSpace: true,
								Children: []Widget{
									chromeBtn(a.retrySB, btnRetryW),
									HSpacer{},
								},
							},
						},
					},
				},
			},
			Composite{
				Layout:     Grid{Columns: 3, Spacing: 10, Alignment: AlignHNearVCenter, MarginsZero: true},
				Background: bg,
				MinSize:    Size{Height: btnH},
				MaxSize:    Size{Height: btnH},
				Children: []Widget{
					Composite{
						Column:  0,
						MinSize: Size{Width: labelColW, Height: btnH},
						MaxSize: Size{Width: labelColW, Height: btnH},
						Layout:  HBox{MarginsZero: true},
						Children: []Widget{
							TextLabel{AssignTo: &a.fileLabel, Text: a.t("name.label"), Alignment: AlignHNearVCenter, TextColor: colorLabel()},
						},
					},
					LineEdit{AssignTo: &a.fileEdit, Column: 1, CueBanner: a.t("name.cue"), StretchFactor: 1, Alignment: AlignHNearVCenter},
					Composite{
						Column:  2,
						MinSize: Size{Width: 36, Height: btnH},
						MaxSize: Size{Width: 36, Height: btnH},
						Layout:  HBox{MarginsZero: true, Alignment: AlignHNearVCenter},
						Children: []Widget{
							TextLabel{Text: ".mp3", Alignment: AlignHNearVCenter, TextColor: colorMuted()},
						},
					},
				},
			},
			Composite{
				Layout:     Grid{Columns: 3, Spacing: 10, Alignment: AlignHNearVCenter, MarginsZero: true},
				Background: bg,
				MinSize:    Size{Height: btnH},
				MaxSize:    Size{Height: btnH},
				Children: []Widget{
					Composite{
						Column:  0,
						MinSize: Size{Width: labelColW, Height: btnH},
						MaxSize: Size{Width: labelColW, Height: btnH},
						Layout:  HBox{MarginsZero: true},
						Children: []Widget{
							TextLabel{AssignTo: &a.folderLabel, Text: a.t("folder.label"), Alignment: AlignHNearVCenter, TextColor: colorLabel()},
						},
					},
					ComboBox{
						AssignTo:              &a.folderCombo,
						Column:                1,
						Editable:              true,
						StretchFactor:         1,
						Alignment:             AlignHNearVCenter,
						OnCurrentIndexChanged: a.onFolderIndexChanged,
						OnEditingFinished:     a.onFolderEditingFinished,
					},
					Composite{
						Column:  2,
						MinSize: Size{Width: btnBrowseW, Height: btnH},
						MaxSize: Size{Width: btnBrowseW, Height: btnH},
						Layout:  VBox{MarginsZero: true},
						Children: []Widget{
							chromeBtn(a.browseSB, btnBrowseW),
						},
					},
				},
			},
			Composite{
				Layout:     HBox{MarginsZero: true, Spacing: 10},
				Background: bg,
				MinSize:    Size{Height: btnH},
				MaxSize:    Size{Height: btnH},
				Children: []Widget{
					HSpacer{},
					chromeBtn(a.downloadSB, btnDownloadW),
				},
			},
			Composite{
				MinSize:            Size{Height: progressH},
				MaxSize:            Size{Height: progressH},
				AlwaysConsumeSpace: true,
				Layout:             VBox{MarginsZero: true, Spacing: 0},
				Children: []Widget{
					CustomWidget{
						AssignTo:            &a.progress,
						MinSize:             Size{Height: progressH},
						MaxSize:             Size{Height: progressH},
						AlwaysConsumeSpace:  true,
						ClearsBackground:    false,
						InvalidatesOnResize: true,
						PaintMode:           PaintBuffered,
						PaintPixels:         a.paintProgressPixels,
					},
				},
			},
			Composite{
				Layout:  HBox{MarginsZero: true, Spacing: 10, Alignment: AlignHNearVCenter},
				MinSize: Size{Height: footerTextH},
				MaxSize: Size{Height: footerTextH},
				Children: []Widget{
					Composite{
						StretchFactor: 1,
						Layout:        VBox{MarginsZero: true, Spacing: 2},
						MinSize:       Size{Height: footerTextH},
						MaxSize:       Size{Height: footerTextH},
						Children: []Widget{
							Composite{
								MinSize:            Size{Height: footerLineH},
								MaxSize:            Size{Height: footerLineH},
								AlwaysConsumeSpace: true,
								Layout:             VBox{MarginsZero: true, Spacing: 0},
								Children:           []Widget{a.statusClip.decl(footerLineH)},
							},
							Composite{
								MinSize:            Size{Height: footerLineH},
								MaxSize:            Size{Height: footerLineH},
								AlwaysConsumeSpace: true,
								Layout:             VBox{MarginsZero: true, Spacing: 0},
								Children:           []Widget{a.pathClip.decl(footerLineH)},
							},
						},
					},
					Composite{
						MinSize:            Size{Width: btnRevealW, Height: btnH},
						MaxSize:            Size{Width: btnRevealW, Height: btnH},
						AlwaysConsumeSpace: true,
						Layout:             VBox{MarginsZero: true},
						Children: []Widget{
							chromeBtn(a.revealSB, btnRevealW),
						},
					},
				},
			},
		},
	}.Create()); err != nil {
		return err
	}

	a.applyChromeAssets()
	a.wireChromeButtons()

	a.updatingLang = true
	if a.lang == i18n.ES {
		_ = a.langCombo.SetCurrentIndex(1)
	} else {
		_ = a.langCombo.SetCurrentIndex(0)
	}
	a.updatingLang = false

	if a.retrySB != nil && a.retrySB.widget != nil {
		a.retrySB.widget.SetVisible(false)
	}
	a.footer = uistate.Reduce(uistate.Footer{}, uistate.Event{Kind: uistate.EvIdle, Line1: a.t("status.nothing")})
	a.applyFooterModel(a.footer)
	a.setTitleText(a.t("preview.hint"))

	a.refreshFolderCombo()
	if strings.TrimSpace(a.current.OutputFolder) != "" {
		if err := settings.FolderAvailable(a.current.OutputFolder); err != nil {
			a.setStatusText(i18n.FolderMessage(a.lang, a.current.OutputFolder, err))
		}
	}
	a.refreshActions()
	a.mw.Run()
	return nil
}

func (a *App) applyChromeAssets() {
	var err error
	a.dlFont, err = walk.NewFont("Segoe UI", 10, walk.FontBold)
	if err != nil {
		a.dlFont, _ = walk.NewFont("Segoe UI", 10, 0)
	}
	a.btnFont, _ = walk.NewFont("Segoe UI", 10, 0)
	a.headingFont, err = walk.NewFont("Segoe UI", 14, walk.FontBold)
	if err != nil {
		a.headingFont, _ = walk.NewFont("Segoe UI", 12, walk.FontBold)
	}
	if a.headingLabel != nil && a.headingFont != nil {
		a.headingLabel.SetFont(a.headingFont)
	}
	if bmp, err := walk.NewBitmapFromImage(appicon.Render(64)); err == nil {
		a.headerBmp = bmp
		if a.headingIcon != nil {
			_ = a.headingIcon.SetImage(bmp)
		}
	}
	if a.mw != nil {
		var ic *walk.Icon
		for _, id := range []int{2, 1, 3} {
			if got, err := walk.NewIconFromResourceId(id); err == nil {
				ic = got
				break
			}
		}
		if ic == nil {
			ic, _ = walk.NewIconFromImage(appicon.Render(32))
		}
		if ic != nil {
			_ = a.mw.SetIcon(ic)
		}
	}
}

func (a *App) wireChromeButtons() {
	if a.pasteSB != nil {
		a.pasteSB.backdrop = colorCard()
		a.pasteSB.wire()
	}
	if a.retrySB != nil {
		a.retrySB.backdrop = colorCard()
		a.retrySB.wire()
	}
	for _, b := range []*styledButton{a.browseSB, a.downloadSB, a.revealSB} {
		if b != nil {
			b.backdrop = colorPage()
			b.wire()
		}
	}
}

func (a *App) downloadEnabled() bool {
	a.mu.Lock()
	mode := a.mode
	a.mu.Unlock()
	if mode != modeIdle {
		return false
	}
	raw := ""
	if a.urlEdit != nil {
		raw = a.urlEdit.Text()
	}
	cleaned, ok := yturl.ReadyForFetch(raw)
	return ok && a.gate.MatchesURL(cleaned)
}

func (a *App) onLanguageChanged() {
	if a.updatingLang || a.langCombo == nil {
		return
	}
	next := i18n.EN
	if a.langCombo.CurrentIndex() == 1 {
		next = i18n.ES
	}
	if next == a.lang {
		return
	}
	a.lang = next
	a.current.Language = next
	_ = a.store.Save(a.current)
	a.applyTranslations()
}

func (a *App) applyTranslations() {
	if a.mw != nil {
		_ = a.mw.SetTitle(a.t("app.title"))
	}
	if a.urlEdit != nil {
		_ = a.urlEdit.SetCueBanner(a.t("url.cue"))
	}
	if a.headingLabel != nil {
		_ = a.headingLabel.SetText(a.t("app.title"))
	}
	if a.fileLabel != nil {
		_ = a.fileLabel.SetText(a.t("name.label"))
	}
	if a.fileEdit != nil {
		_ = a.fileEdit.SetCueBanner(a.t("name.cue"))
	}
	if a.folderLabel != nil {
		_ = a.folderLabel.SetText(a.t("folder.label"))
	}
	a.pasteSB.invalidate()
	a.browseSB.invalidate()
	a.retrySB.invalidate()
	a.downloadSB.invalidate()
	if a.revealSB != nil {
		a.revealSB.invalidate()
	}
	switch a.previewState {
	case previewHint:
		if true {
			a.setTitleText(a.t("preview.hint"))
		}
	case previewLoading:
		if true {
			a.setTitleText(a.t("preview.loading"))
		}
	case previewError:
		if true {
			a.setTitleText(a.t("preview.error"))
		}
	case previewNotVideo:
		if true {
			a.setTitleText(a.t("preview.notVideo"))
		}
	case previewReady:
		if a.durationLabel != nil && a.lastDuration != "" {
			_ = a.durationLabel.SetText(i18n.Format(a.lang, "preview.duration", a.lastDuration))
		}
	}
	a.relocalizeFooter()
}

func (a *App) refreshFolderCombo() {
	if a.folderCombo == nil {
		return
	}
	view := folderui.DisplayOf(a.current.RecentFolders, a.current.OutputFolder)
	a.updatingCombo = true
	defer func() { a.updatingCombo = false }()
	_ = a.folderCombo.SetModel(view.Model)
	if view.Index >= 0 {
		_ = a.folderCombo.SetCurrentIndex(view.Index)
	}
	_ = a.folderCombo.SetText(view.Text)
}

func (a *App) typedFolder() string {
	if a.folderCombo == nil {
		return a.current.OutputFolder
	}
	return a.folderCombo.Text()
}

func (a *App) persistFolder(path string, alertIfMissing bool) error {
	ferr := settings.FolderAvailable(path)
	a.current.ApplyChosenFolder(path, ferr == nil)
	a.current.Language = a.lang
	if err := a.store.Save(a.current); err != nil {
		return fmt.Errorf("%s", i18n.Format(a.lang, "err.settings", err.Error()))
	}
	a.refreshFolderCombo()
	if ferr != nil {
		msg := i18n.FolderMessage(a.lang, path, ferr)
		a.setStatusText(msg)
		if alertIfMissing && strings.TrimSpace(path) != "" {
			walk.MsgBox(a.mw, a.t("app.title"), msg, walk.MsgBoxIconError)
		}
		return ferr
	}
	return nil
}

func (a *App) onFolderIndexChanged() {
	if a.folderCombo == nil {
		return
	}
	idx := a.folderCombo.CurrentIndex()
	if folderui.ShouldIgnoreIndexEvent(a.updatingCombo, idx) {
		return
	}
	path, ok := folderui.PathFromClick(a.current.RecentFolders, idx, a.folderCombo.Text())
	if !ok || path == "" {
		return
	}
	if folderui.AlreadyShowing(a.current.OutputFolder, path) {
		return
	}
	_ = a.persistFolder(path, true)
}

func (a *App) onFolderEditingFinished() {
	if a.updatingCombo {
		return
	}
	path := a.typedFolder()
	if strings.TrimSpace(path) == "" || path == a.current.OutputFolder {
		return
	}
	_ = a.persistFolder(path, true)
}

func (a *App) onBrowse() {
	var owner uintptr
	if a.mw != nil {
		owner = uintptr(a.mw.Handle())
	}
	path, accepted, err := folderpick.PickFolder(owner, a.t("folder.picker"), a.typedFolder())
	if err != nil {
		walk.MsgBox(a.mw, a.t("app.title"), i18n.Format(a.lang, "err.picker", err.Error()), walk.MsgBoxIconError)
		return
	}
	if !accepted {
		return
	}
	_ = a.persistFolder(path, true)
}

func (a *App) currentURLText() string {
	if a.urlEdit == nil {
		return ""
	}
	return a.urlEdit.Text()
}

func (a *App) onPaste() {
	a.mu.Lock()
	mode := a.mode
	a.mu.Unlock()
	if mode == modeSave {
		return
	}
	text, ok, err := clip.ReadUnicodeText()
	hasText := ok && err == nil
	if !hasText {
		a.setStatusKey("status.clipboardEmpty")
		return
	}
	next := ""
	apply := false
	if res := yturl.Analyze(text); res.Kind == yturl.KindVideo {
		next, apply = res.Canonical, true
	} else {
		next, apply = clip.DecidePaste(a.currentURLText(), text, true)
	}
	if !apply {
		a.setStatusKey("status.clipboardEmpty")
		return
	}
	prev := a.currentURLText()
	if a.urlEdit != nil {
		_ = a.urlEdit.SetText(next)
		_ = a.urlEdit.SetFocus()
	}
	// If SetText did not fire TextChanged (same visual result or no-op),
	// still run the existing debounce path when the field actually changed.
	if a.currentURLText() == prev && next != prev {
		a.onURLChanged()
	}
}

func (a *App) committedURL() string {
	if got, ok := a.gate.Snapshot(); ok {
		return got.SourceURL
	}
	return ""
}

func (a *App) onURLChanged() {
	if a.updatingURL {
		return
	}
	raw := a.currentURLText()
	res := yturl.Analyze(raw)
	if res.Kind == yturl.KindVideo && a.urlEdit != nil && raw != res.Canonical {
		a.updatingURL = true
		_ = a.urlEdit.SetText(res.Canonical)
		a.updatingURL = false
		raw = a.currentURLText()
	}
	cleaned, ok := yturl.ReadyForFetch(raw)
	if a.gate.SyncToURL(cleaned, ok) {
		a.cancelIfFetching()
		a.showHint(true)
	}
	a.refreshActions()
	a.debouncer.Trigger(func() {
		a.ui(a.handleDebouncedURL)
	})
}

func (a *App) handleDebouncedURL() {
	a.mu.Lock()
	mode := a.mode
	a.mu.Unlock()
	if mode == modeSave {
		return
	}
	raw := a.currentURLText()
	op, url := preview.Decide(raw, a.gate.PendingURL(), a.committedURL())
	switch op {
	case preview.OpFetch:
		a.startPreview(url)
	case preview.OpClear:
		a.cancelIfFetching()
		a.gate.Invalidate()
		a.showHint(true)
		a.refreshActions()
	}
	if yturl.Analyze(raw).Kind == yturl.KindNotVideo {
		a.showNotAVideo()
		a.refreshActions()
	}
}

func (a *App) onRetry() {
	raw := a.currentURLText()
	url, ok := yturl.ReadyForFetch(raw)
	if !ok {
		return
	}
	a.startPreview(url)
}

func (a *App) cancelIfFetching() {
	a.mu.Lock()
	if a.mode == modeFetch && a.cancel != nil {
		a.cancel()
	}
	a.mu.Unlock()
}

func (a *App) finishIfCurrent(id uint64) {
	a.mu.Lock()
	if a.workID == id {
		a.mode = modeIdle
		a.cancel = nil
	}
	a.mu.Unlock()
	a.ui(a.refreshActions)
}

func (a *App) startPreview(videoURL string) {
	a.mu.Lock()
	if a.mode == modeSave {
		a.mu.Unlock()
		return
	}
	if a.cancel != nil {
		a.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.workID++
	id := a.workID
	a.mode = modeFetch
	a.cancel = cancel
	a.mu.Unlock()

	seq := a.gate.Begin(videoURL)
	a.ui(func() {
		a.showLoading()
		a.refreshActions()
	})
	go func() {
		defer a.finishIfCurrent(id)
		a.runFetch(ctx, seq, videoURL)
	}()
}

func (a *App) onCancel() {
	a.mu.Lock()
	cancel := a.cancel
	mode := a.mode
	a.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	if mode == modeFetch {
		a.restoreIdleOrSuccessFooter()
		return
	}
	a.dispatch(uistate.Event{Kind: uistate.EvCancelled, Line1: a.t("status.cancelling")})
}

func (a *App) onDownload() {
	raw := a.currentURLText()
	videoURL, ok := yturl.ReadyForFetch(raw)
	if !ok {
		return
	}
	fetched, ok := a.gate.Snapshot()
	if !ok || fetched.SourceURL != videoURL {
		return
	}
	folder := a.typedFolder()
	if err := a.persistFolder(folder, true); err != nil {
		return
	}
	userName := ""
	if a.fileEdit != nil {
		userName = a.fileEdit.Text()
	}

	a.mu.Lock()
	if a.mode == modeSave {
		a.mu.Unlock()
		return
	}
	if a.cancel != nil {
		a.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.workID++
	id := a.workID
	a.mode = modeSave
	a.cancel = cancel
	a.mu.Unlock()

	a.lastSavedPath = ""
	a.refreshActions()
	go func() {
		defer a.finishIfCurrent(id)
		a.runSave(ctx, fetched, userName, folder)
	}()
}

func (a *App) runFetch(ctx context.Context, seq uint64, videoURL string) {
	tp, err := a.ensureTools()
	if err != nil {
		a.finishPreviewErr(ctx, seq)
		return
	}
	if ctx.Err() != nil {
		return
	}
	a.dispatch(uistate.Event{Kind: uistate.EvResolving, Line1: a.t("status.loading")})
	dt := download.Tools{Ytdlp: tp.Ytdlp, FFmpegDir: tp.FFmpegDir, Deno: tp.Deno}
	info, err := download.FetchMeta(ctx, dt, videoURL, nil)
	if err != nil {
		a.finishPreviewErr(ctx, seq)
		return
	}
	id, _ := yturl.VideoID(videoURL)
	thumbURL := ""
	var preview image.Image
	img, used, loadErr := metadata.LoadPreview(ctx, metadata.NewThumbnailClient(), metadata.PreviewCandidates(info, id))
	if loadErr != nil && ctx.Err() != nil {
		return
	}
	if loadErr == nil {
		preview = img
		thumbURL = used
	}
	dur := ""
	if info.HasDur {
		dur = metadata.FormatDuration(info.Duration)
	}
	fetched := metadata.Fetched{
		SourceURL:   videoURL,
		Title:       info.Title,
		Duration:    dur,
		DefaultName: metadata.DefaultFilename(info.Title),
		ThumbURL:    thumbURL,
	}
	if !a.gate.Commit(seq, fetched) {
		return
	}
	a.ui(func() {
		a.applyFetched(fetched, preview)
		a.restoreIdleOrSuccessFooter()
		a.refreshActions()
	})
}

func (a *App) finishPreviewErr(ctx context.Context, seq uint64) {
	if ctx.Err() != nil {
		return
	}
	if a.gate.Seq() != seq {
		return
	}
	a.ui(func() {
		a.showError()
		a.refreshActions()
	})
}

func (a *App) runSave(ctx context.Context, fetched metadata.Fetched, userName, folder string) {
	diaglog.Start(appVersion)
	if id, ok := yturl.VideoID(fetched.SourceURL); ok {
		res := infocache.SavedResult(id)
		if res == "" {
			res = "absent"
		}
		diaglog.Event("info_cache", "phase=save result=%s", res)
	}
	ok := false
	reason := "error"
	defer func() { diaglog.End(ok, reason) }()
	tp, err := a.ensureTools()
	if err != nil {
		if ctx.Err() != nil {
			a.dispatch(uistate.Event{Kind: uistate.EvCancelled, Line1: a.t("status.cancelled")})
			return
		}
		a.fail(i18n.Format(a.lang, "err.tools", err.Error()))
		return
	}
	if err := settings.FolderAvailable(folder); err != nil {
		a.fail(i18n.FolderMessage(a.lang, folder, err))
		return
	}
	tempRoot, err := apppaths.TempRoot()
	if err != nil {
		a.fail(err.Error())
		return
	}
	jobDir, err := download.NewJobDir(tempRoot)
	if err != nil {
		a.fail(i18n.Format(a.lang, "err.temp", err.Error()))
		return
	}
	defer func() {
		_ = download.SafeRemoveJobDir(jobDir, tempRoot)
	}()

	base := sanitize.OutputBase(userName, fetched.Title)
	tmpl := download.OutputTemplate(jobDir, "audio")
	dt := download.Tools{Ytdlp: tp.Ytdlp, FFmpegDir: tp.FFmpegDir, Deno: tp.Deno}
	diaglog.Flags(true, true, true, true)
	diaglog.Tool("yt-dlp", download.YtdlpVersion(ctx, dt.Ytdlp))
	infoJSON := a.startSaveFooter(fetched.SourceURL)
	var prog download.View
	err = download.RunDownloadMaybeInfo(ctx, dt, fetched.SourceURL, infoJSON, tmpl, func(line string) {
		nextView, nextFooter, ch := download.ApplyFooter(prog, a.currentFooter(), line, a.lang)
		if !ch.Changed {
			return
		}
		prog = nextView
		a.replaceFooter(nextFooter)
		diaglog.Progress(line, ch.Kind, ch.Determinate, ch.Percent, ch.Speed, ch.ETA, ch.Downloaded, ch.Total, ch.HasPercent, ch.HasTotal, ch.StatusMode)
		a.publishFooter(nextFooter)
	})
	if err != nil {
		if ctx.Err() != nil {
			reason = "cancelled"
			a.dispatch(uistate.Event{Kind: uistate.EvCancelled, Line1: a.t("status.cancelled")})
			return
		}
		reason = "download"
		a.fail(i18n.Format(a.lang, "err.download", err.Error()))
		return
	}
	if err := settings.FolderAvailable(folder); err != nil {
		a.fail(i18n.FolderMessage(a.lang, folder, err))
		return
	}
	saved := a.currentFooter()
	a.dispatch(uistate.Event{
		Kind:  uistate.EvWriting,
		Line1: a.t("status.writing"),
		Line2: i18n.RetainedDownload(a.lang, saved.SavedPct, saved.SavedSpeed),
	})
	finalPath, err := download.PlaceNamedOutput(jobDir, folder, base)
	if err != nil {
		a.fail(err.Error())
		return
	}
	_ = a.persistFolder(folder, false)
	a.lastSavedPath = finalPath
	ok = true
	reason = "ok"
	a.dispatch(uistate.Event{
		Kind:  uistate.EvDone,
		Line1: a.t("status.savedHeader"),
		Line2: finalPath,
		Tip2:  finalPath,
	})
}

func (a *App) ensureTools() (tools.Paths, error) {
	toolsDir, err := apppaths.ToolsDir()
	if err != nil {
		return tools.Paths{}, err
	}
	a.mu.Lock()
	busy := a.mode != modeIdle
	a.mu.Unlock()
	if busy {
		a.dispatch(uistate.Event{Kind: uistate.EvResolving, Line1: a.t("status.tools")})
	}
	paths, err := tools.Ensure(toolsDir, func(status string, percent int) {
		if !busy {
			return
		}
		line := i18n.LocalizeToolsStatus(a.lang, status)
		_ = percent
		a.dispatch(uistate.ToolStatus(line))
	})
	return paths, err
}

func (a *App) applyFetched(fetched metadata.Fetched, img image.Image) {
	a.previewState = previewReady
	a.lastDuration = fetched.Duration
	if true {
		a.setTitleText(fetched.Title)
	}
	if a.durationLabel != nil {
		if fetched.Duration != "" {
			_ = a.durationLabel.SetText(i18n.Format(a.lang, "preview.duration", fetched.Duration))
		} else {
			_ = a.durationLabel.SetText("")
		}
	}
	if a.fileEdit != nil {
		_ = a.fileEdit.SetText(fetched.DefaultName)
	}
	a.setRetryVisible(false)
	a.setThumbnailImage(img)
}

func (a *App) showHint(clearName bool) {
	a.previewState = previewHint
	a.lastDuration = ""
	if true {
		a.setTitleText(a.t("preview.hint"))
	}
	if a.durationLabel != nil {
		_ = a.durationLabel.SetText("")
	}
	if clearName && a.fileEdit != nil {
		_ = a.fileEdit.SetText("")
	}
	a.setRetryVisible(false)
	a.setThumbnailImage(nil)
	a.restoreIdleOrSuccessFooter()
}

func (a *App) showLoading() {
	a.previewState = previewLoading
	if true {
		a.setTitleText(a.t("preview.loading"))
	}
	if a.durationLabel != nil {
		_ = a.durationLabel.SetText("")
	}
	if a.fileEdit != nil {
		_ = a.fileEdit.SetText("")
	}
	a.setRetryVisible(false)
	a.setThumbnailImage(nil)
	a.beginPreviewFooter()
}

func (a *App) showNotAVideo() {
	a.previewState = previewNotVideo
	a.lastDuration = ""
	if true {
		a.setTitleText(a.t("preview.notVideo"))
	}
	if a.durationLabel != nil {
		_ = a.durationLabel.SetText("")
	}
	if a.fileEdit != nil {
		_ = a.fileEdit.SetText("")
	}
	a.setRetryVisible(false)
	a.setThumbnailImage(nil)
	a.restoreIdleOrSuccessFooter()
}

func (a *App) showError() {
	a.previewState = previewError
	if true {
		a.setTitleText(a.t("preview.error"))
	}
	if a.durationLabel != nil {
		_ = a.durationLabel.SetText("")
	}
	a.setRetryVisible(true)
	a.setThumbnailImage(nil)
	a.restoreIdleOrSuccessFooter()
}

func (a *App) setThumbnailImage(img image.Image) {
	if a.thumbImg != nil {
		a.thumbImg.Dispose()
		a.thumbImg = nil
	}
	a.thumbPxW, a.thumbPxH = 0, 0
	if img != nil {
		b := img.Bounds()
		if bmp, err := walk.NewBitmapFromImage(img); err == nil {
			a.thumbImg = bmp
			a.thumbPxW = b.Dx()
			a.thumbPxH = b.Dy()
		}
	}
	if a.thumbWidget != nil {
		a.thumbWidget.Invalidate()
	}
}

func (a *App) fail(msg string) {
	a.dispatch(uistate.Event{Kind: uistate.EvError, Line1: msg})
	a.ui(func() {
		walk.MsgBox(a.mw, a.t("app.title"), msg, walk.MsgBoxIconError)
	})
}

func (a *App) refreshActions() {
	a.mu.Lock()
	mode := a.mode
	a.mu.Unlock()
	editable := mode != modeSave
	if a.browseSB != nil && a.browseSB.widget != nil {
		a.browseSB.widget.SetEnabled(editable)
		a.browseSB.invalidate()
	}
	if a.pasteSB != nil && a.pasteSB.widget != nil {
		a.pasteSB.widget.SetEnabled(editable)
		a.pasteSB.invalidate()
	}
	if a.folderCombo != nil {
		a.folderCombo.SetEnabled(editable)
	}
	if a.fileEdit != nil {
		a.fileEdit.SetEnabled(editable)
	}
	if a.urlEdit != nil {
		a.urlEdit.SetEnabled(editable)
	}
	if a.langCombo != nil {
		a.langCombo.SetEnabled(editable)
	}
	a.downloadSB.invalidate()
}

func (a *App) currentFooter() uistate.Footer {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.footer
}

func (a *App) replaceFooter(f uistate.Footer) {
	a.mu.Lock()
	a.footer = f
	a.mu.Unlock()
}

func (a *App) dispatch(ev uistate.Event) {
	if ev.Kind == "" {
		return
	}
	a.mu.Lock()
	next := uistate.Reduce(a.footer, ev)
	a.footer = next
	a.mu.Unlock()
	a.publishFooter(next)
}

func (a *App) publishFooter(f uistate.Footer) {
	a.ui(func() { a.applyFooterModel(f) })
}

func (a *App) dispatchIdle() {
	a.dispatch(uistate.Event{Kind: uistate.EvIdle, Line1: a.t("status.nothing")})
}

func (a *App) noteStatus(text string) {
	a.mu.Lock()
	f := a.footer
	saved := a.lastSavedPath
	switch f.Stage {
	case uistate.EvResolving, uistate.EvDownloading, uistate.EvConverting, uistate.EvWriting:
		a.mu.Unlock()
		return
	}
	if strings.TrimSpace(text) == "" {
		a.mu.Unlock()
		if saved == "" {
			a.dispatchIdle()
		}
		return
	}
	if f.Stage == uistate.EvDone && saved != "" {
		f.Line1 = text
		a.footer = f
		a.mu.Unlock()
		a.publishFooter(f)
		return
	}
	a.mu.Unlock()
	a.dispatch(uistate.Event{Kind: uistate.EvIdle, Line1: text})
}

func (a *App) setStatusKey(key string, args ...any) {
	a.noteStatus(i18n.Format(a.lang, key, args...))
}

func (a *App) setStatusText(text string) {
	a.noteStatus(text)
}

func (a *App) relocalizeFooter() {
	a.mu.Lock()
	f := a.footer
	lang := a.lang
	a.mu.Unlock()
	switch f.Stage {
	case uistate.EvIdle, "":
		a.dispatch(uistate.Event{Kind: uistate.EvIdle, Line1: i18n.T(lang, "status.nothing")})
	case uistate.EvResolving:
		a.dispatch(uistate.Event{Kind: uistate.EvResolving, Line1: i18n.T(lang, "status.resolving")})
	case uistate.EvDownloading:
		a.dispatch(uistate.Event{
			Kind:        uistate.EvDownloading,
			Determinate: f.Determinate,
			Percent:     f.Percent,
			Speed:       f.SavedSpeed,
			Line1:       i18n.DownloadStatus(lang, f.Percent, f.SavedSpeed, ""),
		})
	case uistate.EvConverting:
		a.dispatch(uistate.Event{
			Kind:  uistate.EvConverting,
			Line1: i18n.T(lang, "status.converting"),
			Line2: i18n.RetainedDownload(lang, f.SavedPct, f.SavedSpeed),
		})
	case uistate.EvWriting:
		a.dispatch(uistate.Event{
			Kind:  uistate.EvWriting,
			Line1: i18n.T(lang, "status.writing"),
			Line2: i18n.RetainedDownload(lang, f.SavedPct, f.SavedSpeed),
		})
	case uistate.EvDone:
		a.dispatch(uistate.Event{
			Kind:  uistate.EvDone,
			Line1: i18n.T(lang, "status.savedHeader"),
			Line2: f.Line2,
			Tip2:  f.Tip2,
		})
	case uistate.EvCancelled:
		a.dispatch(uistate.Event{Kind: uistate.EvCancelled, Line1: i18n.T(lang, "status.cancelled")})
	default:
		a.publishFooter(f)
	}
}

func (a *App) applyFooterModel(f uistate.Footer) {
	a.progressActive = f.BarLive
	a.progressDeterminate = f.Determinate
	a.progressValue = f.Percent
	if f.BarLive && !f.Determinate {
		a.ensureProgressPulse()
	} else {
		a.stopProgressPulse()
	}
	a.statusClip.setText(f.Line1, f.Tip1)
	a.pathClip.setText(f.Line2, f.Tip2)
	a.applyFooterAction(f.Action)
	if a.progress != nil {
		a.progress.Invalidate()
	}
	readDet := a.progressDeterminate
	readPct := a.progressValue
	diaglog.UIApply(f.Stage, f.Determinate, f.Percent, readDet, readPct, f.Line1)
}

func (a *App) applyFooterAction(act uistate.Action) {
	if a.revealSB == nil {
		return
	}
	switch act {
	case uistate.ActionCancel:
		a.revealSB.key = "cancel"
		a.revealSB.icon = iconCancel
		a.revealSB.click = a.onCancel
		a.revealSB.canClick = nil
		if a.revealSB.widget != nil {
			a.revealSB.widget.SetEnabled(true)
		}
	case uistate.ActionRevealOn:
		a.revealSB.key = "reveal"
		a.revealSB.icon = iconFolder
		a.revealSB.click = a.onReveal
		a.revealSB.canClick = nil
		if a.revealSB.widget != nil {
			a.revealSB.widget.SetEnabled(true)
		}
	default:
		a.revealSB.key = "reveal"
		a.revealSB.icon = iconFolder
		a.revealSB.click = a.onReveal
		a.revealSB.canClick = func() bool { return false }
		if a.revealSB.widget != nil {
			a.revealSB.widget.SetEnabled(false)
		}
	}
	a.revealSB.invalidate()
}

func (a *App) setTitleText(full string) {
	a.titleClip.setText(full, full)
}

func (a *App) beginPreviewFooter() {
	a.mu.Lock()
	if a.mode == modeSave {
		a.mu.Unlock()
		return
	}
	a.mu.Unlock()
	a.dispatch(uistate.Event{Kind: uistate.EvResolving, Line1: a.t("status.loading")})
}

func (a *App) restoreIdleOrSuccessFooter() {
	a.mu.Lock()
	if a.mode == modeSave {
		a.mu.Unlock()
		return
	}
	saved := a.lastSavedPath
	a.mu.Unlock()
	if saved != "" {
		a.dispatch(uistate.Event{
			Kind:  uistate.EvDone,
			Line1: a.t("status.savedHeader"),
			Line2: saved,
			Tip2:  saved,
		})
		return
	}
	a.dispatchIdle()
}

func (a *App) startSaveFooter(videoURL string) string {
	if id, ok := yturl.VideoID(videoURL); ok {
		path, age, reason, hit := infocache.Lookup(id)
		if hit {
			diaglog.Event("info_cache", "phase=lookup hit=true age_ms=%d reason=%s", age.Milliseconds(), reason)
			a.dispatch(uistate.Event{
				Kind:        uistate.EvDownloading,
				Determinate: true,
				Percent:     0,
				Line1:       i18n.DownloadStatus(a.lang, 0, "", ""),
			})
			return path
		}
		diaglog.Event("info_cache", "phase=lookup hit=false reason=%s", reason)
		if reason == "expired_url" || reason == "expired_ttl" || reason == "expired" {
			a.dispatch(uistate.Event{Kind: uistate.EvResolving, Line1: a.t("status.metadataExpired")})
			return ""
		}
	}
	a.dispatch(uistate.Event{Kind: uistate.EvResolving, Line1: a.t("status.resolving")})
	return ""
}

func (a *App) setRetryVisible(visible bool) {
	if a.retrySB != nil && a.retrySB.widget != nil {
		a.retrySB.widget.SetVisible(visible)
	}
}

func (a *App) maybeLogPaint() {
	now := time.Now()
	if !a.lastPaintLog.IsZero() && now.Sub(a.lastPaintLog) < 200*time.Millisecond &&
		a.lastPaintDet == a.progressDeterminate &&
		a.lastPaintPct == a.progressValue &&
		a.lastPaintActive == a.progressActive {
		return
	}
	a.lastPaintLog = now
	a.lastPaintDet = a.progressDeterminate
	a.lastPaintPct = a.progressValue
	a.lastPaintActive = a.progressActive
	diaglog.Paint(a.progressDeterminate, a.progressValue, a.progressActive)
}

func (a *App) ensureProgressPulse() {
	a.mu.Lock()
	if a.pulseStop != nil {
		a.mu.Unlock()
		return
	}
	stop := make(chan struct{})
	a.pulseStop = stop
	a.mu.Unlock()
	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				a.ui(func() {
					if !a.progressActive || a.progressDeterminate {
						return
					}
					a.progressPulse = (a.progressPulse + 4) % 100
					if a.progress != nil {
						a.progress.Invalidate()
					}
				})
			}
		}
	}()
}

func (a *App) stopProgressPulse() {
	a.mu.Lock()
	if a.pulseStop != nil {
		close(a.pulseStop)
		a.pulseStop = nil
	}
	a.mu.Unlock()
}

func (a *App) onReveal() {
	path := a.lastSavedPath
	if err := reveal.CheckFile(path); err != nil {
		a.setStatusText(i18n.RevealMessage(a.lang, path, err))
		return
	}
	if err := reveal.OpenAndSelect(path); err != nil {
		a.setStatusText(i18n.RevealMessage(a.lang, path, err))
	}
}

func (a *App) ui(fn func()) {
	if a.mw == nil {
		fn()
		return
	}
	a.mw.Synchronize(fn)
}
