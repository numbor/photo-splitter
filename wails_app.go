package main

import (
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"photo-splitter-go/internal/imageproc"
)

//go:embed wails_assets/*
var wailsEmbeddedAssets embed.FS

type DesktopApp struct {
	ctx context.Context
}

type ProcessFileRequest struct {
	Input           string `json:"input"`
	Output          string `json:"output"`
	JPGQuality      int    `json:"jpgQuality"`
	AutoRotateCrops bool   `json:"autoRotateCrops"`
	AddBorder       bool   `json:"addBorder"`
	EnhanceCrops    bool   `json:"enhanceCrops"`
}

type RotateRequest struct {
	Input      string `json:"input"`
	Angle      int    `json:"angle"`
	JPGQuality int    `json:"jpgQuality"`
}

type OperationResult struct {
	OutputDir    string   `json:"outputDir"`
	BorderedPath string   `json:"borderedPath,omitempty"`
	Photos       []string `json:"photos"`
	Logs         []string `json:"logs"`
}

func runWailsApp() error {
	assets, err := fs.Sub(wailsEmbeddedAssets, "wails_assets")
	if err != nil {
		return fmt.Errorf("caricamento asset Wails fallito: %w", err)
	}

	app := &DesktopApp{}
	return wails.Run(&options.App{
		Title:             "Photo Splitter Go",
		Width:             1080,
		Height:            760,
		MinWidth:          980,
		MinHeight:         700,
		AssetServer:       &assetserver.Options{Assets: assets},
		OnStartup:         app.startup,
		Bind:              []interface{}{app},
		DisableResize:     false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
	})
}

func (a *DesktopApp) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *DesktopApp) DefaultOutputDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "output"
	}
	return filepath.Join(cwd, "output")
}

func (a *DesktopApp) SelectOutputDir(current string) (string, error) {
	if strings.TrimSpace(current) == "" {
		current = a.DefaultOutputDir()
	}

	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:            "Seleziona cartella output",
		DefaultDirectory: current,
	})
}

func (a *DesktopApp) SelectInputFile(current string) (string, error) {
	defaultDir := current
	if strings.TrimSpace(defaultDir) == "" {
		cwd, err := os.Getwd()
		if err == nil {
			defaultDir = cwd
		}
	}

	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:            "Seleziona immagine da elaborare",
		DefaultDirectory: defaultDir,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Immagini", Pattern: "*.jpg;*.jpeg;*.png;*.bmp;*.tif;*.tiff"},
			{DisplayName: "Tutti i file", Pattern: "*.*"},
		},
	})
}

func (a *DesktopApp) ProcessFile(req ProcessFileRequest) (OperationResult, error) {
	inputPath := strings.TrimSpace(req.Input)
	if inputPath == "" {
		return OperationResult{}, fmt.Errorf("seleziona un file immagine")
	}

	outputDir := strings.TrimSpace(req.Output)
	if outputDir == "" {
		outputDir = a.DefaultOutputDir()
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return OperationResult{}, fmt.Errorf("creazione cartella output: %w", err)
	}

	targetDir := filepath.Join(outputDir, time.Now().Format("20060102_150405"))
	procResult, err := imageproc.ProcessTo4PhotosWithOptions(inputPath, targetDir, imageproc.Options{
		JPEGQuality:     req.JPGQuality,
		AutoRotateCrops: req.AutoRotateCrops,
		SkipWhiteBorder: !req.AddBorder,
		SkipEnhancement: !req.EnhanceCrops,
	})
	if err != nil {
		return OperationResult{}, err
	}

	logs := []string{
		"Elaborazione file completata",
		"Input: " + inputPath,
		fmt.Sprintf("Qualità JPG output: %d", req.JPGQuality),
		"Auto rotate crops: " + boolToOnOff(req.AutoRotateCrops),
		"Enhance crops: " + boolToOnOff(req.EnhanceCrops),
		"Add border: " + boolToOnOff(req.AddBorder),
	}

	return OperationResult{
		OutputDir:    targetDir,
		BorderedPath: procResult.BorderedImage,
		Photos:       procResult.Crops,
		Logs:         logs,
	}, nil
}

func (a *DesktopApp) RotatePhoto(req RotateRequest) (string, error) {
	inputPath := strings.TrimSpace(req.Input)
	if inputPath == "" {
		return "", fmt.Errorf("nessuna foto da ruotare")
	}
	if req.Angle == 0 {
		req.Angle = 90
	}

	if err := imageproc.RotateJPEGFile(inputPath, req.Angle, req.JPGQuality); err != nil {
		return "", err
	}

	return inputPath, nil
}

func (a *DesktopApp) GetPhotoPreviewDataURL(path string) (string, error) {
	photoPath := strings.TrimSpace(path)
	if photoPath == "" {
		return "", nil
	}

	bytes, err := os.ReadFile(photoPath)
	if err != nil {
		return "", fmt.Errorf("lettura anteprima foto: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(bytes)
	return "data:image/jpeg;base64," + encoded, nil
}

func (a *DesktopApp) OpenOutputFolder(path string) error {
	folder := strings.TrimSpace(path)
	if folder == "" {
		folder = a.DefaultOutputDir()
	}
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return fmt.Errorf("creazione cartella output: %w", err)
	}

	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer.exe", folder).Start()
	case "darwin":
		return exec.Command("open", folder).Start()
	default:
		return exec.Command("xdg-open", folder).Start()
	}
}

func boolToOnOff(value bool) string {
	if value {
		return "ON"
	}
	return "OFF"
}
