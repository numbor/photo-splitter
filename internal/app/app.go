package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func Run() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("questa GUI è disponibile solo su Windows")
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("impossibile determinare eseguibile: %w", err)
	}

	cwd, _ := os.Getwd()
	defaultOutput := filepath.Join(cwd, "output")

	tempScript, err := os.CreateTemp("", "photo-splitter-gui-*.ps1")
	if err != nil {
		return fmt.Errorf("creazione script temporaneo fallita: %w", err)
	}
	scriptPath := tempScript.Name()
	defer os.Remove(scriptPath)

	if _, err := tempScript.WriteString(winFormsScript); err != nil {
		_ = tempScript.Close()
		return fmt.Errorf("scrittura script GUI fallita: %w", err)
	}
	if err := tempScript.Close(); err != nil {
		return fmt.Errorf("chiusura script GUI fallita: %w", err)
	}

	cmd := exec.Command(
		"powershell",
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
		"-AppExe", exePath,
		"-DefaultOutput", defaultOutput,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("avvio GUI Windows fallito: %w", err)
	}

	return nil
}

const winFormsScript = `
param(
  [Parameter(Mandatory=$true)][string]$AppExe,
  [Parameter(Mandatory=$true)][string]$DefaultOutput
)

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

[System.Windows.Forms.Application]::EnableVisualStyles()

$form = New-Object System.Windows.Forms.Form
$form.Text = 'Photo Splitter Go'
$form.StartPosition = 'CenterScreen'
$form.Size = New-Object System.Drawing.Size(980, 700)

$labelOutput = New-Object System.Windows.Forms.Label
$labelOutput.Text = 'Cartella output'
$labelOutput.Location = New-Object System.Drawing.Point(20, 20)
$labelOutput.Size = New-Object System.Drawing.Size(140, 20)
$form.Controls.Add($labelOutput)

$txtOutput = New-Object System.Windows.Forms.TextBox
$txtOutput.Location = New-Object System.Drawing.Point(20, 45)
$txtOutput.Size = New-Object System.Drawing.Size(760, 25)
$txtOutput.Text = $DefaultOutput
$form.Controls.Add($txtOutput)

$btnBrowseOutput = New-Object System.Windows.Forms.Button
$btnBrowseOutput.Text = 'Scegli...'
$btnBrowseOutput.Location = New-Object System.Drawing.Point(800, 43)
$btnBrowseOutput.Size = New-Object System.Drawing.Size(120, 30)
$form.Controls.Add($btnBrowseOutput)

$folderDialog = New-Object System.Windows.Forms.FolderBrowserDialog
$btnBrowseOutput.Add_Click({
  if ($folderDialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
    $txtOutput.Text = $folderDialog.SelectedPath
  }
})

$labelScan = New-Object System.Windows.Forms.Label
$labelScan.Text = 'Percorso immagine scannerizzata (opzionale)'
$labelScan.Location = New-Object System.Drawing.Point(20, 90)
$labelScan.Size = New-Object System.Drawing.Size(350, 20)
$form.Controls.Add($labelScan)

$txtScan = New-Object System.Windows.Forms.TextBox
$txtScan.Location = New-Object System.Drawing.Point(20, 115)
$txtScan.Size = New-Object System.Drawing.Size(760, 25)
$form.Controls.Add($txtScan)

$btnBrowseScan = New-Object System.Windows.Forms.Button
$btnBrowseScan.Text = 'Apri file...'
$btnBrowseScan.Location = New-Object System.Drawing.Point(800, 113)
$btnBrowseScan.Size = New-Object System.Drawing.Size(120, 30)
$form.Controls.Add($btnBrowseScan)

$fileDialog = New-Object System.Windows.Forms.OpenFileDialog
$fileDialog.Filter = 'Immagini|*.jpg;*.jpeg;*.png;*.bmp;*.tif;*.tiff|Tutti i file|*.*'
$btnBrowseScan.Add_Click({
  if ($fileDialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
    $txtScan.Text = $fileDialog.FileName
  }
})

$labelDpi = New-Object System.Windows.Forms.Label
$labelDpi.Text = 'DPI'
$labelDpi.Location = New-Object System.Drawing.Point(20, 150)
$labelDpi.Size = New-Object System.Drawing.Size(40, 20)
$form.Controls.Add($labelDpi)

$numDpi = New-Object System.Windows.Forms.NumericUpDown
$numDpi.Location = New-Object System.Drawing.Point(65, 148)
$numDpi.Size = New-Object System.Drawing.Size(90, 25)
$numDpi.Minimum = 75
$numDpi.Maximum = 1200
$numDpi.Value = 300
$numDpi.Increment = 25
$form.Controls.Add($numDpi)

$labelBrightness = New-Object System.Windows.Forms.Label
$labelBrightness.Text = 'Brightness'
$labelBrightness.Location = New-Object System.Drawing.Point(180, 150)
$labelBrightness.Size = New-Object System.Drawing.Size(80, 20)
$form.Controls.Add($labelBrightness)

$numBrightness = New-Object System.Windows.Forms.NumericUpDown
$numBrightness.Location = New-Object System.Drawing.Point(265, 148)
$numBrightness.Size = New-Object System.Drawing.Size(90, 25)
$numBrightness.Minimum = -1000
$numBrightness.Maximum = 1000
$numBrightness.Value = 0
$numBrightness.Increment = 50
$form.Controls.Add($numBrightness)

$labelContrast = New-Object System.Windows.Forms.Label
$labelContrast.Text = 'Contrast'
$labelContrast.Location = New-Object System.Drawing.Point(380, 150)
$labelContrast.Size = New-Object System.Drawing.Size(70, 20)
$form.Controls.Add($labelContrast)

$numContrast = New-Object System.Windows.Forms.NumericUpDown
$numContrast.Location = New-Object System.Drawing.Point(455, 148)
$numContrast.Size = New-Object System.Drawing.Size(90, 25)
$numContrast.Minimum = -1000
$numContrast.Maximum = 1000
$numContrast.Value = 0
$numContrast.Increment = 50
$form.Controls.Add($numContrast)

$labelJpgQuality = New-Object System.Windows.Forms.Label
$labelJpgQuality.Text = 'JPG Quality'
$labelJpgQuality.Location = New-Object System.Drawing.Point(570, 150)
$labelJpgQuality.Size = New-Object System.Drawing.Size(80, 20)
$form.Controls.Add($labelJpgQuality)

$numJpgQuality = New-Object System.Windows.Forms.NumericUpDown
$numJpgQuality.Location = New-Object System.Drawing.Point(655, 148)
$numJpgQuality.Size = New-Object System.Drawing.Size(90, 25)
$numJpgQuality.Minimum = 1
$numJpgQuality.Maximum = 100
$numJpgQuality.Value = 95
$numJpgQuality.Increment = 1
$form.Controls.Add($numJpgQuality)

$btnScanAndSplit = New-Object System.Windows.Forms.Button
$btnScanAndSplit.Text = 'Scansiona e separa 4 foto'
$btnScanAndSplit.Location = New-Object System.Drawing.Point(20, 190)
$btnScanAndSplit.Size = New-Object System.Drawing.Size(240, 36)
$form.Controls.Add($btnScanAndSplit)

$btnProcessPath = New-Object System.Windows.Forms.Button
$btnProcessPath.Text = 'Elabora file selezionato'
$btnProcessPath.Location = New-Object System.Drawing.Point(280, 190)
$btnProcessPath.Size = New-Object System.Drawing.Size(220, 36)
$form.Controls.Add($btnProcessPath)

$btnOpenOutput = New-Object System.Windows.Forms.Button
$btnOpenOutput.Text = 'Apri cartella output'
$btnOpenOutput.Location = New-Object System.Drawing.Point(520, 190)
$btnOpenOutput.Size = New-Object System.Drawing.Size(180, 36)
$form.Controls.Add($btnOpenOutput)

$labelLog = New-Object System.Windows.Forms.Label
$labelLog.Text = 'Log'
$labelLog.Location = New-Object System.Drawing.Point(20, 245)
$labelLog.Size = New-Object System.Drawing.Size(120, 20)
$form.Controls.Add($labelLog)

$txtLog = New-Object System.Windows.Forms.TextBox
$txtLog.Multiline = $true
$txtLog.ScrollBars = 'Vertical'
$txtLog.ReadOnly = $true
$txtLog.Location = New-Object System.Drawing.Point(20, 270)
$txtLog.Size = New-Object System.Drawing.Size(900, 360)
$form.Controls.Add($txtLog)

function Append-Log([string]$msg) {
  $timestamp = (Get-Date).ToString('HH:mm:ss')
  $txtLog.AppendText("[$timestamp] $msg" + [Environment]::NewLine)
}

function Invoke-Backend([string[]]$arguments) {
  $psi = New-Object System.Diagnostics.ProcessStartInfo
  $psi.FileName = $AppExe
  $psi.Arguments = [string]::Join(' ', $arguments)
  $psi.RedirectStandardOutput = $true
  $psi.RedirectStandardError = $true
  $psi.UseShellExecute = $false
  $psi.CreateNoWindow = $true

  $proc = New-Object System.Diagnostics.Process
  $proc.StartInfo = $psi
  [void]$proc.Start()

  $stdout = $proc.StandardOutput.ReadToEnd()
  $stderr = $proc.StandardError.ReadToEnd()
  $proc.WaitForExit()

  if ($stdout) { Append-Log $stdout.TrimEnd() }
  if ($stderr) { Append-Log $stderr.TrimEnd() }

  if ($proc.ExitCode -ne 0) {
    [System.Windows.Forms.MessageBox]::Show("Operazione fallita. Controlla il log.", "Errore", 'OK', 'Error') | Out-Null
    return $false
  }

  [System.Windows.Forms.MessageBox]::Show("Operazione completata.", "OK", 'OK', 'Information') | Out-Null
  return $true
}

$btnScanAndSplit.Add_Click({
  if ([string]::IsNullOrWhiteSpace($txtOutput.Text)) {
    [System.Windows.Forms.MessageBox]::Show('Imposta la cartella output.', 'Attenzione', 'OK', 'Warning') | Out-Null
    return
  }

  $outArg = '--output "' + $txtOutput.Text.Replace('"','\"') + '"'
  $dpiArg = '--dpi ' + [int]$numDpi.Value
  $brightnessArg = '--brightness ' + [int]$numBrightness.Value
  $contrastArg = '--contrast ' + [int]$numContrast.Value
  $jpgQualityArg = '--jpg-quality ' + [int]$numJpgQuality.Value
  Append-Log 'Avvio scansione...'
  Append-Log ('Qualita scanner: DPI=' + [int]$numDpi.Value + ', Brightness=' + [int]$numBrightness.Value + ', Contrast=' + [int]$numContrast.Value + ', JPG Quality=' + [int]$numJpgQuality.Value)
  [void](Invoke-Backend @('scan-process', $outArg, $dpiArg, $brightnessArg, $contrastArg, $jpgQualityArg))
})

$btnProcessPath.Add_Click({
  if ([string]::IsNullOrWhiteSpace($txtOutput.Text)) {
    [System.Windows.Forms.MessageBox]::Show('Imposta la cartella output.', 'Attenzione', 'OK', 'Warning') | Out-Null
    return
  }
  if ([string]::IsNullOrWhiteSpace($txtScan.Text)) {
    [System.Windows.Forms.MessageBox]::Show('Seleziona un file immagine.', 'Attenzione', 'OK', 'Warning') | Out-Null
    return
  }

  $inArg = '--input "' + $txtScan.Text.Replace('"','\"') + '"'
  $outArg = '--output "' + $txtOutput.Text.Replace('"','\"') + '"'
  $jpgQualityArg = '--jpg-quality ' + [int]$numJpgQuality.Value
  Append-Log 'Avvio elaborazione file...'
  Append-Log ('Qualita output JPG: ' + [int]$numJpgQuality.Value)
  [void](Invoke-Backend @('process', $inArg, $outArg, $jpgQualityArg))
})

$btnOpenOutput.Add_Click({
  if (-not (Test-Path $txtOutput.Text)) {
    New-Item -ItemType Directory -Path $txtOutput.Text | Out-Null
  }
  Start-Process explorer.exe $txtOutput.Text
})

Append-Log "GUI pronta. Eseguibile backend: $AppExe"
[void]$form.ShowDialog()
`