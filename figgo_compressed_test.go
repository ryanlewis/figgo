package figgo

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// minimalFont is a minimal valid FIGfont for testing
const minimalFont = `flf2a$ 3 2 10 0 1
Test font
   @
   @
   @@
 _ @
|_|@
|_|@@
`

// TestParseFontCompressed tests ParseFont with ZIP-compressed font files
func TestParseFontCompressed(t *testing.T) { //nolint:gocognit // Test function with many test cases
	tests := []struct {
		name        string
		createZip   func() ([]byte, error)
		wantErr     bool
		errContains string
	}{
		{
			name: "valid_single_entry_zip",
			createZip: func() ([]byte, error) {
				var buf bytes.Buffer
				w := zip.NewWriter(&buf)
				f, err := w.Create("test.flf")
				if err != nil {
					return nil, err
				}
				if _, err := f.Write([]byte(minimalFont)); err != nil {
					return nil, err
				}
				if err := w.Close(); err != nil {
					return nil, err
				}
				return buf.Bytes(), nil
			},
			wantErr: false,
		},
		{
			name: "zip_with_multiple_entries_uses_first",
			createZip: func() ([]byte, error) {
				var buf bytes.Buffer
				w := zip.NewWriter(&buf)

				// First entry - this should be used
				f1, err := w.Create("first.flf")
				if err != nil {
					return nil, err
				}
				if _, werr := f1.Write([]byte(minimalFont)); werr != nil {
					return nil, werr
				}

				// Second entry - should be ignored
				f2, err := w.Create("second.flf")
				if err != nil {
					return nil, err
				}
				if _, werr := f2.Write([]byte("invalid content")); werr != nil {
					return nil, werr
				}

				if err := w.Close(); err != nil {
					return nil, err
				}
				return buf.Bytes(), nil
			},
			wantErr: false,
		},
		{
			name: "empty_zip_file",
			createZip: func() ([]byte, error) {
				var buf bytes.Buffer
				w := zip.NewWriter(&buf)
				if err := w.Close(); err != nil {
					return nil, err
				}
				return buf.Bytes(), nil
			},
			wantErr:     true,
			errContains: "empty",
		},
		{
			name: "directory_first_then_file",
			createZip: func() ([]byte, error) {
				var buf bytes.Buffer
				w := zip.NewWriter(&buf)
				// First entry is a directory - should be skipped
				_, err := w.Create("fonts/")
				if err != nil {
					return nil, err
				}
				// Second entry is the actual font file
				f, err := w.Create("fonts/myfont.flf")
				if err != nil {
					return nil, err
				}
				if _, werr := f.Write([]byte(minimalFont)); werr != nil {
					return nil, werr
				}
				if err := w.Close(); err != nil {
					return nil, err
				}
				return buf.Bytes(), nil
			},
			wantErr: false,
		},
		{
			name: "inner_file_no_extension",
			createZip: func() ([]byte, error) {
				var buf bytes.Buffer
				w := zip.NewWriter(&buf)
				// Inner file has no extension - should still work
				f, err := w.Create("FONT")
				if err != nil {
					return nil, err
				}
				if _, werr := f.Write([]byte(minimalFont)); werr != nil {
					return nil, werr
				}
				if err := w.Close(); err != nil {
					return nil, err
				}
				return buf.Bytes(), nil
			},
			wantErr: false,
		},
		{
			name: "zip_with_directory_only",
			createZip: func() ([]byte, error) {
				var buf bytes.Buffer
				w := zip.NewWriter(&buf)
				// Create directory entry
				_, err := w.Create("fonts/")
				if err != nil {
					return nil, err
				}
				if err := w.Close(); err != nil {
					return nil, err
				}
				return buf.Bytes(), nil
			},
			wantErr:     true,
			errContains: "directories",
		},
		{
			name: "corrupted_zip_file",
			createZip: func() ([]byte, error) {
				// Create a valid ZIP then corrupt it
				var buf bytes.Buffer
				w := zip.NewWriter(&buf)
				f, err := w.Create("test.flf")
				if err != nil {
					return nil, err
				}
				if _, err := f.Write([]byte(minimalFont)); err != nil {
					return nil, err
				}
				if err := w.Close(); err != nil {
					return nil, err
				}

				data := buf.Bytes()
				// Corrupt the central directory
				if len(data) > 10 {
					data[len(data)-5] = 0xFF
				}
				return data, nil
			},
			wantErr: true,
		},
		{
			name: "non_zip_data_falls_back_to_regular_parse",
			createZip: func() ([]byte, error) {
				// Return regular FLF data, not a ZIP
				return []byte(minimalFont), nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.createZip()
			if err != nil {
				t.Fatalf("Failed to create test data: %v", err)
			}

			font, err := ParseFont(bytes.NewReader(data))

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseFont() expected error, got nil")
				} else if tt.errContains != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errContains)) {
					t.Errorf("ParseFont() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ParseFont() unexpected error = %v", err)
				}
				if font == nil {
					t.Errorf("ParseFont() returned nil font without error")
				} else if font.Height != 3 {
					t.Errorf("Font.Height = %d, want %d", font.Height, 3)
				}
			}
		})
	}
}

// TestLoadFontCompressed tests LoadFont with ZIP-compressed font files
func TestLoadFontCompressed(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a ZIP-compressed .flf file
	flfPath := filepath.Join(tmpDir, "test.flf")

	// Create ZIP content
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("test.flf")
	if err != nil {
		t.Fatalf("Failed to create ZIP entry: %v", err)
	}
	if _, werr := f.Write([]byte(minimalFont)); werr != nil {
		t.Fatalf("Failed to write to ZIP: %v", werr)
	}
	if cerr := w.Close(); cerr != nil {
		t.Fatalf("Failed to close ZIP: %v", cerr)
	}

	// Write to file
	if werr := os.WriteFile(flfPath, buf.Bytes(), 0o644); werr != nil {
		t.Fatalf("Failed to write test file: %v", werr)
	}

	// Test loading
	font, err := LoadFont(flfPath)
	if err != nil {
		t.Errorf("LoadFont() unexpected error = %v", err)
	}
	if font == nil {
		t.Errorf("LoadFont() returned nil font")
	} else {
		if font.Name != "test" {
			t.Errorf("Font.Name = %q, want %q", font.Name, "test")
		}
		if font.Height != 3 {
			t.Errorf("Font.Height = %d, want %d", font.Height, 3)
		}
	}
}

// TestLoadFontFSCompressed tests LoadFontFS with compressed fonts
func TestLoadFontFSCompressed(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a ZIP-compressed .flf file
	flfPath := filepath.Join(tmpDir, "compressed.flf")

	// Create ZIP content
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("compressed.flf")
	if err != nil {
		t.Fatalf("Failed to create ZIP entry: %v", err)
	}
	if _, werr := f.Write([]byte(minimalFont)); werr != nil {
		t.Fatalf("Failed to write to ZIP: %v", werr)
	}
	if cerr := w.Close(); cerr != nil {
		t.Fatalf("Failed to close ZIP: %v", cerr)
	}

	// Write to file
	if werr := os.WriteFile(flfPath, buf.Bytes(), 0o644); werr != nil {
		t.Fatalf("Failed to write test file: %v", werr)
	}

	// Test loading with os.DirFS
	fsys := os.DirFS(tmpDir)
	font, err := LoadFontFS(fsys, "compressed.flf")
	if err != nil {
		t.Errorf("LoadFontFS() unexpected error = %v", err)
	}
	if font == nil {
		t.Errorf("LoadFontFS() returned nil font")
	} else {
		if font.Name != "compressed" {
			t.Errorf("Font.Name = %q, want %q", font.Name, "compressed")
		}
		if font.Height != 3 {
			t.Errorf("Font.Height = %d, want %d", font.Height, 3)
		}
	}
}

// TestParseFontBytesCompressed tests ParseFontBytes with compressed data
func TestParseFontBytesCompressed(t *testing.T) {
	// Create ZIP content
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("test.flf")
	if err != nil {
		t.Fatalf("Failed to create ZIP entry: %v", err)
	}
	if _, werr := f.Write([]byte(minimalFont)); werr != nil {
		t.Fatalf("Failed to write to ZIP: %v", werr)
	}
	if cerr := w.Close(); cerr != nil {
		t.Fatalf("Failed to close ZIP: %v", cerr)
	}

	data := buf.Bytes()

	font, err := ParseFontBytes(data)
	if err != nil {
		t.Errorf("ParseFontBytes() unexpected error = %v", err)
	}
	if font == nil {
		t.Errorf("ParseFontBytes() returned nil font")
	} else if font.Height != 3 {
		t.Errorf("Font.Height = %d, want %d", font.Height, 3)
	}
}

// TestDetectZipMagic tests ZIP magic byte detection
func TestDetectZipMagic(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "valid_zip_magic",
			data:     []byte("PK\x03\x04rest of data"),
			expected: true,
		},
		{
			name:     "invalid_magic_pk_only",
			data:     []byte("PKrest of data"),
			expected: false,
		},
		{
			name:     "flf_header",
			data:     []byte("flf2a$ 3 2 10 0 1"),
			expected: false,
		},
		{
			name:     "too_short",
			data:     []byte("PK"),
			expected: false,
		},
		{
			name:     "empty",
			data:     []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isZipFile(bytes.NewReader(tt.data))
			if result != tt.expected {
				t.Errorf("isZipFile() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Helper function to check if data is a ZIP file
func isZipFile(r io.Reader) bool {
	// This is a test helper - the actual implementation will be in figgo.go
	magic := make([]byte, 4)
	n, err := r.Read(magic)
	if n < 4 || err != nil {
		return false
	}
	return bytes.Equal(magic, []byte("PK\x03\x04"))
}
