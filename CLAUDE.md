# CLAUDE.md

This file provides guidance for code assistants when working with this repository.

## ⚠️ CRITICAL RULES - ALWAYS FOLLOW

### Commit Messages
- **Subject line**: Maximum 50 characters, lowercase, no period
- **Format**: `type: brief description` (types: feat, fix, test, docs, refactor, perf, chore)
- **Body**: 2-4 lines MAX explaining what and why, not how
- **Examples**:
  ```
  ✅ GOOD: feat: implement full-width rendering - closes #9
  ```

### Other Critical Rules
- Prefer editing existing files; create new files only when the task requires it (e.g., adding a new package or test).
- Do not proactively create documentation files (*.md) unless explicitly requested.
- NEVER commit if `just ci` fails or reports warnings
- When running potentially destructive commands, explain what they do first

## Project Overview

Figgo is a high-performance Go library and CLI for rendering FIGlet-compatible ASCII art with correct layout handling (kerning/smushing) and a clean, race-safe API. The project targets FIGfont v2 compliance with ASCII (32-126) support.

## Core Architecture

The codebase follows a clean separation of concerns:

- **`figgo.go`**: Main public API entry point providing `ParseFont()`, `Render()`, and option functions
- **`types.go`**: Core type definitions including the immutable `Font` struct and error constants
- **`layout.go`**: Layout bitmask definitions and fitting modes (Full-width, Kerning, Smushing)
- **`font_cache.go`**: LRU font cache with SHA256 content-based keys for performance
- **`internal/parser/`**: FIGfont file parsing logic with lazy trim computation and memory pooling
- **`internal/renderer/`**: Text rendering engine with smushing rules and render buffer pooling
- **`cmd/figgo/`**: CLI application for command-line FIGlet rendering
- **`golden_test.go`**: Golden test harness for FIGlet compatibility verification

The Font type is immutable and thread-safe, allowing concurrent use without locking. Layout handling uses a normalized bitmask system combining fitting modes with smushing rules.

## Key Documentation

- `docs/figfont-spec.txt`: Specification for FIGlet fonts
- `docs/prd.md`: Product Requirements Document
- `docs/spec-compliance.md`: Tracking of compliance with the spec

- Reference C implementation: https://raw.githubusercontent.com/cmatsuoka/figlet/refs/heads/master/figlet.c
## Available Test Fonts

The project includes 4 FIGlet fonts for testing:
- **`standard.flf`**: Default font, medium size (28KB)
- **`slant.flf`**: Slanted text style (16KB)
- **`small.flf`**: Compact font (12KB)
- **`big.flf`**: Large block letters (26KB)

## Build and Development Commands

```bash
# Build the binary
just build
# or: go build -v -o figgo ./cmd/figgo

# Run all tests with race detection
just test
# or: go test -v -race ./...

# Run linting (golangci-lint or go vet fallback)
just lint

# Format code (goimports + gofmt)
just fmt

# Run a single test
go test -v -run TestSpecificFunction ./...

# Generate test coverage
just coverage

# Generate golden test files (uses system figlet)
just generate-goldens
# or: go run ./cmd/generate-goldens
# With specific options:
go run ./cmd/generate-goldens -fonts "standard slant" -layouts "full kern smush"
# Strict mode (fail on warnings):
go run ./cmd/generate-goldens -strict

# Run golden tests
go test -run TestGoldenFiles
# or quick subset:
go test -run TestGoldenFilesSubset

# Run CI checks locally (lint + test + build)
just ci

# Run benchmarks
just bench
# or: go test -bench=. -benchmem ./...

# Manage dependencies
just mod
```

## Testing Strategy

1. **Golden Tests**: Compare output against the reference `figlet` implementation
   - Located in `testdata/goldens/`
   - Generated with `cmd/generate-goldens` using `figlet -w 80` and, for smushing, `-s`
   - Covers multiple fonts (`fonts/*.flf`), layouts (full/kern/smush), and representative samples

2. **Unit Tests**: Test individual components and smushing rules
   - `layout_test.go`: Layout validation and normalization
   - `types_test.go`: Font type behavior
   - `internal/renderer/smushing_test.go`: Smushing rule tests

3. **Property/Correctness**: Parser validation and race-safety
   - Race detection via `go test -race`
   - Concurrent rendering tests

### Testing Tips

- When comparing outputs between figgo and figlet, its easier and more insightful to compare column-by-column instead of row-by-row.

## Smushing Rules Implementation

The renderer implements 6 horizontal controlled smushing rules with strict precedence (matching FIGlet):

1. **Hardblank**: Two hardblanks merge to one (highest controlled precedence)
2. **Equal**: Identical characters merge
3. **Underscore**: `_` merges into border chars (`|/\[]{}()<>`)
4. **Hierarchy**: Stronger replaces weaker: `|` > `/\` > `[]` > `{}` > `()` > `<>`
5. **Opposite pair**: `[]`, `{}`, `()` (and their reverses) become `|`
6. **Big X**: `/\` → `|`, `\/` → `Y`, `><` → `X`

- **Universal smushing**: When smushing is enabled but no rule bits are set, the overlap prefers the right character in LTR and left in RTL; hardblanks yield to visible characters.
- **No matching rule**: When controlled rules are present but none matches, characters do not overlap (kerning fallback).

## Key Implementation Notes

- **Layout Normalization**: Font headers provide `OldLayout` and possibly `FullLayout`. The public `Font.Layout` is normalized from header values; rule selection for smushing follows FIGlet (`-s`) semantics.
- **Hardblank Handling**: Replace with spaces only after final rendering
- **Print Direction**: 0=LTR, 1=RTL — affects smushing decisions and boundary calculations; the algorithm handles both directions.
- **Error Policy**: Unknown runes → `?`, missing glyphs → `ErrUnsupportedRune`
- **Performance Optimizations**:
  - Memory pooling for parser buffers (64KB-4MB) and render buffers
  - LRU font cache with SHA256 content-based keys
  - Lazy computation of glyph trim data on first access
  - Precomputed trim widths to avoid repeated scanning
  - `strings.Builder` for efficient string concatenation

## GitHub Workflow

The project uses GitHub Actions for CI with the following jobs:
- **Lint**: golangci-lint with comprehensive rules (`.golangci.yml`)
- **Test**: Matrix testing across Go 1.22/1.23 and Linux/macOS/Windows
- **Build**: Cross-compilation for multiple GOOS/GOARCH combinations
- **Benchmark**: PR-triggered performance testing

### Before Committing - STOP AND CHECK:
1. Is subject line ≤ 50 characters?
2. Did you remove ALL watermarks and signatures?
3. Is the message concise (what, not how)?
4. Run: `git log -1 --oneline` to verify

### Commit Message Template:
```
<type>: <description under 50 chars>

<2-4 line body if needed>
<Closes #N if applicable>
```

## Debug Mode (Comprehensive, On/Off)

A single, comprehensive debug mode is available to trace parsing, layout decisions, and renderer internals.

- Enable: set `FIGGO_DEBUG=1` or use CLI flags `--debug`, `--debug-file`, `--debug-pretty`.
- Format: JSON Lines by default; pretty human-readable output with `--debug-pretty`.
- Scope: Per-render session with a unique ID; events cover parser (summary), API (layout merge), renderer (start/end, overlap calculations, smush decisions, flush/split), and output.
- Library: pass a session via `figgo.WithDebug(session)`; see `internal/debug/` for event types and sinks.

This mode is zero-cost when disabled and provides structured, machine-parsable traces when enabled.
