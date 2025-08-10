# Figgo — Spec Compliance Guide

This document is the **source of truth** for how Figgo interprets **FIGfont v2** and maps font header data to Figgo's runtime `Layout` flags and rendering behavior. When a PR claims "spec compliant," reviewers should validate it against this page.

**Status**: ✅ **FULLY COMPLIANT** with FIGfont v2 specification for required character parsing (as of issue #6 implementation)

---

## 1) Purpose & Scope

* Covers **horizontal** rendering only (MVP): fitting modes, smushing rules, print direction, and hardblank handling.
* Defines how to read FIGfont headers and normalize layout information.
* Establishes a **test matrix** and a **review checklist** for compliance.

---

## 2) Parsed Header Fields

* Signature & version (expects `flf2a`‑style header)
* **Hardblank** character
* **Height**, **Baseline**, **MaxLen**
* **OldLayout** (legacy) and **FullLayout** (bitmask)
* **PrintDirection** (0 = LTR, 1 = RTL)
* **CommentLines**
* Glyph blocks for ASCII 32–126 plus German characters 196, 214, 220, 228, 246, 252, 223 (full spec compliance)

---

## 3) Layout Normalization

Figgo normalizes FIGfont layout data into a single `Layout` bitmask used at render time. The normalization process handles both `OldLayout` and `FullLayout` with proper precedence as per FIGfont v2 specification.

### 3.1 OldLayout → Layout

Valid range: `-1..63`
* **`-1`** → Horizontal **Full width** (no overlap)
* **`0..63`** → Horizontal layout modes encoded as follows:
  * **`0`** → Horizontal **Fitting** (kerning - minimal spacing without character overlap)
  * **`1..63`** → Horizontal **Smushing** with controlled rules:
    * Bits 0-5 encode smushing rules 1-6 respectively
    * Example: `3` = bits 0+1 = rules 1+2 enabled
    * Example: `63` = all 6 rules enabled

**Note**: OldLayout cannot express universal smushing. When `OldLayout = 0`, it means kerning/fitting for backward compatibility, NOT universal smushing.

Invalid values (`< -1` or `> 63`) produce an error.

### 3.2 FullLayout → Layout

Valid range: `0..32767`

#### Horizontal Layout (bits 0-7):
* **Bits 0-5** — Smushing rules:
  * Bit 0 → `RuleEqualChar`
  * Bit 1 → `RuleUnderscore`
  * Bit 2 → `RuleHierarchy`
  * Bit 3 → `RuleOppositePair`
  * Bit 4 → `RuleBigX`
  * Bit 5 → `RuleHardblank`
* **Bit 6** — Horizontal fitting → `FitKerning`
* **Bit 7** — Horizontal smushing → `FitSmushing`

#### Vertical Layout (bits 8-14):
* **Bits 8-12** — Vertical smushing rules 1-5
* **Bit 13** — Vertical fitting
* **Bit 14** — Vertical smushing

#### Layout Mode Determination:
* **Universal smushing**: Smushing bit set (7 or 14) with NO rule bits for that axis
* **Controlled smushing**: Smushing bit set with rule bits
* **Fitting/Kerning**: Fitting bit set (6 or 13)
* **Full width/height**: No mode bits set (default)

### 3.3 Precedence Rules

* If `FullLayout` is present in the header (tracked via field count), it **completely overrides** `OldLayout`
* If `FullLayout` is absent, derive horizontal layout from `OldLayout`; vertical defaults to full height
* `FullLayout = 0` with presence flag set means full width/height (different from absent)

### 3.4 Implementation Details

The normalization is implemented via:
```go
func NormalizeLayoutFromHeader(oldLayout int, fullLayout int, fullLayoutSet bool) (NormalizedLayout, error)
```

This function:
1. Validates ranges (`OldLayout`: -1..63, `FullLayout`: 0..32767)
2. Applies precedence rules based on `fullLayoutSet`
3. Returns a `NormalizedLayout` struct with separate horizontal/vertical modes and rules
4. Converts to the simplified `Layout` bitmask for rendering via `ToLayout()`

---

## 4) Validation Rules

* Exactly **one** fitting mode must be active at render time.

  * Both `FitKerning` **and** `FitSmushing` set → **`ErrLayoutConflict`**.
  * Neither set → **`FitFullWidth`**.
* **Rule bits only have effect when `FitSmushing` is active**. With `FitKerning`/`FitFullWidth`, rule bits are ignored.
* It is valid to have `FitSmushing` with **no rule bits** set → **universal smushing only**.

---

## 5) Controlled Smushing Rule Precedence (Horizontal)

When smushing is active and an overlap column is considered, apply rules **in order**:

### Rule 1: Equal Character
Identical non‑space, non‑hardblank characters merge into one.
```
Left: "H"    Right: "H"    Result: "H"
Left: "#"    Right: "#"    Result: "#"
Left: "@"    Right: "@"    Result: "@"
```
**Note**: Hardblanks do NOT smush under this rule - they only smush via Rule 6.

### Rule 2: Underscore
`_` merges with border characters (`|/\\[]{}()<>`), keeping the border char.
```
Left: "_"    Right: "|"    Result: "|"
Left: "|"    Right: "_"    Result: "|"
Left: "_"    Right: "/"    Result: "/"
Left: "_"    Right: "["    Result: "["
```

### Rule 3: Hierarchy
Class priority: `|` > `/\\` > `[]` > `{}` > `()` > `<>`; when classes differ, the higher priority (earlier in list) wins.
```
Left: "/"    Right: "|"    Result: "|"  (| beats /)
Left: "["    Right: "/"    Result: "/"  (/ beats [)
Left: "{"    Right: "]"    Result: "]"  (] beats {)
Left: "("    Right: "}"    Result: "}"  (} beats ()
```

### Rule 4: Opposite Pairs
Matching bracket pairs merge into `|`.
```
Left: "["    Right: "]"    Result: "|"
Left: "{"    Right: "}"    Result: "|"
Left: "("    Right: ")"    Result: "|"
```

### Rule 5: Big‑X
Diagonal pairs form specific patterns per FIGfont v2 spec.
```
Left: "/"    Right: "\\"   Result: "|"  (/\ → |)
Left: "\\"   Right: "/"    Result: "Y"  (\/ → Y)
Left: ">"    Right: "<"    Result: "X"  (>< → X)
```

### Rule 6: Hardblank
Two hardblanks merge into one (replaced with space at final output).
```
Left: "$"    Right: "$"    Result: "$"  (if $ is hardblank)
```

### Universal Smushing
Universal smushing only applies when smushing is enabled but **NO controlled rules are defined** (bits 0-5 all clear):
- The later character (right) **overrides** the earlier character at overlapping positions
- Visible characters always override spaces and hardblanks
- When controlled rules ARE defined but no rule matches at a position, fall back to **kerning** (no smush)

---

## 6) Overlap Selection Algorithm

For each glyph boundary, choose the **maximum** overlap where:
- **With controlled rules**: Every overlapped column satisfies a defined rule
- **With universal smushing**: Use universal override semantics
- **When no rule matches**: Fall back to kerning distance

### Step-by-Step Process

```
Glyph A (right edge)    Glyph B (left edge)
      ...|                 |...
      ...#                 #...
      .../                 \\...
```

1. **Calculate maximum potential overlap** (limited by glyph widths)
2. **For each overlap amount (max → 1)**:
   - Check each overlapped column pair
   - Verify ALL columns satisfy a rule (controlled or universal)
   - If yes: use this overlap amount
   - If no: try smaller overlap
3. **If no valid overlap**: fall back to kerning distance

### Example: Attempting 2-column overlap

```
Trying overlap=2:
  Column 1: '|' + '|' → '|' (Rule 1: Equal) ✓
  Column 2: '#' + '#' → '#' (Rule 1: Equal) ✓
Result: Valid 2-column overlap

Trying overlap=2:
  Column 1: '/' + '\\' → '|' (Rule 5: Big-X) ✓
  Column 2: ' ' + 'a' → 'a' (Universal) ✓
Result: Valid 2-column overlap

Trying overlap=2:
  Column 1: 'a' + 'b' → ??? (No rule applies) ✗
Result: Reduce to overlap=1 or kerning
```

---

## 7) Hardblank Handling

* Treat the hardblank as a **non‑space** during collision checks.
* After final composition, **replace all hardblanks with spaces** in the rendered output.

---

## 8) Print Direction

* Default to the font's `PrintDirection` (0 = LTR, 1 = RTL). Allow user override via `WithPrintDirection`.
* For RTL, compose glyphs right‑to‑left; rule precedence remains unchanged for per‑column decisions.

---

## 9) Unknown Rune Handling

When rendering text, runes are handled as follows:

1. **If rune is in font's Glyphs map** → render normally
2. **If rune is outside ASCII 32-126 or absent from font**:
   * Replace with configured "unknown rune" replacement
   * Default replacement is `'?'`
   * Callers can override via `WithUnknownRune(r rune)` option
   * CLI users can override via `-u, --unknown-rune <rune>` flag
3. **If replacement rune is not in font**:
   * Library: fall back to `'?'` if available, else return `ErrUnsupportedRune`
   * CLI: warn and fall back to `'?'` if available, else exit with error

### CLI Flag Formats

The `-u, --unknown-rune` flag accepts:
* Literal character: `*`, `?`
* Escaped Unicode: `\uXXXX`, `\UXXXXXXXX`
* Unicode notation: `U+XXXX`
* Decimal: `63`
* Hexadecimal: `0x3F`

---

## 10) Test Matrix (Minimum)

* **Fonts:** `standard`, `slant`, `small`, `big`
* **Layouts:** full‑width, kerning, smushing (rules on/off)
* **Inputs:**

  * `"Hello, World!"` — Basic test
  * `"FIGgo 1.0"` — Mixed alphanumeric
  * `"|/\\[]{}()<>"` — Rule trigger set
  * `"The quick brown fox jumps over the lazy dog"` — Full alphabet
  * Long ASCII line (> 120 chars) — Stress test
  * `""` — Empty string (edge case)
  * `" "` — Single space (edge case)
  * `"a"` — Single character (edge case)
  * `"   "` — Multiple spaces (edge case)
  * `"$$$$"` — Consecutive hardblanks (edge case)
  * `"\t\n\r"` — Control characters (edge case)
  * Mixed case with symbols: `"!@#$%^&*()_+-=[]{}"` (edge case)

---

## 11) Known Deviations

* *(None yet.)* Add entries here if the implementation intentionally diverges from FIGfont behavior; include rationale and tests.

---

## 12) PR Review Checklist (Plain Bullets)

* Header parsing matches this page (unit tests included)
* OldLayout → Layout normalization covered by tests (range validation, mode conversion)
* FullLayout bit mapping and validation rules enforced (unit/property tests)
* FullLayout presence tracking via field count implemented
* Precedence rules tested (FullLayout overrides OldLayout when present)
* Universal vs controlled smushing distinction tested
* Controlled smushing precedence verified with targeted glyph‑pair tests
* Goldens regenerated with `tools/generate_goldens.sh` and committed
* CI compares Figgo output byte‑for‑byte to goldens (fails on drift)
* No hardblank leakage in final output; replacement verified
* RTL (`PrintDirection=1`) behavior verified (font default + override)
* Unknown‑rune policy exercised (replacement char configurable)
* Docs updated (this page + PRD + fonts licensing)

---
