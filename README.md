# Photo Splitter Go

Applicativo in Go con interfaccia grafica desktop basata su Wails per:
- caricare una singola immagine iniziale,
- acquisire una scansione da scanner TWAIN tramite `NAPS2.Console.exe`,
- aggiungere un leggero bordo bianco in modo nativo,
- individuare 4 fotografie presenti nell'immagine,
- fare crop e salvare 4 immagini separate.

## Requisiti

- Go 1.22+
- Microsoft Edge WebView2 Runtime
- NAPS2 portable estratto in `nasp32/naps2-8.2.1-win-x64/App/NAPS2.Console.exe`
   (in alternativa imposta `NAPS2_CONSOLE_PATH`)

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

Nella finestra puoi usare questi flussi:

1. **Elabora file selezionato**
   - scegli un'immagine già presente su disco e avvia il crop.
   - puoi impostare la qualità di salvataggio dei JPG output (`JPG Quality`).
   - puoi attivare/disattivare la rotazione automatica dei crop (`Ruota automaticamente i crop di 90° a destra`).
   - puoi attivare/disattivare il miglioramento automatico dei crop (`Migliora automaticamente i crop`).
   - puoi attivare/disattivare l'aggiunta di bordo bianco (`Aggiungi bordo bianco all'immagine`).
   - al termine mostra le anteprime delle 4 foto croppate.
   - ogni anteprima ha pulsante di rotazione (`Ruota 90°`).
2. **Scansiona via TWAIN**
   - usa `NAPS2.Console.exe` con driver TWAIN.
   - puoi scegliere i DPI (`DPI scansione TWAIN`).
   - con `Carica dispositivi TWAIN` l'app richiama `--listdevices --driver twain` e mostra i device disponibili.
   - puoi selezionare un dispositivo dalla lista, oppure inserirlo manualmente (match parziale del nome).
   - la scansione viene salvata in `output/raw_scans` e caricata automaticamente nel campo input.
3. **Apri cartella output**
   - apre direttamente la cartella risultati in Esplora File.

## Output

Nella cartella output viene creato un sottofolder timestamp con:
- `input_bordered.png`
- `photo_1.jpg`
- `photo_2.jpg`
- `photo_3.jpg`
- `photo_4.jpg`

Le scansioni TWAIN vengono salvate in:
- `output/raw_scans/scan_YYYYMMDD_HHMMSS.jpg`

## Note tecniche

- Il bordo bianco è aggiunto tramite `image/draw` (standard library Go).
- Formati input supportati nativamente: JPEG, PNG, BMP, TIFF, GIF.
- Il rilevamento delle 4 foto usa proiezioni orizzontali/verticali e ricerca della valle centrale.
- Se il rilevamento è incerto, viene usato un fallback in 4 quadranti.
- Su ogni foto croppata può essere applicato un miglioramento qualità (auto-level, auto-gamma, saturazione +15%, sharpen leggero), attivo di default.
- Su ogni foto croppata viene rimosso automaticamente il bordo bianco residuo ai margini.
- Il bordo bianco è configurabile con `--add-border=true|false`.

## CLI tecnica (usata internamente dalla GUI)

 `photo-splitter.exe process --input "D:\\path\\scan.tiff" --output "D:\\scan\\project\\photo-splitter-go\\output" --jpg-quality 100 --auto-rotate-crops=false --add-border=false --enhance-crops=true`
 `photo-splitter.exe rotate --input "D:\\scan\\project\\photo-splitter-go\\output\\20260326_123000\\photo_1.jpg" --angle 90 --jpg-quality 100`
