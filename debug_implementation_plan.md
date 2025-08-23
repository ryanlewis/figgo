# Debug System Implementation Plan for Figgo

## Executive Summary

This document outlines the implementation of a comprehensive debug system for the figgo library. The system provides deep visibility into the rendering pipeline with a single on/off switch, zero overhead when disabled, and structured output for easy analysis.

## Design Principles

1. **Single Switch**: `FIGGO_DEBUG=1` or `--debug` flag enables everything
2. **Zero Overhead**: No performance impact when disabled
3. **Session Scoped**: Each render gets unique session ID for concurrent safety
4. **Comprehensive Coverage**: Every decision point logged with context
5. **Machine Parsable**: JSON Lines by default, pretty format optional
6. **Surgical Integration**: Minimal changes to existing code

## Architecture Overview

### Core Components

```
internal/debug/
├── debug.go        # Core debug system, session management
├── sink.go         # Output sinks (JSON, pretty, file)
├── events.go       # Typed event definitions
└── debug_test.go   # Testing and benchmarks
```

### Data Flow

```
[Environment/CLI] → [Session Creation] → [Debug Handle] → [Renderer State]
                                              ↓
                                         [Event Emission]
                                              ↓
                                    [Sink (JSON/Pretty)] → [Output]
```

## Detailed Implementation

### 1. Core Debug Infrastructure (`internal/debug/`)

#### `debug.go` - Core System

```go
package debug

import (
    "fmt"
    "os"
    "sync/atomic"
    "time"
)

// Global enabled flag - set once at startup
var enabled uint32

// SetEnabled configures debug mode (called once at startup)
func SetEnabled(on bool) {
    if on {
        atomic.StoreUint32(&enabled, 1)
    }
}

// Enabled returns true if debug mode is active
func Enabled() bool {
    return atomic.LoadUint32(&enabled) == 1
}

// Debug represents a debug session
type Debug struct {
    sessionID string
    sink      Sink
    startTime time.Time
}

// StartSession creates a new debug session
func StartSession(sink Sink) *Debug {
    if !Enabled() {
        return nil
    }
    return &Debug{
        sessionID: generateSessionID(),
        sink:      sink,
        startTime: time.Now(),
    }
}

// Emit sends an event to the sink (fast-path for nil check)
func (d *Debug) Emit(phase, event string, data interface{}) {
    if d == nil {
        return
    }
    
    evt := Event{
        Timestamp: time.Now().Format(time.RFC3339Nano),
        SessionID: d.sessionID,
        Phase:     phase,
        Event:     event,
        Data:      data,
    }
    
    d.sink.Write(evt)
}

// Close flushes and closes the debug session
func (d *Debug) Close() error {
    if d == nil {
        return nil
    }
    
    // Emit session end event
    elapsed := time.Since(d.startTime).Milliseconds()
    d.Emit("session", "End", map[string]int64{"elapsed_ms": elapsed})
    
    return d.sink.Close()
}
```

#### `sink.go` - Output Sinks

```go
package debug

import (
    "bufio"
    "encoding/json"
    "fmt"
    "io"
    "os"
)

// Sink is the interface for debug output destinations
type Sink interface {
    Write(event Event) error
    Flush() error
    Close() error
}

// JSONSink writes JSON Lines format
type JSONSink struct {
    w       *bufio.Writer
    encoder *json.Encoder
}

func NewJSONSink(w io.Writer) *JSONSink {
    bw := bufio.NewWriter(w)
    return &JSONSink{
        w:       bw,
        encoder: json.NewEncoder(bw),
    }
}

func (s *JSONSink) Write(event Event) error {
    return s.encoder.Encode(event)
}

func (s *JSONSink) Flush() error {
    return s.w.Flush()
}

func (s *JSONSink) Close() error {
    return s.Flush()
}

// PrettySink writes human-readable format
type PrettySink struct {
    w *bufio.Writer
}

func NewPrettySink(w io.Writer) *PrettySink {
    return &PrettySink{
        w: bufio.NewWriter(w),
    }
}

func (s *PrettySink) Write(event Event) error {
    // Format: [timestamp] [phase/event] data
    fmt.Fprintf(s.w, "[%s] [%s/%s]\n", event.Timestamp, event.Phase, event.Event)
    
    // Pretty print data based on type
    switch d := event.Data.(type) {
    case SmushAmountRowData:
        fmt.Fprintf(s.w, "  session: %s\n", event.SessionID)
        fmt.Fprintf(s.w, "  glyph: %d, row: %d\n", d.GlyphIdx, d.Row)
        fmt.Fprintf(s.w, "  boundaries: line=%d(%s), char=%d(%s)\n", 
            d.LineBoundaryIdx, runeStr(d.Ch1Code),
            d.CharBoundaryIdx, runeStr(d.Ch2Code))
        fmt.Fprintf(s.w, "  amount: %d %s → %d\n", 
            d.AmtBefore, d.PlusReason, d.AmtAfter)
        if d.RTL {
            fmt.Fprintf(s.w, "  direction: RTL\n")
        }
    // ... handle other event types ...
    default:
        // Generic formatting for unknown types
        fmt.Fprintf(s.w, "  data: %+v\n", d)
    }
    
    return nil
}

// runeStr formats a rune code for display: 'X' (0x58)
func runeStr(code int) string {
    if code == 0 {
        return "NUL"
    }
    if code >= 32 && code < 127 {
        return fmt.Sprintf("'%c' (0x%X)", rune(code), code)
    }
    return fmt.Sprintf("0x%X", code)
}
```

#### `events.go` - Event Definitions

```go
package debug

// Event is the base envelope for all debug events
type Event struct {
    Timestamp string      `json:"ts"`
    SessionID string      `json:"session_id"`
    Phase     string      `json:"phase"`
    Event     string      `json:"event"`
    Data      interface{} `json:"data"`
}

// Parser Events

type FontHeaderData struct {
    Hardblank    int    `json:"hardblank"`
    Height       int    `json:"height"`
    Baseline     int    `json:"baseline"`
    MaxLength    int    `json:"max_length"`
    OldLayout    int    `json:"old_layout"`
    FullLayout   int    `json:"full_layout"`
    PrintDir     int    `json:"print_dir"`
    CommentLines int    `json:"comment_lines"`
}

type LayoutNormalizedData struct {
    OldLayout     int    `json:"old_layout"`
    FullLayout    int    `json:"full_layout"`
    FullLayoutSet bool   `json:"full_layout_set"`
    NormalizedH   int    `json:"normalized_horz"`
    NormalizedV   int    `json:"normalized_vert"`
    Rationale     string `json:"rationale"`
}

type GlyphStatsData struct {
    RequiredCount int      `json:"required_count"`
    OptionalCount int      `json:"optional_count"`
    MissingList   []string `json:"missing_list,omitempty"`
}

// API Events

type OptionsData struct {
    Layout         *int  `json:"layout,omitempty"`
    Width          *int  `json:"width,omitempty"`
    PrintDirection *int  `json:"print_direction,omitempty"`
    UnknownRune    *int  `json:"unknown_rune,omitempty"`
    TrimWhitespace bool  `json:"trim_whitespace"`
}

type LayoutMergeData struct {
    RequestedLayout int    `json:"requested_layout"`
    FontDefaults    int    `json:"font_defaults"`
    InjectedRules   int    `json:"injected_rules"`
    FinalSmushMode  int    `json:"final_smush_mode"`
    Rationale       string `json:"rationale"`
}

// Renderer Events

type StartData struct {
    CharHeight   int `json:"char_height"`
    Hardblank    int `json:"hardblank"`
    WidthLimit   int `json:"width_limit"`
    PrintDir     int `json:"print_dir"`
    SmushMode    int `json:"smush_mode"`
}

type GlyphData struct {
    Index        int  `json:"index"`
    Rune         int  `json:"rune"`
    GlyphWidth   int  `json:"glyph_width"`
    SpaceGlyph   bool `json:"space_glyph"`
    UnknownSubst bool `json:"unknown_subst,omitempty"`
}

type SmushAmountRowData struct {
    GlyphIdx        int    `json:"glyph_idx"`
    Row             int    `json:"row"`
    LineBoundaryIdx int    `json:"line_boundary_idx"`
    CharBoundaryIdx int    `json:"char_boundary_idx"`
    Ch1Code         int    `json:"ch1_code"`  // Numeric codepoint
    Ch2Code         int    `json:"ch2_code"`  // Numeric codepoint
    AmtBefore       int    `json:"amt_before"`
    PlusReason      string `json:"plus_reason"` // "none"|"ch1_null"|"smush(l,r)"
    AmtAfter        int    `json:"amt_after"`
    RTL             bool   `json:"rtl"`
}

type SmushDecisionData struct {
    Row        int    `json:"row"`
    Col        int    `json:"col"`
    LchCode    int    `json:"lch_code"`    // Numeric codepoint
    RchCode    int    `json:"rch_code"`    // Numeric codepoint
    ResultCode int    `json:"result_code"` // Numeric codepoint
    Rule       string `json:"rule"`        // Rule name
}

type RowAppendData struct {
    Row        int `json:"row"`
    StartPos   int `json:"start_pos"`
    CharCount  int `json:"char_count"`
    EndBefore  int `json:"end_before"`
    EndAfter   int `json:"end_after"`
}

type SplitData struct {
    Reason      string `json:"reason"`       // "width"|"wordbreak"
    FSMPrev     int    `json:"fsm_prev"`
    FSMNext     int    `json:"fsm_next"`
    OutlineLen  int    `json:"outline_len"`
}

type FlushData struct {
    RowLengthsBefore []int `json:"row_lengths_before"` // Capped at 32
    RowLengthsAfter  []int `json:"row_lengths_after"`  // Capped at 32
}

type EndData struct {
    TotalLines int   `json:"total_lines"`
    TotalRunes int   `json:"total_runes"`
    ElapsedMs  int64 `json:"elapsed_ms"`
}

// Output Events

type WriteRowData struct {
    RowIdx               int  `json:"row_idx"`
    Trimmed              bool `json:"trimmed"`
    HardblankReplacements int  `json:"hardblank_replacements"`
}

type WriteDoneData struct {
    BytesWritten int `json:"bytes_written"`
}

// Error Events

type ErrorData struct {
    Type    string `json:"type"`
    Message string `json:"message"`
    Context map[string]interface{} `json:"context,omitempty"`
}
```

### 2. Integration Points

#### API Layer (`figgo.go`)

```go
// Add to options struct
type options struct {
    layout         *Layout
    printDirection *int
    unknownRune    *rune
    trimWhitespace bool
    width          *int
    debug          *debug.Debug  // NEW: Session-scoped debug handle
}

// Add debug option function
func WithDebug(d *debug.Debug) Option {
    return func(o *options) { o.debug = d }
}

// Modify RenderTo to log layout merge and normalized layout (API layer)
func (f *Font) RenderTo(w io.Writer, text string, opts ...Option) error {
    // ... existing setup ...

    // Handle FitSmushing rule injection as today; also log decisions
    const ruleMask = RuleEqualChar | RuleUnderscore | RuleHierarchy | RuleOppositePair | RuleBigX | RuleHardblank
    requestedLayout := 0
    if options.layout != nil { requestedLayout = int(*options.layout) }

    injected := 0
    rationale := ""
    if options.layout != nil && (*options.layout & FitSmushing) != 0 && ((*options.layout & ruleMask) == 0) {
        fontRules := f.Layout & ruleMask
        if fontRules == 0 {
            fontRules = ruleMask
            rationale = "FitSmushing requested; no font defaults; injecting all rules"
        } else {
            rationale = "FitSmushing requested; using font default rules"
        }
        injected = int(fontRules)
        newLayout := *options.layout | fontRules
        options.layout = &newLayout
    }

    finalLayout := 0
    if options.layout != nil { finalLayout = int(*options.layout) }
    finalSmushMode := layoutToSmushMode(finalLayout)

    if options.debug != nil {
        // Log layout merge
        options.debug.Emit("api", "LayoutMerge", debug.LayoutMergeData{
            RequestedLayout: requestedLayout,
            FontDefaults:    int(f.Layout),
            InjectedRules:   injected,
            FinalLayout:     finalLayout,
            FinalSmushMode:  finalSmushMode,
            Rationale:       rationale,
        })

        // Optional: Log normalized layout derived from font header (no parser import here)
        options.debug.Emit("api", "LayoutNormalized", debug.LayoutNormalizedData{
            OldLayout:     f.OldLayout,
            FullLayout:    0,          // if exposed; otherwise omit or set 0
            FullLayoutSet: false,      // if exposed; otherwise omit or set false
            NormalizedH:   int(ModeFromLayout(finalLayout)), // or map to existing API if available
            NormalizedV:   0,
            Rationale:     "normalized in API after header parse",
        })
    }

    // Pass debug to renderer
    pf := convertToParserFont(f)
    return renderer.RenderTo(w, text, pf, options.toInternal())
}

// Modify toInternal to pass debug handle
func (o *options) toInternal() renderer.Options {
    opts := renderer.Options{ TrimWhitespace: o.trimWhitespace, Debug: o.debug }
    // ... rest of conversion ...
    return opts
}
```

#### Parser Integration (`internal/parser/parser.go`)

```go
// Keep parser free of figgo imports to avoid cycles.
// Optionally, emit summary events only (no normalization here).
// After parsing:
//  - FontHeader (hardblank, height, baseline, maxlen, old_layout, print_dir, comment_lines)
//  - GlyphStats (present/missing required glyphs)
// Leave LayoutNormalized to API layer where normalization already happens.
```

#### Renderer Integration (`internal/renderer/`)

##### `types.go` - Add debug to structures

```go
// Add to Options struct
type Options struct {
    Layout         int
    PrintDirection *int
    UnknownRune    *rune
    TrimWhitespace bool
    Width          *int
    Debug          *debug.Debug  // NEW: Session debug handle
}

// Add to renderState struct
type renderState struct {
    // ... existing fields ...
    debug *debug.Debug  // NEW: Fast-path field check
}
```

##### `renderer.go` - Core renderer hooks

```go
func RenderTo(output io.Writer, text string, font *parser.Font, options Options) error {
    // Initialize state with debug handle
    state := &renderState{
        // ... existing initialization ...
        debug: options.Debug,  // Carry debug handle for fast checks
    }
    
    // Log render start
    if state.debug != nil {
        state.debug.Emit("render", "Start", debug.StartData{
            CharHeight:   state.charHeight,
            Hardblank:    int(state.hardblank),
            WidthLimit:   state.outlineLenLimit,
            PrintDir:     state.right2left,
            SmushMode:    state.smushMode,
        })
    }
    
    // ... existing rendering ...
    
    // Log render end
    if state.debug != nil {
        elapsed := time.Since(startTime).Milliseconds()
        state.debug.Emit("render", "End", debug.EndData{
            TotalLines: lineCount,
            TotalRunes: runeCount,
            ElapsedMs:  elapsed,
        })
    }
    
    return nil
}

// Modify addChar to log glyph addition
func (state *renderState) addChar(ch rune) bool {
    // ... existing logic to get glyph ...
    
    // Log glyph processing
    if state.debug != nil {
        state.debug.Emit("render", "Glyph", debug.GlyphData{
            Index:        state.inputCount,
            Rune:         int(ch),
            GlyphWidth:   state.currentCharWidth,
            SpaceGlyph:   state.processingSpaceGlyph,
            UnknownSubst: wasSubstituted,
        })
    }
    
    // ... rest of addChar logic ...
    
    // Log row append after successful addition
    if state.debug != nil && added {
        for row := 0; row < state.charHeight; row++ {
            state.debug.Emit("render", "RowAppend", debug.RowAppendData{
                Row:       row,
                StartPos:  startPos[row],
                CharCount: charCount[row],
                EndBefore: endBefore[row],
                EndAfter:  state.rowLengths[row],
            })
        }
    }
    
    return added
}

// Modify splitLine to log splits
func (state *renderState) splitLine() {
    if state.debug != nil {
        reason := "width"
        if state.wordbreakmode > 0 {
            reason = "wordbreak"
        }
        
        state.debug.Emit("render", "Split", debug.SplitData{
            Reason:     reason,
            FSMPrev:    prevFSM,
            FSMNext:    state.wordbreakmode,
            OutlineLen: state.outlineLen,
        })
    }
    
    // ... existing split logic ...
}

// Modify flushLine to log flush
func (state *renderState) flushLine() {
    if state.debug != nil {
        // Cap row lengths to avoid huge output
        before := state.rowLengths
        if len(before) > 32 {
            before = before[:32]
        }
        
        state.debug.Emit("render", "Flush", debug.FlushData{
            RowLengthsBefore: before,
        })
    }
    
    // ... existing flush logic ...
    
    if state.debug != nil {
        after := state.rowLengths
        if len(after) > 32 {
            after = after[:32]
        }
        // Update with after state
        state.debug.Emit("render", "FlushAfter", map[string][]int{
            "row_lengths_after": after,
        })
    }
}
```

##### `smushing.go` - Detailed smush logging

```go
// Keep a single implementation; gate event emission with state.debug != nil
func (state *renderState) smushAmount() int {
    maxSmush := state.currentCharWidth
    for row := 0; row < state.charHeight; row++ {
        // ... existing boundary and amt calculation ...
        plus := "none"
        final := amt
        if ch1 == 0 { plus = "ch1_null"; final = amt + 1 } else if ch2 != 0 && state.smush(ch1, ch2) != 0 { plus = "smush(l,r)"; final = amt + 1 }
        if state.debug != nil {
            state.debug.Emit("render", "SmushAmountRow", debug.SmushAmountRowData{
                GlyphIdx:        state.inputCount,
                Row:             row,
                LineBoundaryIdx: lineBoundary,
                CharBoundaryIdx: charBoundary,
                Ch1Code:         int(ch1),
                Ch2Code:         int(ch2),
                AmtBefore:       amt,
                PlusReason:      plus,
                AmtAfter:        final,
                RTL:             state.right2left != 0,
            })
        }
        if final < maxSmush { maxSmush = final }
    }
    return maxSmush
}

// Keep smush() pure; emit SmushDecision at call sites in addChar() where row/col are known.
// Add a debug-only helper to classify the rule when result != 0:
//   func (state *renderState) ruleName(lch, rch, result rune) string { ... }
```

#### CLI Integration (`cmd/figgo/main.go`)

```go
// Add flags: --debug, --debug-file, --debug-pretty. Initialize in run() and map to env.
// Use a single session sink; pass via figgo.WithDebug to rendering. No changes to ParseFont signature.
// Keep existing run() int pattern and defer session.Close().
```

### 3. Testing Strategy

#### Unit Tests (`internal/debug/debug_test.go`)

```go
package debug

import (
    "bytes"
    "encoding/json"
    "testing"
)

func TestDebugDisabled(t *testing.T) {
    // Ensure debug is disabled
    SetEnabled(false)
    
    var buf bytes.Buffer
    sink := NewJSONSink(&buf)
    d := StartSession(sink)
    
    // Should return nil when disabled
    if d != nil {
        t.Error("StartSession should return nil when disabled")
    }
    
    // Emit should be no-op
    d.Emit("test", "Event", nil)
    
    if buf.Len() > 0 {
        t.Error("Events emitted when debug disabled")
    }
}

func TestJSONOutput(t *testing.T) {
    SetEnabled(true)
    defer SetEnabled(false)
    
    var buf bytes.Buffer
    sink := NewJSONSink(&buf)
    d := StartSession(sink)
    
    // Emit test event
    d.Emit("test", "TestEvent", map[string]string{
        "key": "value",
    })
    
    d.Close()
    
    // Parse and verify JSON
    var event Event
    if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
        t.Fatalf("Failed to parse JSON: %v", err)
    }
    
    if event.Phase != "test" || event.Event != "TestEvent" {
        t.Errorf("Unexpected event: %+v", event)
    }
}

func TestPrettyOutput(t *testing.T) {
    SetEnabled(true)
    defer SetEnabled(false)
    
    var buf bytes.Buffer
    sink := NewPrettySink(&buf)
    d := StartSession(sink)
    
    d.Emit("render", "SmushAmountRow", SmushAmountRowData{
        Row:         1,
        Ch1Code:     124,  // '|'
        Ch2Code:     47,   // '/'
        AmtBefore:   2,
        PlusReason:  "smush(|,/)",
        AmtAfter:    3,
    })
    
    output := buf.String()
    if !strings.Contains(output, "'|' (0x7C)") {
        t.Error("Pretty output should show rune with code")
    }
}

// Benchmark to verify zero overhead when disabled
func BenchmarkEmitDisabled(b *testing.B) {
    SetEnabled(false)
    var buf bytes.Buffer
    sink := NewJSONSink(&buf)
    d := StartSession(sink)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        d.Emit("test", "Event", nil)
    }
    
    if buf.Len() > 0 {
        b.Error("Buffer should be empty when disabled")
    }
}

// Benchmark hot path with debug
func BenchmarkSmushWithDebug(b *testing.B) {
    // Compare performance with and without debug
    // Ensure minimal overhead in hot loops
}
```

#### Golden Test Integration (`golden_test.go`)

```go
// Modify golden test to support debug on failure
func TestGoldenFiles(t *testing.T) {
    // ... existing test setup ...
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // ... existing test logic ...
            
            if !match && os.Getenv("FIGGO_DEBUG") == "1" {
                // Re-run with debug enabled
                debugFile := fmt.Sprintf("testdata/debug/%s.jsonl", tc.name)
                
                // Create debug sink
                f, _ := os.Create(debugFile)
                defer f.Close()
                sink := debug.NewJSONSink(f)
                d := debug.StartSession(sink)
                defer d.Close()
                
                // Re-render with debug
                _, _ = font.Render(tc.input, figgo.WithDebug(d))
                
                // Log debug file location
                t.Logf("Debug trace saved to %s", debugFile)
                
                // Extract and display key decisions
                firstSmush := extractFirstSmushDecision(debugFile)
                if firstSmush != "" {
                    t.Logf("First smush decision: %s", firstSmush)
                }
            }
        })
    }
}
```

### 4. Example Outputs

#### JSON Lines Format (Default)

```json
{"ts":"2025-01-23T10:15:30.123456Z","session_id":"abc123","phase":"parse","event":"FontHeader","data":{"hardblank":36,"height":6,"baseline":5,"max_length":16,"old_layout":15143,"full_layout":0,"print_dir":0,"comment_lines":23}}
{"ts":"2025-01-23T10:15:30.124567Z","session_id":"abc123","phase":"parse","event":"LayoutNormalized","data":{"old_layout":15143,"full_layout":0,"full_layout_set":false,"normalized_horz":128,"normalized_vert":0,"rationale":"Using OldLayout as FullLayout not set"}}
{"ts":"2025-01-23T10:15:30.234567Z","session_id":"abc123","phase":"render","event":"Start","data":{"char_height":6,"hardblank":36,"width_limit":80,"print_dir":0,"smush_mode":199}}
{"ts":"2025-01-23T10:15:30.345678Z","session_id":"abc123","phase":"render","event":"SmushAmountRow","data":{"glyph_idx":2,"row":1,"line_boundary_idx":8,"char_boundary_idx":0,"ch1_code":124,"ch2_code":47,"amt_before":2,"plus_reason":"smush(|,/)","amt_after":3,"rtl":false}}
{"ts":"2025-01-23T10:15:30.456789Z","session_id":"abc123","phase":"render","event":"SmushDecision","data":{"row":1,"col":8,"lch_code":124,"rch_code":47,"result_code":124,"rule":"hierarchy"}}
```

#### Pretty Format (Human-Readable)

```
[2025-01-23T10:15:30.123456Z] [parse/FontHeader]
  session: abc123
  hardblank: '$' (0x24)
  height: 6, baseline: 5
  max_length: 16
  old_layout: 15143, full_layout: 0
  print_dir: LTR

[2025-01-23T10:15:30.234567Z] [render/Start]
  session: abc123
  char_height: 6
  hardblank: '$' (0x24)
  width_limit: 80
  print_dir: LTR
  smush_mode: 0xC7 (SMSmush | SMKern | Equal | Lowline | Hierarchy)

[2025-01-23T10:15:30.345678Z] [render/SmushAmountRow]
  session: abc123
  glyph: 2, row: 1
  boundaries: line=8('|' (0x7C)), char=0('/' (0x2F))
  amount: 2 +1 (smush) → 3

[2025-01-23T10:15:30.456789Z] [render/SmushDecision]
  session: abc123
  position: row=1, col=8
  characters: '|' (0x7C) + '/' (0x2F)
  result: '|' (0x7C)
  rule: hierarchy
```

### 5. Performance Considerations

#### Zero Overhead When Disabled

- Single atomic load for global enabled check
- Nil pointer check in hot paths (single CPU instruction)
- No allocations or string building when disabled
- Inline-able fast paths

#### Minimal Overhead When Enabled

- Pre-allocated event structs where possible
- Numeric codepoints avoid string encoding
- Buffered I/O for output
- Rule name computation only when debugging
- Lazy formatting in sinks

#### Hot Path Optimization

```go
// Fast path in every hot function
if state.debug == nil {
    return state.doWorkInternal()
}
// Debug path with logging
return state.doWorkWithDebug()
```

### 6. Usage Examples

#### Environment Variable

```bash
# Basic debug to stderr
FIGGO_DEBUG=1 ./figgo "Hello"

# Debug to file with JSON
FIGGO_DEBUG=1 FIGGO_DEBUG_FILE=trace.jsonl ./figgo "Hello"

# Pretty format to stderr
FIGGO_DEBUG=1 FIGGO_DEBUG_JSON=0 ./figgo "Hello"
```

#### CLI Flags

```bash
# Debug to stderr (JSON)
./figgo --debug "Hello"

# Debug to file (JSON)
./figgo --debug --debug-file trace.jsonl "Hello"

# Debug with pretty format
./figgo --debug --debug-pretty "Hello"
```

#### Programmatic API

```go
// Create debug session
var debugHandle *debug.Debug
if os.Getenv("FIGGO_DEBUG") == "1" {
    sink := debug.NewJSONSink(os.Stderr)
    debugHandle = debug.StartSession(sink)
    defer debugHandle.Close()
}

// Parse font with debug
font, err := figgo.ParseFontWithDebug(reader, debugHandle)

// Render with debug
output, err := font.Render("Hello World",
    figgo.WithDebug(debugHandle),
    figgo.WithLayout(figgo.FitSmushing),
)
```

#### Analysis Tools

```bash
# Count smush decisions by rule
cat trace.jsonl | jq -r 'select(.event=="SmushDecision") | .data.rule' | sort | uniq -c

# Find all hierarchy rule applications
cat trace.jsonl | jq 'select(.event=="SmushDecision" and .data.rule=="hierarchy")'

# Extract overlap amounts
cat trace.jsonl | jq -r 'select(.event=="SmushAmountRow") | "\(.data.row): \(.data.amt_before) → \(.data.amt_after)"'

# Get timing information
cat trace.jsonl | jq 'select(.event=="End") | .data.elapsed_ms'
```

### 7. Implementation Order

1. **Phase 1: Core Infrastructure**
   - Create `internal/debug/` package
   - Implement basic `debug.go`, `sink.go`, `events.go`
   - Add unit tests

2. **Phase 2: API Integration**
   - Modify `figgo.go` to accept debug handle
   - Add `WithDebug()` option
   - Pass through to renderer

3. **Phase 3: Parser Integration**
   - Add debug parameter to `Parse()`
   - Log header, normalization, stats
   - Test with sample fonts

4. **Phase 4: Renderer Integration**
   - Add debug to `renderState`
   - Implement fast-path checks
   - Add basic Start/End events

5. **Phase 5: Detailed Renderer Logging**
   - Split `smushAmount` into internal/debug versions
   - Add per-row logging
   - Log smush decisions with rules

6. **Phase 6: CLI Integration**
   - Add CLI flags
   - Environment variable support
   - Test end-to-end

7. **Phase 7: Testing & Documentation**
   - Golden test integration
   - Performance benchmarks
   - Usage documentation

### 8. Success Criteria

- [ ] Single switch enables all debugging
- [ ] Zero overhead when disabled (benchmarks confirm)
- [ ] All smush decisions logged with rule names
- [ ] Per-row overlap calculations visible
- [ ] Layout normalization rationale captured
- [ ] JSON Lines format parseable
- [ ] Pretty format human-readable
- [ ] Session IDs prevent log mixing
- [ ] Golden tests can save debug traces
- [ ] CLI flags work as specified

### 9. Future Enhancements

- Web-based trace viewer
- Diff tool for comparing traces
- Statistical analysis mode
- Performance profiling integration
- Replay mode from trace files
- Visual overlay showing decisions

## Conclusion

This debug system provides comprehensive visibility into figgo's rendering pipeline with minimal performance impact. The single-switch design makes it easy to enable when needed, while the structured output enables both human analysis and automated tooling. The surgical integration approach ensures the core rendering logic remains clean and maintainable.
