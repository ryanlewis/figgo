package debug

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestDebugDisabled(t *testing.T) {
	// Ensure debug is disabled
	SetEnabled(false)
	defer SetEnabled(false)

	var buf bytes.Buffer
	sink := NewJSONSink(&buf)
	session := NewSession(sink)

	// Should return nil when disabled
	if session != nil {
		t.Error("NewSession should return nil when disabled")
	}

	// Emit should be no-op on nil session
	session.Emit("test", "Event", nil)

	if buf.Len() > 0 {
		t.Error("Events emitted when debug disabled")
	}
}

func TestDebugEnabled(t *testing.T) {
	SetEnabled(true)
	defer SetEnabled(false)

	var buf bytes.Buffer
	sink := NewJSONSink(&buf)
	session := NewSession(sink)

	if session == nil {
		t.Fatal("NewSession should return non-nil when enabled")
	}

	// Emit test event
	session.Emit("test", "TestEvent", map[string]string{
		"key": "value",
	})

	if err := session.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Parse and verify JSON lines
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) < 3 { // Start, TestEvent, End
		t.Fatalf("Expected at least 3 lines, got %d", len(lines))
	}

	// Verify first event is session start
	var startEvent Event
	if err := json.Unmarshal([]byte(lines[0]), &startEvent); err != nil {
		t.Fatalf("Failed to parse start event: %v", err)
	}
	if startEvent.Phase != "session" || startEvent.Event != "Start" {
		t.Errorf("Expected session/Start, got %s/%s", startEvent.Phase, startEvent.Event)
	}

	// Verify test event
	var testEvent Event
	if err := json.Unmarshal([]byte(lines[1]), &testEvent); err != nil {
		t.Fatalf("Failed to parse test event: %v", err)
	}
	if testEvent.Phase != "test" || testEvent.Event != "TestEvent" {
		t.Errorf("Expected test/TestEvent, got %s/%s", testEvent.Phase, testEvent.Event)
	}
	if testEvent.SessionID == "" {
		t.Error("Session ID should not be empty")
	}

	// Verify last event is session end
	var endEvent Event
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &endEvent); err != nil {
		t.Fatalf("Failed to parse end event: %v", err)
	}
	if endEvent.Phase != "session" || endEvent.Event != "End" {
		t.Errorf("Expected session/End, got %s/%s", endEvent.Phase, endEvent.Event)
	}
}

func TestJSONSink(t *testing.T) {
	var buf bytes.Buffer
	sink := NewJSONSink(&buf)

	event := Event{
		Timestamp: "2025-01-01T00:00:00Z",
		SessionID: "abc123",
		Phase:     "test",
		Event:     "TestEvent",
		Data:      map[string]int{"count": 42},
	}

	if err := sink.Write(event); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := sink.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	var parsed Event
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed.Phase != "test" || parsed.Event != "TestEvent" {
		t.Errorf("Unexpected event: %+v", parsed)
	}
}

func TestPrettySink(t *testing.T) {
	var buf bytes.Buffer
	sink := NewPrettySink(&buf)

	event := Event{
		Timestamp: "2025-01-01T00:00:00Z",
		SessionID: "abc123",
		Phase:     "render",
		Event:     "SmushAmountRow",
		Data: SmushAmountRowData{
			GlyphIdx:        2,
			Row:             1,
			LineBoundaryIdx: 8,
			CharBoundaryIdx: 0,
			Ch1:             '|',
			Ch2:             '/',
			AmountBefore:    2,
			AmountAfter:     3,
			Reason:          "smushable",
			RTL:             false,
		},
	}

	if err := sink.Write(event); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := sink.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "'|' (0x7C)") {
		t.Errorf("Pretty output should show rune with code, got: %s", output)
	}
	if !strings.Contains(output, "smushable") {
		t.Errorf("Pretty output should show reason, got: %s", output)
	}
}

func TestFormatSmushRules(t *testing.T) {
	tests := []struct {
		name      string
		smushMode int
		want      []string
	}{
		{"none", 0, []string{"None"}},
		{"kern only", 64, []string{"SMKern"}},
		{"smush only", 128, []string{"SMSmush"}},
		{"smush with equal", 128 | 1, []string{"SMSmush", "Equal"}},
		{"smush all rules", 128 | 63, []string{"SMSmush", "Equal", "Lowline", "Hierarchy", "Pair", "BigX", "Hardblank"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSmushRules(tt.smushMode)
			if len(got) != len(tt.want) {
				t.Errorf("FormatSmushRules(%d) = %v, want %v", tt.smushMode, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("FormatSmushRules(%d)[%d] = %v, want %v", tt.smushMode, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestClassifySmushRule(t *testing.T) {
	tests := []struct {
		name      string
		lch       rune
		rch       rune
		result    rune
		smushMode int
		want      string
	}{
		{"space left", ' ', 'X', 'X', 128, "space"},
		{"space right", 'X', ' ', 'X', 128, "space"},
		{"universal", 'A', 'B', 'B', 128, "universal"},
		{"equal", 'X', 'X', 'X', 128 | 1, "equal"},
		{"underscore", '_', '|', '|', 128 | 2, "underscore"},
		{"hierarchy", '|', '/', '|', 128 | 4, "hierarchy"},
		{"pair", '[', ']', '|', 128 | 8, "pair"},
		{"bigx slash", '/', '\\', '|', 128 | 16, "bigx"},
		{"bigx backslash", '\\', '/', 'Y', 128 | 16, "bigx"},
		{"bigx angle", '>', '<', 'X', 128 | 16, "bigx"},
		{"kerning", 'A', 'B', 'B', 64, "kerning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifySmushRule(tt.lch, tt.rch, tt.result, tt.smushMode)
			if got != tt.want {
				t.Errorf("ClassifySmushRule('%c', '%c', '%c', %d) = %v, want %v",
					tt.lch, tt.rch, tt.result, tt.smushMode, got, tt.want)
			}
		})
	}
}

func TestSessionID(t *testing.T) {
	SetEnabled(true)
	defer SetEnabled(false)

	var buf bytes.Buffer
	sink := NewJSONSink(&buf)
	session := NewSession(sink)

	if session == nil {
		t.Fatal("NewSession should return non-nil when enabled")
	}

	id := session.SessionID()
	if id == "" {
		t.Error("SessionID should not be empty")
	}
	if len(id) != 8 { // 4 bytes hex encoded = 8 characters
		t.Errorf("SessionID should be 8 characters, got %d", len(id))
	}

	session.Close()
}

func TestNilSessionSafety(t *testing.T) {
	// All operations on nil session should be safe
	var session *Session

	// Should not panic
	session.Emit("test", "Event", nil)

	if err := session.Close(); err != nil {
		t.Errorf("Close on nil session should return nil, got %v", err)
	}

	if id := session.SessionID(); id != "" {
		t.Errorf("SessionID on nil session should return empty, got %v", id)
	}
}

// BenchmarkEmitDisabled verifies zero overhead when debug is disabled.
func BenchmarkEmitDisabled(b *testing.B) {
	SetEnabled(false)

	var buf bytes.Buffer
	sink := NewJSONSink(&buf)
	session := NewSession(sink)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.Emit("test", "Event", nil)
	}

	if buf.Len() > 0 {
		b.Error("Buffer should be empty when disabled")
	}
}

// BenchmarkEmitEnabled measures overhead when debug is enabled.
func BenchmarkEmitEnabled(b *testing.B) {
	SetEnabled(true)
	defer SetEnabled(false)

	var buf bytes.Buffer
	sink := NewJSONSink(&buf)
	session := NewSession(sink)

	data := SmushAmountRowData{
		GlyphIdx:        1,
		Row:             0,
		LineBoundaryIdx: 5,
		CharBoundaryIdx: 0,
		Ch1:             '|',
		Ch2:             '/',
		AmountBefore:    2,
		AmountAfter:     3,
		Reason:          "smushable",
		RTL:             false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.Emit("render", "SmushAmountRow", data)
	}
}
