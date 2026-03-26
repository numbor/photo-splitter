//go:build windows

package scan

import (
	"fmt"
	"os/exec"
	"strings"
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

function Set-WiaProperty($wiaItem, $propertyId, $value) {
  try {
    $prop = $wiaItem.Properties | Where-Object { $_.PropertyID -eq $propertyId }
    if ($null -ne $prop) {
      $prop.Value = $value
    }
  } catch {
  }
}

# 6147: Horizontal Resolution (DPI)
# 6148: Vertical Resolution (DPI)
# 6154: Brightness
# 6155: Contrast
Set-WiaProperty $item 6147 %d
Set-WiaProperty $item 6148 %d
Set-WiaProperty $item 6154 %d
Set-WiaProperty $item 6155 %d

$formatGuid = '%s'
$image = $item.Transfer($formatGuid)
$image.SaveFile('%s')
`, opt.DPI, opt.DPI, opt.Brightness, opt.Contrast, formatGUID, escapedPath)

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scansione fallita: %w: %s", err, string(out))
	}
	return nil
}
