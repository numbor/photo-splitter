package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"photo-splitter-go/internal/app"
	"photo-splitter-go/internal/imageproc"
	"photo-splitter-go/internal/scan"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "process":
			if err := runProcessCmd(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			return
		case "scan-process":
			if err := runScanProcessCmd(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			return
		}
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func runProcessCmd(args []string) error {
	fs := flag.NewFlagSet("process", flag.ContinueOnError)
	input := fs.String("input", "", "percorso immagine scannerizzata")
	output := fs.String("output", "", "cartella output")
	jpgQuality := fs.Int("jpg-quality", 95, "qualita JPG output (1-100)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *input == "" {
		return fmt.Errorf("flag obbligatoria: --input")
	}
	if *output == "" {
		cwd, _ := os.Getwd()
		*output = filepath.Join(cwd, "output")
	}

	ts := time.Now().Format("20060102_150405")
	targetDir := filepath.Join(*output, ts)
	result, err := imageproc.ProcessTo4PhotosWithOptions(*input, targetDir, imageproc.Options{
		JPEGQuality: *jpgQuality,
	})
	if err != nil {
		return err
	}

	fmt.Printf("JPG_QUALITY=%d\n", *jpgQuality)
	fmt.Printf("OUTPUT_DIR=%s\n", targetDir)
	fmt.Printf("BORDERED=%s\n", result.BorderedImage)
	for _, p := range result.Crops {
		fmt.Printf("PHOTO=%s\n", p)
	}
	return nil
}

func runScanProcessCmd(args []string) error {
	fs := flag.NewFlagSet("scan-process", flag.ContinueOnError)
	output := fs.String("output", "", "cartella output")
	dpi := fs.Int("dpi", 300, "risoluzione scanner DPI (75-1200)")
	brightness := fs.Int("brightness", 0, "luminosita scanner (-1000..1000)")
	contrast := fs.Int("contrast", 0, "contrasto scanner (-1000..1000)")
	jpgQuality := fs.Int("jpg-quality", 95, "qualita JPG output (1-100)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *output == "" {
		cwd, _ := os.Getwd()
		*output = filepath.Join(cwd, "output")
	}

	ts := time.Now().Format("20060102_150405")
	scanDir := filepath.Join(*output, "raw_scans")
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		return err
	}

	scanPath := filepath.Join(scanDir, "scan_"+ts+".tiff")
	opts := scan.Options{
		DPI:        *dpi,
		Brightness: *brightness,
		Contrast:   *contrast,
	}
	if err := scan.AcquireScanTIFFWithOptions(scanPath, opts); err != nil {
		return err
	}

	targetDir := filepath.Join(*output, ts)
	result, err := imageproc.ProcessTo4PhotosWithOptions(scanPath, targetDir, imageproc.Options{
		JPEGQuality: *jpgQuality,
	})
	if err != nil {
		return err
	}

	fmt.Printf("SCAN=%s\n", scanPath)
	fmt.Printf("SCAN_DPI=%d\n", opts.DPI)
	fmt.Printf("SCAN_BRIGHTNESS=%d\n", opts.Brightness)
	fmt.Printf("SCAN_CONTRAST=%d\n", opts.Contrast)
	fmt.Printf("JPG_QUALITY=%d\n", *jpgQuality)
	fmt.Printf("OUTPUT_DIR=%s\n", targetDir)
	fmt.Printf("BORDERED=%s\n", result.BorderedImage)
	for _, p := range result.Crops {
		fmt.Printf("PHOTO=%s\n", p)
	}
	return nil
}
