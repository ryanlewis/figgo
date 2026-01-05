// Package debug provides comprehensive tracing for figgo's rendering pipeline.
//
// The debug system follows these principles:
//   - Single switch: FIGGO_DEBUG=1 or --debug enables everything
//   - Zero overhead: No performance impact when disabled
//   - Session scoped: Each render gets unique session ID for concurrent safety
//   - Machine parsable: JSON Lines by default, pretty format optional
package debug

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"sync/atomic"
	"time"
)

// enabled is the global debug flag - set once at startup.
var enabled uint32

// SetEnabled configures debug mode globally.
// This should be called once at program startup.
func SetEnabled(on bool) {
	if on {
		atomic.StoreUint32(&enabled, 1)
	} else {
		atomic.StoreUint32(&enabled, 0)
	}
}

// Enabled returns true if debug mode is active.
func Enabled() bool {
	return atomic.LoadUint32(&enabled) == 1
}

// InitFromEnv initialises debug settings from environment variables.
// Recognised variables:
//   - FIGGO_DEBUG=1: Enable debug mode
//   - FIGGO_DEBUG_PRETTY=1: Use pretty output format
func InitFromEnv() {
	if os.Getenv("FIGGO_DEBUG") == "1" {
		SetEnabled(true)
	}
}

// Session represents a debug session for a single render operation.
// Sessions are safe for concurrent use within a single render but should
// not be shared across multiple concurrent renders.
type Session struct {
	sessionID string
	sink      Sink
	startTime time.Time
}

// NewSession creates a new debug session with the provided sink.
// Returns nil if debug mode is not enabled.
func NewSession(sink Sink) *Session {
	if !Enabled() {
		return nil
	}
	if sink == nil {
		return nil
	}

	s := &Session{
		sessionID: generateSessionID(),
		sink:      sink,
		startTime: time.Now(),
	}

	// Emit session start event
	s.Emit("session", "Start", map[string]interface{}{
		"version": "1.0",
	})

	return s
}

// SessionID returns the unique identifier for this session.
func (s *Session) SessionID() string {
	if s == nil {
		return ""
	}
	return s.sessionID
}

// Emit sends an event to the sink.
// This is a no-op if the session is nil (fast-path for disabled debug).
func (s *Session) Emit(phase, event string, data interface{}) {
	if s == nil {
		return
	}

	evt := Event{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		SessionID: s.sessionID,
		Phase:     phase,
		Event:     event,
		Data:      data,
	}

	// Write errors are intentionally ignored - debug failures should not break normal operation
	//nolint:errcheck // Debug sink errors are non-critical
	s.sink.Write(evt)
}

// Close flushes and closes the debug session.
// This should be called when the render operation completes.
func (s *Session) Close() error {
	if s == nil {
		return nil
	}

	// Emit session end event
	elapsed := time.Since(s.startTime).Milliseconds()
	s.Emit("session", "End", map[string]int64{
		"elapsed_ms": elapsed,
	})

	return s.sink.Close()
}

// generateSessionID creates a unique session identifier.
func generateSessionID() string {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to time-based ID if crypto/rand fails
		return hex.EncodeToString([]byte{
			byte(time.Now().UnixNano() >> 24),
			byte(time.Now().UnixNano() >> 16),
			byte(time.Now().UnixNano() >> 8),
			byte(time.Now().UnixNano()),
		})
	}
	return hex.EncodeToString(b)
}

// Event is the base envelope for all debug events.
type Event struct {
	Timestamp string      `json:"ts"`
	SessionID string      `json:"session_id"`
	Phase     string      `json:"phase"`
	Event     string      `json:"event"`
	Data      interface{} `json:"data"`
}
