# Product Requirements Document: Figgo - Modern FIGlet Library for Go

## Executive Summary

### Product Vision
Build a high-performance, specification-compliant FIGlet text rendering library for Go that prioritizes production readiness, modern development practices, and developer experience.

### Problem Statement
Existing Go FIGlet libraries suffer from:
- Poor error handling (panic/fatal on invalid input)
- Incomplete FIGfont specification implementation
- Lack of comprehensive testing
- Performance bottlenecks and memory inefficiencies
- Outdated Go patterns (no modules, old stdlib usage)
- Tight coupling to stdout instead of flexible buffer output

### Solution
A ground-up implementation that:
- Correctly implements core FIGfont specification
- Uses modern Go patterns and error handling
- Starts simple and evolves based on real usage
- Focuses on correctness before optimization
- Grows features incrementally with user needs

---

## Product Requirements

### Functional Requirements

#### Core Features (P0 - MVP)
1. **Basic FIGfont Parsing**
   - Parse standard.flf font file
   - Support ASCII character set (32-126)
   - Extract font metadata (height, baseline)

2. **Simple Text Rendering**
   - Render ASCII text with full-width layout
   - Return rendered text as string
   - Maintain consistent character height

3. **Error Handling**
   - Return errors instead of panicking
   - Provide clear error messages

#### Core Features (P1 - Extended)
1. **Additional Fonts**
   - Load multiple standard fonts (slant, small, big)
   - Load fonts from filesystem
   - List available fonts

2. **Layout Modes**
   - Fitting/kerning layout
   - Basic smushing support

3. **Flexible Output**
   - Write to io.Writer
   - Support custom line endings

#### Advanced Features (P2 - Production)
1. **Full Smushing Rules**
   - Complete FIGfont specification smushing
   - Universal smushing mode

2. **Performance Optimizations**
   - Font caching
   - Buffer pooling
   - Concurrent rendering

3. **Production Features**
   - Context support
   - Metrics hooks
   - Thread-safety guarantees

### Non-Functional Requirements

#### Performance
- **Latency**: < 1ms p50, < 5ms p99 for 20-character strings
- **Throughput**: > 10,000 renders/second on modern hardware
- **Memory**: < 10KB per render for standard fonts
- **Allocations**: Zero allocations for cached font renders
- **Concurrency**: Support 10,000+ concurrent operations

#### Reliability
- **Error Rate**: Zero panics in production
- **Compatibility**: 100% FIGfont specification compliance
- **Testing**: > 90% code coverage
- **Stability**: No breaking API changes after v1.0

#### Usability
- **API Simplicity**: Render text in 3 lines of code
- **Documentation**: Godoc for all public APIs
- **Examples**: Working examples for all major features
- **Error Messages**: Clear, actionable error descriptions

#### Maintainability
- **Code Quality**: Pass all standard Go linters
- **Modularity**: Clear separation of concerns
- **Testability**: Mockable interfaces for testing
- **Versioning**: Semantic versioning compliance

---

## Success Criteria

### Functional Success
- [ ] 100% FIGfont specification compliance
- [ ] All standard fonts rendering correctly
- [ ] All smushing rules implemented
- [ ] Error handling without panics
- [ ] Thread-safe operations

### Performance Success
- [ ] < 1ms median latency for 20-char strings
- [ ] < 5ms p99 latency under load
- [ ] Zero allocations for cached renders
- [ ] < 10KB memory per render
- [ ] > 10,000 renders/second throughput

### Quality Success
- [ ] > 90% test coverage
- [ ] Zero critical bugs in production
- [ ] All linters passing
- [ ] Comprehensive documentation
- [ ] Working examples for all features

### Adoption Success
- [ ] Easy migration from existing libraries
- [ ] Positive developer feedback
- [ ] Active community contributions
- [ ] Production usage confirmed
- [ ] Performance benchmarks published

---

## Risk Analysis

### Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| FIGfont spec ambiguity | High | Medium | Study reference implementation, test extensively |
| Performance regression | Medium | Medium | Continuous benchmarking, profiling |
| Memory leaks | High | Low | Use tools like pprof, stress testing |
| API design flaws | High | Low | Review with Go community, iterate in v0 |
| Font compatibility issues | Medium | Medium | Test all standard fonts, fuzzing |

### Project Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Scope creep | Medium | High | Strict phase boundaries, defer P2 features |
| Time overrun | Low | Medium | Focus on MVP first, iterate |
| Adoption challenges | Medium | Medium | Clear migration path, compatibility layer |

---

## Implementation Plan

### Phase 1: MVP - Make It Work

#### Goals
- Minimal working implementation
- Core functionality without optimizations
- Foundation for future enhancements

#### Deliverables
- [ ] Initialize go.mod for `github.com/yourusername/figgo`
- [ ] Create simple directory structure
- [ ] Parse standard.flf font only
- [ ] Implement FLF header parsing for standard font
- [ ] Parse ASCII characters (32-126)
- [ ] Implement full-width rendering only (no smushing)
- [ ] Create simple Engine with Render(text) method
- [ ] Add golden file tests against FIGlet output
- [ ] Write basic README with usage example
- [ ] Ensure zero panics on any input

#### Directory Structure
```
figgo/
├── go.mod              # Module definition
├── LICENSE             # MIT License
├── README.md           # Basic usage
├── figgo.go            # Main API
├── figgo_test.go       # Core tests
├── parser.go           # Font parser
├── parser_test.go      # Parser tests
├── fonts/
│   └── standard.flf    # Embedded standard font
└── testdata/
    └── golden/         # Expected outputs
```

### Phase 2: Core Features - Make It Right

#### Goals
- Multiple font support
- Layout modes implementation
- Improved API ergonomics

#### Deliverables

**Multiple Fonts**
- [ ] Add slant, small, and big fonts
- [ ] Implement LoadFont method
- [ ] Add font listing capability
- [ ] Test each font rendering

**Basic Layout Modes**
- [ ] Implement fitting/kerning layout
- [ ] Add basic smushing (equal character rule only)
- [ ] Test layout modes

**API Improvements**
- [ ] Add RenderWithFont method
- [ ] Support loading fonts from filesystem
- [ ] Improve error messages
- [ ] Add basic examples

#### Additional Files
```
Add to structure:
├── font.go             # Font types
├── layout.go           # Layout modes
├── fonts/
│   ├── slant.flf
│   ├── small.flf
│   └── big.flf
└── examples/
    └── basic/
        └── main.go
```

### Phase 3: Production Ready - Make It Fast

#### Goals
- Full specification compliance
- Performance optimization
- Production-grade features

#### Deliverables

**Full Specification Compliance**
- [ ] Implement all smushing rules
- [ ] Add universal smushing mode
- [ ] Complete FIGfont spec compliance
- [ ] Validate against reference implementation

**Performance Optimizations**
- [ ] Add font caching
- [ ] Implement buffer pooling (if benchmarks show need)
- [ ] Profile and optimize hot paths
- [ ] Add concurrent rendering support
- [ ] Benchmark against performance targets

**Production Features**
- [ ] Add context support for cancellation
- [ ] Implement RenderTo(io.Writer) method
- [ ] Ensure thread-safety
- [ ] Add metrics hooks (if requested)

**Polish**
- [ ] Complete documentation
- [ ] Add CLI tool
- [ ] Create migration guide from other libraries
- [ ] Add more examples

#### Final Structure
```
Add to structure:
├── smushing.go         # Full smushing rules
├── cache.go            # Performance optimizations
├── cmd/figgo/          # CLI tool
└── docs/               # Full documentation
```

---

## Technical Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────┐
│                     Public API Layer                     │
│                     (figgo.Engine)                       │
└────────────┬────────────────────────────────┬───────────┘
             │                                │
┌────────────▼──────────┐        ┌───────────▼───────────┐
│    Font Management    │        │   Rendering Engine    │
│  (Parser, Cache, Loader)       │  (Layout, Smushing)   │
└────────────┬──────────┘        └───────────┬───────────┘
             │                                │
┌────────────▼────────────────────────────────▼───────────┐
│                    Resource Management                   │
│              (Buffer Pool, Memory Management)            │
└──────────────────────────────────────────────────────────┘
             │                                │
┌────────────▼──────────┐        ┌───────────▼───────────┐
│    Embedded Fonts     │        │    File System I/O    │
│     (go:embed)        │        │   (Custom Fonts)      │
└───────────────────────┘        └───────────────────────┘
```

### Core Components

#### 1. Main Engine (`figgo.go`)
```go
// Simple initial API - Phase 1
type Engine struct {
    font *Font
}

// MVP methods
func New() (*Engine, error)                           // Loads embedded standard font
func (e *Engine) Render(text string) (string, error)  // Simple rendering

// Phase 2 additions
func (e *Engine) LoadFont(name string) error
func (e *Engine) RenderWithFont(text, fontName string) (string, error)
func (e *Engine) ListFonts() []string

// Phase 3 additions
func (e *Engine) RenderTo(w io.Writer, text string) error
func (e *Engine) RenderContext(ctx context.Context, text string) (string, error)
```

#### 2. Font System (`font.go`, `parser.go`)
```go
// Font representation
type Font struct {
    // Metadata
    Name      string
    Height    int
    Baseline  int
    MaxLength int
    
    // Layout configuration
    Hardblank    rune
    PrintDirection int
    LayoutMode     LayoutMode
    SmushingRules  SmushingRules
    
    // Character data
    Characters map[rune]*Character
    
    // Performance (Phase 3)
    precomputed map[string][]string // Common words cache
}

// Character representation
type Character struct {
    Rune   rune
    Width  int
    Lines  []string
    
    // Smushing optimization (Phase 2+)
    LeftEdge  []rune // Leftmost smushable character per line
    RightEdge []rune // Rightmost smushable character per line
}

// Parser interface
type FontParser interface {
    Parse(r io.Reader) (*Font, error)
    ParseFile(path string) (*Font, error)
}

// Font manager interface (Phase 2+)
type FontManager interface {
    Get(name string) (*Font, error)
    Load(name string, font *Font) error
    List() []string
    Clear()
}
```

#### 3. Rendering System (`renderer.go`, `layout.go`, `smushing.go`)
```go
// Renderer interface
type Renderer interface {
    Render(font *Font, text string, opts RenderOptions) ([]string, error)
}

// Layout engine (Phase 2+)
type LayoutEngine interface {
    Layout(chars []*Character, mode LayoutMode, rules SmushingRules) []string
}

// Layout modes
type LayoutMode int

const (
    LayoutFullWidth LayoutMode = iota // No kerning or smushing (Phase 1)
    LayoutFitting                      // Kerning only (Phase 2)
    LayoutSmushing                     // Full smushing (Phase 2)
    LayoutControlled                   // Controlled smushing (Phase 3)
)

// Smushing rules (bit flags) - Phase 2+
type SmushingRules uint32

const (
    SmushEqual      SmushingRules = 1 << iota // Rule 1
    SmushUnderscore                            // Rule 2
    SmushHierarchy                             // Rule 3
    SmushPair                                  // Rule 4
    SmushBigX                                  // Rule 5
    SmushHardblank                             // Rule 6
    SmushKern       SmushingRules = 128       // Kerning
    SmushDefault    SmushingRules = 256       // Default
)

// Render options (Phase 2+)
type RenderOptions struct {
    Layout        LayoutMode
    SmushingRules SmushingRules
    MaxWidth      int
    LineEnding    string
}
```

#### 4. Configuration & Options
```go
// Phase 2: Font selection
func (e *Engine) SetFont(name string) error

// Phase 3: Render options
type RenderOption func(*renderConfig)
func WithLayout(mode LayoutMode) RenderOption
func WithMaxWidth(width int) RenderOption
func WithLineEnding(ending string) RenderOption
```

---

## API Design & Examples

### Phase 1 - MVP Usage
```go
package main

import (
    "fmt"
    "log"
    
    "github.com/yourusername/figgo"
)

func main() {
    // Simple usage - one way to do it
    engine, err := figgo.New()
    if err != nil {
        log.Fatal(err)
    }
    
    output, err := engine.Render("Hello World")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(output)
}
```

### Phase 2 - Extended Usage
```go
package main

import (
    "fmt"
    "log"
    
    "github.com/yourusername/figgo"
)

func main() {
    engine, err := figgo.New()
    if err != nil {
        log.Fatal(err)
    }
    
    // Load different font
    err = engine.LoadFont("slant")
    if err != nil {
        log.Fatal(err)
    }
    
    // Render with specific font
    output, err := engine.RenderWithFont("Hello", "slant")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(output)
    
    // List available fonts
    fonts := engine.ListFonts()
    fmt.Println("Available fonts:", fonts)
}
```

### Phase 3 - Production Usage
```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "os"
    "time"
    
    "github.com/yourusername/figgo"
)

func main() {
    engine, err := figgo.New()
    if err != nil {
        log.Fatal(err)
    }
    
    // Context support for timeout
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    output, err := engine.RenderContext(ctx, "Timeout Test")
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            log.Println("Render timeout")
        }
    } else {
        fmt.Println(output)
    }
    
    // Write directly to io.Writer
    err = engine.RenderTo(os.Stdout, "Direct Write")
    if err != nil {
        log.Fatal(err)
    }
    
    // Advanced options (Phase 3)
    output, err = engine.RenderWithOptions("Custom", 
        figgo.WithLayout(figgo.LayoutSmushing),
        figgo.WithMaxWidth(80),
        figgo.WithLineEnding("\r\n"),
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(output)
}
```

### CLI Tool Usage (Phase 3)
```bash
# Basic usage
figgo "Hello World"

# With font selection
figgo -f slant "Hello World"

# With layout options
figgo -l smushing -w 80 "Hello World"

# List available fonts
figgo --list-fonts

# Read from stdin
echo "Hello" | figgo

# Output to file
figgo "Hello" > output.txt
```

---

## Testing Strategy

### Unit Testing
- **Coverage Target**: > 90%
- **Test Categories**:
  - Parser correctness
  - Renderer accuracy
  - Layout calculations
  - Smushing logic
  - Cache behavior
  - Error handling

### Integration Testing
- End-to-end rendering tests
- Multi-font scenarios
- Concurrent usage patterns
- Memory leak detection
- Performance regression tests

### Compliance Testing
```go
// Example compliance test
func TestFIGletCompliance(t *testing.T) {
    tests := []struct {
        name     string
        text     string
        font     string
        layout   LayoutMode
        expected string // From reference FIGlet
    }{
        {
            name:   "standard_font_basic",
            text:   "Hello",
            font:   "standard",
            layout: LayoutSmushing,
            expected: loadGoldenFile("standard_hello_smush.txt"),
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            engine, _ := figgo.New()
            result, _ := engine.RenderWithFont(tt.text, tt.font)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Fuzzing
```go
func FuzzFontParser(f *testing.F) {
    // Seed with valid headers
    f.Add([]byte("flf2a$ 8 6 15 -1 16\n"))
    
    f.Fuzz(func(t *testing.T, data []byte) {
        parser := NewParser()
        _, err := parser.Parse(bytes.NewReader(data))
        // Must not panic, errors are acceptable
        if err != nil {
            t.Skip() // Invalid input, skip
        }
    })
}
```

### Benchmarking
```go
func BenchmarkRender(b *testing.B) {
    engine, _ := figgo.New()
    text := "Hello World"
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        _, err := engine.Render(text)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkConcurrentRender(b *testing.B) {
    engine, _ := figgo.New()
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, _ = engine.Render("Test")
        }
    })
}
```

### Golden File Testing
```go
func TestGoldenFiles(t *testing.T) {
    // Compare output against known good FIGlet output
    goldenDir := "testdata/golden"
    
    tests, _ := filepath.Glob(filepath.Join(goldenDir, "*.txt"))
    for _, golden := range tests {
        name := filepath.Base(golden)
        t.Run(name, func(t *testing.T) {
            expected, _ := os.ReadFile(golden)
            
            // Parse test parameters from filename
            // e.g., "standard_hello_smushing.txt"
            parts := strings.Split(name, "_")
            font := parts[0]
            text := parts[1]
            layout := parts[2]
            
            engine, _ := figgo.New()
            actual, _ := engine.RenderWithFont(text, font)
            
            assert.Equal(t, string(expected), actual)
        })
    }
}
```

---

## Appendices

### Appendix A: FIGfont Specification Summary

The FIGfont format consists of:

1. **Header Line**: Contains font metadata
   ```
   flf2a$ 8 6 15 -1 16
   ```
   - Magic: "flf2a"
   - Hardblank: '$'
   - Height: 8 lines
   - Baseline: 6
   - Max length: 15
   - Old layout: -1 (full width)
   - Comment lines: 16

2. **Comments**: Font information and copyright

3. **Character Data**: ASCII art for each character
   - Required: ASCII 32-126
   - Optional: Extended characters
   - Format: Multiple lines ending with endmark

### Appendix B: Smushing Rules Detail

1. **Equal Character**: `||` becomes `|`
2. **Underscore**: `_` combines with `|/\[]{}()<>`
3. **Hierarchy**: Based on character class ranking
4. **Opposite Pair**: `[]` becomes `|`, etc.
5. **Big X**: `/\` becomes `X`
6. **Hardblank**: Hardblank can be smushed

### Appendix C: Performance Baseline

Target performance metrics based on production requirements:
- Render time: < 1ms (cached fonts)
- Memory usage: < 10KB per render
- Allocations: Minimal per render operation
- Concurrency: Full support for concurrent renders

### Appendix D: Reference Implementation

The reference FIGlet implementation (C) can be found at:
- Repository: https://github.com/cmatsuoka/figlet
- Specification: http://www.figlet.org/figfont.html

Key differences in Figgo implementation:
- Pure Go (no CGO)
- Buffer-based instead of stdout
- Modern error handling
- Performance optimizations
- Production features (context, metrics)

### Appendix E: Migration Guide

For users migrating from other Go FIGlet libraries:

**From go-figlet:**
```go
// Old
figlet.Render("text")

// New - Figgo
engine, _ := figgo.New()
output, _ := engine.Render("text")
```

**From figlet4go:**
```go
// Old
ascii := figlet4go.NewAsciiRender()
options := figlet4go.NewRenderOptions()
ascii.Render("text", options)

// New - Figgo
engine, _ := figgo.New()
output, _ := engine.Render("text")
```

---

## Conclusion

This PRD defines a comprehensive plan for building Figgo, a modern, production-ready FIGlet library for Go. The implementation focuses on correctness, performance, and developer experience while maintaining full specification compliance.

The phased approach allows for incremental delivery of value while maintaining high quality standards. With proper testing and benchmarking throughout development, we can ensure Figgo meets all stated requirements and becomes the go-to FIGlet library for Go developers.