package figgo

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func loadTestFont(t *testing.T) *Font {
	t.Helper()
	font, err := LoadFont("fonts/standard.flf")
	if err != nil {
		t.Fatalf("LoadFont: %v", err)
	}
	return font
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	original := loadTestFont(t)

	data, err := encodeFont(original)
	if err != nil {
		t.Fatalf("encodeFont: %v", err)
	}

	decoded, err := decodeFont(data)
	if err != nil {
		t.Fatalf("decodeFont: %v", err)
	}

	// Verify metadata.
	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Layout != original.Layout {
		t.Errorf("Layout: got %d, want %d", decoded.Layout, original.Layout)
	}
	if decoded.Hardblank != original.Hardblank {
		t.Errorf("Hardblank: got %q, want %q", decoded.Hardblank, original.Hardblank)
	}
	if decoded.Height != original.Height {
		t.Errorf("Height: got %d, want %d", decoded.Height, original.Height)
	}
	if decoded.Baseline != original.Baseline {
		t.Errorf("Baseline: got %d, want %d", decoded.Baseline, original.Baseline)
	}
	if decoded.MaxLen != original.MaxLen {
		t.Errorf("MaxLen: got %d, want %d", decoded.MaxLen, original.MaxLen)
	}
	if decoded.OldLayout != original.OldLayout {
		t.Errorf("OldLayout: got %d, want %d", decoded.OldLayout, original.OldLayout)
	}
	if decoded.PrintDirection != original.PrintDirection {
		t.Errorf("PrintDirection: got %d, want %d", decoded.PrintDirection, original.PrintDirection)
	}
	if decoded.CommentLines != original.CommentLines {
		t.Errorf("CommentLines: got %d, want %d", decoded.CommentLines, original.CommentLines)
	}

	// Verify render output matches.
	origOut, err := Render("Hello", original)
	if err != nil {
		t.Fatalf("Render original: %v", err)
	}
	decodedOut, err := Render("Hello", decoded)
	if err != nil {
		t.Fatalf("Render decoded: %v", err)
	}
	if origOut != decodedOut {
		t.Errorf("Render output mismatch:\noriginal:\n%s\ndecoded:\n%s", origOut, decodedOut)
	}
}

func TestDecodeInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"too short", []byte{1, 2, 3}},
		{"bad magic", []byte("BADMAGXX")},
		{"bad version", append(diskCacheMagic[:], 0xFF, 0xFF)},
		{"truncated gob", append(append(diskCacheMagic[:], 1, 0), 0xFF, 0xFF)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeFont(tt.data)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestDiskCacheGetPut(t *testing.T) {
	dir := t.TempDir()
	dc := newDiskCache(DiskCacheConfig{Dir: dir, MaxEntries: 5})
	if dc == nil {
		t.Fatal("newDiskCache returned nil")
	}

	font := loadTestFont(t)
	hash := "testhash123"

	// Miss.
	if got := dc.get(hash); got != nil {
		t.Fatal("expected nil on cache miss")
	}

	// Put.
	dc.put(hash, font)

	// Hit.
	got := dc.get(hash)
	if got == nil {
		t.Fatal("expected cache hit")
	}

	// Verify render matches.
	origOut, _ := Render("Test", font)
	gotOut, _ := Render("Test", got)
	if origOut != gotOut {
		t.Error("render output mismatch after disk cache round-trip")
	}

	// Verify .gob file exists.
	if _, err := os.Stat(filepath.Join(dir, hash+".gob")); err != nil {
		t.Errorf("expected .gob file: %v", err)
	}

	// Verify meta.json exists.
	if _, err := os.Stat(filepath.Join(dir, "meta.json")); err != nil {
		t.Errorf("expected meta.json: %v", err)
	}
}

func TestDiskCacheLRUEviction(t *testing.T) {
	dir := t.TempDir()
	dc := newDiskCache(DiskCacheConfig{Dir: dir, MaxEntries: 3})

	font := loadTestFont(t)

	// Fill cache beyond capacity.
	dc.put("hash1", font)
	dc.put("hash2", font)
	dc.put("hash3", font)
	dc.put("hash4", font) // should evict hash1

	// hash1 should be evicted.
	if _, err := os.Stat(filepath.Join(dir, "hash1.gob")); !os.IsNotExist(err) {
		t.Error("expected hash1.gob to be evicted")
	}
	if dc.get("hash1") != nil {
		t.Error("expected hash1 to be evicted from meta")
	}

	// hash2-4 should still be present.
	for _, h := range []string{"hash2", "hash3", "hash4"} {
		if dc.get(h) == nil {
			t.Errorf("expected %s to be present", h)
		}
	}
}

func TestDiskCacheLRUAccessOrder(t *testing.T) {
	dir := t.TempDir()
	dc := newDiskCache(DiskCacheConfig{Dir: dir, MaxEntries: 3})

	font := loadTestFont(t)

	dc.put("hash1", font)
	dc.put("hash2", font)
	dc.put("hash3", font)

	// Access hash1 to make it MRU.
	dc.get("hash1")

	// Add hash4 — should evict hash2 (LRU), not hash1.
	dc.put("hash4", font)

	if dc.get("hash1") == nil {
		t.Error("hash1 should survive (was accessed recently)")
	}
	if dc.get("hash2") != nil {
		t.Error("hash2 should be evicted (LRU)")
	}
}

func TestDiskCacheCorruptFile(t *testing.T) {
	dir := t.TempDir()
	dc := newDiskCache(DiskCacheConfig{Dir: dir, MaxEntries: 5})

	font := loadTestFont(t)
	dc.put("hashX", font)

	// Corrupt the .gob file.
	os.WriteFile(filepath.Join(dir, "hashX.gob"), []byte("corrupt"), 0o644)

	// Should silently return nil and clean up.
	if got := dc.get("hashX"); got != nil {
		t.Error("expected nil for corrupt cache entry")
	}

	// Entry should be removed from meta.
	if dc.findEntry("hashX") >= 0 {
		t.Error("corrupt entry should be removed from meta")
	}
}

func TestDiskCacheMissingFile(t *testing.T) {
	dir := t.TempDir()
	dc := newDiskCache(DiskCacheConfig{Dir: dir, MaxEntries: 5})

	font := loadTestFont(t)
	dc.put("hashY", font)

	// Delete the .gob file.
	os.Remove(filepath.Join(dir, "hashY.gob"))

	// Should silently return nil and clean up meta.
	if got := dc.get("hashY"); got != nil {
		t.Error("expected nil for missing cache file")
	}
	if dc.findEntry("hashY") >= 0 {
		t.Error("missing entry should be removed from meta")
	}
}

func TestDiskCacheClear(t *testing.T) {
	dir := t.TempDir()
	dc := newDiskCache(DiskCacheConfig{Dir: dir, MaxEntries: 5})

	font := loadTestFont(t)
	dc.put("h1", font)
	dc.put("h2", font)

	dc.clear()

	if dc.get("h1") != nil || dc.get("h2") != nil {
		t.Error("cache should be empty after clear")
	}

	// .gob files should be removed.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".gob" {
			t.Errorf("expected .gob files to be removed, found %s", e.Name())
		}
	}
}

func TestDiskCacheDuplicatePut(t *testing.T) {
	dir := t.TempDir()
	dc := newDiskCache(DiskCacheConfig{Dir: dir, MaxEntries: 5})

	font := loadTestFont(t)
	dc.put("dup", font)
	dc.put("dup", font) // should be no-op

	if len(dc.meta.Entries) != 1 {
		t.Errorf("expected 1 meta entry, got %d", len(dc.meta.Entries))
	}
}

func TestDiskCacheConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	dc := newDiskCache(DiskCacheConfig{Dir: dir, MaxEntries: 20})

	font := loadTestFont(t)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			hash := "concurrent" + string(rune('0'+i))
			dc.put(hash, font)
			dc.get(hash)
		}(i)
	}
	wg.Wait()
}

func TestFontCacheWithDiskCache(t *testing.T) {
	dir := t.TempDir()
	cfg := DiskCacheConfig{Dir: dir, MaxEntries: 5}

	cache1 := NewFontCache(10, WithDiskCache(cfg))

	// Load via file path — populates both memory and disk caches.
	font1, err := cache1.LoadFont("fonts/standard.flf")
	if err != nil {
		t.Fatalf("LoadFont: %v", err)
	}

	// Create a fresh in-memory cache pointing at the same disk dir.
	cache2 := NewFontCache(10, WithDiskCache(cfg))

	// Load again — should hit disk cache.
	font2, err := cache2.LoadFont("fonts/standard.flf")
	if err != nil {
		t.Fatalf("LoadFont from disk: %v", err)
	}

	out1, _ := Render("Hi", font1)
	out2, _ := Render("Hi", font2)
	if out1 != out2 {
		t.Error("render output mismatch after disk cache recovery")
	}
}

func TestFontCacheParseFontWithDiskCache(t *testing.T) {
	dir := t.TempDir()
	cfg := DiskCacheConfig{Dir: dir, MaxEntries: 5}

	data, err := os.ReadFile("fonts/small.flf")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	cache1 := NewFontCache(10, WithDiskCache(cfg))

	// Parse — populates disk cache.
	font1, err := cache1.ParseFont(data)
	if err != nil {
		t.Fatalf("ParseFont: %v", err)
	}

	// Fresh in-memory cache, same disk dir.
	cache2 := NewFontCache(10, WithDiskCache(cfg))

	// Parse again — should hit disk cache.
	font2, err := cache2.ParseFont(data)
	if err != nil {
		t.Fatalf("ParseFont from disk: %v", err)
	}

	out1, _ := Render("Test", font1)
	out2, _ := Render("Test", font2)
	if out1 != out2 {
		t.Error("render output mismatch after disk cache recovery for ParseFont")
	}
}

func TestEnableDefaultDiskCache(t *testing.T) {
	// Save and restore the default cache.
	saved := defaultCache
	defer func() { defaultCache = saved }()

	dir := t.TempDir()
	EnableDefaultDiskCache(DiskCacheConfig{Dir: dir, MaxEntries: 5})

	if defaultCache.disk == nil {
		t.Error("expected disk cache to be enabled on default cache")
	}

	_, err := LoadFontCached("fonts/standard.flf")
	if err != nil {
		t.Fatalf("LoadFontCached: %v", err)
	}

	// Verify a .gob file was created.
	entries, _ := os.ReadDir(dir)
	gobCount := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".gob" {
			gobCount++
		}
	}
	if gobCount == 0 {
		t.Error("expected at least one .gob file in disk cache dir")
	}
}

func BenchmarkDiskCacheDecode(b *testing.B) {
	font := loadBenchFont(b)

	data, err := encodeFont(font)
	if err != nil {
		b.Fatalf("encodeFont: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := decodeFont(data)
		if err != nil {
			b.Fatalf("decodeFont: %v", err)
		}
	}
}

func BenchmarkDiskCacheVsParse(b *testing.B) {
	raw, err := os.ReadFile("fonts/standard.flf")
	if err != nil {
		b.Fatalf("ReadFile: %v", err)
	}

	font, err := ParseFontBytes(raw)
	if err != nil {
		b.Fatalf("ParseFontBytes: %v", err)
	}
	encoded, err := encodeFont(font)
	if err != nil {
		b.Fatalf("encodeFont: %v", err)
	}

	b.Run("Parse", func(b *testing.B) {
		for b.Loop() {
			_, err := ParseFontBytes(raw)
			if err != nil {
				b.Fatalf("ParseFontBytes: %v", err)
			}
		}
	})

	b.Run("DiskDecode", func(b *testing.B) {
		for b.Loop() {
			_, err := decodeFont(encoded)
			if err != nil {
				b.Fatalf("decodeFont: %v", err)
			}
		}
	})
}

func loadBenchFont(b *testing.B) *Font {
	b.Helper()
	font, err := LoadFont("fonts/standard.flf")
	if err != nil {
		b.Fatalf("LoadFont: %v", err)
	}
	return font
}
