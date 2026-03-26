@echo off
setlocal

cd /d "%~dp0"

echo [1/4] Verifica Go...
go version >nul 2>&1
if errorlevel 1 (
  echo ERRORE: Go non trovato nel PATH.
  echo Installa Go e riapri il terminale.
  pause
  exit /b 1
)

echo [2/4] Download dipendenze...
go mod tidy
if errorlevel 1 (
  echo ERRORE: go mod tidy fallito.
  pause
  exit /b 1
)

echo [3/4] Build applicazione...
go build -ldflags "-H windowsgui" -o photo-splitter.exe .
if errorlevel 1 (
  echo ERRORE: build fallita.
  pause
  exit /b 1
)

echo [4/4] Avvio applicazione...
start "Photo Splitter" "%cd%\photo-splitter.exe"

echo Completato.
exit /b 0
