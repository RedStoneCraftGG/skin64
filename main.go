package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// Load Image
func loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// Save image
func saveImage(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

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

func countTransparent(img image.Image, x, y, width, height int, alphaThreshold uint32) int {
	count := 0
	for xx := 0; xx < width; xx++ {
		allTransparent := true
		for yy := 0; yy < height; yy++ {
			if alpha255(img.At(x+xx, y+yy)) > alphaThreshold {
				allTransparent = false
				break
			}
		}
		if allTransparent {
			count++
		}
	}
	return count
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

func fillBottom(src image.Image) (image.Image, bool) {
	bounds := src.Bounds()
	if bounds.Dx() != 64 || bounds.Dy() != 64 {
		return src, false
	}

	alphaThreshold := uint32(0)

	if !isFullyTransparent(src, 32, 48, 16, 16, alphaThreshold) {
		return src, false
	}
	if !isFullyTransparent(src, 16, 48, 16, 16, alphaThreshold) {
		return src, false
	}

	armRects412 := [][4]int{
		{40, 52, 4, 12}, {32, 52, 4, 12}, // swap pair
		{44, 52, 4, 12}, {36, 52, 4, 12}, // mirror-in-place
	}

	armRects44 := [][4]int{{36, 48, 4, 4}, {40, 48, 4, 4}}
	legRects412 := [][4]int{
		{24, 52, 4, 12}, {16, 52, 4, 12}, // swap pair
		{20, 52, 4, 12}, {28, 52, 4, 12}, // mirror-in-place
	}

	legRects44 := [][4]int{{20, 48, 4, 4}, {24, 48, 4, 4}}

	for _, r := range legRects412 {
		if !isFullyTransparent(src, r[0], r[1], r[2], r[3], alphaThreshold) {
			return src, false
		}
	}
	for _, r := range legRects44 {
		if !isFullyTransparent(src, r[0], r[1], r[2], r[3], alphaThreshold) {
			return src, false
		}
	}

	for _, r := range armRects412 {
		fullyTransparent := isFullyTransparent(src, r[0], r[1], r[2], r[3], alphaThreshold)
		if fullyTransparent {
			continue
		}
		transparentCols := countTransparent(src, r[0], r[1], r[2], r[3], alphaThreshold)
		if transparentCols > 1 {
			return src, false
		}
	}

	for _, r := range armRects44 {
		if !isFullyTransparent(src, r[0], r[1], r[2], r[3], alphaThreshold) {
			return src, false
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
	mirrorPlace := func(x, y, width, height int) {
		for yy := 0; yy < height; yy++ {
			for xx := 0; xx < width/2; xx++ {
				left := dst.At(x+xx, y+yy)
				right := dst.At(x+(width-1-xx), y+yy)
				dst.Set(x+xx, y+yy, right)
				dst.Set(x+(width-1-xx), y+yy, left)
			}
		}
	}
	swapMirror := func(ax, ay, width, height, bx, by int) {
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

	topColsRight := countTransparentRight(src, 44, 16, 8, 4, alphaThreshold)
	armColsRight := countTransparentRight(src, 40, 20, 16, 12, alphaThreshold)
	isSlimTop := topColsRight >= 1 && topColsRight <= 2
	isSlimBody := armColsRight >= 1 && armColsRight <= 2

	copyPart(40, 16, 16, 16, 32, 48)
	copyPart(0, 16, 16, 16, 16, 48)
	swapMirror(40, 52, 4, 12, 32, 52)

	if isSlimBody {
		copyMirror(44, 20, 3, 12, 36, 52)
		copyMirror(51, 20, 3, 12, 43, 52)
	} else {
		mirrorPlace(44, 52, 4, 12)
		mirrorPlace(36, 52, 4, 12)
	}

	if isSlimTop {
		copyMirror(44, 16, 3, 4, 36, 48)
		copyMirror(47, 16, 3, 4, 39, 48)
	} else {
		mirrorPlace(36, 48, 4, 4)
		mirrorPlace(40, 48, 4, 4)
	}

	swapMirror(24, 52, 4, 12, 16, 52)
	mirrorPlace(20, 52, 4, 12)
	mirrorPlace(28, 52, 4, 12)
	mirrorPlace(20, 48, 4, 4)
	mirrorPlace(24, 48, 4, 4)

	return dst, true
}

func convertTo64(src image.Image) image.Image {
	w := 64
	h := 64
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))

	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			dst.Set(x, y, src.At(x, y))
		}
	}

	copyPart := func(srcX, srcY, width, height, dstX, dstY int) {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				color := src.At(srcX+x, srcY+y)
				dst.Set(dstX+x, dstY+y, color)
			}
		}
	}

	mirrorPlace := func(x, y, width, height int) {
		for yy := 0; yy < height; yy++ {
			for xx := 0; xx < width/2; xx++ {
				left := dst.At(x+xx, y+yy)
				right := dst.At(x+(width-1-xx), y+yy)
				dst.Set(x+xx, y+yy, right)
				dst.Set(x+(width-1-xx), y+yy, left)
			}
		}
	}

	swapMirror := func(ax, ay, width, height, bx, by int) {
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

	alphaThreshold := uint32(0)
	topColsRight := countTransparentRight(src, 44, 16, 8, 4, alphaThreshold)
	armColsRight := countTransparentRight(src, 40, 20, 16, 12, alphaThreshold)
	isSlimTop := topColsRight >= 1 && topColsRight <= 2
	isSlimBody := armColsRight >= 1 && armColsRight <= 2

	copyPart(40, 16, 16, 16, 32, 48)
	copyPart(0, 16, 16, 16, 16, 48)
	swapMirror(40, 52, 4, 12, 32, 52)
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
		mirrorPlace(44, 52, 4, 12)
		mirrorPlace(36, 52, 4, 12)
	}

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
		mirrorPlace(36, 48, 4, 4)
		mirrorPlace(40, 48, 4, 4)
	}

	swapMirror(24, 52, 4, 12, 16, 52)
	mirrorPlace(20, 52, 4, 12)
	mirrorPlace(28, 52, 4, 12)
	mirrorPlace(20, 48, 4, 4)
	mirrorPlace(24, 48, 4, 4)

	return dst
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: skin64.exe <file/folder path>")
		return
	}

	inputPath := os.Args[1]
	info, err := os.Stat(inputPath)
	if err != nil {
		fmt.Println("Failed to access path:", err)
		return
	}

	if info.IsDir() {
		outputDir := filepath.Join(inputPath, "converted")
		err := os.MkdirAll(outputDir, 0755)
		if err != nil {
			fmt.Println("Failed to create output directory:", err)
			return
		}

		err = filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && filepath.Base(path) == "converted" {
				return filepath.SkipDir
			}
			if !info.IsDir() && filepath.Ext(path) == ".png" {
				img, err := loadImage(path)
				if err != nil {
					fmt.Println("Failed to load", path, ":", err)
					return nil
				}

				bounds := img.Bounds()
				width := bounds.Dx()
				height := bounds.Dy()

				baseName := info.Name()
				if len(baseName) > 4 && baseName[len(baseName)-4:] == ".png" {
					baseName = baseName[:len(baseName)-4]
				}

				switch {
				case width == 64 && height == 32:
					converted := convertTo64(img)
					outPath := filepath.Join(outputDir, baseName+"_converted.png")
					saveImage(converted, outPath)
					fmt.Println("Successfully converted 64x32:", path, "->", outPath)
				case width == 64 && height == 64:
					filled, ok := fillBottom(img)
					if ok {
						outPath := filepath.Join(outputDir, baseName+"_fixed.png")
						saveImage(filled, outPath)
						fmt.Println("Successfully fixed 64x64 (fill bottom):", path, "->", outPath)
					} else {
						fmt.Println("Skipped 64x64 (bottom does not meet requirements):", path)
					}
				default:
					fmt.Println("Skipped (unsupported size):", path)
				}
			}
			return nil
		})

		if err != nil {
			fmt.Println("An error occurred:", err)
		}
	} else {
		if filepath.Ext(inputPath) != ".png" {
			fmt.Println("File is not PNG:", inputPath)
			return
		}
		img, err := loadImage(inputPath)
		if err != nil {
			fmt.Println("Failed to load", inputPath, ":", err)
			return
		}

		bounds := img.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		dir := filepath.Dir(inputPath)
		baseName := filepath.Base(inputPath)
		if len(baseName) > 4 && baseName[len(baseName)-4:] == ".png" {
			baseName = baseName[:len(baseName)-4]
		}

		switch {
		case width == 64 && height == 32:
			converted := convertTo64(img)
			outPath := filepath.Join(dir, baseName+"_converted.png")
			saveImage(converted, outPath)
			fmt.Println("Successfully converted 64x32:", inputPath, "->", outPath)
		case width == 64 && height == 64:
			filled, ok := fillBottom(img)
			if ok {
				outPath := filepath.Join(dir, baseName+"_fixed.png")
				saveImage(filled, outPath)
				fmt.Println("Successfully fixed 64x64 (fill bottom):", inputPath, "->", outPath)
			} else {
				fmt.Println("Skipped 64x64 (bottom does not meet requirements):", inputPath)
			}
		default:
			fmt.Println("Skipped (unsupported size):", inputPath)
		}
	}
}
