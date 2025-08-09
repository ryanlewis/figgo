# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## ⚠️ CRITICAL RULES - ALWAYS FOLLOW

### Commit Messages
- **Subject line**: Maximum 50 characters, lowercase, no period
- **Format**: `type: brief description` (types: feat, fix, test, docs, refactor, perf, chore)
- **Body**: 2-4 lines MAX explaining what and why, not how
- **NEVER include**: AI watermarks, "Generated with Claude", emoji, signatures, "Co-Authored-By"
- **Examples**:
  ```
  ✅ GOOD: feat: implement full-width rendering - closes #9
  ❌ BAD:  feat: implement Full-Width rendering mode with comprehensive tests - closes #9 🤖 Generated with Claude
  ```

### Other Critical Rules
- NEVER create files unless absolutely necessary - always prefer editing existing files
- NEVER proactively create documentation files (*.md) unless explicitly requested
- When running potentially destructive commands, explain what they do first

## Project Overview

Figgo is a high-performance Go library and CLI for rendering FIGlet-compatible ASCII art with correct layout handling (kerning/smushing) and a clean, race-safe API. The project targets FIGfont v2 compliance with ASCII (32-126) support.

## Core Architecture

The codebase follows a clean separation of concerns:

- **`figgo.go`**: Main public API entry point providing `ParseFont()`, `Render()`, and option functions
- **`types.go`**: Core type definitions including the immutable `Font` struct and error constants
- **`layout.go`**: Layout bitmask definitions and fitting modes (Full-width, Kerning, Smushing)
- **`internal/parser/`**: FIGfont file parsing logic, converting .flf format to internal representation
- **`internal/renderer/`**: Text rendering engine implementing horizontal fitting and smushing rules
- **`cmd/figgo/`**: CLI application for command-line FIGlet rendering

The Font type is immutable and thread-safe, allowing concurrent use without locking. Layout handling uses a normalized bitmask system combining fitting modes with smushing rules.

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
   - Located in `testdata/goldens/`
   - Generated via `tools/generate-goldens.sh`
   - Test matrix: fonts × layouts × sample strings

2. **Unit Tests**: Test individual components and smushing rules
   - `layout_test.go`: Layout validation and normalization
   - `types_test.go`: Font type behavior

3. **Property Tests**: Fuzz parser and ensure race-safety

## Smushing Rules Implementation

The renderer implements 6 horizontal controlled smushing rules with strict precedence:

1. **Equal character**: Identical non-space characters merge
2. **Underscore**: `_` with border chars (`|/\[]{}()<>`) yields border
3. **Hierarchy**: `|` > `/\` > `[]` > `{}` > `()`
4. **Opposite pairs**: `[]`, `{}`, `()` become `|`
5. **Big X**: `/\` → `X`, `><` → `X`
6. **Hardblank**: Two hardblanks merge to one

Universal smushing applies when no rule matches: take non-space character, never smush hardblank collisions.

## Key Implementation Notes

- **Layout Normalization**: Convert OldLayout (-1/-2/-3) to modern Layout bitmask on font parse
- **Hardblank Handling**: Replace with spaces only after final rendering
- **Print Direction**: 0=LTR, 1=RTL - apply after smushing
- **Error Policy**: Unknown runes → `?`, missing glyphs → `ErrUnsupportedRune`
- **Performance**: Use `strings.Builder`, minimize allocations, precompute glyph trim widths

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
NO WATERMARKS. NO "Generated with". NO "Co-Authored-By: Claude".

## Current Status

The CLI at `cmd/figgo/main.go` is a work-in-progress stub. Core library functionality for font parsing and rendering is being implemented according to the Product Requirements Document (`docs/prd.md`).

### Recent Updates (Issue #6 - Glyph Parser)

The glyph parser is now **fully spec-compliant** with FIGfont v2:
- ✅ Parses all 102 required characters: ASCII 32-126 (95) + German 196,214,220,228,246,252,223 (7)
- ✅ Dynamic endmark detection from glyph data
- ✅ Support for empty FIGcharacters (zero-width)
- ✅ Handles single/double/multiple endmarks correctly
- ✅ Unicode support for hardblank and endmark characters
- ✅ Graceful handling of partial fonts (backward compatibility)