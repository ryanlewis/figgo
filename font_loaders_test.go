package figgo

import (
	"bytes"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

// testFonts will be used when actual font files exist
// For now, tests use fstest.MapFS

// Test ParseFont with valid font data
func TestParseFont(t *testing.T) {
	tests := []struct {
		name      string
		fontData  string
		wantErr   bool
		checkFont func(t *testing.T, f *Font)
	}{
		{
			name: "valid minimal font",
			fontData: `flf2a$ 4 3 10 -1 5
Test font
$@
$@
$@
$@@
 @
|@
 @
 @@
`,
			wantErr: false,
			checkFont: func(t *testing.T, f *Font) {
				if f == nil {
					t.Fatal("expected non-nil font")
				}
				if f.Height != 4 {
					t.Errorf("Height = %d, want 4", f.Height)
				}
				if f.Baseline != 3 {
					t.Errorf("Baseline = %d, want 3", f.Baseline)
				}
				if f.MaxLen != 10 {
					t.Errorf("MaxLen = %d, want 10", f.MaxLen)
				}
				if f.OldLayout != -1 {
					t.Errorf("OldLayout = %d, want -1", f.OldLayout)
				}
				if f.Hardblank != '$' {
					t.Errorf("Hardblank = %c, want $", f.Hardblank)
				}
			},
		},
		{
			name: "font with full layout",
			fontData: `flf2a$ 4 3 10 15 5 0 24463
Test font with full layout
$@
$@
$@
$@@
 @
|@
 @
 @@
`,
			wantErr: false,
			checkFont: func(t *testing.T, f *Font) {
				if f == nil {
					t.Fatal("expected non-nil font")
				}
				// With OldLayout=15 and FullLayout=24463, layout should be normalized
				// OldLayout 15 = kerning + all horizontal smushing rules
				// FullLayout 24463 should be validated and normalized
				if f.FullLayout == 0 {
					t.Error("FullLayout should be non-zero after normalization")
				}
			},
		},
		{
			name:     "empty reader",
			fontData: "",
			wantErr:  true,
		},
		{
			name:     "invalid header",
			fontData: "not a valid font header\n",
			wantErr:  true,
		},
		{
			name: "missing glyphs",
			fontData: `flf2a$ 4 3 10 -1 5
Test font
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.fontData)
			font, err := ParseFont(r)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFont() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFont != nil {
				tt.checkFont(t, font)
			}
		})
	}
}

// Test ParseFont with io.Reader edge cases
func TestParseFontReaderBehavior(t *testing.T) {
	validFont := `flf2a$ 4 3 10 -1 5
Test font
$@
$@
$@
$@@
 @
|@
 @
 @@
`

	t.Run("multiple reads from same reader", func(t *testing.T) {
		r := strings.NewReader(validFont)

		// First parse should succeed
		font1, err := ParseFont(r)
		if err != nil {
			t.Fatalf("first ParseFont() error = %v", err)
		}
		if font1 == nil {
			t.Fatal("first ParseFont() returned nil font")
		}

		// Second parse should fail as reader is exhausted
		font2, err := ParseFont(r)
		if err == nil {
			t.Error("second ParseFont() should fail on exhausted reader")
		}
		if font2 != nil {
			t.Error("second ParseFont() should return nil font")
		}
	})

	t.Run("reader that returns error", func(t *testing.T) {
		expectedErr := errors.New("read error")
		r := &errorReader{err: expectedErr}

		font, err := ParseFont(r)
		if err == nil {
			t.Error("ParseFont() should propagate reader error")
		}
		if font != nil {
			t.Error("ParseFont() should return nil font on error")
		}
	})

	t.Run("large font data", func(t *testing.T) {
		// Create a large but valid font header
		var buf bytes.Buffer
		buf.WriteString("flf2a$ 4 3 10 -1 5\n")
		buf.WriteString("Test font\n")

		// Add minimal glyphs
		for i := 0; i < 2; i++ {
			buf.WriteString("$@\n$@\n$@\n$@@\n")
		}

		font, err := ParseFont(&buf)
		if err != nil {
			t.Errorf("ParseFont() with large data error = %v", err)
		}
		if font == nil {
			t.Error("ParseFont() should return valid font")
		}
	})
}

// Test LoadFontFS with valid filesystem
func TestLoadFontFS(t *testing.T) {
	// Create a test filesystem
	testFS := fstest.MapFS{
		"fonts/standard.flf": &fstest.MapFile{
			Data: []byte(`flf2a$ 4 3 10 -1 5
Standard font
$@
$@
$@
$@@
 @
|@
 @
 @@
`),
		},
		"fonts/slant.flf": &fstest.MapFile{
			Data: []byte(`flf2a# 4 3 10 -1 5
Slant font
#@
#@
#@
#@@
 @
|@
 @
 @@
`),
		},
		"fonts/subdir/mini.flf": &fstest.MapFile{
			Data: []byte(`flf2a@ 3 2 4 -1 3
Mini font
 @
 @
 @@
|@
 @
 @@
`),
		},
		"fonts/invalid.txt": &fstest.MapFile{
			Data: []byte("not a font file"),
		},
		"fonts/empty.flf": &fstest.MapFile{
			Data: []byte(""),
		},
	}

	tests := []struct {
		name      string
		fs        fs.FS
		path      string
		wantErr   bool
		checkFont func(t *testing.T, f *Font)
	}{
		{
			name:    "load standard font",
			fs:      testFS,
			path:    "fonts/standard.flf",
			wantErr: false,
			checkFont: func(t *testing.T, f *Font) {
				if f == nil {
					t.Fatal("expected non-nil font")
				}
				if f.Height != 4 {
					t.Errorf("Height = %d, want 4", f.Height)
				}
				if f.Hardblank != '$' {
					t.Errorf("Hardblank = %c, want $", f.Hardblank)
				}
			},
		},
		{
			name:    "load slant font",
			fs:      testFS,
			path:    "fonts/slant.flf",
			wantErr: false,
			checkFont: func(t *testing.T, f *Font) {
				if f == nil {
					t.Fatal("expected non-nil font")
				}
				if f.Height != 4 {
					t.Errorf("Height = %d, want 4", f.Height)
				}
				if f.Hardblank != '#' {
					t.Errorf("Hardblank = %c, want #", f.Hardblank)
				}
			},
		},
		{
			name:    "load font from subdirectory",
			fs:      testFS,
			path:    "fonts/subdir/mini.flf",
			wantErr: false,
			checkFont: func(t *testing.T, f *Font) {
				if f == nil {
					t.Fatal("expected non-nil font")
				}
				if f.Height != 3 {
					t.Errorf("Height = %d, want 3", f.Height)
				}
			},
		},
		{
			name:    "non-existent file",
			fs:      testFS,
			path:    "fonts/missing.flf",
			wantErr: true,
		},
		{
			name:    "invalid font file",
			fs:      testFS,
			path:    "fonts/invalid.txt",
			wantErr: true,
		},
		{
			name:    "empty font file",
			fs:      testFS,
			path:    "fonts/empty.flf",
			wantErr: true,
		},
		{
			name:    "nil filesystem",
			fs:      nil,
			path:    "fonts/standard.flf",
			wantErr: true,
		},
		{
			name:    "empty path",
			fs:      testFS,
			path:    "",
			wantErr: true,
		},
		{
			name:    "path with .. traversal",
			fs:      testFS,
			path:    "../fonts/standard.flf",
			wantErr: true,
		},
		{
			name:    "path with .. in filename (not traversal)",
			fs:      testFS,
			path:    "fonts/my..font.flf",
			wantErr: true, // Will fail because file doesn't exist, not validation
		},
		{
			name:    "path with backslash",
			fs:      testFS,
			path:    "fonts\\standard.flf",
			wantErr: true,
		},
		{
			name:    "absolute path",
			fs:      testFS,
			path:    "/fonts/standard.flf",
			wantErr: true,
		},
		{
			name:    "path with ./",
			fs:      testFS,
			path:    "./fonts/standard.flf",
			wantErr: true,
		},
		{
			name:    "path is just .",
			fs:      testFS,
			path:    ".",
			wantErr: true,
		},
		{
			name:    "complex traversal attempt",
			fs:      testFS,
			path:    "fonts/../../../secret.flf",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			font, err := LoadFontFS(tt.fs, tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFontFS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFont != nil {
				tt.checkFont(t, font)
			}
		})
	}
}

// Test LoadFontFS with embedded fonts
func TestLoadFontFSEmbedded(t *testing.T) {
	t.Skip("Skipping embedded font test until actual font files are added")
}

// Test Name propagation from filename
func TestLoadFontFS_NamePropagation(t *testing.T) {
	testFS := fstest.MapFS{
		"fonts/my-custom-font.flf": &fstest.MapFile{
			Data: []byte(`flf2a$ 4 3 10 -1 5
Test font
$@
$@
$@
$@@
 @
|@
 @
 @@
`),
		},
		"deep/nested/path/special.flf": &fstest.MapFile{
			Data: []byte(`flf2a$ 4 3 10 -1 5
Test font
$@
$@
$@
$@@
 @
|@
 @
 @@
`),
		},
	}

	tests := []struct {
		path     string
		wantName string
	}{
		{
			path:     "fonts/my-custom-font.flf",
			wantName: "my-custom-font",
		},
		{
			path:     "deep/nested/path/special.flf",
			wantName: "special",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			font, err := LoadFontFS(testFS, tt.path)
			if err != nil {
				t.Fatalf("LoadFontFS() error = %v", err)
			}
			if font.Name != tt.wantName {
				t.Errorf("Font.Name = %q, want %q", font.Name, tt.wantName)
			}
		})
	}
}

// Test concurrent access to Font (verify thread-safety)
func TestFontConcurrentAccess(t *testing.T) {
	fontData := `flf2a$ 4 3 10 -1 5
Test font
$@
$@
$@
$@@
 @
|@
 @
 @@
H@
H@
H@
H@@
e@
e@
e@
e@@
l@
l@
l@
l@@
o@
o@
o@
o@@
`
	r := strings.NewReader(fontData)
	font, err := ParseFont(r)
	if err != nil {
		t.Fatalf("ParseFont() error = %v", err)
	}

	// Run multiple goroutines accessing the font concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Try to render text (this will fail until Render is fully implemented)
			_, _ = Render("Hello", font)

			// Access font properties
			_ = font.Height
			_ = font.Baseline
			_ = font.MaxLen
			_ = font.OldLayout
			_ = font.FullLayout
			_ = font.PrintDirection
			_ = font.Hardblank
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Test integration with real font files
func TestIntegrationWithRealFonts(t *testing.T) {
	// Create test filesystem with real-like font data
	testFS := fstest.MapFS{
		"standard.flf": &fstest.MapFile{
			Data: []byte(`flf2a$ 4 3 10 -1 5
Standard test font
$@
$@
$@
$@@
_@
|@
 @
 @@
`),
		},
	}

	t.Run("load and use standard font", func(t *testing.T) {
		font, err := LoadFontFS(testFS, "standard.flf")
		if err != nil {
			t.Fatalf("LoadFontFS() error = %v", err)
		}

		// Verify font loaded correctly
		if font.Height != 4 {
			t.Errorf("Height = %d, want 4", font.Height)
		}
		if font.Baseline != 3 {
			t.Errorf("Baseline = %d, want 3", font.Baseline)
		}

		// Test that we can attempt to render with it (may fail until renderer is complete)
		_, _ = Render("Test", font)
	})
}

// errorReader is a test helper that always returns an error
type errorReader struct {
	err error
}

func (r *errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

// Test ParseFontBytes convenience function
func TestParseFontBytes(t *testing.T) {
	fontData := []byte(`flf2a$ 4 3 10 -1 5
Test font
$@
$@
$@
$@@
 @
|@
 @
 @@
`)

	font, err := ParseFontBytes(fontData)
	if err != nil {
		t.Fatalf("ParseFontBytes() error = %v", err)
	}
	if font == nil {
		t.Fatal("ParseFontBytes() returned nil font")
	}
	if font.Height != 4 {
		t.Errorf("Height = %d, want 4", font.Height)
	}

	// Test with invalid data
	_, err = ParseFontBytes([]byte("invalid"))
	if err == nil {
		t.Error("ParseFontBytes() should return error for invalid data")
	}

	// Test with empty data
	_, err = ParseFontBytes([]byte(""))
	if err == nil {
		t.Error("ParseFontBytes() should return error for empty data")
	}
}
