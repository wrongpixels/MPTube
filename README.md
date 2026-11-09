# MPTube

A small Windows app that saves the audio of a public YouTube video as an MP3 file.

## What it does

- Paste a YouTube link (`youtube.com/watch`, `youtu.be`, Shorts, Music). The title, duration, and thumbnail load on their own. **Paste** reads the clipboard only when you click it. Playlist and tracking parameters are dropped. A playlist with no video is not downloaded.
- Change the file name if you want. The default is a cleaned version of the title. If that name already exists, the app writes `Title (1).mp3`, `Title (2).mp3`, and so on.
- Choose a destination folder with the usual Windows folder window. The last folder, recent folders, and language are kept in `%APPDATA%\YouTubeMP3Saver\settings.json`. A folder that is not available stays in the list with its saved path. The app does not clear, replace, or create it.
- Saves the best available audio as MP3 (`bestaudio/best`, `--audio-quality 0`). It does not download a separate video just to extract the sound. The result is lossy MP3.
- English and Spanish. If Windows starts in a language beginning with `es`, the app starts in Spanish; otherwise English. Switching language does not clear the link, file name, folder, or history.
- After a file is written, **Show in folder** opens Explorer with that MP3 selected.

## Download

Windows 10 or later, 64-bit. Get the executable from the [Releases](https://github.com/wrongpixels/MPTube/releases) page.

## Requirements

There is no Python, Java, or administrator install. The executable does not bundle the helper programs. The first time a preview or download needs them, the app downloads these official builds over HTTPS, checks their SHA-256 checksums, and stores them in `%LOCALAPPDATA%\YouTubeMP3Saver\tools\`:

- yt-dlp 2026.10.26 (`yt-dlp.exe`)
- FFmpeg 9.0.2 essentials (`ffmpeg.exe`, from Gyan / GyanD)
- Deno 2.9.8 (`deno.exe`), the JavaScript runtime yt-dlp uses for full YouTube support

If those exact versions are already in the cache, they are reused. A file whose checksum does not match is discarded. Downloads can still fail if YouTube changes something.

## Build from source

Go 1.26. From the repository root, on Linux or Windows:

```
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-H windowsgui -s -w" -o dist/MPTube.exe ./cmd/youtubemp3
```

`scripts/build.sh` runs that command. A Windows application manifest and the app icon are embedded via `cmd/youtubemp3/rsrc_windows_amd64.syso` when present.

## Personal use

Use it only with videos you own or are authorized to download. Do not use it to copy other people's work or to bypass YouTube restrictions, DRM, or private/account-only content. Respect YouTube's terms and copyright. The app does not log in, store cookies, or send anything on your behalf.

The software is provided as is, without warranty of any kind.

## Licenses

The app license, third-party notices, and the helper-tool checksums are in [LICENSES/](LICENSES/).
