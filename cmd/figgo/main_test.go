package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestParseUnknownRune(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    rune
		wantErr bool
	}{
		// Literal characters
		{"literal star", "*", '*', false},
		{"literal question", "?", '?', false},
		{"literal underscore", "_", '_', false},
		{"literal unicode emoji", "☺", '☺', false},

		// Escaped Unicode
		{"unicode escape \\u2588", "\\u2588", '\u2588', false},
		{"unicode escape \\u003F", "\\u003F", '?', false},
		{"unicode escape \\U00002588", "\\U00002588", '\u2588', false},

		// Unicode notation
		{"unicode U+2588", "U+2588", '\u2588', false},
		{"unicode u+2588", "u+2588", '\u2588', false},
		{"unicode U+003F", "U+003F", '?', false},

		// Decimal
		{"decimal 63", "63", '?', false},
		{"decimal 42", "42", '*', false},
		{"decimal 9608", "9608", '█', false},

		// Hexadecimal
		{"hex 0x3F", "0x3F", '?', false},
		{"hex 0x2A", "0x2A", '*', false},
		{"hex 0X3F", "0X3F", '?', false},

		// Invalid inputs
		{"empty string", "", 0, true},
		{"invalid unicode escape", "\\u", 0, true},
		{"invalid unicode notation", "U+", 0, true},
		{"invalid hex", "0x", 0, true},
		{"multi-rune literal", "abc", 0, true},
		{"invalid format", "xyz", 0, true},

		// Invalid rune values
		{"beyond max rune", "U+110000", 0, true}, // > utf8.MaxRune
		{"negative decimal", "-1", 0, true},
		{"surrogate start", "U+D800", 0, true}, // UTF-16 surrogate
		{"surrogate end", "U+DFFF", 0, true},   // UTF-16 surrogate
		{"surrogate mid", "0xDC00", 0, true},   // UTF-16 surrogate

		// Exact length validation
		{"unicode escape too short", "\\u258", 0, true},
		{"unicode escape too long", "\\u25888", 0, true},
		{"unicode U escape too short", "\\U0002588", 0, true},
		{"unicode U escape too long", "\\U000025888", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseUnknownRune(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseUnknownRune(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseUnknownRune(%q) = %v (U+%04X), want %v (U+%04X)",
					tt.input, got, got, tt.want, tt.want)
			}
		})
	}
}

// projectRoot returns the absolute path to the project root directory.
func projectRoot() string {
	_, file, _, _ := runtime.Caller(0)
	// cmd/figgo/main_test.go -> project root
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

// TestCLIConcurrentSubprocesses verifies the CLI can be invoked concurrently
// without race conditions or interference between processes.
func TestCLIConcurrentSubprocesses(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	root := projectRoot()
	fontsDir := filepath.Join(root, "fonts")

	// Build the binary once for all concurrent invocations
	binPath := t.TempDir() + "/figgo-test"
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, out)
	}

	// Test cases with different inputs (using absolute font paths)
	inputs := []struct {
		text string
		font string
	}{
		{"Hello", filepath.Join(fontsDir, "standard.flf")},
		{"World", filepath.Join(fontsDir, "standard.flf")},
		{"Test", filepath.Join(fontsDir, "small.flf")},
		{"FIGlet", filepath.Join(fontsDir, "slant.flf")},
		{"Go", filepath.Join(fontsDir, "big.flf")},
		{"123", filepath.Join(fontsDir, "standard.flf")},
		{"ABC", filepath.Join(fontsDir, "small.flf")},
		{"XYZ", filepath.Join(fontsDir, "slant.flf")},
	}

	const concurrency = 10 // Number of concurrent invocations per input

	var wg sync.WaitGroup
	errCh := make(chan error, len(inputs)*concurrency)
	results := make(chan struct {
		input  string
		output string
	}, len(inputs)*concurrency)

	// Spawn multiple concurrent processes
	for _, tc := range inputs {
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(text, font string) {
				defer wg.Done()

				cmd := exec.Command(binPath, "-f", font, text)
				var stdout, stderr bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr

				if err := cmd.Run(); err != nil {
					errCh <- err
					return
				}

				results <- struct {
					input  string
					output string
				}{text, stdout.String()}
			}(tc.text, tc.font)
		}
	}

	wg.Wait()
	close(errCh)
	close(results)

	// Check for errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		t.Errorf("got %d errors during concurrent execution: %v", len(errs), errs[:min(5, len(errs))])
	}

	// Verify outputs are consistent (same input produces same output)
	outputsByInput := make(map[string]string)
	for r := range results {
		if existing, ok := outputsByInput[r.input]; ok {
			if existing != r.output {
				t.Errorf("inconsistent output for input %q:\nfirst:\n%s\nlater:\n%s",
					r.input, existing, r.output)
			}
		} else {
			outputsByInput[r.input] = r.output
		}
	}

	// Verify we got results for all inputs
	if len(outputsByInput) != len(inputs) {
		t.Errorf("expected results for %d inputs, got %d", len(inputs), len(outputsByInput))
	}
}
