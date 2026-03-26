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
$form.Size = New-Object System.Drawing.Size(1080, 760)
$form.MinimumSize = New-Object System.Drawing.Size(980, 700)
$form.Font = New-Object System.Drawing.Font('Segoe UI', 9)

$labelOutput = New-Object System.Windows.Forms.Label
$labelOutput.Text = 'Cartella output'
$labelOutput.Location = New-Object System.Drawing.Point(20, 20)
$labelOutput.Size = New-Object System.Drawing.Size(140, 20)
$form.Controls.Add($labelOutput)

$txtOutput = New-Object System.Windows.Forms.TextBox
$txtOutput.Location = New-Object System.Drawing.Point(20, 45)
$txtOutput.Size = New-Object System.Drawing.Size(760, 25)
$txtOutput.Text = $DefaultOutput
$txtOutput.Anchor = [System.Windows.Forms.AnchorStyles]::Top -bor [System.Windows.Forms.AnchorStyles]::Left -bor [System.Windows.Forms.AnchorStyles]::Right
$form.Controls.Add($txtOutput)

$btnBrowseOutput = New-Object System.Windows.Forms.Button
$btnBrowseOutput.Text = 'Scegli...'
$btnBrowseOutput.Location = New-Object System.Drawing.Point(800, 43)
$btnBrowseOutput.Size = New-Object System.Drawing.Size(120, 30)
$btnBrowseOutput.Anchor = [System.Windows.Forms.AnchorStyles]::Top -bor [System.Windows.Forms.AnchorStyles]::Right
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
$txtScan.Anchor = [System.Windows.Forms.AnchorStyles]::Top -bor [System.Windows.Forms.AnchorStyles]::Left -bor [System.Windows.Forms.AnchorStyles]::Right
$form.Controls.Add($txtScan)

$btnBrowseScan = New-Object System.Windows.Forms.Button
$btnBrowseScan.Text = 'Apri file...'
$btnBrowseScan.Location = New-Object System.Drawing.Point(800, 113)
$btnBrowseScan.Size = New-Object System.Drawing.Size(120, 30)
$btnBrowseScan.Anchor = [System.Windows.Forms.AnchorStyles]::Top -bor [System.Windows.Forms.AnchorStyles]::Right
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

$labelScanFormat = New-Object System.Windows.Forms.Label
$labelScanFormat.Text = 'Formato scansione'
$labelScanFormat.Location = New-Object System.Drawing.Point(760, 150)
$labelScanFormat.Size = New-Object System.Drawing.Size(140, 20)
$labelScanFormat.Anchor = [System.Windows.Forms.AnchorStyles]::Top -bor [System.Windows.Forms.AnchorStyles]::Right
$form.Controls.Add($labelScanFormat)

$cmbScanFormat = New-Object System.Windows.Forms.ComboBox
$cmbScanFormat.Location = New-Object System.Drawing.Point(760, 170)
$cmbScanFormat.Size = New-Object System.Drawing.Size(160, 25)
$cmbScanFormat.DropDownStyle = [System.Windows.Forms.ComboBoxStyle]::DropDownList
[void]$cmbScanFormat.Items.Add('JPEG')
[void]$cmbScanFormat.Items.Add('TIFF')
$cmbScanFormat.SelectedIndex = 0
$cmbScanFormat.Anchor = [System.Windows.Forms.AnchorStyles]::Top -bor [System.Windows.Forms.AnchorStyles]::Right
$form.Controls.Add($cmbScanFormat)

$chkAutoRotateCrops = New-Object System.Windows.Forms.CheckBox
$chkAutoRotateCrops.Text = 'Ruota automaticamente i crop di 90° a destra'
$chkAutoRotateCrops.Location = New-Object System.Drawing.Point(20, 176)
$chkAutoRotateCrops.Size = New-Object System.Drawing.Size(340, 20)
$chkAutoRotateCrops.Checked = $true
$form.Controls.Add($chkAutoRotateCrops)

$chkAddBorder = New-Object System.Windows.Forms.CheckBox
$chkAddBorder.Text = 'Aggiungi bordo bianco alla scansione'
$chkAddBorder.Location = New-Object System.Drawing.Point(380, 176)
$chkAddBorder.Size = New-Object System.Drawing.Size(300, 20)
$chkAddBorder.Checked = $true
$form.Controls.Add($chkAddBorder)

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
$txtLog.Size = New-Object System.Drawing.Size(430, 360)
$txtLog.Font = New-Object System.Drawing.Font('Consolas', 9)
$form.Controls.Add($txtLog)

$labelPreview = New-Object System.Windows.Forms.Label
$labelPreview.Text = 'Anteprime foto croppate'
$labelPreview.Location = New-Object System.Drawing.Point(480, 245)
$labelPreview.Size = New-Object System.Drawing.Size(220, 20)
$form.Controls.Add($labelPreview)

$previewBoxes = @()
$previewLabels = @()
$rotateButtons = @()
$photoPaths = @('','','','')

for ($i = 0; $i -lt 4; $i++) {
  $row = [int][Math]::Floor($i / 2)
  $col = $i % 2

  $baseX = 480 + ($col * 220)
  $baseY = 270 + ($row * 180)

  $lbl = New-Object System.Windows.Forms.Label
  $lbl.Text = 'Foto ' + ($i + 1)
  $lbl.Location = New-Object System.Drawing.Point($baseX, $baseY)
  $lbl.Size = New-Object System.Drawing.Size(70, 20)
  $form.Controls.Add($lbl)
  $previewLabels += $lbl

  $pb = New-Object System.Windows.Forms.PictureBox
  $pb.Location = New-Object System.Drawing.Point($baseX, ($baseY + 20))
  $pb.Size = New-Object System.Drawing.Size(200, 120)
  $pb.BorderStyle = [System.Windows.Forms.BorderStyle]::FixedSingle
  $pb.SizeMode = [System.Windows.Forms.PictureBoxSizeMode]::Zoom
  $form.Controls.Add($pb)
  $previewBoxes += $pb

  $btnRotate = New-Object System.Windows.Forms.Button
  $btnRotate.Text = 'Ruota 90°'
  $btnRotate.Location = New-Object System.Drawing.Point($baseX, ($baseY + 145))
  $btnRotate.Size = New-Object System.Drawing.Size(100, 28)
  $btnRotate.Tag = $i
  $btnRotate.Enabled = $false
  $form.Controls.Add($btnRotate)
  $rotateButtons += $btnRotate
}

function Update-ResponsiveLayout() {
  $clientW = $form.ClientSize.Width
  $clientH = $form.ClientSize.Height

  $browseWidth = 120
  $sideMargin = 20
  $gap = 10

  $btnBrowseOutput.Location = New-Object System.Drawing.Point(($clientW - $sideMargin - $browseWidth), 43)
  $txtOutput.Width = [Math]::Max(320, $btnBrowseOutput.Left - $gap - $txtOutput.Left)

  $btnBrowseScan.Location = New-Object System.Drawing.Point(($clientW - $sideMargin - $browseWidth), 113)
  $txtScan.Width = [Math]::Max(320, $btnBrowseScan.Left - $gap - $txtScan.Left)

  $cmbScanFormat.Location = New-Object System.Drawing.Point(($clientW - $sideMargin - $cmbScanFormat.Width), 170)
  $labelScanFormat.Location = New-Object System.Drawing.Point($cmbScanFormat.Left, 150)

  $contentTop = 270
  $contentBottomMargin = 20
  $contentH = [Math]::Max(220, $clientH - $contentTop - $contentBottomMargin)

  $leftPaneW = [Math]::Max(360, [Math]::Floor($clientW * 0.42))
  $txtLog.SetBounds(20, $contentTop, $leftPaneW, $contentH)
  $labelLog.Location = New-Object System.Drawing.Point(20, ($contentTop - 25))

  $previewStartX = $txtLog.Right + 24
  $previewRightMargin = 20
  $previewW = [Math]::Max(300, $clientW - $previewStartX - $previewRightMargin)
  $previewH = $contentH
  $labelPreview.Location = New-Object System.Drawing.Point($previewStartX, ($contentTop - 25))

  $gridGap = 14
  $cellW = [Math]::Max(130, [Math]::Floor(($previewW - $gridGap) / 2))
  $cellH = [Math]::Max(140, [Math]::Floor(($previewH - $gridGap) / 2))

  for ($i = 0; $i -lt 4; $i++) {
    $row = [int][Math]::Floor($i / 2)
    $col = $i % 2

    $x = $previewStartX + ($col * ($cellW + $gridGap))
    $y = $contentTop + ($row * ($cellH + $gridGap))

    $imgHeight = [Math]::Max(84, $cellH - 52)
    $btnY = $y + $cellH - 28

    $previewLabels[$i].SetBounds($x, $y, 90, 18)
    $previewBoxes[$i].SetBounds($x, ($y + 18), $cellW, $imgHeight)
    $rotateButtons[$i].SetBounds($x, $btnY, [Math]::Min(120, $cellW), 26)
  }
}

$form.Add_Shown({
  Update-ResponsiveLayout
})

$form.Add_Resize({
  Update-ResponsiveLayout
})

function Append-Log([string]$msg) {
  $timestamp = (Get-Date).ToString('HH:mm:ss')
  $txtLog.AppendText("[$timestamp] $msg" + [Environment]::NewLine)
}

function To-OnOff([bool]$value) {
  if ($value) {
    return 'ON'
  }
  return 'OFF'
}

function Clear-PreviewImage([System.Windows.Forms.PictureBox]$pb) {
  if ($null -ne $pb.Image) {
    $img = $pb.Image
    $pb.Image = $null
    $img.Dispose()
  }
}

function Set-Preview([int]$index, [string]$path) {
  if ($index -lt 0 -or $index -ge $previewBoxes.Count) {
    return
  }

  $pb = $previewBoxes[$index]
  Clear-PreviewImage $pb

  if ([string]::IsNullOrWhiteSpace($path) -or -not (Test-Path $path)) {
    $photoPaths[$index] = ''
    $rotateButtons[$index].Enabled = $false
    return
  }

  try {
    $bytes = [System.IO.File]::ReadAllBytes($path)
    $ms = New-Object System.IO.MemoryStream(, $bytes)
    $img = [System.Drawing.Image]::FromStream($ms)
    $bmp = New-Object System.Drawing.Bitmap($img)
    $img.Dispose()
    $ms.Dispose()
    $pb.Image = $bmp
    $photoPaths[$index] = $path
    $rotateButtons[$index].Enabled = $true
  } catch {
    Append-Log ('Errore caricamento anteprima: ' + $_.Exception.Message)
    $photoPaths[$index] = ''
    $rotateButtons[$index].Enabled = $false
  }
}

function Update-PreviewsFromOutput([string]$stdout) {
  $paths = @()
  foreach ($rawLine in ([System.Text.RegularExpressions.Regex]::Split($stdout, "\r?\n"))) {
    $line = $rawLine.Trim()
    if ($line.StartsWith('PHOTO=')) {
      $paths += $line.Substring(6).Trim()
    }
  }

  for ($i = 0; $i -lt 4; $i++) {
    if ($i -lt $paths.Count) {
      Set-Preview $i $paths[$i]
    } else {
      Set-Preview $i ''
    }
  }
}

function Invoke-Backend([string[]]$arguments, [switch]$NoSuccessPopup) {
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

  $ok = ($proc.ExitCode -eq 0)
  if ($proc.ExitCode -ne 0) {
    [System.Windows.Forms.MessageBox]::Show("Operazione fallita. Controlla il log.", "Errore", 'OK', 'Error') | Out-Null
    return @{ Success = $false; Stdout = $stdout; Stderr = $stderr }
  }

  if (-not $NoSuccessPopup) {
    [System.Windows.Forms.MessageBox]::Show("Operazione completata.", "OK", 'OK', 'Information') | Out-Null
  }
  return @{ Success = $ok; Stdout = $stdout; Stderr = $stderr }
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
  $scanFormatValue = $cmbScanFormat.SelectedItem.ToString().ToLower()
  $scanFormatArg = '--scan-format ' + $scanFormatValue
  $jpgQualityArg = '--jpg-quality ' + [int]$numJpgQuality.Value
  $autoRotateArg = '--auto-rotate-crops=' + $chkAutoRotateCrops.Checked.ToString().ToLower()
  $addBorderArg = '--add-border=true'
  Append-Log 'Avvio scansione...'
  Append-Log ('Formato scansione: ' + $cmbScanFormat.SelectedItem)
  Append-Log ('Preprocessing: AddBorder=ON (forzato su scansione), AutoRotateCrops=' + (To-OnOff $chkAutoRotateCrops.Checked))
  Append-Log ('Qualita scanner: DPI=' + [int]$numDpi.Value + ', Brightness=' + [int]$numBrightness.Value + ', Contrast=' + [int]$numContrast.Value + ', JPG Quality=' + [int]$numJpgQuality.Value + ', AutoRotateCrops=' + $chkAutoRotateCrops.Checked + ', AddBorder=true')
  $res = Invoke-Backend @('scan-process', $outArg, $scanFormatArg, $dpiArg, $brightnessArg, $contrastArg, $jpgQualityArg, $autoRotateArg, $addBorderArg)
  if ($res.Success) {
    Update-PreviewsFromOutput $res.Stdout
  }
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
  $autoRotateArg = '--auto-rotate-crops=' + $chkAutoRotateCrops.Checked.ToString().ToLower()
  $addBorderArg = '--add-border=' + $chkAddBorder.Checked.ToString().ToLower()
  Append-Log 'Avvio elaborazione file...'
  Append-Log ('Preprocessing: AddBorder=' + (To-OnOff $chkAddBorder.Checked) + ', AutoRotateCrops=' + (To-OnOff $chkAutoRotateCrops.Checked))
  Append-Log ('Qualita output JPG: ' + [int]$numJpgQuality.Value + ', AutoRotateCrops=' + $chkAutoRotateCrops.Checked + ', AddBorder=' + $chkAddBorder.Checked)
  $res = Invoke-Backend @('process', $inArg, $outArg, $jpgQualityArg, $autoRotateArg, $addBorderArg)
  if ($res.Success) {
    Update-PreviewsFromOutput $res.Stdout
  }
})

$btnOpenOutput.Add_Click({
  if (-not (Test-Path $txtOutput.Text)) {
    New-Item -ItemType Directory -Path $txtOutput.Text | Out-Null
  }
  Start-Process explorer.exe $txtOutput.Text
})

for ($i = 0; $i -lt $rotateButtons.Count; $i++) {
  $btn = $rotateButtons[$i]
  $btn.Add_Click({
    $idx = [int]$this.Tag
    $path = $photoPaths[$idx]
    if ([string]::IsNullOrWhiteSpace($path) -or -not (Test-Path $path)) {
      [System.Windows.Forms.MessageBox]::Show('Nessuna foto da ruotare.', 'Attenzione', 'OK', 'Warning') | Out-Null
      return
    }

    $inArg = '--input "' + $path.Replace('"','\"') + '"'
    $angleArg = '--angle 90'
    $jpgQualityArg = '--jpg-quality ' + [int]$numJpgQuality.Value
    Append-Log ('Rotazione foto ' + ($idx + 1) + ' di 90 gradi...')
    $res = Invoke-Backend @('rotate', $inArg, $angleArg, $jpgQualityArg) -NoSuccessPopup
    if ($res.Success) {
      Set-Preview $idx $path
      Append-Log ('Foto ' + ($idx + 1) + ' ruotata con successo.')
    }
  })
}

Append-Log "GUI pronta. Eseguibile backend: $AppExe"
Append-Log ('Preprocessing default: AddBorder=' + (To-OnOff $chkAddBorder.Checked) + ', AutoRotateCrops=' + (To-OnOff $chkAutoRotateCrops.Checked))
[void]$form.ShowDialog()
`
