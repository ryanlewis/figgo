# Figgo

[![Coverage Status](https://coveralls.io/repos/github/ryanlewis/figgo/badge.svg?branch=main)](https://coveralls.io/github/ryanlewis/figgo?branch=main)

A high-performance Go library and CLI for rendering FIGlet-compatible ASCII art.

## Features

- FIGfont v2 specification compliant
- Correct layout handling (full-width, kerning, smushing)
- All 6 horizontal controlled smushing rules + universal smushing
- Thread-safe, immutable font API — safe for concurrent use
- LRU font cache (in-memory) with optional on-disk binary cache
- LTR and RTL print direction support
- Compressed font support (ZIP)

## Installation

### Library

```bash
go get github.com/ryanlewis/figgo
```

Requires Go 1.24 or later.

### CLI

```bash
go install github.com/ryanlewis/figgo/cmd/figgo@latest
```

## Usage

### As a Library

```go
package main

import (
    "fmt"
    "log"

    "github.com/ryanlewis/figgo"
)

func main() {
    font, err := figgo.LoadFont("fonts/standard.flf")
    if err != nil {
        log.Fatal(err)
    }

    output, err := figgo.Render("Hello, World!", font)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(output)
}
```

### Render Options

```go
// Force a specific layout mode
output, _ := figgo.Render("Hello", font, figgo.WithLayout(figgo.FitSmushing))

// Set output width (default 80)
output, _ := figgo.Render("Hello", font, figgo.WithWidth(120))

// Right-to-left rendering
output, _ := figgo.Render("Hello", font, figgo.WithPrintDirection(1))

// Replace unsupported characters instead of erroring
output, _ := figgo.Render("Hello 🎉", font, figgo.WithUnknownRune('?'))

// Trim trailing whitespace from each line
output, _ := figgo.Render("Hello", font, figgo.WithTrimWhitespace(true))
```

### Font Loading

```go
// Load from file path
font, err := figgo.LoadFont("fonts/standard.flf")

// Parse from io.Reader
font, err := figgo.ParseFont(reader)

// Parse from byte slice
font, err := figgo.ParseFontBytes(data)

// Load from a directory by name
font, err := figgo.LoadFontDir("/usr/share/figlet", "standard")

// Load from an fs.FS (e.g., embedded fonts)
font, err := figgo.LoadFontFS(myFS, "fonts/standard.flf")
```

### Font Caching

Figgo includes a two-tier font cache for long-running applications:

```go
// In-memory LRU cache (global convenience functions)
font, err := figgo.LoadFontCached("fonts/standard.flf")
font, err := figgo.ParseFontCached(data)

// Custom cache instance
cache := figgo.NewFontCache(50) // up to 50 fonts in memory
font, err := cache.LoadFont("fonts/standard.flf")

// Enable on-disk binary cache for faster restarts
cache := figgo.NewFontCache(50, figgo.WithDiskCache(figgo.DiskCacheConfig{
    MaxEntries: 20, // max fonts on disk
}))

// Or enable disk caching on the global default cache
figgo.EnableDefaultDiskCache(figgo.DiskCacheConfig{})

// Cache stats
stats := figgo.DefaultCacheStats()
fmt.Printf("Hit rate: %.1f%%\n", stats.HitRate())
```

The disk cache serializes parsed fonts to `os.UserCacheDir()/figgo/fonts/` by default. It uses LRU eviction, atomic writes, and silently falls back to parsing on any error.

### As a CLI Tool

```bash
# Basic usage
figgo "Hello, World!"

# Specify a font
figgo -f fonts/slant.flf "Hello"

# Set output width
figgo -w 120 "Hello, World!"

# Force smushing layout
figgo -s "Hello"

# Debug mode (JSON trace output)
figgo --debug "Hello"
```

## Project Structure

```
figgo.go              Main public API (LoadFont, Render, options)
types.go              Core types (Font, Layout, Option)
layout.go             Layout bitmask definitions and fitting modes
font_cache.go         In-memory LRU font cache
disk_cache.go         On-disk binary font cache (opt-in)
internal/parser/      FIGfont file parsing with lazy trim computation
internal/renderer/    Rendering engine with smushing rules
internal/debug/       Structured debug tracing (JSON Lines)
cmd/figgo/            CLI application
cmd/generate-goldens/ Golden test file generator
```

## Development

```bash
# Run tests with race detection
just test

# Run linting
just lint

# Format code
just fmt

# Build the binary
just build

# Run CI checks locally (lint + test + build)
just ci

# Run benchmarks
just bench

# Generate golden test files (requires system figlet)
just generate-goldens
```

## License

MIT