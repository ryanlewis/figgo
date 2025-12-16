package debug

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Sink is the interface for debug output destinations.
type Sink interface {
	Write(event Event) error
	Flush() error
	Close() error
}

// JSONSink writes events in JSON Lines format.
type JSONSink struct {
	w       *bufio.Writer
	encoder *json.Encoder
}

// NewJSONSink creates a new JSON Lines sink writing to w.
func NewJSONSink(w io.Writer) *JSONSink {
	bw := bufio.NewWriter(w)
	return &JSONSink{
		w:       bw,
		encoder: json.NewEncoder(bw),
	}
}

// Write encodes and writes an event as a JSON line.
func (s *JSONSink) Write(event Event) error {
	return s.encoder.Encode(event)
}

// Flush writes any buffered data to the underlying writer.
func (s *JSONSink) Flush() error {
	return s.w.Flush()
}

// Close flushes the buffer.
func (s *JSONSink) Close() error {
	return s.Flush()
}

// PrettySink writes events in human-readable format.
type PrettySink struct {
	w *bufio.Writer
}

// NewPrettySink creates a new pretty-format sink writing to w.
func NewPrettySink(w io.Writer) *PrettySink {
	return &PrettySink{
		w: bufio.NewWriter(w),
	}
}

// Write formats and writes an event in human-readable format.
func (s *PrettySink) Write(event Event) error {
	// Format: [timestamp] [phase/event]
	fmt.Fprintf(s.w, "[%s] [%s/%s] session=%s\n", event.Timestamp, event.Phase, event.Event, event.SessionID)

	// Pretty print data based on type
	switch d := event.Data.(type) {
	case SmushAmountRowData:
		s.writeSmushAmountRow(d)
	case SmushDecisionData:
		s.writeSmushDecision(d)
	case RenderStartData:
		s.writeRenderStart(d)
	case RenderEndData:
		s.writeRenderEnd(d)
	case GlyphData:
		s.writeGlyph(d)
	case SplitData:
		s.writeSplit(d)
	case FlushData:
		s.writeFlush(d)
	case RowAppendData:
		s.writeRowAppend(d)
	case LayoutMergeData:
		s.writeLayoutMerge(d)
	case map[string]interface{}:
		s.writeMap(d)
	case map[string]int64:
		s.writeMapInt64(d)
	default:
		fmt.Fprintf(s.w, "  data: %+v\n", d)
	}

	return nil
}

func (s *PrettySink) writeSmushAmountRow(d SmushAmountRowData) {
	fmt.Fprintf(s.w, "  glyph: %d, row: %d\n", d.GlyphIdx, d.Row)
	fmt.Fprintf(s.w, "  boundaries: line=%d (%s), char=%d (%s)\n",
		d.LineBoundaryIdx, runeStr(d.Ch1),
		d.CharBoundaryIdx, runeStr(d.Ch2))
	fmt.Fprintf(s.w, "  amount: %d %s → %d\n", d.AmountBefore, d.Reason, d.AmountAfter)
	if d.RTL {
		fmt.Fprintf(s.w, "  direction: RTL\n")
	}
}

func (s *PrettySink) writeSmushDecision(d SmushDecisionData) {
	fmt.Fprintf(s.w, "  position: row=%d, col=%d\n", d.Row, d.Col)
	fmt.Fprintf(s.w, "  characters: %s + %s → %s\n",
		runeStr(d.Lch), runeStr(d.Rch), runeStr(d.Result))
	fmt.Fprintf(s.w, "  rule: %s\n", d.Rule)
}

func (s *PrettySink) writeRenderStart(d RenderStartData) {
	fmt.Fprintf(s.w, "  text: %q (length: %d)\n", d.Text, d.TextLength)
	fmt.Fprintf(s.w, "  char_height: %d, hardblank: %s\n", d.CharHeight, runeStr(d.Hardblank))
	fmt.Fprintf(s.w, "  width_limit: %d, print_dir: %s\n", d.WidthLimit, dirStr(d.PrintDir))
	fmt.Fprintf(s.w, "  smush_mode: 0x%02X (%s)\n", d.SmushMode, strings.Join(d.SmushRules, "|"))
}

func (s *PrettySink) writeRenderEnd(d RenderEndData) {
	fmt.Fprintf(s.w, "  total_lines: %d, total_runes: %d, total_glyphs: %d\n",
		d.TotalLines, d.TotalRunes, d.TotalGlyphs)
	fmt.Fprintf(s.w, "  elapsed_ms: %d, bytes_written: %d\n", d.ElapsedMs, d.BytesWritten)
}

func (s *PrettySink) writeGlyph(d GlyphData) {
	fmt.Fprintf(s.w, "  index: %d, rune: %s, width: %d\n", d.Index, runeStr(d.Rune), d.Width)
	if d.SpaceGlyph {
		fmt.Fprintf(s.w, "  space_glyph: true\n")
	}
	if d.UnknownSubst {
		fmt.Fprintf(s.w, "  unknown_subst: true\n")
	}
}

func (s *PrettySink) writeSplit(d SplitData) {
	fmt.Fprintf(s.w, "  reason: %s, position: %d\n", d.Reason, d.Position)
	fmt.Fprintf(s.w, "  fsm: %d → %d, outline_len: %d\n", d.FSMPrev, d.FSMNext, d.OutlineLen)
}

func (s *PrettySink) writeFlush(d FlushData) {
	fmt.Fprintf(s.w, "  line_number: %d\n", d.LineNumber)
	fmt.Fprintf(s.w, "  row_lengths_before: %v\n", d.RowLengthsBefore)
	fmt.Fprintf(s.w, "  row_lengths_after: %v\n", d.RowLengthsAfter)
}

func (s *PrettySink) writeRowAppend(d RowAppendData) {
	fmt.Fprintf(s.w, "  row: %d, start_pos: %d, char_count: %d\n", d.Row, d.StartPos, d.CharCount)
	fmt.Fprintf(s.w, "  end: %d → %d\n", d.EndBefore, d.EndAfter)
}

func (s *PrettySink) writeLayoutMerge(d LayoutMergeData) {
	fmt.Fprintf(s.w, "  requested: 0x%02X, font_defaults: 0x%02X\n", d.RequestedLayout, d.FontDefaults)
	fmt.Fprintf(s.w, "  injected_rules: 0x%02X, final: 0x%02X\n", d.InjectedRules, d.FinalLayout)
	fmt.Fprintf(s.w, "  final_smush_mode: 0x%02X\n", d.FinalSmushMode)
	fmt.Fprintf(s.w, "  rationale: %s\n", d.Rationale)
}

func (s *PrettySink) writeMap(d map[string]interface{}) {
	for k, v := range d {
		fmt.Fprintf(s.w, "  %s: %v\n", k, v)
	}
}

func (s *PrettySink) writeMapInt64(d map[string]int64) {
	for k, v := range d {
		fmt.Fprintf(s.w, "  %s: %d\n", k, v)
	}
}

// Flush writes any buffered data to the underlying writer.
func (s *PrettySink) Flush() error {
	return s.w.Flush()
}

// Close flushes the buffer.
func (s *PrettySink) Close() error {
	return s.Flush()
}

// runeStr formats a rune for display: 'X' (0x58) or NUL for 0.
func runeStr(r rune) string {
	if r == 0 {
		return "NUL"
	}
	if r >= 32 && r < 127 {
		return fmt.Sprintf("'%c' (0x%02X)", r, r)
	}
	return fmt.Sprintf("0x%02X", r)
}

// dirStr converts print direction to a string.
func dirStr(dir int) string {
	if dir == 0 {
		return "LTR"
	}
	return "RTL"
}
