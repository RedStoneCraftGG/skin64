package skin64

import (
	"image"
	"image/color"
)

func alpha255(c color.Color) uint32 {
	_, _, _, a := c.RGBA()
	return a >> 8
}

func isFullyTransparent(img image.Image, x, y, width, height int, alphaThreshold uint32) bool {
	for yy := 0; yy < height; yy++ {
		for xx := 0; xx < width; xx++ {
			if alpha255(img.At(x+xx, y+yy)) > alphaThreshold {
				return false
			}
		}
	}
	return true
}

func countTransparentRight(img image.Image, x, y, width, height int, alphaThreshold uint32) int {
	count := 0
	for off := 0; off < width; off++ {
		colX := x + (width - 1 - off)
		allTransparent := true
		for yy := 0; yy < height; yy++ {
			if alpha255(img.At(colX, y+yy)) > alphaThreshold {
				allTransparent = false
				break
			}
		}
		if allTransparent {
			count++
		} else {
			break
		}
	}
	return count
}

func mirrorPlace(dst *image.NRGBA, x, y, width, height int) {
	for yy := 0; yy < height; yy++ {
		for xx := 0; xx < width/2; xx++ {
			left := dst.At(x+xx, y+yy)
			right := dst.At(x+(width-1-xx), y+yy)
			dst.Set(x+xx, y+yy, right)
			dst.Set(x+(width-1-xx), y+yy, left)
		}
	}
}

func swapMirror(dst *image.NRGBA, ax, ay, width, height, bx, by int) {
	bufA := make([]color.Color, width*height)
	bufB := make([]color.Color, width*height)
	for yy := 0; yy < height; yy++ {
		for xx := 0; xx < width; xx++ {
			bufA[yy*width+xx] = dst.At(ax+xx, ay+yy)
			bufB[yy*width+xx] = dst.At(bx+xx, by+yy)
		}
	}
	for yy := 0; yy < height; yy++ {
		for xx := 0; xx < width; xx++ {
			srcX := width - 1 - xx
			dst.Set(bx+xx, by+yy, bufA[yy*width+srcX])
		}
	}
	for yy := 0; yy < height; yy++ {
		for xx := 0; xx < width; xx++ {
			srcX := width - 1 - xx
			dst.Set(ax+xx, ay+yy, bufB[yy*width+srcX])
		}
	}
}

func convertTo64(src image.Image) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, 64, 64))

	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			dst.Set(x, y, src.At(x, y))
		}
	}

	copyPart := func(srcX, srcY, width, height, dstX, dstY int) {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				c := src.At(srcX+x, srcY+y)
				dst.Set(dstX+x, dstY+y, c)
			}
		}
	}

	alphaThreshold := uint32(0)
	topColsRight := countTransparentRight(src, 44, 16, 8, 4, alphaThreshold)
	armColsRight := countTransparentRight(src, 40, 20, 16, 12, alphaThreshold)
	isSlimTop := topColsRight >= 1 && topColsRight <= 2
	isSlimBody := armColsRight >= 1 && armColsRight <= 2

	copyPart(40, 16, 16, 16, 32, 48)
	copyPart(0, 16, 16, 16, 16, 48)

	// Arms body
	swapMirror(dst, 40, 52, 4, 12, 32, 52)
	if isSlimBody {
		for y := 0; y < 12; y++ {
			for x := 0; x < 3; x++ {
				c1 := src.At(44+(3-1-x), 20+y)
				dst.Set(36+x, 52+y, c1)
				c2 := src.At(51+(3-1-x), 20+y)
				dst.Set(43+x, 52+y, c2)
			}
		}
	} else {
		mirrorPlace(dst, 44, 52, 4, 12)
		mirrorPlace(dst, 36, 52, 4, 12)
	}

	// Arms top
	if isSlimTop {
		for y := 0; y < 4; y++ {
			for x := 0; x < 3; x++ {
				c1 := src.At(44+(3-1-x), 16+y)
				dst.Set(36+x, 48+y, c1)
				c2 := src.At(47+(3-1-x), 16+y)
				dst.Set(39+x, 48+y, c2)
			}
		}
	} else {
		mirrorPlace(dst, 36, 48, 4, 4)
		mirrorPlace(dst, 40, 48, 4, 4)
	}

	// Legs
	swapMirror(dst, 24, 52, 4, 12, 16, 52)
	mirrorPlace(dst, 20, 52, 4, 12)
	mirrorPlace(dst, 28, 52, 4, 12)
	mirrorPlace(dst, 20, 48, 4, 4)
	mirrorPlace(dst, 24, 48, 4, 4)

	return dst
}

func fillBottom(src image.Image) (*image.NRGBA, bool) {
	bounds := src.Bounds()
	if bounds.Dx() != 64 || bounds.Dy() != 64 {
		return nil, false
	}

	alphaThreshold := uint32(0)
	if !isFullyTransparent(src, 32, 48, 16, 16, alphaThreshold) {
		return nil, false
	}
	if !isFullyTransparent(src, 16, 48, 16, 16, alphaThreshold) {
		return nil, false
	}

	legRects412 := [][4]int{{24, 52, 4, 12}, {16, 52, 4, 12}, {20, 52, 4, 12}, {28, 52, 4, 12}}
	legRects44 := [][4]int{{20, 48, 4, 4}, {24, 48, 4, 4}}
	for _, r := range legRects412 {
		if !isFullyTransparent(src, r[0], r[1], r[2], r[3], alphaThreshold) {
			return nil, false
		}
	}
	for _, r := range legRects44 {
		if !isFullyTransparent(src, r[0], r[1], r[2], r[3], alphaThreshold) {
			return nil, false
		}
	}

	armRects412 := [][4]int{{40, 52, 4, 12}, {32, 52, 4, 12}, {44, 52, 4, 12}, {36, 52, 4, 12}}
	for _, r := range armRects412 {
		if !isFullyTransparent(src, r[0], r[1], r[2], r[3], alphaThreshold) {
			left := countTransparentRight(src, r[0], r[1], r[2], r[3], alphaThreshold)
			if left > 1 {
				return nil, false
			}
		}
	}
	for _, r := range [][4]int{{36, 48, 4, 4}, {40, 48, 4, 4}} {
		if !isFullyTransparent(src, r[0], r[1], r[2], r[3], alphaThreshold) {
			return nil, false
		}
	}

	dst := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			dst.Set(x, y, src.At(x, y))
		}
	}

	copyPart := func(srcX, srcY, width, height, dstX, dstY int) {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				c := src.At(srcX+x, srcY+y)
				dst.Set(dstX+x, dstY+y, c)
			}
		}
	}
	copyMirror := func(srcX, srcY, width, height, dstX, dstY int) {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				c := src.At(srcX+(width-1-x), srcY+y)
				dst.Set(dstX+x, dstY+y, c)
			}
		}
	}

	topColsRight := countTransparentRight(src, 44, 16, 8, 4, alphaThreshold)
	armColsRight := countTransparentRight(src, 40, 20, 16, 12, alphaThreshold)
	isSlimTop := topColsRight >= 1 && topColsRight <= 2
	isSlimBody := armColsRight >= 1 && armColsRight <= 2

	copyPart(40, 16, 16, 16, 32, 48)
	copyPart(0, 16, 16, 16, 16, 48)

	swapMirror(dst, 40, 52, 4, 12, 32, 52)
	if isSlimBody {
		copyMirror(44, 20, 3, 12, 36, 52)
		copyMirror(51, 20, 3, 12, 43, 52)
	} else {
		mirrorPlace(dst, 44, 52, 4, 12)
		mirrorPlace(dst, 36, 52, 4, 12)
	}
	if isSlimTop {
		copyMirror(44, 16, 3, 4, 36, 48)
		copyMirror(47, 16, 3, 4, 39, 48)
	} else {
		mirrorPlace(dst, 36, 48, 4, 4)
		mirrorPlace(dst, 40, 48, 4, 4)
	}

	swapMirror(dst, 24, 52, 4, 12, 16, 52)
	mirrorPlace(dst, 20, 52, 4, 12)
	mirrorPlace(dst, 28, 52, 4, 12)
	mirrorPlace(dst, 20, 48, 4, 4)
	mirrorPlace(dst, 24, 48, 4, 4)

	return dst, true
}

func ConvertSize64(img image.Image) (*image.NRGBA, bool, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	switch {
	case w == 64 && h == 32:
		out := convertTo64(img)
		return out, true, nil
	case w == 64 && h == 64:
		if out, ok := fillBottom(img); ok {
			return out, true, nil
		}
		wrapped := image.NewNRGBA(image.Rect(0, 0, 64, 64))
		for y := 0; y < 64; y++ {
			for x := 0; x < 64; x++ {
				wrapped.Set(x, y, img.At(x, y))
			}
		}
		return wrapped, false, nil
	default:
		return nil, false, ErrUnsupportedSize
	}
}

var ErrUnsupportedSize = errUnsupportedSize{}

type errUnsupportedSize struct{}

func (e errUnsupportedSize) Error() string { return "unsupported image size (expected 64x32 or 64x64)" }
