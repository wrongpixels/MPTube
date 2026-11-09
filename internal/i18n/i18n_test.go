package i18n

import (
	"errors"
	"strings"
	"testing"

	"youtubemp3/internal/settings"
)

func TestCatalogCoverage(t *testing.T) {
	if len(catalog[EN]) == 0 || len(catalog[ES]) == 0 {
		t.Fatal("empty catalog")
	}
	for k, ev := range catalog[EN] {
		sv, ok := catalog[ES][k]
		if !ok {
			t.Fatalf("missing Spanish key %q", k)
		}
		if k != "status.hint" && ev == "" {
			t.Fatalf("empty English %q", k)
		}
		if k != "status.hint" && sv == "" {
			t.Fatalf("empty Spanish %q", k)
		}
	}
	for k := range catalog[ES] {
		if _, ok := catalog[EN][k]; !ok {
			t.Fatalf("Spanish extra key %q", k)
		}
	}
	if T(ES, "download") != "Descargar MP3" {
		t.Fatalf("download es=%q", T(ES, "download"))
	}
	if T(EN, "download") != "Download MP3" {
		t.Fatalf("download en=%q", T(EN, "download"))
	}
	if T(ES, "cancel") != "Cancelar" || T(EN, "cancel") != "Cancel" {
		t.Fatal("cancel")
	}
	if T(ES, "paste") != "Pegar" || T(EN, "paste") != "Paste" {
		t.Fatal("paste")
	}
	if T(ES, "status.clipboardEmpty") == T(EN, "status.clipboardEmpty") {
		t.Fatal("clipboard empty should differ")
	}
	if T(ES, "preview.notVideo") == T(EN, "preview.notVideo") || T(ES, "status.notVideo") == T(EN, "status.notVideo") {
		t.Fatal("not-video should differ")
	}
	if !strings.HasPrefix(T(EN, "status.saved"), "Downloaded!") || !strings.HasPrefix(T(ES, "status.saved"), "¡Descargado!") {
		t.Fatalf("saved en=%q es=%q", T(EN, "status.saved"), T(ES, "status.saved"))
	}
}

func TestResolveLanguage(t *testing.T) {
	if Resolve("es", "en-US") != ES {
		t.Fatal("saved es wins")
	}
	if Resolve("en", "es-ES") != EN {
		t.Fatal("saved en wins")
	}
	if Resolve("", "es-ES") != ES {
		t.Fatal("os spanish")
	}
	if Resolve("", "es_MX.UTF-8") != ES {
		t.Fatal("os spanish underscore")
	}
	if Resolve("", "en-GB") != EN {
		t.Fatal("os english")
	}
	if Resolve("", "") != EN {
		t.Fatal("empty defaults to en")
	}
	if Normalize("ES-es") != ES || Normalize("en-US") != EN {
		t.Fatal("normalize")
	}
}

func TestFolderMessage(t *testing.T) {
	en := FolderMessage(EN, `E:\Music`, settings.ErrFolderUnavailable)
	es := FolderMessage(ES, `E:\Music`, settings.ErrFolderUnavailable)
	if !strings.Contains(en, `E:\Music`) || !strings.Contains(es, `E:\Music`) {
		t.Fatal("path missing")
	}
	if !strings.Contains(en, "not changed") || !strings.Contains(es, "no se ha modificado") {
		t.Fatalf("en=%q es=%q", en, es)
	}
	if FolderMessage(ES, "", settings.ErrEmptyFolder) == FolderMessage(EN, "", settings.ErrEmptyFolder) {
		t.Fatal("empty folder should differ by language")
	}
	if FolderMessage(EN, "x", errors.New("other")) == "" {
		t.Fatal("fallback")
	}
}

func TestLocalizeToolsStatus(t *testing.T) {
	got := LocalizeToolsStatus(ES, "Downloading yt-dlp 2026.10.26 from the official source…")
	if got != "Descargando yt-dlp 2026.10.26…" {
		t.Fatalf("got %q", got)
	}
	if LocalizeToolsStatus(ES, "Helper tools are ready.") != "Las herramientas están listas." {
		t.Fatal("ready")
	}
}

func TestDownloadStatusAndReveal(t *testing.T) {
	if got := DownloadStatus(ES, 40, "", ""); !strings.Contains(got, "40") || !strings.Contains(got, "Descargando audio") {
		t.Fatalf("basic: %q", got)
	}
	if got := DownloadStatus(EN, 40, "1.2 MiB/s", "12s"); !strings.Contains(got, "1.2 MiB/s") || !strings.Contains(got, "ETA 12s") {
		t.Fatalf("full: %q", got)
	}
	if T(ES, "reveal") != "Mostrar en carpeta" || T(EN, "reveal") != "Show in folder" {
		t.Fatal("reveal label")
	}
	if T(ES, "status.loading") != "Cargando vídeo…" || T(EN, "status.loading") != "Loading video…" {
		t.Fatalf("loading %q %q", T(EN, "status.loading"), T(ES, "status.loading"))
	}
	if T(ES, "status.converting") != "Convirtiendo a MP3…" {
		t.Fatal("converting")
	}
	miss := RevealMessage(ES, `E:\Music\a.mp3`, errors.New("downloaded file is not available"))
	if !strings.Contains(miss, `E:\Music\a.mp3`) || strings.Contains(miss, "Could not open") {
		t.Fatalf("missing: %q", miss)
	}
}

func TestDownloadBytesStatus(t *testing.T) {
	got := DownloadBytesStatus(ES, "1.2 MiB", "400 KiB/s")
	if !strings.Contains(got, "1.2 MiB") || !strings.Contains(got, "400 KiB/s") || !strings.Contains(got, "Descargando audio") {
		t.Fatalf("got %q", got)
	}
	got = DownloadBytesStatus(EN, "800 KiB", "")
	if !strings.Contains(got, "800 KiB") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "%") {
		t.Fatalf("must not invent percent: %q", got)
	}
}

func TestNothingResolvingAndRetained(t *testing.T) {
	if T(ES, "status.nothing") != "Nada descargado aún" || T(EN, "status.nothing") != "Nothing downloaded yet" {
		t.Fatal("nothing")
	}
	if T(ES, "status.resolving") != "Obteniendo datos del vídeo…" {
		t.Fatal("resolving")
	}
	got := RetainedDownload(ES, 100, "15.7 MiB/s")
	if !strings.Contains(got, "100") || !strings.Contains(got, "15.7") || !strings.Contains(got, "Descarga") {
		t.Fatalf("retained %q", got)
	}
	if RetainedDownload(EN, 0, "1 MiB/s") != "" {
		t.Fatal("no retain without percent")
	}
}
