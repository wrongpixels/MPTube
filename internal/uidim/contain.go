package uidim

// Contain fits imgW×imgH inside boundsW×boundsH without cropping or stretching.
// The result is centered. A square or portrait source keeps equal scale on both
// axes and leaves horizontal margins; a source wider than the slot leaves
// vertical margins. A 16:9 source in a 16:9 slot fills it.
//
// Dimensions are pixels. The preview slot stays 160×90 at 96 DPI (ThumbWidth ×
// ThumbHeight); callers pass the widget's pixel bounds and the decoded image's
// pixel bounds. Do not scale the source by DPI again.
func Contain(boundsW, boundsH, imgW, imgH int) (x, y, w, h int) {
	if boundsW <= 0 || boundsH <= 0 || imgW <= 0 || imgH <= 0 {
		return 0, 0, 0, 0
	}
	scale := float64(boundsW) / float64(imgW)
	if sy := float64(boundsH) / float64(imgH); sy < scale {
		scale = sy
	}
	w = int(float64(imgW) * scale)
	h = int(float64(imgH) * scale)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if w > boundsW {
		w = boundsW
	}
	if h > boundsH {
		h = boundsH
	}
	x = (boundsW - w) / 2
	y = (boundsH - h) / 2
	return
}
