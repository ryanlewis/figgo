# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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
- NEVER create files unless absolutely necessary - always prefer editing existing files
- NEVER proactively create documentation files (*.md) unless explicitly requested
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
- **`golden_test.go`**: Comprehensive golden test harness for FIGlet compatibility verification

The Font type is immutable and thread-safe, allowing concurrent use without locking. Layout handling uses a normalized bitmask system combining fitting modes with smushing rules.

## Key documentation

- `docs/figfont-spec.txt`: Specification for FIGlet fonts
- `docs/prd.md`: Product Requirements Document
- `docs/spec-compliance.md`: Tracking of compliance with the spec

- https://raw.githubusercontent.com/cmatsuoka/figlet/refs/heads/master/figlet.c: Reference C implementation
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

# Generate golden test files
./tools/generate-goldens.sh
# or with specific fonts/layouts:
FONTS="standard slant" LAYOUTS="full kern smush" ./tools/generate-goldens.sh
# or in strict CI mode (fail on warnings):
STRICT=1 ./tools/generate-goldens.sh

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

1. **Golden Tests**: Compare output against C figlet reference implementation
   - Located in `testdata/goldens/` (144 test files)
   - Generated via `tools/generate-goldens.sh`
   - Test matrix: 4 fonts × 3 layouts × 12 sample strings
   - Current pass rate: 38% (56/144 tests passing)
   - Breakdown by layout:
     - Full-width: 4% passing (spacing issues)
     - Kerning: 67% passing (best performance)
     - Smushing: 46% passing (partial implementation)

2. **Unit Tests**: Test individual components and smushing rules
   - `layout_test.go`: Layout validation and normalization
   - `types_test.go`: Font type behavior
   - `internal/renderer/smushing_test.go`: Smushing rule tests

3. **Property Tests**: Fuzz parser and ensure race-safety
   - Race detection via `go test -race`
   - Concurrent rendering tests

## Smushing Rules Implementation

The renderer implements 6 horizontal controlled smushing rules with strict precedence:

1. **Equal character**: Identical non-space, non-hardblank characters merge
2. **Underscore**: `_` with border chars (`|/\[]{}()<>`) yields border
3. **Hierarchy**: `|` > `/\` > `[]` > `{}` > `()` > `<>`
4. **Opposite pairs**: `[]`, `{}`, `()` (and their reverses) become `|`
5. **Big X**: `/\` → `|`, `\/` → `Y`, `><` → `X`
6. **Hardblank**: Two hardblanks merge to one

Universal smushing applies only when NO controlled rules are defined: later character overrides earlier at overlap position. When controlled rules ARE defined but no rule matches, fall back to kerning.

## Key Implementation Notes

- **Layout Normalization**: Convert OldLayout (-1 or 0..63) to modern Layout bitmask on font parse
- **Hardblank Handling**: Replace with spaces only after final rendering
- **Print Direction**: 0=LTR, 1=RTL - apply after smushing
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

## Current Status

Core library functionality is implemented with comprehensive testing infrastructure. The project is actively working toward full FIGlet compatibility.

### Recent Updates

#### Issue #26 - Golden Test Suite (Completed)
- ✅ Comprehensive golden test harness (`golden_test.go`)
- ✅ 144 golden test files covering 4 fonts × 3 layouts × 12 samples
- ✅ Enhanced `generate-goldens.sh` with CI support (STRICT mode)
- ✅ YAML front matter with full metadata in markdown format
- ✅ Current compliance: 38% overall (56/144 tests passing)

#### Issue #6 - Glyph Parser (Completed)
The glyph parser is **fully spec-compliant** with FIGfont v2:
- ✅ Parses all 102 required characters: ASCII 32-126 (95) + German 196,214,220,228,246,252,223 (7)
- ✅ Dynamic endmark detection from glyph data
- ✅ Support for empty FIGcharacters (zero-width)
- ✅ Handles single/double/multiple endmarks correctly
- ✅ Unicode support for hardblank and endmark characters
- ✅ Graceful handling of partial fonts (backward compatibility)

### Performance Enhancements
- Memory pooling for parser and renderer (reduces allocations)
- LRU font caching with content-based keys
- Lazy computation patterns for expensive operations
- Thread-safe concurrent rendering support
