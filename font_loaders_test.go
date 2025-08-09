package figgo

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

// testFonts will be used when actual font files exist
// For now, tests use fstest.MapFS

// checkMinimalFont validates basic font properties
func checkMinimalFont(t *testing.T, f *Font) {
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
}

// checkLayoutFont validates layout normalization
func checkLayoutFont(t *testing.T, f *Font) {
	if f == nil {
		t.Fatal("expected non-nil font")
	}
	// With OldLayout=15 and FullLayout=24463, layout should be normalized
	// OldLayout 15 = kerning + all horizontal smushing rules
	// FullLayout 24463 should be validated and normalized
	if f.Layout == 0 {
		t.Error("Layout should be non-zero after normalization")
	}
}

// Test ParseFont with valid font data
func TestParseFont(t *testing.T) {
	tests := []struct {
		name      string
		fontData  string
		wantErr   bool
		checkFunc func(t *testing.T, f *Font)
	}{
		{
			name: "valid minimal font",
			fontData: `flf2a$ 4 3 10 -1 1
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
			wantErr:   false,
			checkFunc: checkMinimalFont,
		},
		{
			name: "font with full layout",
			fontData: `flf2a$ 4 3 10 15 1 0 24463
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
			wantErr:   false,
			checkFunc: checkLayoutFont,
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
			fontData: `flf2a$ 4 3 10 -1 1
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

			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, font)
			}
		})
	}
}

// Test ParseFont with io.Reader edge cases
func TestParseFontReaderBehavior(t *testing.T) {
	validFont := `flf2a$ 4 3 10 -1 1
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

// createTestFS creates a test filesystem with various font files
func createTestFS() fstest.MapFS {
	return fstest.MapFS{
		"fonts/standard.flf": &fstest.MapFile{
			Data: []byte(`flf2a$ 4 3 10 -1 1
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
			Data: []byte(`flf2a# 4 3 10 -1 1
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
}

// checkStandardFont validates standard font properties
func checkStandardFont(t *testing.T, f *Font) {
	if f == nil {
		t.Fatal("expected non-nil font")
	}
	if f.Height != 4 {
		t.Errorf("Height = %d, want 4", f.Height)
	}
	if f.Hardblank != '$' {
		t.Errorf("Hardblank = %c, want $", f.Hardblank)
	}
}

// checkSlantFont validates slant font properties
func checkSlantFont(t *testing.T, f *Font) {
	if f == nil {
		t.Fatal("expected non-nil font")
	}
	if f.Height != 4 {
		t.Errorf("Height = %d, want 4", f.Height)
	}
	if f.Hardblank != '#' {
		t.Errorf("Hardblank = %c, want #", f.Hardblank)
	}
}

// checkMiniFont validates mini font properties
func checkMiniFont(t *testing.T, f *Font) {
	if f == nil {
		t.Fatal("expected non-nil font")
	}
	if f.Height != 3 {
		t.Errorf("Height = %d, want 3", f.Height)
	}
}

// Test LoadFontFS with valid filesystem
func TestLoadFontFS(t *testing.T) {
	testFS := createTestFS()

	tests := []struct {
		name      string
		fs        fs.FS
		path      string
		wantErr   bool
		checkFunc func(t *testing.T, f *Font)
	}{
		{
			name:      "load standard font",
			fs:        testFS,
			path:      "fonts/standard.flf",
			wantErr:   false,
			checkFunc: checkStandardFont,
		},
		{
			name:      "load slant font",
			fs:        testFS,
			path:      "fonts/slant.flf",
			wantErr:   false,
			checkFunc: checkSlantFont,
		},
		{
			name:      "load font from subdirectory",
			fs:        testFS,
			path:      "fonts/subdir/mini.flf",
			wantErr:   false,
			checkFunc: checkMiniFont,
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

			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, font)
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
	fontData := `flf2a$ 4 3 10 -1 1
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
			_ = font.Layout
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
	fontData := []byte(`flf2a$ 4 3 10 -1 1
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

	// Test that Name field is not set
	if font.Name != "" {
		t.Errorf("ParseFontBytes() should not set Name field, got %q", font.Name)
	}
}

// Test FS path semantics with comprehensive edge cases
func TestLoadFontFS_PathSemantics(t *testing.T) {
	validFont := []byte(`flf2a$ 4 3 10 -1 1
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

	testFS := fstest.MapFS{
		"fonts/standard.flf": &fstest.MapFile{Data: validFont},
		"fonts/my..font.flf": &fstest.MapFile{Data: validFont},
	}

	tests := []struct {
		name      string
		path      string
		wantErr   bool
		errString string // substring that should be in error message
	}{
		{
			name:      "path traversal with ../",
			path:      "fonts/../secret.flf",
			wantErr:   true,
			errString: "invalid fs path", // fs.ValidPath rejects this first
		},
		{
			name:      "backslash in path",
			path:      "fonts\\standard.flf",
			wantErr:   true,
			errString: "backslashes not allowed",
		},
		{
			name:      "absolute path",
			path:      "/abs/font.flf",
			wantErr:   true,
			errString: "absolute paths not allowed",
		},
		{
			name:      "path with ./",
			path:      "./fonts/standard.flf",
			wantErr:   true,
			errString: "invalid fs path",
		},
		{
			name:      "double dots in filename (allowed)",
			path:      "fonts/my..font.flf",
			wantErr:   false,
			errString: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadFontFS(testFS, tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFontFS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errString != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errString) {
					t.Errorf("LoadFontFS() error = %v, should contain %q", err, tt.errString)
				}
			}
		})
	}
}

// Test name derivation from various path formats
func TestLoadFont_NameDerivation(t *testing.T) {
	// Create a temporary font file
	tmpDir := t.TempDir()
	fontPath := tmpDir + "/SLANT.FLF"

	fontData := `flf2a$ 4 3 10 -1 1
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

	err := os.WriteFile(fontPath, []byte(fontData), 0o644)
	if err != nil {
		t.Fatalf("Failed to create temp font file: %v", err)
	}

	font, err := LoadFont(fontPath)
	if err != nil {
		t.Fatalf("LoadFont() error = %v", err)
	}

	expectedName := "SLANT"
	if font.Name != expectedName {
		t.Errorf("LoadFont() Name = %q, want %q", font.Name, expectedName)
	}
}

// Test LoadFontFS name derivation with various path formats
func TestLoadFontFS_NameDerivation(t *testing.T) {
	validFont := []byte(`flf2a$ 4 3 10 -1 1
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

	testFS := fstest.MapFS{
		"fonts/Weird.Name.flf": &fstest.MapFile{Data: validFont},
		"deep/path/script.flf": &fstest.MapFile{Data: validFont},
		"simple.flf":           &fstest.MapFile{Data: validFont},
	}

	tests := []struct {
		path         string
		expectedName string
	}{
		{
			path:         "fonts/Weird.Name.flf",
			expectedName: "Weird.Name",
		},
		{
			path:         "deep/path/script.flf",
			expectedName: "script",
		},
		{
			path:         "simple.flf",
			expectedName: "simple",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			font, err := LoadFontFS(testFS, tt.path)
			if err != nil {
				t.Fatalf("LoadFontFS() error = %v", err)
			}
			if font.Name != tt.expectedName {
				t.Errorf("LoadFontFS() Name = %q, want %q", font.Name, tt.expectedName)
			}
		})
	}
}

// Test layout precedence with FullLayout overriding OldLayout
func TestLayoutPrecedence(t *testing.T) {
	// Font where both OldLayout and FullLayout are present
	fontData := `flf2a$ 4 3 10 31 1 0 64
Test font with layout precedence
$@
$@
$@
$@@
 @
|@
 @
 @@
`

	font, err := ParseFontBytes([]byte(fontData))
	if err != nil {
		t.Fatalf("ParseFontBytes() error = %v", err)
	}

	// OldLayout=31 would be smushing with rules 1-5
	// FullLayout=64 should win and give us FitKerning (fitting mode)
	expectedLayout := FitKerning
	if font.Layout != expectedLayout {
		t.Errorf("Layout precedence: got %v, want %v (FullLayout should override OldLayout)",
			font.Layout, expectedLayout)
	}
}

// Test comprehensive loader seams for edge cases and security
func TestLoaderSeams(t *testing.T) {
	validFont := []byte(`flf2a$ 4 3 10 -1 1
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

	testFS := fstest.MapFS{
		"fonts/standard.flf":          &fstest.MapFile{Data: validFont},
		"fonts/my..font.flf":          &fstest.MapFile{Data: validFont},
		"fonts/SLANT.FLF":             &fstest.MapFile{Data: validFont},
		"fonts/control.flc":           &fstest.MapFile{Data: validFont}, // Future control file support
		"fonts/subdir/my.font.v1.flf": &fstest.MapFile{Data: validFont}, // Complex name
	}

	tests := []struct {
		name        string
		path        string
		wantErr     bool
		errContains string
		wantName    string // Expected font name if successful
	}{
		// Security tests - should all fail
		{
			name:        "path traversal with ../",
			path:        "fonts/../secret.flf",
			wantErr:     true,
			errContains: "invalid fs path",
		},
		{
			name:        "backslash in path",
			path:        "fonts\\std.flf",
			wantErr:     true,
			errContains: "backslashes not allowed",
		},
		{
			name:        "explicit backslash test",
			path:        "a\\b.flf",
			wantErr:     true,
			errContains: "backslashes not allowed",
		},
		{
			name:        "absolute path",
			path:        "/abs/standard.flf",
			wantErr:     true,
			errContains: "absolute paths not allowed",
		},
		// Valid cases
		{
			name:     "double dots in filename (no traversal)",
			path:     "fonts/my..font.flf",
			wantErr:  false,
			wantName: "my..font",
		},
		{
			name:     "uppercase extension",
			path:     "fonts/SLANT.FLF",
			wantErr:  false,
			wantName: "SLANT",
		},
		{
			name:     "control file extension (.flc)",
			path:     "fonts/control.flc",
			wantErr:  false,
			wantName: "control",
		},
		{
			name:     "complex name with dots",
			path:     "fonts/subdir/my.font.v1.flf",
			wantErr:  false,
			wantName: "my.font.v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			font, err := LoadFontFS(testFS, tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFontFS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err == nil {
					t.Error("LoadFontFS() should have returned an error")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("LoadFontFS() error = %v, should contain %q", err, tt.errContains)
				}
			} else {
				if font == nil {
					t.Error("LoadFontFS() should return valid font")
				} else if tt.wantName != "" && font.Name != tt.wantName {
					t.Errorf("LoadFontFS() Name = %q, want %q", font.Name, tt.wantName)
				}
			}
		})
	}
}

// Test that directories are properly rejected
func TestLoadFontFS_DirectoryRejection(t *testing.T) {
	// Create a filesystem with a directory entry
	testFS := fstest.MapFS{
		"fonts": &fstest.MapFile{
			Mode: fs.ModeDir,
		},
		"fonts/standard.flf": &fstest.MapFile{
			Data: []byte(`flf2a$ 4 3 10 -1 1
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
		name        string
		path        string
		wantErr     bool
		errContains string
	}{
		{
			name:        "directory path rejected",
			path:        "fonts",
			wantErr:     true,
			errContains: "is a directory",
		},
		{
			name:        "file path succeeds",
			path:        "fonts/standard.flf",
			wantErr:     false,
			errContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadFontFS(testFS, tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFontFS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("LoadFontFS() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

// Test FullLayoutSet precedence scenarios
func TestFullLayoutSetPrecedence(t *testing.T) {
	tests := []struct {
		name            string
		oldLayout       int
		fullLayout      int
		fullLayoutSet   bool
		expectNonZero   bool   // Layout field should be non-zero after normalization
		expectedFitMode string // Expected fitting mode description
	}{
		{
			name:            "FullLayout set overrides OldLayout",
			oldLayout:       31, // Would be controlled smushing with rules 1-5
			fullLayout:      64, // Horizontal fitting
			fullLayoutSet:   true,
			expectNonZero:   true,
			expectedFitMode: "Kerning", // FitKerning
		},
		{
			name:            "FullLayout set to universal smushing",
			oldLayout:       0,   // Would be fitting
			fullLayout:      128, // Universal smushing
			fullLayoutSet:   true,
			expectNonZero:   true,
			expectedFitMode: "Smushing", // FitSmushing
		},
		{
			name:            "No FullLayout falls back to OldLayout",
			oldLayout:       15, // Controlled smushing with rules 1-4
			fullLayout:      0,  // Not relevant when fullLayoutSet=false
			fullLayoutSet:   false,
			expectNonZero:   true,
			expectedFitMode: "Smushing", // FitSmushing with rules
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create font data with specific layout values
			fontHeader := fmt.Sprintf("flf2a$ 4 3 10 %d 1", tt.oldLayout)
			if tt.fullLayoutSet {
				fontHeader = fmt.Sprintf("flf2a$ 4 3 10 %d 1 0 %d", tt.oldLayout, tt.fullLayout)
			}

			fontData := fontHeader + `
Test font with layout precedence
$@
$@
$@
$@@
 @
|@
 @
 @@
`

			font, err := ParseFontBytes([]byte(fontData))
			if err != nil {
				t.Fatalf("ParseFontBytes() error = %v", err)
			}

			if tt.expectNonZero && font.Layout == 0 {
				t.Errorf("Layout should be non-zero after normalization, got %v", font.Layout)
			}

			// Verify the layout is reasonable (has the expected fitting mode)
			layoutStr := font.Layout.String()
			if !strings.Contains(layoutStr, tt.expectedFitMode) {
				t.Errorf("Layout %v should contain %q, got %q", font.Layout, tt.expectedFitMode, layoutStr)
			}
		})
	}
}
