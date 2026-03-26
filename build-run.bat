@echo off
setlocal

cd /d "%~dp0"

echo [1/5] Verifica Go...
go version >nul 2>&1
if errorlevel 1 (
  echo ERRORE: Go non trovato nel PATH.
  echo Installa Go e riapri il terminale.
  pause
  exit /b 1
)

echo [2/5] Verifica Microsoft Edge WebView2 Runtime...
powershell -NoProfile -ExecutionPolicy Bypass -Command ^
  "$paths=@('HKLM:\SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}','HKLM:\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}','HKCU:\SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}'); $ok=$false; foreach($p in $paths){ try { $v=(Get-ItemProperty -Path $p -Name pv -ErrorAction Stop).pv; if($v){$ok=$true; break} } catch {} }; if(-not $ok){ exit 1 }"
if errorlevel 1 (
  echo ERRORE: Microsoft Edge WebView2 Runtime non trovato.
  echo Installa WebView2 Runtime e riprova:
  echo https://developer.microsoft.com/microsoft-edge/webview2/
  pause
  exit /b 1
)

echo [3/5] Download dipendenze...
go mod tidy
if errorlevel 1 (
  echo ERRORE: go mod tidy fallito.
  pause
  exit /b 1
)

echo [4/5] Build applicazione...
go build -tags production -ldflags "-H windowsgui" -o photo-splitter.exe .
if errorlevel 1 (
  echo ERRORE: build fallita.
  pause
  exit /b 1
)

echo [5/5] Avvio applicazione...
start "Photo Splitter" "%cd%\photo-splitter.exe"

echo Completato.
exit /b 0
