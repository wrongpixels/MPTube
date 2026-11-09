package i18n

import (
	"errors"
	"fmt"
	"strings"

	"youtubemp3/internal/settings"
)

const (
	EN = "en"
	ES = "es"
)

var catalog = map[string]map[string]string{
	EN: {
		"app.title":                        "YouTube MP3 Saver",
		"url.cue":                          "Paste a YouTube link",
		"paste":                            "Paste",
		"name.label":                       "File name",
		"name.cue":                         "Video title",
		"folder.label":                     "Folder",
		"folder.browse":                    "Browse",
		"folder.picker":                    "Choose folder to save MP3 files",
		"download":                         "Download MP3",
		"cancel":                           "Cancel",
		"retry":                            "Retry",
		"lang.en":                          "EN",
		"lang.es":                          "ES",
		"preview.hint":                     "Paste a YouTube link",
		"preview.loading":                  "Loading…",
		"preview.error":                    "Couldn't load this video.",
		"preview.notVideo":                 "This link is not a single YouTube video.",
		"preview.duration":                 "Duration: %s",
		"status.hint":                      "",
		"status.clipboardEmpty":            "The clipboard doesn't have a text link. Copy a YouTube link and try again.",
		"status.notVideo":                  "This is not a single video. Playlists and other YouTube pages cannot be downloaded.",
		"status.loading":                   "Loading video…",
		"status.tools":                     "Checking helper tools…",
		"status.downloadingTool":           "Downloading %s…",
		"status.toolsReady":                "Helper tools are ready.",
		"status.cancelled":                 "Cancelled.",
		"status.cancelling":                "Cancelling…",
		"status.saving":                    "Saving: %s",
		"status.nothing":                   "Nothing downloaded yet",
		"status.resolving":                 "Fetching video data…",
		"status.metadataExpired":           "Video data expired. Fetching again…",
		"status.downloadRetained":          "Download: %d %% · %s",
		"status.downloadRetainedPct":       "Download: %d %%",
		"status.savedHeader":               "Downloaded!",
		"status.downloadingAudio":          "Downloading audio… %d%%",
		"status.downloadingAudioSpeed":     "Downloading audio… %d%% • %s",
		"status.downloadingAudioFull":      "Downloading audio… %d%% • %s • ETA %s",
		"status.downloadingAudioBytes":     "Downloading audio… %s • %s",
		"status.downloadingAudioBytesOnly": "Downloading audio… %s",
		"status.converting":                "Converting to MP3…",
		"status.writing":                   "Saving file…",
		"status.saved":                     "Downloaded!\n%s",
		"reveal":                           "Show in folder",
		"err.revealMissing":                "The downloaded file is no longer available:\n%s\n\nIt may have been moved, renamed, or the drive disconnected.\nThe saved folder setting was not changed.",
		"err.revealFailed":                 "Could not open the folder:\n%s",
		"err.tools":                        "Could not prepare helper tools:\n%s",
		"err.temp":                         "Could not create a temporary folder:\n%s",
		"err.settings":                     "Could not save settings: %s",
		"err.download":                     "%s",
		"err.folderEmpty":                  "No output folder is set. Choose a folder to save MP3 files.",
		"err.folderMissing":                "The selected folder is not available:\n%s\n\nIt may be on a disconnected drive or unreachable network location.\nThe saved path was not changed.",
		"err.folderNotDir":                 "The selected path is not a folder:\n%s\n\nThe saved path was not changed.",
		"err.picker":                       "Could not open the folder picker:\n%s",
	},
	ES: {
		"app.title":                        "YouTube MP3 Saver",
		"url.cue":                          "Pega un enlace de YouTube",
		"paste":                            "Pegar",
		"name.label":                       "Nombre",
		"name.cue":                         "Título del vídeo",
		"folder.label":                     "Carpeta",
		"folder.browse":                    "Examinar",
		"folder.picker":                    "Elige la carpeta para guardar los MP3",
		"download":                         "Descargar MP3",
		"cancel":                           "Cancelar",
		"retry":                            "Reintentar",
		"lang.en":                          "EN",
		"lang.es":                          "ES",
		"preview.hint":                     "Pega un enlace de YouTube",
		"preview.loading":                  "Cargando…",
		"preview.error":                    "No se ha podido cargar este vídeo.",
		"preview.notVideo":                 "Este enlace no es un vídeo de YouTube.",
		"preview.duration":                 "Duración: %s",
		"status.hint":                      "",
		"status.clipboardEmpty":            "El portapapeles no tiene un enlace de texto. Copia un enlace de YouTube e inténtalo de nuevo.",
		"status.notVideo":                  "Esto no es un vídeo concreto. No se pueden descargar listas de reproducción ni otras páginas de YouTube.",
		"status.loading":                   "Cargando vídeo…",
		"status.tools":                     "Comprobando las herramientas…",
		"status.downloadingTool":           "Descargando %s…",
		"status.toolsReady":                "Las herramientas están listas.",
		"status.cancelled":                 "Cancelado.",
		"status.cancelling":                "Cancelando…",
		"status.saving":                    "Guardando: %s",
		"status.nothing":                   "Nada descargado aún",
		"status.resolving":                 "Obteniendo datos del vídeo…",
		"status.metadataExpired":           "Los datos del vídeo han caducado. Obteniendo de nuevo…",
		"status.downloadRetained":          "Descarga: %d %% · %s",
		"status.downloadRetainedPct":       "Descarga: %d %%",
		"status.savedHeader":               "¡Descargado!",
		"status.downloadingAudio":          "Descargando audio… %d%%",
		"status.downloadingAudioSpeed":     "Descargando audio… %d%% • %s",
		"status.downloadingAudioFull":      "Descargando audio… %d%% • %s • ETA %s",
		"status.downloadingAudioBytes":     "Descargando audio… %s • %s",
		"status.downloadingAudioBytesOnly": "Descargando audio… %s",
		"status.converting":                "Convirtiendo a MP3…",
		"status.writing":                   "Guardando el archivo…",
		"status.saved":                     "¡Descargado!\n%s",
		"reveal":                           "Mostrar en carpeta",
		"err.revealMissing":                "El archivo descargado ya no está disponible:\n%s\n\nPuede haberse movido, haberse renombrado o estar en una unidad desconectada.\nLa carpeta guardada en los ajustes no se ha modificado.",
		"err.revealFailed":                 "No se ha podido abrir la carpeta:\n%s",
		"err.tools":                        "No se han podido preparar las herramientas:\n%s",
		"err.temp":                         "No se ha podido crear una carpeta temporal:\n%s",
		"err.settings":                     "No se han podido guardar los ajustes: %s",
		"err.download":                     "%s",
		"err.folderEmpty":                  "No hay una carpeta de destino. Elige dónde guardar los MP3.",
		"err.folderMissing":                "La carpeta seleccionada no está disponible:\n%s\n\nPuede estar en una unidad desconectada o en una ubicación de red inaccesible.\nLa ruta guardada no se ha modificado.",
		"err.folderNotDir":                 "La ruta seleccionada no es una carpeta:\n%s\n\nLa ruta guardada no se ha modificado.",
		"err.picker":                       "No se ha podido abrir el selector de carpetas:\n%s",
	},
}

func Normalize(code string) string {
	c := strings.ToLower(strings.TrimSpace(code))
	c = strings.ReplaceAll(c, "_", "-")
	if c == ES || strings.HasPrefix(c, "es-") {
		return ES
	}
	if c == EN || strings.HasPrefix(c, "en-") {
		return EN
	}
	return ""
}

func FromOSLocale(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	tag = strings.ReplaceAll(tag, "_", "-")
	if tag == "" {
		return EN
	}
	// e.g. es, es-ES, es-419.utf8
	if dot := strings.IndexByte(tag, '.'); dot >= 0 {
		tag = tag[:dot]
	}
	if at := strings.IndexByte(tag, '@'); at >= 0 {
		tag = tag[:at]
	}
	if Normalize(tag) == ES || strings.HasPrefix(tag, "es") {
		return ES
	}
	return EN
}

func Resolve(saved, osLocale string) string {
	if n := Normalize(saved); n != "" {
		return n
	}
	return FromOSLocale(osLocale)
}

func T(lang, key string) string {
	lang = Resolve(lang, "")
	if m, ok := catalog[lang]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	if s, ok := catalog[EN][key]; ok {
		return s
	}
	return key
}

func Format(lang, key string, args ...any) string {
	return fmt.Sprintf(T(lang, key), args...)
}

func FolderMessage(lang, path string, err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, settings.ErrEmptyFolder) {
		return T(lang, "err.folderEmpty")
	}
	if errors.Is(err, settings.ErrNotAFolder) {
		return Format(lang, "err.folderNotDir", path)
	}
	return Format(lang, "err.folderMissing", path)
}

func DownloadStatus(lang string, percent int, speed, eta string) string {
	switch {
	case strings.TrimSpace(speed) != "" && strings.TrimSpace(eta) != "":
		return Format(lang, "status.downloadingAudioFull", percent, speed, eta)
	case strings.TrimSpace(speed) != "":
		return Format(lang, "status.downloadingAudioSpeed", percent, speed)
	default:
		return Format(lang, "status.downloadingAudio", percent)
	}
}

func DownloadBytesStatus(lang, size, speed string) string {
	if strings.TrimSpace(speed) != "" {
		return Format(lang, "status.downloadingAudioBytes", size, speed)
	}
	return Format(lang, "status.downloadingAudioBytesOnly", size)
}

func RetainedDownload(lang string, percent int, speed string) string {
	if percent <= 0 {
		return ""
	}
	if strings.TrimSpace(speed) != "" {
		return Format(lang, "status.downloadRetained", percent, speed)
	}
	return Format(lang, "status.downloadRetainedPct", percent)
}

func RevealMessage(lang, path string, err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if msg == "downloaded file is not available" || strings.Contains(msg, "not available") {
		return Format(lang, "err.revealMissing", path)
	}
	return Format(lang, "err.revealFailed", msg)
}

func Keys() []string {
	keys := make([]string, 0, len(catalog[EN]))
	for k := range catalog[EN] {
		keys = append(keys, k)
	}
	return keys
}

func LocalizeToolsStatus(lang, status string) string {
	const prefix = "Downloading "
	const suffix = " from the official source…"
	if strings.HasPrefix(status, prefix) && strings.HasSuffix(status, suffix) {
		name := strings.TrimSuffix(strings.TrimPrefix(status, prefix), suffix)
		return Format(lang, "status.downloadingTool", name)
	}
	if status == "Helper tools are ready." {
		return T(lang, "status.toolsReady")
	}
	return status
}
