package figgo

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var diskCacheMagic = [6]byte{'F', 'I', 'G', 'G', 'O', 0}

const diskCacheVersion uint16 = 1

// DiskCacheConfig configures the on-disk font cache.
type DiskCacheConfig struct {
	// Dir is the directory for cached font files.
	// If empty, defaults to os.UserCacheDir()/figgo/fonts/.
	Dir string

	// MaxEntries is the maximum number of fonts stored on disk.
	// When exceeded, the least recently used entry is evicted.
	// Default is 20 if zero.
	MaxEntries int
}

type diskCache struct {
	mu         sync.Mutex
	dir        string
	maxEntries int
	meta       *diskCacheMeta
}

type diskCacheMeta struct {
	Version int              `json:"version"`
	Entries []diskCacheEntry `json:"entries"`
}

type diskCacheEntry struct {
	Hash string `json:"hash"`
}

type fontGobEntry struct {
	Glyphs         map[rune][]string
	Name           string
	Layout         uint32
	Hardblank      rune
	Height         int
	Baseline       int
	MaxLen         int
	OldLayout      int
	PrintDirection int
	CommentLines   int
}

func fontToGobEntry(f *Font) fontGobEntry {
	return fontGobEntry{
		Glyphs:         f.glyphs,
		Name:           f.Name,
		Layout:         uint32(f.Layout),
		Hardblank:      f.Hardblank,
		Height:         f.Height,
		Baseline:       f.Baseline,
		MaxLen:         f.MaxLen,
		OldLayout:      f.OldLayout,
		PrintDirection: f.PrintDirection,
		CommentLines:   f.CommentLines,
	}
}

func gobEntryToFont(e fontGobEntry) *Font {
	// Gob decoder allocates fresh maps/slices, so no deep copy needed.
	return &Font{
		glyphs:         e.Glyphs,
		Name:           e.Name,
		Layout:         Layout(e.Layout),
		Hardblank:      e.Hardblank,
		Height:         e.Height,
		Baseline:       e.Baseline,
		MaxLen:         e.MaxLen,
		OldLayout:      e.OldLayout,
		PrintDirection: e.PrintDirection,
		CommentLines:   e.CommentLines,
	}
}

func encodeFont(f *Font) ([]byte, error) {
	var buf bytes.Buffer

	buf.Write(diskCacheMagic[:])
	if err := binary.Write(&buf, binary.LittleEndian, diskCacheVersion); err != nil {
		return nil, fmt.Errorf("write version: %w", err)
	}

	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(fontToGobEntry(f)); err != nil {
		return nil, fmt.Errorf("gob encode: %w", err)
	}

	return buf.Bytes(), nil
}

func decodeFont(data []byte) (*Font, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short: %d bytes", len(data))
	}

	var magic [6]byte
	copy(magic[:], data[:6])
	if magic != diskCacheMagic {
		return nil, fmt.Errorf("invalid magic: %x", magic)
	}

	version := binary.LittleEndian.Uint16(data[6:8])
	if version != diskCacheVersion {
		return nil, fmt.Errorf("unsupported version: %d", version)
	}

	var entry fontGobEntry
	dec := gob.NewDecoder(bytes.NewReader(data[8:]))
	if err := dec.Decode(&entry); err != nil {
		return nil, fmt.Errorf("gob decode: %w", err)
	}

	return gobEntryToFont(entry), nil
}

// newDiskCache returns nil if the cache directory cannot be resolved.
func newDiskCache(cfg DiskCacheConfig) *diskCache {
	dir := cfg.Dir
	if dir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return nil
		}
		dir = filepath.Join(base, "figgo", "fonts")
	}

	maxEntries := cfg.MaxEntries
	if maxEntries <= 0 {
		maxEntries = 20
	}

	dc := &diskCache{
		dir:        dir,
		maxEntries: maxEntries,
	}
	dc.meta = dc.loadMeta()
	return dc
}

// get returns nil if not found or on any error (silent fallback).
func (dc *diskCache) get(hash string) *Font {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	idx := dc.findEntry(hash)
	if idx < 0 {
		return nil
	}

	data, err := os.ReadFile(dc.gobPath(hash))
	if err != nil {
		dc.removeEntryAt(idx)
		dc.saveMeta()
		return nil
	}

	font, err := decodeFont(data)
	if err != nil {
		os.Remove(dc.gobPath(hash))
		dc.removeEntryAt(idx)
		dc.saveMeta()
		return nil
	}

	dc.moveToFront(idx)

	return font
}

func (dc *diskCache) put(hash string, font *Font) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.findEntry(hash) >= 0 {
		return
	}

	data, err := encodeFont(font)
	if err != nil {
		return
	}

	if err := atomicWriteFile(dc.gobPath(hash), data); err != nil {
		return
	}

	dc.meta.Entries = append([]diskCacheEntry{{Hash: hash}}, dc.meta.Entries...)

	for len(dc.meta.Entries) > dc.maxEntries {
		tail := dc.meta.Entries[len(dc.meta.Entries)-1]
		os.Remove(dc.gobPath(tail.Hash))
		dc.meta.Entries = dc.meta.Entries[:len(dc.meta.Entries)-1]
	}

	dc.saveMeta()
}

func (dc *diskCache) clear() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	for _, e := range dc.meta.Entries {
		os.Remove(dc.gobPath(e.Hash))
	}
	dc.meta.Entries = nil
	dc.saveMeta()
}

func (dc *diskCache) gobPath(hash string) string {
	return filepath.Join(dc.dir, hash+".gob")
}

func (dc *diskCache) metaPath() string {
	return filepath.Join(dc.dir, "meta.json")
}

func (dc *diskCache) findEntry(hash string) int {
	for i, e := range dc.meta.Entries {
		if e.Hash == hash {
			return i
		}
	}
	return -1
}

func (dc *diskCache) removeEntryAt(idx int) {
	dc.meta.Entries = append(dc.meta.Entries[:idx], dc.meta.Entries[idx+1:]...)
}

func (dc *diskCache) moveToFront(idx int) {
	if idx == 0 {
		return
	}
	entry := dc.meta.Entries[idx]
	dc.meta.Entries = append(dc.meta.Entries[:idx], dc.meta.Entries[idx+1:]...)
	dc.meta.Entries = append([]diskCacheEntry{entry}, dc.meta.Entries...)
}

func (dc *diskCache) loadMeta() *diskCacheMeta {
	data, err := os.ReadFile(dc.metaPath())
	if err != nil {
		return &diskCacheMeta{Version: 1}
	}
	var meta diskCacheMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return &diskCacheMeta{Version: 1}
	}
	return &meta
}

func (dc *diskCache) saveMeta() {
	data, err := json.Marshal(dc.meta)
	if err != nil {
		return
	}
	atomicWriteFile(dc.metaPath(), data)
}

// atomicWriteFile writes data to path via temp file + rename.
func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
