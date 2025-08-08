# Figgo — Fonts & Licensing

This page documents which fonts Figgo ships with, how to add more, and how we handle licensing/attribution.

> **Policy:** Keep the bundled set **minimal** and **permissively licensed**. Prefer linking to external font packs or a separate repo to avoid license churn in the main module.

---

## Bundled fonts (proposed minimal set)

* `standard.flf` — Public domain, created by Glenn Chappell & Ian Chai (1991-1993)
* `slant.flf` — Public domain, created by Glenn Chappell (1993)

These fonts are part of the original FIGlet distribution and have been explicitly released into the public domain by their creators.

---

## Font Sources

### Official Repositories

* **Original FIGlet fonts**: https://github.com/cmatsuoka/figlet (includes standard set)
* **FIGlet-fonts collection**: https://github.com/xero/figlet-fonts (community contributed)
* **Toilet fonts**: http://caca.zoy.org/wiki/toilet (additional compatible fonts)

### Font Discovery

Figgo discovers fonts through the following mechanisms:

1. **Embedded fonts** (compile-time):
   ```go
   //go:embed fonts/*.flf
   var embeddedFonts embed.FS
   ```

2. **Runtime paths** (in order of precedence):
   - Current directory: `./fonts/`
   - User config: `~/.config/figgo/fonts/`
   - System-wide: `/usr/share/figgo/fonts/`
   - Custom paths via `WithFontPath(path string)` option

3. **Font Registry**:
   - Built-in fonts are auto-registered at init
   - Runtime fonts discovered on first `ListFonts()` call
   - Cached for the session duration

---

## Directory Structure

```
figgo/
├── fonts/                    # Embedded fonts (compile-time)
│   ├── standard.flf         # Default font
│   ├── slant.flf           # Secondary font
│   └── ATTRIBUTIONS.md     # License details
└── tests/
    └── fonts/               # Test-only fonts (not embedded)
        ├── test_small.flf   # Minimal font for unit tests
        └── test_broken.flf  # Invalid font for error testing
```

---

## Attributions

Create `fonts/ATTRIBUTIONS.md` for any bundled fonts. Each entry should include:

* **Name** and short description
* **Author** and year
* **License** (SPDX identifier if possible)
* **Source URL** (where the `.flf` was obtained)

Example entry:

```md
### standard.flf
- Author: Glenn Chappell & Ian Chai
- Year: 1991-1993
- License: Public Domain
- Source: https://github.com/cmatsuoka/figlet/blob/master/fonts/standard.flf
- Notes: Original FIGlet font, explicitly released to public domain

### slant.flf
- Author: Glenn Chappell
- Year: 1993
- License: Public Domain
- Source: https://github.com/cmatsuoka/figlet/blob/master/fonts/slant.flf
```

---

## Font Embedding Strategy

### Compile-time Embedding

For the MVP, fonts are embedded at compile time using `go:embed`:

```go
package figgo

import (
    "embed"
    "io/fs"
)

//go:embed fonts/*.flf
var embeddedFonts embed.FS

// Access embedded fonts
func init() {
    fontsFS, _ := fs.Sub(embeddedFonts, "fonts")
    fs.WalkDir(fontsFS, ".", func(path string, d fs.DirEntry, err error) error {
        if strings.HasSuffix(path, ".flf") {
            // Auto-register embedded font
        }
        return nil
    })
}
```

### Runtime Loading (Phase 2)

Future support for loading fonts from filesystem:

```go
// Load from custom fs.FS (zip, embed, disk)
engine.LoadFontFS(customFS, "fancy.flf")

// Load from disk path
engine.LoadFontPath("/usr/share/fonts/figlet/fancy.flf")
```

---

## Adding fonts locally (for development/tests)

1. Place `.flf` files under `fonts/` (do **not** commit unless license is permissive and attribution is included)
2. Update `fonts/ATTRIBUTIONS.md` with complete license details
3. Verify the font loads correctly:
   ```bash
   go test ./... -run TestFontParsing
   ```
4. Generate goldens for the new font:
   ```bash
   FONTS="newfont" ./tools/generate-goldens.sh
   ```

---

## License Compatibility Guide

### Acceptable Licenses (can bundle)
* Public Domain
* MIT, BSD (2/3-clause)
* Apache 2.0
* CC0, CC-BY

### Requires Attribution (can bundle with notice)
* CC-BY-SA (include attribution in ATTRIBUTIONS.md)
* Artistic License (include notice)

### Cannot Bundle (link externally)
* GPL (any version) - unless entire project is GPL
* Commercial/proprietary fonts
* Fonts with unclear licensing

---

## Non-goals (MVP)

* `.flc` control files
* Compressed/zipped font ingestion
* Font downloading from remote URLs
* Font format conversion

---

## Font Validation

The parser validates fonts and returns specific errors:

* **Missing header**: `ErrBadFontFormat: invalid header signature`
* **Invalid dimensions**: `ErrBadFontFormat: invalid height/baseline`
* **Missing glyphs**: `ErrBadFontFormat: incomplete character set`
* **Malformed data**: `ErrBadFontFormat: invalid glyph data`

---

## Notes

* The **hardblank** is font-specific; verify your renderer replaces it with space at the end of composition
* Some community fonts have inconsistent headers; the parser should fail with `ErrBadFontFormat` if key fields are missing
* Font names are derived from filenames (minus `.flf` extension) and must be unique within the registry
* Fonts are immutable once loaded; concurrent access is safe without locking