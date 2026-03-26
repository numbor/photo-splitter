package imageproc

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"os"
)

func RotateJPEGFile(path string, angle int, quality int) error {
	normalizedAngle := ((angle % 360) + 360) % 360
	if normalizedAngle != 90 && normalizedAngle != 180 && normalizedAngle != 270 {
		return fmt.Errorf("angolo non supportato: %d (usa 90, 180, 270)", angle)
	}

	opt := Options{JPEGQuality: quality}.normalized()

	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("lettura file: %w", err)
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("decode immagine: %w", err)
	}

	rotated := rotateImage(src, normalizedAngle)

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("apertura file output: %w", err)
	}
	defer f.Close()

	if err := jpeg.Encode(f, rotated, &jpeg.Options{Quality: opt.JPEGQuality}); err != nil {
		return fmt.Errorf("encoding jpeg: %w", err)
	}

	return nil
}

func rotateImage(src image.Image, angle int) image.Image {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()

	switch angle {
	case 90:
		dst := image.NewRGBA(image.Rect(0, 0, sh, sw))
		for y := 0; y < sh; y++ {
			for x := 0; x < sw; x++ {
				dst.Set(sh-1-y, x, src.At(b.Min.X+x, b.Min.Y+y))
			}
		}
		return dst
	case 180:
		dst := image.NewRGBA(image.Rect(0, 0, sw, sh))
		for y := 0; y < sh; y++ {
			for x := 0; x < sw; x++ {
				dst.Set(sw-1-x, sh-1-y, src.At(b.Min.X+x, b.Min.Y+y))
			}
		}
		return dst
	case 270:
		dst := image.NewRGBA(image.Rect(0, 0, sh, sw))
		for y := 0; y < sh; y++ {
			for x := 0; x < sw; x++ {
				dst.Set(y, sw-1-x, src.At(b.Min.X+x, b.Min.Y+y))
			}
		}
		return dst
	default:
		dst := image.NewRGBA(image.Rect(0, 0, sw, sh))
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
		return dst
	}
}
