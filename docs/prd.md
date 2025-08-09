# Figgo — Product Requirements Document (v1.1)

## 1) Summary

**Figgo** is a Go library and CLI for FIGlet-compatible ASCII art rendering with **correct layout handling** (kerning/smushing) and a **clean, race-safe API**. MVP targets **horizontal rendering** with **FIGfont v2** compliance and **ASCII (32–126)** only.

---

## 2) Goals / Non-Goals

**Goals (MVP)**

* Parse **FIGfont v2** headers & glyphs.
* Implement **horizontal fitting**:

  * **Full-width**, **Kerning**, **Smushing** (controlled + universal).
* Implement **controlled smushing rules (1–6)** with correct precedence.
* Deterministic, **race-safe** rendering with immutable fonts.
* Golden-test parity with the **C `figlet`** output for selected fonts.
* Ergonomic Go API + small CLI parity (`-f`, `-w`, `--list-fonts`).

**Non-Goals (MVP)**

* Unicode beyond ASCII (roadmap, see §4).
* Vertical smushing (roadmap).
* ~~Reading `.flc` control files or compressed fonts.~~ ✅ **Implemented in #21**
* Layout overrides that are not in the font (beyond explicit options).

---

## 3) Terminology (pinned)

* **FIGfont v2**: The font file format (`.flf`) Figgo targets.
* **Hardblank**: The special placeholder character defined in the header; replaced with spaces at the end.
* **Fitting modes**:

  * **Full-width**: No overlap; fixed glyph widths.
  * **Kerning**: Minimize inter-glyph spaces without overlapping pixels.
  * **Smushing**: Allow overlaps by rules (controlled) or fallback (universal).
* **Controlled smushing**: Applying specific rule set encoded in the font header.
* **Universal smushing**: If no rule applies, overlap by taking the non-space of either side (with hardblank protection).
* **Print direction**: 0 = left-to-right, 1 = right-to-left (honor font default; allow override).

---

## 4) Scope & Roadmap

**MVP (this PRD)**

* ASCII (32–126), horizontal only.
* Full-width, kerning, smushing (rules 1–6).
* Print direction honored; no vertical layout.
* CLI basics.

**Phase 2**

* Latin-1 extension set; ~~optional `.flc` support.~~ ✅ **Implemented in #21**
* Font discovery from `fs.FS` (embed/zip/disk) with `WithFS`.

**Phase 3**

* Opt-in **UTF-8** (per-font support), vertical smushing, word-wrap/refill.

**Unknown runes policy (MVP)**: replace with `?` (configurable via option).

---

## 5) Public API (finalized)

```go
package figgo

// Font is immutable & shareable across goroutines.
type Font struct {
    Name           string
    Hardblank      rune
    Height         int
    Baseline       int
    MaxLen         int
    OldLayout      int    // preserved for reference
    FullLayout     Layout // normalized bitmask, see §7
    PrintDirection int    // 0 LTR, 1 RTL
    CommentLines   int
    Glyphs         map[rune][]string // height lines per rune
}

// Loaders
func ParseFont(r io.Reader) (*Font, error)
func LoadFontFS(fsys fs.FS, path string) (*Font, error)

// Rendering (stateless; Font is read-only)
func Render(text string, f *Font, opts ...Option) (string, error)
func RenderTo(w io.Writer, text string, f *Font, opts ...Option) error

// One-line usage examples:
// output, err := figgo.Render("Hello World", font)
// out, _ := figgo.Render(
//     "Hello",
//     font,
//     figgo.WithLayout(figgo.FitSmushing|figgo.RuleEqualChar|figgo.RuleBigX),
// )

// Options pattern
type Option interface{ apply(*settings) }

func WithLayout(layout Layout) Option              // override fitting + rules
func WithPrintDirection(dir int) Option            // 0 / 1
func WithMaxWidth(width int, mode WrapMode) Option // future (no-op in MVP)
func WithJustify(j Justify) Option                 // future (no-op in MVP)

// Errors
var (
    ErrUnknownFont     = errors.New("figgo: unknown font")
    ErrUnsupportedRune = errors.New("figgo: unsupported rune")
    ErrBadFontFormat   = errors.New("figgo: invalid .flf")
    ErrLayoutConflict  = errors.New("figgo: conflicting layout flags")
)
```

---

## 6) Rendering Semantics (MVP)

### 6.1 Pipeline

**State Diagram:**

```
┌─────────┐     ┌──────────┐     ┌─────────┐     ┌──────────┐
│ Parse   │ ─▶  │ Filter   │ ─▶  │ Fetch   │ ─▶  │  Layout  │
│  Font   │     │  ASCII   │     │ Glyphs  │     │  Glyphs  │
└─────────┘     └──────────┘     └─────────┘     └──────────┘
      │                                           │
      ▼                                           ▼
┌─────────┐     ┌──────────┐     ┌─────────┐     ┌──────────┐
│ Return  │ ◀── │  Apply   │ ◀── │ Replace │ ◀── │ Smush/   │
│ Output  │     │Direction │     │Hardblank│     │  Kern    │
└─────────┘     └──────────┘     └─────────┘     └──────────┘
```

**Steps:**

1. Parse font header → normalize **FullLayout** (see §7).
2. Convert input to runes; **filter to ASCII** (replace unknown runes).
3. For each rune, fetch glyph (slice of `height` strings).
4. Start with empty buffer; append glyphs line-by-line using **Fitting**:

   * **Full-width**: concatenate.
   * **Kerning**: trim interstitial spaces to minimal non-overlap.
   * **Smushing**: compute minimal overlap; attempt **controlled rules** in precedence order; if none apply → **universal smush** (except when hardblank collision forbids).
   * **Overlap selection rule:** choose the *maximum* overlap where **every** overlapped column satisfies either a controlled rule **or** the universal rule; otherwise fall back to the **kerning distance**.
5. After final line assembly, **replace hardblanks** with spaces.
6. Apply **print direction** (reverse horizontally if `dir == 1`).

### 6.2 Controlled Smushing Rules (horizontal)

**Precedence (top→down)**. If a rule matches, it decides the overlapped column:

1. **Equal character** — identical non-space, keep that char.
2. **Underscore** — `_` + border chars (`|/\\[]{}()<>`) → border char.
3. **Hierarchy** — class order `|` > `/\\` > `[]` > `{}` > `()`; higher class survives.
4. **Opposite pairs** — `[]`, `{}`, `()` → `|`.
5. **Big X** — `/\\` → `X`, `><` → `X`.
6. **Hardblank** — two hardblanks smush to one hardblank.

If none of the active rules match, but overlap is allowed, fall back to **universal**: take right if left is space, left if right is space; otherwise **no smush** (keep kerning distance). Hardblank collisions **never** universal-smush.

*(Vertical rules are out of scope for MVP.)*

---

## 7) Layout Bitmask (normalized)

Internally, Figgo uses a single `Layout` bitmask:

```go
type Layout uint32

const (
    // Fitting mode (choose exactly one)
    FitFullWidth Layout = 0
    FitKerning   Layout = 1 << 6
    FitSmushing  Layout = 1 << 7

    // Horizontal smushing rules (enable any subset)
    RuleEqualChar   Layout = 1 << 0
    RuleUnderscore  Layout = 1 << 1
    RuleHierarchy   Layout = 1 << 2
    RuleOpposite    Layout = 1 << 3
    RuleBigX        Layout = 1 << 4
    RuleHardblank   Layout = 1 << 5

    // (Reserved for future vertical rules)
)
```

**Normalization:**

* If font uses **OldLayout** (`-1 full`, `-2 kern`, `-3 smush`), convert to `Fit*` + default rules (if any) according to spec.
* If font uses **FullLayout** (bitmask), import as-is.
* If both are present, **FullLayout wins**.

**Validation:**

* Enforce **exactly one** fitting mode at render time (default from font, override with `WithLayout`).
* If **both** `FitKerning` **and** `FitSmushing` are set, return **`ErrLayoutConflict`**.
* If **neither** bit 6 nor 7 is set, Figgo uses **`FitFullWidth`**.
* **Rule bits only have effect when `FitSmushing` is active**; with `FitKerning`/`FitFullWidth` they are ignored.
* It’s valid to have `FitSmushing` with **no rule bits** set → **universal smushing only**.

---

## 8) File Loading & Fonts

* Support `.flf` uncompressed and `.flc` compressed (ZIP) from `io.Reader` and `fs.FS`.
* **Font is immutable**; safe to share across goroutines.
* No word-level caches in `Font`. If caching is needed later, use a separate LRU keyed by `(fontID, layout, text)`.

---

## 9) CLI (minimal parity)

* `figgo -f <font> "Hello"`
* `figgo --list-fonts`
* `figgo -w <columns>` (MVP may ignore wrapping; print warning)

Defaults mirror `figlet` where practical (smushing on if the font dictates it).

---

## 10) Performance & Concurrency

**Targets** *(measured on Go 1.22, 8‑core dev box; indicative)*

* p50 render `"The quick brown fox"` with `standard.flf` in **< 50µs**, **< 4 allocs/op** with pooling.
* Throughput: **\~1M glyph merges/sec** in a single goroutine (smushing enabled) is a stretch goal.

**Approach**

* Precompute per-glyph **left/right trim widths** for faster fitting.
* Use `strings.Builder` + `[]byte` scratch buffers from `sync.Pool`.
* Avoid allocations in hot loops; no reflection, no maps per-call.
* No locking on `Font`—it’s read-only.

---

## 11) Testing & Verification

**Golden tests** *(oracle = C `figlet`)*

* Matrix: fonts `{standard, slant, small, big}` × layouts `{full, kern, smush}` × sample strings `{short, mixed symbols, long}`.
* Commit **fixtures** and a `tools/generate_goldens.sh` script that re-generates them.
* Test compares **byte-for-byte** output.

**Property/Unit tests**

* Tiny glyph pairs for each **rule** to assert precedence/behavior.
* Fuzz the **parser** (header + glyph lines).
* **Race test**: concurrent `Render` with the same `Font`.

**Benchmarks**

* Public `benchmarks/` with controlled inputs and README.

---

## 12) Error Handling & Edge Cases

* **Unknown rune** → `?` (configurable).
* **Missing glyph** in font → `ErrUnsupportedRune`.
* **Hardblank** must never be produced in final output except via rule 6 before replacement; final render replaces it with space.
* `PrintDirection = 1` reverses the composition order; smushing logic must still respect rule precedence.

---

## 13) Licensing & Fonts

* Ship a **small, permissive** font set in-repo (e.g., `standard.flf`) with attributions in `docs/fonts.md`.
* Provide a `fonts/README.md` explaining licenses and where to obtain more.

---

## 14) Deliverables

**MVP (PR milestone)**

* `figgo` package with API in §5.
* Parser + renderer with rules in §6, layout in §7.
* CLI (`cmd/figgo`).
* Tests (goldens + units + fuzz target scaffold).
* Benchmarks.
* Docs:

  * `docs/prd.md` (this file)
  * `docs/spec-compliance.md` — shows the **OldLayout → Layout** normalization and the horizontal rule examples.
  * `docs/fonts.md` — bundled fonts and licensing.

---

## 15) Open Questions (decide in review)

* Default **fitting** when a font encodes both old/new layout fields — **proposal**: prefer new `FullLayout`.
* **Wrap semantics** when `MaxWidth` is specified later: break **between words** under kerning/smushing? (Phase 3.)
* Should CLI default to **font’s** print direction, with `--rtl/--ltr` override? (**Yes**, proposal.)

---

## 16) Appendix — Rule Examples (one-line sketches)

* **Equal char**: `'#' '#'` over `'#' '#'` → `'#'`
* **Underscore**: `'_'` + `'|'` → `'|'`
* **Hierarchy**: `'/'` vs `'|'` → `'|'` wins
* **Opposite pairs**: `'('` + `')'` → `'|'`
* **Big X**: `'/'` + `'\\'` → `'X'`, `'>'` + `'<'` → `'X'`
* **Hardblank**: hardblank + hardblank → hardblank (then replaced with space at end)

---

## Implementation Notes (non-normative)

* Keep the **smushing function** tiny and branch-predictable; compute the minimal overlap across lines first, then decide the join per the precedence list.
* Store per-glyph **trailing/leading space counts** per line to speed fitting.
