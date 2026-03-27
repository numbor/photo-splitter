package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"photo-splitter-go/internal/imageproc"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "process":
			if err := runProcessCmd(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			return
		case "rotate":
			if err := runRotateCmd(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			return
		}
	}

	if err := runWailsApp(); err != nil {
		log.Fatal(err)
	}
}

func runProcessCmd(args []string) error {
	fs := flag.NewFlagSet("process", flag.ContinueOnError)
	input := fs.String("input", "", "percorso immagine da elaborare")
	output := fs.String("output", "", "cartella output")
	jpgQuality := fs.Int("jpg-quality", 100, "qualita JPG output (1-100)")
	autoRotateCrops := fs.Bool("auto-rotate-crops", true, "ruota automaticamente di 90° a destra ogni crop")
	addBorder := fs.Bool("add-border", true, "aggiunge il bordo bianco prima del rilevamento e crop")
	enhanceCrops := fs.Bool("enhance-crops", true, "applica miglioramento automatico ai crop finali")
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
		JPEGQuality:     *jpgQuality,
		AutoRotateCrops: *autoRotateCrops,
		SkipWhiteBorder: !*addBorder,
		SkipEnhancement: !*enhanceCrops,
	})
	if err != nil {
		return err
	}

	fmt.Printf("JPG_QUALITY=%d\n", *jpgQuality)
	fmt.Printf("AUTO_ROTATE_CROPS=%t\n", *autoRotateCrops)
	fmt.Printf("ADD_BORDER=%t\n", *addBorder)
	fmt.Printf("ENHANCE_CROPS=%t\n", *enhanceCrops)
	fmt.Printf("OUTPUT_DIR=%s\n", targetDir)
	fmt.Printf("BORDERED=%s\n", result.BorderedImage)
	for _, p := range result.Crops {
		fmt.Printf("PHOTO=%s\n", p)
	}
	return nil
}

func runRotateCmd(args []string) error {
	fs := flag.NewFlagSet("rotate", flag.ContinueOnError)
	input := fs.String("input", "", "percorso immagine JPG da ruotare")
	angle := fs.Int("angle", 90, "angolo rotazione (90,180,270)")
	jpgQuality := fs.Int("jpg-quality", 100, "qualita JPG output (1-100)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *input == "" {
		return fmt.Errorf("flag obbligatoria: --input")
	}

	if err := imageproc.RotateJPEGFile(*input, *angle, *jpgQuality); err != nil {
		return err
	}

	fmt.Printf("ROTATED=%s\n", *input)
	fmt.Printf("ANGLE=%d\n", *angle)
	fmt.Printf("JPG_QUALITY=%d\n", *jpgQuality)
	return nil
}
