package main

import (
	"archive/zip"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	UseSubfolders   *bool  `json:"useSubfolders"`
}

type RotateRequest struct {
	Input      string `json:"input"`
	Angle      int    `json:"angle"`
	JPGQuality int    `json:"jpgQuality"`
}

type ScanRequest struct {
	Output  string `json:"output"`
	Profile string `json:"profile"`
}

type NAPS2Profile struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	DeviceName  string `json:"deviceName"`
	IsDefault   bool   `json:"isDefault"`
}

type scanProfilesXML struct {
	Profiles []scanProfileXML `xml:"ScanProfile"`
}

type scanProfileXML struct {
	DisplayName string `xml:"DisplayName"`
	IsDefault   bool   `xml:"IsDefault"`
	Device      struct {
		ID   string `xml:"ID"`
		Name string `xml:"Name"`
	} `xml:"Device"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
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
		Width:             1280,
		Height:            900,
		MinWidth:          1100,
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

func (a *DesktopApp) defaultDataDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "data"
	}
	return filepath.Join(cwd, "data")
}

func (a *DesktopApp) DefaultOutputDir() string {
	return filepath.Join(a.defaultDataDir(), "output")
}

func (a *DesktopApp) defaultRawScansDir() string {
	return filepath.Join(a.defaultDataDir(), "raw_scans")
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

	useSubfolders := true
	if req.UseSubfolders != nil {
		useSubfolders = *req.UseSubfolders
	}

	targetDir := outputDir
	startIndex := 1
	if useSubfolders {
		targetDir = filepath.Join(outputDir, time.Now().Format("20060102_150405"))
	} else {
		seqStart, seqErr := nextSequentialPhotoIndex(outputDir)
		if seqErr != nil {
			return OperationResult{}, seqErr
		}
		startIndex = seqStart
	}

	procResult, err := imageproc.ProcessTo4PhotosWithOptions(inputPath, targetDir, imageproc.Options{
		JPEGQuality:      req.JPGQuality,
		AutoRotateCrops:  req.AutoRotateCrops,
		SkipWhiteBorder:  !req.AddBorder,
		SkipEnhancement:  !req.EnhanceCrops,
		SequentialNaming: !useSubfolders,
		StartIndex:       startIndex,
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
		"Use output subfolders: " + boolToOnOff(useSubfolders),
	}

	return OperationResult{
		OutputDir:    targetDir,
		BorderedPath: procResult.BorderedImage,
		Photos:       procResult.Crops,
		Logs:         logs,
	}, nil
}

func nextSequentialPhotoIndex(outputDir string) (int, error) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return 0, fmt.Errorf("lettura cartella output: %w", err)
	}

	pattern := regexp.MustCompile(`(?i)^photo_(\d+)\.jpe?g$`)
	maxFound := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		match := pattern.FindStringSubmatch(entry.Name())
		if len(match) != 2 {
			continue
		}
		n, convErr := strconv.Atoi(match[1])
		if convErr != nil {
			continue
		}
		if n > maxFound {
			maxFound = n
		}
	}

	return maxFound + 1, nil
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

func (a *DesktopApp) ScanWithNAPS2(req ScanRequest) (string, error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("la scansione NAPS2 e supportata solo su Windows")
	}
	if strings.TrimSpace(req.Profile) == "" {
		return "", fmt.Errorf("seleziona un profilo scanner")
	}

	rawScansDir := a.defaultRawScansDir()
	if err := os.MkdirAll(rawScansDir, 0o755); err != nil {
		return "", fmt.Errorf("creazione cartella raw_scans: %w", err)
	}

	scanPath := filepath.Join(rawScansDir, "scan_"+time.Now().Format("20060102_150405")+".jpg")
	naps2Path, args := a.buildNAPS2ScanCommand(scanPath, req.Profile)

	cmd := exec.Command(naps2Path, args...)
	hideExternalConsoleWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("acquisizione NAPS2 fallita: %s", msg)
	}

	if _, err := os.Stat(scanPath); err != nil {
		return "", fmt.Errorf("scansione completata ma file non trovato: %s", scanPath)
	}

	return scanPath, nil
}

func (a *DesktopApp) PreviewNAPS2ScanCommand(req ScanRequest) string {
	rawScansDir := a.defaultRawScansDir()
	scanPath := filepath.Join(rawScansDir, "scan_<timestamp>.jpg")
	naps2Path, args := a.buildNAPS2ScanCommand(scanPath, req.Profile)
	return commandLinePreview(naps2Path, args)
}

func (a *DesktopApp) ListNAPS2Profiles() ([]NAPS2Profile, error) {
	profilesPath := a.resolveNAPS2ProfilesPath()
	if profilesPath == "" {
		return nil, fmt.Errorf("file profiles.xml non trovato")
	}

	xmlBytes, err := os.ReadFile(profilesPath)
	if err != nil {
		return nil, fmt.Errorf("lettura profiles.xml fallita: %w", err)
	}

	var parsed scanProfilesXML
	if err := xml.Unmarshal(xmlBytes, &parsed); err != nil {
		return nil, fmt.Errorf("parsing profiles.xml fallito: %w", err)
	}

	profiles := make([]NAPS2Profile, 0, len(parsed.Profiles))
	for _, p := range parsed.Profiles {
		displayName := strings.TrimSpace(p.DisplayName)
		if displayName == "" {
			displayName = strings.TrimSpace(p.Device.Name)
		}
		if displayName == "" {
			displayName = strings.TrimSpace(p.Device.ID)
		}
		if displayName == "" {
			continue
		}

		profiles = append(profiles, NAPS2Profile{
			ID:          strings.TrimSpace(p.Device.ID),
			DisplayName: displayName,
			DeviceName:  strings.TrimSpace(p.Device.Name),
			IsDefault:   p.IsDefault,
		})
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("nessun profilo scanner trovato in profiles.xml")
	}

	return profiles, nil
}

func (a *DesktopApp) EnsureNAPS2Portable() (string, error) {
	if runtime.GOOS != "windows" {
		return "Skip bootstrap NAPS2: non Windows.", nil
	}

	if existing := a.resolveNAPS2ConsolePath(); existing != "NAPS2.Console.exe" {
		return "NAPS2 trovato: " + existing, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("lettura working directory fallita: %w", err)
	}

	baseDir := filepath.Join(cwd, "naps2")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", fmt.Errorf("creazione cartella naps2 fallita: %w", err)
	}

	downloadURL := fetchNAPS2PortableURL()
	zipPath := filepath.Join(baseDir, "naps2-portable.zip")
	if err := downloadFile(downloadURL, zipPath); err != nil {
		return "", fmt.Errorf("download NAPS2 portable fallito: %w", err)
	}

	if err := unzipArchive(zipPath, baseDir); err != nil {
		return "", fmt.Errorf("decompressione NAPS2 portable fallita: %w", err)
	}

	if resolved := a.resolveNAPS2ConsolePath(); resolved != "NAPS2.Console.exe" {
		return "NAPS2 scaricato e pronto: " + resolved, nil
	}

	return "", fmt.Errorf("NAPS2 decompresso ma NAPS2.Console.exe non trovato")
}

func (a *DesktopApp) LaunchNAPS2GUI() (string, error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("NAPS2 GUI supportata solo su Windows")
	}

	if _, err := a.EnsureNAPS2Portable(); err != nil {
		return "", err
	}

	guiPath := a.resolveNAPS2GUIPath()
	if guiPath == "" {
		return "", fmt.Errorf("NAPS2 GUI non trovata")
	}

	cmd := exec.Command(guiPath)
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("avvio NAPS2 GUI fallito: %w", err)
	}

	return guiPath, nil
}

func (a *DesktopApp) resolveNAPS2ConsolePath() string {
	if envPath := strings.TrimSpace(os.Getenv("NAPS2_CONSOLE_PATH")); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		candidates := []string{filepath.Join(cwd, "NAPS2.Console.exe")}
		for _, candidate := range candidates {
			if _, statErr := os.Stat(candidate); statErr == nil {
				return candidate
			}
		}

		baseDir := filepath.Join(cwd, "naps2")
		if found := findNAPS2ConsoleUnder(baseDir); found != "" {
			return found
		}
	}

	return "NAPS2.Console.exe"
}

func (a *DesktopApp) resolveNAPS2GUIPath() string {
	if envPath := strings.TrimSpace(os.Getenv("NAPS2_GUI_PATH")); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	if consolePath := a.resolveNAPS2ConsolePath(); consolePath != "NAPS2.Console.exe" {
		baseDir := filepath.Dir(consolePath)
		candidates := []string{
			filepath.Join(baseDir, "NAPS2.Portable.exe"),
			filepath.Join(baseDir, "NAPS2.exe"),
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		fallbacks := []string{
			filepath.Join(cwd, "naps2", "naps2-8.2.1-win-x64", "NAPS2.Portable.exe"),
			filepath.Join(cwd, "naps2", "naps2-8.2.1-win-x64", "App", "NAPS2.exe"),
		}
		for _, candidate := range fallbacks {
			if _, statErr := os.Stat(candidate); statErr == nil {
				return candidate
			}
		}
	}

	return ""
}

func (a *DesktopApp) buildNAPS2ScanCommand(scanPath, profile string) (string, []string) {
	naps2Path := a.resolveNAPS2ConsolePath()
	args := []string{
		"--profile", strings.TrimSpace(profile),
		"-o", scanPath,
		"-f",
	}
	return naps2Path, args
}

func (a *DesktopApp) resolveNAPS2ProfilesPath() string {
	if envPath := strings.TrimSpace(os.Getenv("NAPS2_PROFILES_PATH")); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	if consolePath := a.resolveNAPS2ConsolePath(); consolePath != "NAPS2.Console.exe" {
		appDir := filepath.Dir(consolePath)
		baseDir := filepath.Dir(appDir)
		candidates := []string{
			filepath.Join(baseDir, "Data", "profiles.xml"),
			filepath.Join(appDir, "Data", "profiles.xml"),
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	fallbacks := []string{
		filepath.Join(cwd, "naps2", "Data", "profiles.xml"),
		filepath.Join(cwd, "nasp32", "Data", "profiles.xml"),
		filepath.Join(cwd, "naps2", "naps2-8.2.1-win-x64", "Data", "profiles.xml"),
	}
	for _, candidate := range fallbacks {
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
	}

	return ""
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

func findNAPS2ConsoleUnder(baseDir string) string {
	info, err := os.Stat(baseDir)
	if err != nil || !info.IsDir() {
		return ""
	}

	var found string
	_ = filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d == nil || d.IsDir() {
			return nil
		}
		if strings.EqualFold(d.Name(), "NAPS2.Console.exe") {
			found = path
			return io.EOF
		}
		return nil
	})
	return found
}

func fetchNAPS2PortableURL() string {
	const fallback = "https://github.com/cyanfish/naps2/releases/download/v8.2.1/naps2-8.2.1-win-x64.zip"
	const apiURL = "https://api.github.com/repos/cyanfish/naps2/releases/latest"

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return fallback
	}
	req.Header.Set("User-Agent", "photo-splitter-go")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fallback
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fallback
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return fallback
	}

	for _, asset := range rel.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), "-win-x64.zip") && asset.BrowserDownloadURL != "" {
			return asset.BrowserDownloadURL
		}
	}

	return fallback
}

func downloadFile(url, destination string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "photo-splitter-go")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http status %s", resp.Status)
	}

	tmp := destination + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, resp.Body); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmp, destination); err != nil {
		return err
	}
	return nil
}

func unzipArchive(zipPath, destinationDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(destinationDir, 0o755); err != nil {
		return err
	}

	for _, f := range r.File {
		targetPath := filepath.Join(destinationDir, f.Name)
		cleanDest := filepath.Clean(destinationDir) + string(os.PathSeparator)
		cleanTarget := filepath.Clean(targetPath)
		if !strings.HasPrefix(cleanTarget, cleanDest) {
			return fmt.Errorf("percorso zip non valido: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
			return err
		}

		src, err := f.Open()
		if err != nil {
			return err
		}

		dst, err := os.OpenFile(cleanTarget, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
		if err != nil {
			_ = src.Close()
			return err
		}

		if _, err := io.Copy(dst, src); err != nil {
			_ = dst.Close()
			_ = src.Close()
			return err
		}
		if err := dst.Close(); err != nil {
			_ = src.Close()
			return err
		}
		if err := src.Close(); err != nil {
			return err
		}
	}

	return nil
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

func (a *DesktopApp) OpenImageWithSystemViewer(path string) error {
	imagePath := strings.TrimSpace(path)
	if imagePath == "" {
		return fmt.Errorf("nessuna immagine da aprire")
	}

	info, err := os.Stat(imagePath)
	if err != nil {
		return fmt.Errorf("immagine non trovata: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("il percorso indicato e una cartella, non un file")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", imagePath)
		hideExternalConsoleWindow(cmd)
	case "darwin":
		cmd = exec.Command("open", imagePath)
	default:
		cmd = exec.Command("xdg-open", imagePath)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("apertura visualizzatore immagine fallita: %w", err)
	}

	return nil
}

func boolToOnOff(value bool) string {
	if value {
		return "ON"
	}
	return "OFF"
}
