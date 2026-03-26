package imageproc

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
)

type Result struct {
	BorderedImage string
	Crops         []string
	Rectangles    []image.Rectangle
}

type Options struct {
	JPEGQuality     int
	AutoRotateCrops bool
	SkipWhiteBorder bool
	SkipEnhancement bool
	DPI             int
}

func (o Options) normalized() Options {
	if o.JPEGQuality <= 0 {
		o.JPEGQuality = 100
	}
	if o.JPEGQuality < 1 {
		o.JPEGQuality = 1
	}
	if o.JPEGQuality > 100 {
		o.JPEGQuality = 100
	}
	return o
}

func ProcessTo4Photos(inputPath, outputDir string) (Result, error) {
	return ProcessTo4PhotosWithOptions(inputPath, outputDir, Options{AutoRotateCrops: true})
}

func ProcessTo4PhotosWithOptions(inputPath, outputDir string, options Options) (Result, error) {
	opt := options.normalized()

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("creazione output dir: %w", err)
	}

	f, err := os.Open(inputPath)
	if err != nil {
		return Result{}, fmt.Errorf("apertura immagine elaborazione: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return Result{}, fmt.Errorf("decode immagine elaborazione: %w", err)
	}

	workingImage := image.Image(img)
	borderedPath := ""
	if !opt.SkipWhiteBorder {
		bordered := addWhiteBorderImage(img)
		borderedPath = filepath.Join(outputDir, "scan_bordered.png")
		if err := savePNG(borderedPath, bordered); err != nil {
			return Result{}, err
		}
		workingImage = bordered
	}

	rects := detect4Regions(workingImage)
	if len(rects) != 4 {
		return Result{}, fmt.Errorf("rilevamento foto fallito: aree trovate=%d", len(rects))
	}

	cropFiles := make([]string, 0, 4)
	for i, rect := range rects {
		outPath := filepath.Join(outputDir, fmt.Sprintf("photo_%d.jpg", i+1))
		if err := cropToJPEG(workingImage, rect, outPath, opt.JPEGQuality, opt.AutoRotateCrops, opt.SkipEnhancement, opt.DPI); err != nil {
			return Result{}, fmt.Errorf("crop foto %d: %w", i+1, err)
		}
		cropFiles = append(cropFiles, outPath)
	}

	return Result{
		BorderedImage: borderedPath,
		Crops:         cropFiles,
		Rectangles:    rects,
	}, nil
}

func addWhiteBorderImage(img image.Image) *image.RGBA {
	b := img.Bounds()
	border := 12
	outRect := image.Rect(0, 0, b.Dx()+border*2, b.Dy()+border*2)
	canvas := image.NewRGBA(outRect)

	draw.Draw(canvas, outRect, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(canvas, image.Rect(border, border, border+b.Dx(), border+b.Dy()), img, b.Min, draw.Src)
	return canvas
}

func savePNG(outputPath string, img image.Image) error {
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creazione immagine bordata: %w", err)
	}
	defer out.Close()

	if err := png.Encode(out, img); err != nil {
		return fmt.Errorf("encoding immagine bordata png: %w", err)
	}

	return nil
}

func detect4Regions(img image.Image) []image.Rectangle {
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w < 100 || h < 100 {
		return nil
	}

	colDark := make([]int, w)
	rowDark := make([]int, h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if isDark(img.At(b.Min.X+x, b.Min.Y+y)) {
				colDark[x]++
				rowDark[y]++
			}
		}
	}

	colMinDark := max(8, h/80)
	rowMinDark := max(8, w/80)

	left := firstAbove(colDark, colMinDark)
	right := lastAbove(colDark, colMinDark)
	top := firstAbove(rowDark, rowMinDark)
	bottom := lastAbove(rowDark, rowMinDark)

	if left < 0 || right <= left || top < 0 || bottom <= top {
		return fallbackQuadrants(b)
	}

	splitX := valleyNearCenter(colDark, left, right)
	splitY := valleyNearCenter(rowDark, top, bottom)

	if splitX <= left+40 || splitX >= right-40 || splitY <= top+40 || splitY >= bottom-40 {
		return fallbackQuadrants(b)
	}

	gapX := max(3, (right-left)/120)
	gapY := max(3, (bottom-top)/120)

	rects := []image.Rectangle{
		image.Rect(b.Min.X+left, b.Min.Y+top, b.Min.X+splitX-gapX, b.Min.Y+splitY-gapY),
		image.Rect(b.Min.X+splitX+gapX, b.Min.Y+top, b.Min.X+right, b.Min.Y+splitY-gapY),
		image.Rect(b.Min.X+left, b.Min.Y+splitY+gapY, b.Min.X+splitX-gapX, b.Min.Y+bottom),
		image.Rect(b.Min.X+splitX+gapX, b.Min.Y+splitY+gapY, b.Min.X+right, b.Min.Y+bottom),
	}

	for i := range rects {
		rects[i] = normalizeRect(rects[i], b)
	}

	if !validRects(rects) {
		return fallbackQuadrants(b)
	}

	return rects
}

func cropToJPEG(src image.Image, rect image.Rectangle, outputPath string, jpegQuality int, autoRotateCrops bool, skipEnhancement bool, dpi int) error {
	crop := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(crop, crop.Bounds(), src, rect.Min, draw.Src)
	trimmed := trimWhiteBorder(crop)
	finalImage := image.Image(trimmed)
	if !skipEnhancement {
		finalImage = enhancePhotoQuality(trimmed)
	}
	if autoRotateCrops {
		finalImage = rotateImage(finalImage, 90)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if dpi > 0 {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, finalImage, &jpeg.Options{Quality: jpegQuality}); err != nil {
			return err
		}
		imgBytes := buf.Bytes()
		setJFIFDPI(imgBytes, dpi)
		_, err = f.Write(imgBytes)
		return err
	}
	return jpeg.Encode(f, finalImage, &jpeg.Options{Quality: jpegQuality})
}

// setJFIFDPI patches the JFIF APP0 segment of an in-memory JPEG to set the DPI metadata.
// JFIF APP0 layout from byte 0: FF D8 (SOI) | FF E0 (APP0) | len[2] | "JFIF\0"[5] | ver[2] | units[1] | Xdensity[2] | Ydensity[2]
func setJFIFDPI(data []byte, dpi int) {
	if len(data) < 18 {
		return
	}
	if data[0] != 0xFF || data[1] != 0xD8 || data[2] != 0xFF || data[3] != 0xE0 {
		return
	}
	if data[6] != 'J' || data[7] != 'F' || data[8] != 'I' || data[9] != 'F' || data[10] != 0x00 {
		return
	}
	data[13] = 1 // density units: 1 = DPI
	data[14] = byte(dpi >> 8)
	data[15] = byte(dpi & 0xFF)
	data[16] = byte(dpi >> 8)
	data[17] = byte(dpi & 0xFF)
}

func trimWhiteBorder(src image.Image) image.Image {
	b := src.Bounds()
	width := b.Dx()
	height := b.Dy()
	if width < 8 || height < 8 {
		return src
	}

	const minBorderWhiteRatio = 0.965
	const maxTrimRatio = 0.22

	maxTrimX := max(1, int(float64(width)*maxTrimRatio))
	maxTrimY := max(1, int(float64(height)*maxTrimRatio))

	borderWhiteRatioRow := func(y int, x1 int, x2 int) float64 {
		if x2 < x1 {
			return 0
		}
		count := 0
		total := x2 - x1 + 1
		for x := x1; x <= x2; x++ {
			if isBorderWhitePixel(src.At(x, y)) {
				count++
			}
		}
		return float64(count) / float64(total)
	}

	borderWhiteRatioCol := func(x int, y1 int, y2 int) float64 {
		if y2 < y1 {
			return 0
		}
		count := 0
		total := y2 - y1 + 1
		for y := y1; y <= y2; y++ {
			if isBorderWhitePixel(src.At(x, y)) {
				count++
			}
		}
		return float64(count) / float64(total)
	}

	top := b.Min.Y
	for step := 0; step < maxTrimY && top < b.Max.Y-1; step++ {
		ratio := borderWhiteRatioRow(top, b.Min.X, b.Max.X-1)
		if ratio < minBorderWhiteRatio {
			break
		}
		top++
	}

	bottom := b.Max.Y - 1
	for step := 0; step < maxTrimY && bottom > top; step++ {
		ratio := borderWhiteRatioRow(bottom, b.Min.X, b.Max.X-1)
		if ratio < minBorderWhiteRatio {
			break
		}
		bottom--
	}

	left := b.Min.X
	for step := 0; step < maxTrimX && left < b.Max.X-1; step++ {
		ratio := borderWhiteRatioCol(left, top, bottom)
		if ratio < minBorderWhiteRatio {
			break
		}
		left++
	}

	right := b.Max.X - 1
	for step := 0; step < maxTrimX && right > left; step++ {
		ratio := borderWhiteRatioCol(right, top, bottom)
		if ratio < minBorderWhiteRatio {
			break
		}
		right--
	}

	if left >= right || top >= bottom {
		return src
	}

	trimRect := image.Rect(left, top, right+1, bottom+1)
	out := image.NewRGBA(image.Rect(0, 0, trimRect.Dx(), trimRect.Dy()))
	draw.Draw(out, out.Bounds(), src, trimRect.Min, draw.Src)
	return out
}

func isBorderWhitePixel(c colorLike) bool {
	r, g, b, _ := c.RGBA()
	r8 := int(r >> 8)
	g8 := int(g >> 8)
	b8 := int(b >> 8)

	if r8 < 230 || g8 < 230 || b8 < 230 {
		return false
	}

	maxC := max(r8, max(g8, b8))
	minC := min(r8, min(g8, b8))
	return (maxC - minC) <= 24
}

func enhancePhotoQuality(src image.Image) *image.RGBA {
	base := toRGBA(src)
	leveled := autoLevel(base)
	gammaCorrected := autoGamma(leveled)
	modulated := modulateSaturation(gammaCorrected, 1.15)
	return sharpen(modulated)
}

func toRGBA(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

func autoLevel(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))

	// Build per-channel histograms.
	var histR, histG, histB [256]int
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := src.At(x, y).RGBA()
			histR[r>>8]++
			histG[g>>8]++
			histB[bl>>8]++
		}
	}

	// Use 1st/99th percentile to ignore scanner noise outliers instead of
	// absolute min/max (a single dark pixel would otherwise prevent any stretch).
	total := b.Dx() * b.Dy()
	clip := max(1, total/100)

	percLow := func(hist *[256]int) uint8 {
		cum := 0
		for i := 0; i < 256; i++ {
			cum += hist[i]
			if cum >= clip {
				return uint8(i)
			}
		}
		return 0
	}
	percHigh := func(hist *[256]int) uint8 {
		cum := 0
		for i := 255; i >= 0; i-- {
			cum += hist[i]
			if cum >= clip {
				return uint8(i)
			}
		}
		return 255
	}

	minR, maxR := percLow(&histR), percHigh(&histR)
	minG, maxG := percLow(&histG), percHigh(&histG)
	minB, maxB := percLow(&histB), percHigh(&histB)

	scale := func(v, minV, maxV uint8) uint8 {
		if maxV <= minV {
			return v
		}
		value := (int(v) - int(minV)) * 255 / (int(maxV) - int(minV))
		return uint8(clampInt(value, 0, 255))
	}

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := src.At(x, y).RGBA()
			dst.SetRGBA(x-b.Min.X, y-b.Min.Y, color.RGBA{
				R: scale(uint8(r>>8), minR, maxR),
				G: scale(uint8(g>>8), minG, maxG),
				B: scale(uint8(bl>>8), minB, maxB),
				A: uint8(a >> 8),
			})
		}
	}

	return dst
}

func autoGamma(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))

	var total float64
	pixelCount := float64(b.Dx() * b.Dy())
	if pixelCount <= 0 {
		return src
	}

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := src.At(x, y).RGBA()
			l := (0.2126*float64(r>>8) + 0.7152*float64(g>>8) + 0.0722*float64(bl>>8)) / 255.0
			total += l
		}
	}

	mean := total / pixelCount
	if mean <= 0 {
		mean = 0.01
	}
	if mean >= 1 {
		mean = 0.99
	}

	gamma := math.Log(0.5) / math.Log(mean)
	gamma = clampFloat(gamma, 0.6, 1.8)

	correct := func(v uint8) uint8 {
		vf := float64(v) / 255.0
		out := math.Pow(vf, gamma) * 255.0
		return uint8(clampInt(int(math.Round(out)), 0, 255))
	}

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := src.At(x, y).RGBA()
			dst.SetRGBA(x-b.Min.X, y-b.Min.Y, color.RGBA{
				R: correct(uint8(r >> 8)),
				G: correct(uint8(g >> 8)),
				B: correct(uint8(bl >> 8)),
				A: uint8(a >> 8),
			})
		}
	}

	return dst
}

func modulateSaturation(src *image.RGBA, saturationScale float64) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := src.At(x, y).RGBA()
			rf := float64(r>>8) / 255.0
			gf := float64(g>>8) / 255.0
			bf := float64(bl>>8) / 255.0

			h, s, v := rgbToHSV(rf, gf, bf)
			s = clampFloat(s*saturationScale, 0, 1)
			r2, g2, b2 := hsvToRGB(h, s, v)

			dst.SetRGBA(x-b.Min.X, y-b.Min.Y, color.RGBA{
				R: uint8(clampInt(int(math.Round(r2*255)), 0, 255)),
				G: uint8(clampInt(int(math.Round(g2*255)), 0, 255)),
				B: uint8(clampInt(int(math.Round(b2*255)), 0, 255)),
				A: uint8(a >> 8),
			})
		}
	}

	return dst
}

func sharpen(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	blur := boxBlur3x3(src)
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r1, g1, b1, a := src.At(x, y).RGBA()
			r2, g2, b2, _ := blur.At(x-b.Min.X, y-b.Min.Y).RGBA()

			outR := int(r1>>8) + (int(r1>>8) - int(r2>>8))
			outG := int(g1>>8) + (int(g1>>8) - int(g2>>8))
			outB := int(b1>>8) + (int(b1>>8) - int(b2>>8))

			dst.SetRGBA(x-b.Min.X, y-b.Min.Y, color.RGBA{
				R: uint8(clampInt(outR, 0, 255)),
				G: uint8(clampInt(outG, 0, 255)),
				B: uint8(clampInt(outB, 0, 255)),
				A: uint8(a >> 8),
			})
		}
	}

	return dst
}

func boxBlur3x3(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))

	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			var sumR, sumG, sumB, sumA, count int
			for ky := -1; ky <= 1; ky++ {
				ny := y + ky
				if ny < 0 || ny >= b.Dy() {
					continue
				}
				for kx := -1; kx <= 1; kx++ {
					nx := x + kx
					if nx < 0 || nx >= b.Dx() {
						continue
					}
					r, g, bl, a := src.At(b.Min.X+nx, b.Min.Y+ny).RGBA()
					sumR += int(r >> 8)
					sumG += int(g >> 8)
					sumB += int(bl >> 8)
					sumA += int(a >> 8)
					count++
				}
			}

			dst.SetRGBA(x, y, color.RGBA{
				R: uint8(sumR / count),
				G: uint8(sumG / count),
				B: uint8(sumB / count),
				A: uint8(sumA / count),
			})
		}
	}

	return dst
}

func rgbToHSV(r, g, b float64) (float64, float64, float64) {
	maxC := math.Max(r, math.Max(g, b))
	minC := math.Min(r, math.Min(g, b))
	delta := maxC - minC

	h := 0.0
	if delta != 0 {
		switch maxC {
		case r:
			h = math.Mod((g-b)/delta, 6)
		case g:
			h = ((b-r)/delta + 2)
		default:
			h = ((r-g)/delta + 4)
		}
		h *= 60
		if h < 0 {
			h += 360
		}
	}

	s := 0.0
	if maxC != 0 {
		s = delta / maxC
	}

	v := maxC
	return h, s, v
}

func hsvToRGB(h, s, v float64) (float64, float64, float64) {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := v - c

	var rf, gf, bf float64
	switch {
	case h < 60:
		rf, gf, bf = c, x, 0
	case h < 120:
		rf, gf, bf = x, c, 0
	case h < 180:
		rf, gf, bf = 0, c, x
	case h < 240:
		rf, gf, bf = 0, x, c
	case h < 300:
		rf, gf, bf = x, 0, c
	default:
		rf, gf, bf = c, 0, x
	}

	return rf + m, gf + m, bf + m
}

func clampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func clampInt(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func fallbackQuadrants(bounds image.Rectangle) []image.Rectangle {
	mx := bounds.Min.X + bounds.Dx()/2
	my := bounds.Min.Y + bounds.Dy()/2
	margX := max(4, bounds.Dx()/200)
	margY := max(4, bounds.Dy()/200)

	return []image.Rectangle{
		normalizeRect(image.Rect(bounds.Min.X, bounds.Min.Y, mx-margX, my-margY), bounds),
		normalizeRect(image.Rect(mx+margX, bounds.Min.Y, bounds.Max.X, my-margY), bounds),
		normalizeRect(image.Rect(bounds.Min.X, my+margY, mx-margX, bounds.Max.Y), bounds),
		normalizeRect(image.Rect(mx+margX, my+margY, bounds.Max.X, bounds.Max.Y), bounds),
	}
}

func isDark(c colorLike) bool {
	r, g, b, _ := c.RGBA()
	gray := 0.2126*float64(r>>8) + 0.7152*float64(g>>8) + 0.0722*float64(b>>8)
	return gray < 235
}

type colorLike interface {
	RGBA() (r, g, b, a uint32)
}

func firstAbove(values []int, threshold int) int {
	for i, v := range values {
		if v >= threshold {
			return i
		}
	}
	return -1
}

func lastAbove(values []int, threshold int) int {
	for i := len(values) - 1; i >= 0; i-- {
		if values[i] >= threshold {
			return i
		}
	}
	return -1
}

func valleyNearCenter(values []int, start, end int) int {
	center := (start + end) / 2
	halfWindow := int(math.Max(20, float64((end-start)/4)))
	l := max(start+20, center-halfWindow)
	r := min(end-20, center+halfWindow)
	best := center
	bestVal := math.MaxInt
	for i := l; i <= r; i++ {
		if values[i] < bestVal {
			bestVal = values[i]
			best = i
		}
	}
	return best
}

func normalizeRect(r, bounds image.Rectangle) image.Rectangle {
	if r.Min.X < bounds.Min.X {
		r.Min.X = bounds.Min.X
	}
	if r.Min.Y < bounds.Min.Y {
		r.Min.Y = bounds.Min.Y
	}
	if r.Max.X > bounds.Max.X {
		r.Max.X = bounds.Max.X
	}
	if r.Max.Y > bounds.Max.Y {
		r.Max.Y = bounds.Max.Y
	}
	return r
}

func validRects(rects []image.Rectangle) bool {
	for _, r := range rects {
		if r.Dx() < 60 || r.Dy() < 60 {
			return false
		}
	}
	return len(rects) == 4
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
