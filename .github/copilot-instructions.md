# Project Guidelines

## Scope
- These instructions apply to the whole workspace.
- Keep changes minimal and aligned with the existing Go + Wails architecture.
- Prefer updating existing files over adding new files unless necessary.

## Build And Run
- Preferred Windows command: .\\build-run.bat
- Manual build:
  - go mod tidy
  - go build -tags production -ldflags "-H windowsgui" -o photo-splitter.exe .
- Basic validation after code changes:
  - go build ./...
- Optional tests (when present):
  - go test ./...

## Architecture
- Entry and CLI routing: main.go
- Desktop backend bindings (Wails): wails_app.go
- Frontend UI (embedded): wails_assets/index.html
- Image pipeline and crop logic: internal/imageproc/pipeline.go
- JPEG rotation utility: internal/imageproc/rotate.go
- Platform-specific command behavior: cmd_windows.go and cmd_other.go

## Conventions
- Use Italian for user-facing logs and labels in the UI.
- Keep API and JSON field names stable unless the task requires a breaking change.
- Default output directory is data/output.
- Scanner raw images are stored in data/raw_scans and should remain independent from output selection.
- Preserve the current output modes:
  - Subfolder mode: timestamp subdirectory under output.
  - Flat mode: sequential photo naming in output root.
- Do not change NAPS2 path resolution order unless explicitly requested:
  - NAPS2_CONSOLE_PATH env var first, then project fallback path.

## Pitfalls
- NAPS2 scanning is Windows-only; avoid introducing non-Windows assumptions into scan flow.
- Process and rotate CLI outputs are parsed by tools/UI; keep parse-friendly KEY=VALUE lines stable.
- When touching output naming logic, avoid accidental overwrite in flat sequential mode.

## Docs
- Use README.md as the primary project documentation source.
- Link to README.md for usage details instead of duplicating long walkthroughs in new docs.
