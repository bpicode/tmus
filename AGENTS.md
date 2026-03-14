# AI Agent & Repository Guidelines

This document contains high-level, persistent rules for AI agents and contributors working on `tmus`.
**Do not update this file with day-to-day implementation details, minor refactors, or directory lists that are self-evident from the code.**

## Architecture & Boundaries
- **Strict Separation of Concerns**: Keep UI logic (`internal/ui/`) completely isolated from core playback, library, or metadata logic (`internal/app/`). Never import `ui` packages into `app`.
- **UI Framework**: Uses Bubble Tea (`charmbracelet/bubbletea`) for the Terminal UI.
- **Audio Engine**: Uses `github.com/gopxl/beep/v2` (and `oto` underneath) for audio decoding and playback (`internal/app/player/`). Add new formats via dedicated decoders.
- **Archive Support**: Uses `github.com/mholt/archives` to abstract archive file access (`internal/app/archive/`).
- **CLI & Config**: Cobra manages the CLI (`internal/cmd/`); configuration is handled via TOML (`internal/config/`).

## Coding Conventions
- **Standard Go**: Use tabs for indentation, `gofmt` style, short descriptive names, and document exported symbols.
- **Error Handling**: Bubble up errors with context rather than panicking.
- **Naming**: Packages are lower-case, singular where possible.

## Testing
- **Location**: Place unit tests alongside the code they test (e.g., `feature_test.go` next to `feature.go`).
- **Style**: Prefer table-driven tests.
- **Requirement**: Run `go test ./...` and `go vet ./...` before considering a feature complete.

## Commit Guidelines
- Use concise, imperative summaries (e.g., "Fix memory leak in lyrics parser").
- Keep commits focused; avoid mixing refactors with new features.
