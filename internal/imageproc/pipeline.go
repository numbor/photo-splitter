package imageproc

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
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
}

func (o Options) normalized() Options {
	if o.JPEGQuality <= 0 {
		o.JPEGQuality = 95
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

	borderedPath := filepath.Join(outputDir, "scan_bordered.jpg")
	if err := addWhiteBorder(inputPath, borderedPath, opt.JPEGQuality); err != nil {
		return Result{}, err
	}

	f, err := os.Open(borderedPath)
	if err != nil {
		return Result{}, fmt.Errorf("apertura immagine bordata: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return Result{}, fmt.Errorf("decode immagine bordata: %w", err)
	}

	rects := detect4Regions(img)
	if len(rects) != 4 {
		return Result{}, fmt.Errorf("rilevamento foto fallito: aree trovate=%d", len(rects))
	}

	cropFiles := make([]string, 0, 4)
	for i, rect := range rects {
		outPath := filepath.Join(outputDir, fmt.Sprintf("photo_%d.jpg", i+1))
		if err := cropToJPEG(img, rect, outPath, opt.JPEGQuality, opt.AutoRotateCrops); err != nil {
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

func addWhiteBorder(inputPath, outputPath string, jpegQuality int) error {
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("apertura immagine input: %w", err)
	}
	if len(raw) == 0 {
		return fmt.Errorf("immagine input vuota: %s", inputPath)
	}

	img, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		headLen := min(16, len(raw))
		return fmt.Errorf("decode immagine input: %w (size=%d, header=% X)", err, len(raw), raw[:headLen])
	}
	_ = format

	b := img.Bounds()
	border := 12
	outRect := image.Rect(0, 0, b.Dx()+border*2, b.Dy()+border*2)
	canvas := image.NewRGBA(outRect)

	draw.Draw(canvas, outRect, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(canvas, image.Rect(border, border, border+b.Dx(), border+b.Dy()), img, b.Min, draw.Src)

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creazione immagine bordata: %w", err)
	}
	defer out.Close()

	if err := jpeg.Encode(out, canvas, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return fmt.Errorf("encoding immagine bordata: %w", err)
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

func cropToJPEG(src image.Image, rect image.Rectangle, outputPath string, jpegQuality int, autoRotateCrops bool) error {
	crop := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(crop, crop.Bounds(), src, rect.Min, draw.Src)
	finalImage := image.Image(crop)
	if autoRotateCrops {
		finalImage = rotateImage(crop, 90)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return jpeg.Encode(f, finalImage, &jpeg.Options{Quality: jpegQuality})
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
