package metadata

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

// DecodePreview decodes a still thumbnail. JPEG, PNG, GIF, and WebP (VP8 / VP8L)
// are registered. Animated WebP is left unsupported and returns an error so the
// caller can try another candidate.
func DecodePreview(data []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	return img, format, nil
}
