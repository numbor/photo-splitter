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
	"strconv"
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

type ScanRequest struct {
	Output string `json:"output"`
	DPI    int    `json:"dpi"`
	Device string `json:"device"`
}

type DeviceListResult struct {
	Devices []string `json:"devices"`
	Raw     string   `json:"raw"`
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

func (a *DesktopApp) ScanPhotoTWAIN(req ScanRequest) (string, error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("la scansione TWAIN e supportata solo su Windows")
	}

	outputDir := strings.TrimSpace(req.Output)
	if outputDir == "" {
		outputDir = a.DefaultOutputDir()
	}
	rawScansDir := filepath.Join(outputDir, "raw_scans")
	if err := os.MkdirAll(rawScansDir, 0o755); err != nil {
		return "", fmt.Errorf("creazione cartella raw_scans: %w", err)
	}

	dpi := req.DPI
	if dpi <= 0 {
		dpi = 300
	}

	scanPath := filepath.Join(rawScansDir, "scan_"+time.Now().Format("20060102_150405")+".jpg")
	naps2Path, args := a.buildTWAINScanCommand(req, scanPath)

	cmd := exec.Command(naps2Path, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("acquisizione TWAIN fallita: %s", msg)
	}

	if _, err := os.Stat(scanPath); err != nil {
		return "", fmt.Errorf("scansione completata ma file non trovato: %s", scanPath)
	}

	return scanPath, nil
}

func (a *DesktopApp) PreviewTWAINScanCommand(req ScanRequest) string {
	outputDir := strings.TrimSpace(req.Output)
	if outputDir == "" {
		outputDir = a.DefaultOutputDir()
	}
	dpi := req.DPI
	if dpi <= 0 {
		dpi = 300
	}

	rawScansDir := filepath.Join(outputDir, "raw_scans")
	scanPath := filepath.Join(rawScansDir, "scan_<timestamp>.jpg")
	naps2Path, args := a.buildTWAINScanCommand(ScanRequest{DPI: dpi, Device: req.Device}, scanPath)
	return commandLinePreview(naps2Path, args)
}

func (a *DesktopApp) PreviewListTWAINDevicesCommand() string {
	naps2Path := a.resolveNAPS2ConsolePath()
	args := []string{"--listdevices", "--driver", "twain"}
	return commandLinePreview(naps2Path, args)
}

func (a *DesktopApp) ListTWAINDevices() (DeviceListResult, error) {
	if runtime.GOOS != "windows" {
		return DeviceListResult{}, fmt.Errorf("la scansione TWAIN e supportata solo su Windows")
	}

	naps2Path := a.resolveNAPS2ConsolePath()
	cmd := exec.Command(naps2Path, "--listdevices", "--driver", "twain")
	output, err := cmd.CombinedOutput()
	raw := strings.TrimSpace(string(output))
	if err != nil {
		msg := raw
		if msg == "" {
			msg = err.Error()
		}
		return DeviceListResult{}, fmt.Errorf("elenco dispositivi TWAIN fallito: %s", msg)
	}

	lines := strings.Split(raw, "\n")
	devices := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		devices = append(devices, trimmed)
	}

	return DeviceListResult{Devices: devices, Raw: raw}, nil
}

func (a *DesktopApp) resolveNAPS2ConsolePath() string {
	if envPath := strings.TrimSpace(os.Getenv("NAPS2_CONSOLE_PATH")); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		candidates := []string{
			filepath.Join(cwd, "nasp32", "naps2-8.2.1-win-x64", "App", "NAPS2.Console.exe"),
			filepath.Join(cwd, "..", "photo-splitter-go", "nasp32", "naps2-8.2.1-win-x64", "App", "NAPS2.Console.exe"),
			filepath.Join(cwd, "NAPS2.Console.exe"),
		}
		for _, candidate := range candidates {
			if _, statErr := os.Stat(candidate); statErr == nil {
				return candidate
			}
		}
	}

	return "NAPS2.Console.exe"
}

func (a *DesktopApp) buildTWAINScanCommand(req ScanRequest, scanPath string) (string, []string) {
	naps2Path := a.resolveNAPS2ConsolePath()
	dpi := req.DPI
	if dpi <= 0 {
		dpi = 300
	}

	args := []string{
		"-o", scanPath,
		"-f",
		"--driver", "twain",
		"--dpi", strconv.Itoa(dpi),
	}
	device := strings.TrimSpace(req.Device)
	if device != "" {
		args = append(args, "--noprofile", "--device", device)
	}
	return naps2Path, args
}

func commandLinePreview(exe string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, quoteCommandArg(exe))
	for _, a := range args {
		parts = append(parts, quoteCommandArg(a))
	}
	return strings.Join(parts, " ")
}

func quoteCommandArg(value string) string {
	if value == "" {
		return "\"\""
	}
	if !strings.ContainsAny(value, " \t\n\"") {
		return value
	}
	escaped := strings.ReplaceAll(value, "\"", "\\\"")
	return "\"" + escaped + "\""
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
