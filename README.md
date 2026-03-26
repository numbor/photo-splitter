# Photo Splitter Go

Applicativo in Go con interfaccia grafica desktop basata su Wails per:
- acquisire una scansione da scanner (Windows/WIA),
- aggiungere un leggero bordo bianco in modo nativo,
- individuare 4 fotografie presenti nello scan,
- fare crop e salvare 4 immagini separate.

## Requisiti

- Windows (per scansione via WIA)
- Go 1.22+
- Scanner compatibile WIA
- Microsoft Edge WebView2 Runtime

Nessun uso di ImageMagick.
Nessuna dipendenza da gcc/MinGW.

## Avvio rapido

Dalla cartella `photo-splitter-go`:

```powershell
.\build-run.bat
```

In alternativa:

```powershell
go mod tidy
go build -tags production -ldflags "-H windowsgui" -o photo-splitter.exe .
.\photo-splitter.exe
```

Quando parte, il programma apre una finestra desktop Wails.

## Uso GUI

Nella finestra puoi scegliere questi flussi:

1. **Scansiona da scanner**
   - usa WIA e poi elabora automaticamente.
   - usa di default scansione `TIFF` per qualità più alta; `JPEG` resta disponibile se vuoi più velocità.
   - applica sempre il bordo bianco alla scansione appena acquisita (migliora il rilevamento delle 4 foto).
   - puoi impostare qualità scanner: `DPI`, `Brightness`, `Contrast`.
   - puoi scegliere il formato scansione (`JPEG` o `TIFF`), default `TIFF` per qualità maggiore.
   - puoi impostare la qualità di salvataggio dei JPG output (`JPG Quality`).
   - puoi attivare/disattivare la rotazione automatica dei crop (`Ruota automaticamente i crop di 90° a destra`).
   - puoi attivare/disattivare il miglioramento automatico dei crop (`Migliora automaticamente i crop`).
   - il toggle bordo nella GUI si applica al flusso **Elabora file selezionato**.
   - al termine mostra le anteprime delle 4 foto croppate.
   - ogni anteprima ha pulsante di rotazione (`Ruota 90°`).
2. **Elabora file selezionato**
   - scegli una scansione già presente su disco e avvia il crop.
3. **Apri cartella output**
   - apre direttamente la cartella risultati in Esplora File.

## Output

Nella cartella output viene creato un sottofolder timestamp con:
- `scan_bordered.png`
- `photo_1.jpg`
- `photo_2.jpg`
- `photo_3.jpg`
- `photo_4.jpg`

In output vengono inoltre mantenute:
- `raw_scans` (scansioni da scanner)

## Note tecniche

- Il bordo bianco è aggiunto tramite `image/draw` (standard library Go).
- Formati input supportati nativamente: JPEG, PNG, BMP, TIFF, GIF.
- Il rilevamento delle 4 foto usa proiezioni orizzontali/verticali e ricerca della valle centrale.
- Se il rilevamento è incerto, viene usato un fallback in 4 quadranti.
- Su ogni foto croppata può essere applicato un miglioramento qualità (auto-level, auto-gamma, saturazione +15%, sharpen leggero), attivo di default.
- Su ogni foto croppata viene rimosso automaticamente il bordo bianco residuo ai margini.
- In `scan-process`, il bordo bianco è sempre attivo per migliorare il riconoscimento; in `process` è configurabile (`--add-border=true|false`).
- In `scan-process`, il formato di scansione è configurabile con `--scan-format=jpeg|tiff` (default `tiff` per qualità maggiore).

## CLI tecnica (usata internamente dalla GUI)

 `photo-splitter.exe scan-process --output "D:\\scan\\project\\photo-splitter-go\\output" --dpi 600 --brightness 0 --contrast 0`
 `photo-splitter.exe scan-process --output "D:\\scan\\project\\photo-splitter-go\\output" --scan-format jpeg --dpi 600 --brightness 0 --contrast 0 --jpg-quality 100 --auto-rotate-crops=true --enhance-crops=true`
 `photo-splitter.exe scan-process --output "D:\\scan\\project\\photo-splitter-go\\output" --scan-format tiff --dpi 600 --brightness 0 --contrast 0 --jpg-quality 100 --auto-rotate-crops=true --enhance-crops=true`
 `photo-splitter.exe process --input "D:\\path\\scan.tiff" --output "D:\\scan\\project\\photo-splitter-go\\output" --jpg-quality 100 --auto-rotate-crops=false --add-border=false --enhance-crops=true`
 `photo-splitter.exe rotate --input "D:\\scan\\project\\photo-splitter-go\\output\\20260326_123000\\photo_1.jpg" --angle 90 --jpg-quality 100`
