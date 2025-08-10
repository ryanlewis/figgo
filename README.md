# Figgo

[![Coverage Status](https://coveralls.io/repos/github/ryanlewis/figgo/badge.svg?branch=main)](https://coveralls.io/github/ryanlewis/figgo?branch=main)

A high-performance Go library and CLI for rendering FIGlet-compatible ASCII art.

## Features

- FIGfont v2 specification compliant
- Correct layout handling (kerning/smushing)
- Thread-safe, immutable font API
- High performance with minimal allocations

## Installation

```bash
go get github.com/ryanlewis/figgo
```

## Usage

### As a Library

```go
package main

import (
    "fmt"
    "github.com/ryanlewis/figgo"
)

func main() {
    font, err := figgo.ParseFont("path/to/font.flf")
    if err != nil {
        panic(err)
    }
    
    output := figgo.Render(font, "Hello World")
    fmt.Println(output)
}
```

### As a CLI Tool

```bash
# Build the CLI
go build -o figgo ./cmd/figgo

# Render text
./figgo -f standard "Hello World"
```

## Project Structure

- `figgo.go` - Main library API
- `types.go` - Core type definitions
- `layout.go` - Layout handling and smushing rules
- `internal/parser/` - FIGfont file parsing
- `internal/renderer/` - Text rendering engine
- `cmd/figgo/` - CLI application

## Development

```bash
# Run tests
just test

# Run linting
just lint

# Build the project
just build

# Run CI checks locally
just ci
```

## License

MIT