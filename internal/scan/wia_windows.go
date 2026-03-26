//go:build windows

package scan

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

func AcquireScanJPEG(outputPath string) error {
	return acquireScanWithFormat(outputPath, Options{}, "{B96B3CAE-0728-11D3-9D7B-0000F81EF32E}")
}

func AcquireScanJPEGWithOptions(outputPath string, options Options) error {
	return acquireScanWithFormat(outputPath, options, "{B96B3CAE-0728-11D3-9D7B-0000F81EF32E}")
}

func AcquireScanTIFFWithOptions(outputPath string, options Options) error {
	return acquireScanWithFormat(outputPath, options, "{B96B3CB1-0728-11D3-9D7B-0000F81EF32E}")
}

func acquireScanWithFormat(outputPath string, options Options, formatGUID string) error {
	opt := options.normalized()
	escapedPath := strings.ReplaceAll(outputPath, "'", "''")
	script := fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$manager = New-Object -ComObject WIA.DeviceManager
if ($manager.DeviceInfos.Count -lt 1) {
  throw 'Nessuno scanner rilevato via WIA.'
}
$device = $null
$maxRetry = 5
for ($attempt = 1; $attempt -le $maxRetry; $attempt++) {
  try {
    $device = $manager.DeviceInfos.Item(1).Connect()
    break
  } catch {
    if ($attempt -eq $maxRetry) {
      throw ('Impossibile connettersi allo scanner dopo ' + $maxRetry + ' tentativi: ' + $_.Exception.Message)
    }
    Start-Sleep -Seconds 3
  }
}
if ($device.Items.Count -lt 1) {
  throw 'Lo scanner non espone item acquisibili.'
}
$item = $device.Items.Item(1)

function Get-WiaProperty($wiaItem, $propertyId) {
  return $wiaItem.Properties | Where-Object { $_.PropertyID -eq $propertyId } | Select-Object -First 1
}

function Set-WiaProperty($wiaItem, $propertyId, $value) {
  $prop = Get-WiaProperty $wiaItem $propertyId
  if ($null -eq $prop) {
    return $null
  }
  try {
    $prop.Value = $value
  } catch {
  }
  return $prop
}

function Resolve-SupportedDpi($wiaItem, $preferredDpi) {
  $xres = Get-WiaProperty $wiaItem 6147
  if ($null -eq $xres) {
    return $preferredDpi
  }

  $values = @()
  try {
    if ($xres.SubType -eq 2 -and $xres.SubTypeValues) {
      $values = @($xres.SubTypeValues | ForEach-Object { [int]$_ })
    }
  } catch {
  }

  if ($values.Count -eq 0) {
    return $preferredDpi
  }

  $sorted = $values | Sort-Object
  $eligible = $sorted | Where-Object { $_ -le $preferredDpi }
  if ($eligible.Count -gt 0) {
    return [int]($eligible | Select-Object -Last 1)
  }

  return [int]($sorted | Select-Object -First 1)
}

function Get-WiaPropertyMax($wiaItem, $propertyId, $fallbackValue) {
  $prop = Get-WiaProperty $wiaItem $propertyId
  if ($null -eq $prop) {
    return $fallbackValue
  }
  try {
    if ($prop.SubType -eq 1 -and $prop.SubTypeMax) {
      return [int]$prop.SubTypeMax
    }
  } catch {
}
  return [int]$prop.Value
}

$effectiveDpi = Resolve-SupportedDpi $item %d

# 6147: Horizontal Resolution (DPI)
# 6148: Vertical Resolution (DPI)
# 6149: Horizontal Start Position
# 6150: Vertical Start Position
# 6151: Horizontal Extent
# 6152: Vertical Extent
# 6154: Brightness
# 6155: Contrast
Set-WiaProperty $item 6147 $effectiveDpi | Out-Null
Set-WiaProperty $item 6148 $effectiveDpi | Out-Null
Set-WiaProperty $item 6149 0 | Out-Null
Set-WiaProperty $item 6150 0 | Out-Null

$maxHorizontalExtent = Get-WiaPropertyMax $item 6151 0
$maxVerticalExtent = Get-WiaPropertyMax $item 6152 0
if ($maxHorizontalExtent -gt 0) {
  Set-WiaProperty $item 6151 $maxHorizontalExtent | Out-Null
}
if ($maxVerticalExtent -gt 0) {
  Set-WiaProperty $item 6152 $maxVerticalExtent | Out-Null
}

Set-WiaProperty $item 6154 %d | Out-Null
Set-WiaProperty $item 6155 %d | Out-Null

$formatGuid = '%s'
$image = $item.Transfer($formatGuid)
$image.SaveFile('%s')
`, opt.DPI, opt.Brightness, opt.Contrast, formatGUID, escapedPath)

	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scansione fallita: %w: %s", err, string(out))
	}
	return nil
}
