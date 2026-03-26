# Photo Splitter Go

Applicativo in Go (solo librerie native) con interfaccia grafica a finestre di Windows per:
- acquisire una scansione da scanner (Windows/WIA),
- aggiungere un leggero bordo bianco in modo nativo,
- individuare 4 fotografie presenti nello scan,
- fare crop e salvare 4 immagini separate.

## Requisiti

- Windows (per scansione via WIA)
- Go 1.22+
- Scanner compatibile WIA

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
go build -o photo-splitter.exe .
.\photo-splitter.exe
```

Quando parte, il programma apre una finestra Windows (WinForms).

## Uso GUI

Nella finestra puoi scegliere questi flussi:

1. **Scansiona da scanner**
   - usa WIA e poi elabora automaticamente.
   - salva la scansione principale in formato `TIFF` per preservare qualità.
   - puoi impostare qualità scanner: `DPI`, `Brightness`, `Contrast`.
   - puoi impostare la qualità di salvataggio dei JPG output (`JPG Quality`).
2. **Elabora file selezionato**
   - scegli una scansione già presente su disco e avvia il crop.
3. **Apri cartella output**
   - apre direttamente la cartella risultati in Esplora File.

## Output

Nella cartella output viene creato un sottofolder timestamp con:
- `scan_bordered.jpg`
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

## CLI tecnica (usata internamente dalla GUI)

- `photo-splitter.exe scan-process --output "D:\\scan\\project\\photo-splitter-go\\output" --dpi 300 --brightness 0 --contrast 0`
- `photo-splitter.exe scan-process --output "D:\\scan\\project\\photo-splitter-go\\output" --dpi 300 --brightness 0 --contrast 0 --jpg-quality 95`
- `photo-splitter.exe process --input "D:\\path\\scan.tiff" --output "D:\\scan\\project\\photo-splitter-go\\output" --jpg-quality 90`
