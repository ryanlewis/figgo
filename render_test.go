package figgo

import (
	"strings"
	"sync"
	"testing"
)

// createTestFontForRender creates a minimal font for rendering tests
func createTestFontForRender() (*Font, error) {
	// We need to use the createTestFont from figgo_test.go approach
	// but with modifications for our test cases
	fontData := `flf2a$ 4 3 10 -1 1
Test font
$@
$@
$@
$@@
H@
H@
H@
H@@
I@
I@
I@
I@@
 @
 @
 @
 @@
$@
$@
$@
$@@
~@
~@
~@
~@@
`
	// Create font with glyphs mapped to actual positions
	font, err := ParseFontBytes([]byte(fontData))
	if err != nil {
		return nil, err
	}

	// The parser maps these to ASCII positions 32-37
	// We need to manually adjust our font's glyphs map for testing
	// This is a workaround since we're not providing full ASCII set
	font.glyphs = map[rune][]string{
		' ': {" ", " ", " ", " "},
		'H': {"H", "H", "H", "H"},
		'I': {"I", "I", "I", "I"},
		'$': {"$", "$", "$", "$"},
		'~': {"~", "~", "~", "~"},
	}

	return font, nil
}

func TestRender_FullWidth(t *testing.T) {
	font, err := createTestFontForRender()
	if err != nil {
		t.Fatalf("Failed to create test font: %v", err)
	}

	tests := []struct {
		name    string
		text    string
		opts    []Option
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "simple HI with FitFullWidth",
			text: "HI",
			opts: []Option{WithLayout(FitFullWidth)},
			want: "HI\nHI\nHI\nHI",
		},
		{
			name: "single character H",
			text: "H",
			opts: []Option{WithLayout(FitFullWidth)},
			want: "H\nH\nH\nH",
		},
		{
			name: "with spaces",
			text: "H I",
			opts: []Option{WithLayout(FitFullWidth)},
			want: "H I\nH I\nH I\nH I",
		},
		{
			name: "RTL print direction",
			text: "HI",
			opts: []Option{
				WithLayout(FitFullWidth),
				WithPrintDirection(1),
			},
			want: "IH\nIH\nIH\nIH",
		},
		{
			name: "hardblank character $",
			text: "$",
			opts: []Option{WithLayout(FitFullWidth)},
			want: " \n \n \n ", // $ glyph contains hardblanks, which become spaces
		},
		{
			name: "character with tilde ~",
			text: "~",
			opts: []Option{WithLayout(FitFullWidth)},
			want: "~\n~\n~\n~",
		},
		{
			name:    "unsupported character",
			text:    "X", // We don't have X in our font
			opts:    []Option{WithLayout(FitFullWidth)},
			wantErr: true,
			errMsg:  "unsupported rune",
		},
		{
			name: "empty text",
			text: "",
			opts: []Option{WithLayout(FitFullWidth)},
			want: "\n\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.text, font, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Render() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}
			if got != tt.want {
				t.Errorf("Render() output mismatch:\ngot:\n%q\nwant:\n%q", got, tt.want)
				// Visual comparison
				t.Logf("Visual comparison:\nGot:\n%s\n\nWant:\n%s", got, tt.want)
			}
		})
	}
}

func TestRender_NilFont(t *testing.T) {
	_, err := Render("test", nil, WithLayout(FitFullWidth))
	if err == nil {
		t.Error("Expected error for nil font")
	}
	if err != ErrUnknownFont {
		t.Errorf("Expected ErrUnknownFont, got %v", err)
	}
}

func TestRender_DefaultToFontLayout(t *testing.T) {
	// Create a font with a specific default layout (-1 = Full-Width)
	fontData := `flf2a$ 4 3 10 -1 1
Test font with Full-Width default (OldLayout=-1)
 @
 @
 @
 @@
H@
H@
H@
H@@
`
	font, err := ParseFontBytes([]byte(fontData))
	if err != nil {
		t.Fatalf("Failed to parse font: %v", err)
	}

	// Manually set the H glyph for testing
	font.glyphs = map[rune][]string{
		'H': {"H", "H", "H", "H"},
	}

	// Render without specifying layout - should use font's default (Full-Width)
	output, err := Render("H", font)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	expected := "H\nH\nH\nH"
	if output != expected {
		t.Errorf("Default layout rendering mismatch:\ngot:\n%q\nwant:\n%q", output, expected)
	}
}

// TestRender_ConcurrentSameFont verifies concurrent Render calls with the same
// Font instance produce consistent, correct outputs without race conditions.
func TestRender_ConcurrentSameFont(t *testing.T) {
	font, err := createTestFontForRender()
	if err != nil {
		t.Fatalf("Failed to create test font: %v", err)
	}

	// Test cases to render concurrently
	testCases := []struct {
		text     string
		expected string
	}{
		{"H", "H\nH\nH\nH"},
		{"I", "I\nI\nI\nI"},
		{"HI", "HI\nHI\nHI\nHI"},
		{"H I", "H I\nH I\nH I\nH I"},
	}

	const goroutines = 50
	const iterations = 20

	var wg sync.WaitGroup
	errCh := make(chan error, goroutines*len(testCases)*iterations)

	// Spawn many goroutines all rendering with the same font
	for g := 0; g < goroutines; g++ {
		for _, tc := range testCases {
			wg.Add(1)
			go func(text, expected string) {
				defer wg.Done()
				for i := 0; i < iterations; i++ {
					output, err := Render(text, font, WithLayout(FitFullWidth))
					if err != nil {
						errCh <- err
						return
					}
					if output != expected {
						errCh <- &concurrentRenderError{
							text:     text,
							expected: expected,
							got:      output,
						}
						return
					}
				}
			}(tc.text, tc.expected)
		}
	}

	wg.Wait()
	close(errCh)

	// Check for errors
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("concurrent rendering produced %d errors", len(errors))
		for i, err := range errors {
			if i >= 5 {
				t.Errorf("  ... and %d more errors", len(errors)-5)
				break
			}
			t.Errorf("  error %d: %v", i+1, err)
		}
	}
}

type concurrentRenderError struct {
	text     string
	expected string
	got      string
}

func (e *concurrentRenderError) Error() string {
	return "output mismatch for " + e.text + ": got " + e.got + ", want " + e.expected
}
