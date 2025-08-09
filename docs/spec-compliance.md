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

Figgo normalizes FIGfont layout data into a single `Layout` bitmask used at render time.

### 3.1 OldLayout → Layout

```
OldLayout = -1  => FitFullWidth
OldLayout = -2  => FitKerning
OldLayout = -3  => FitSmushing  (no explicit rule bits ⇒ universal smushing only)
```

If `FullLayout` is also present, **`FullLayout` takes precedence** (see below).

### 3.2 FullLayout (font) → Layout (Figgo) — horizontal subset

* **Bit 0** — Equal character smushing → `RuleEqualChar`
* **Bit 1** — Underscore smushing → `RuleUnderscore`
* **Bit 2** — Hierarchy smushing → `RuleHierarchy`
* **Bit 3** — Opposite pair smushing → `RuleOpposite`
* **Bit 4** — Big‑X smushing → `RuleBigX`
* **Bit 5** — Hardblank smushing → `RuleHardblank`
* **Bit 6** — Kerning (mutually exclusive with smushing) → `FitKerning`
* **Bit 7** — Smushing (mutually exclusive with kerning) → `FitSmushing`

*If neither bit 6 nor 7 is set, Figgo uses `FitFullWidth`.*

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
Identical non‑space characters merge into one.
```
Left: "H"    Right: "H"    Result: "H"
Left: "#"    Right: "#"    Result: "#"
Left: "@"    Right: "@"    Result: "@"
```

### Rule 2: Underscore
`_` merges with border characters (`|/\\[]{}()<>`), keeping the border char.
```
Left: "_"    Right: "|"    Result: "|"
Left: "|"    Right: "_"    Result: "|"
Left: "_"    Right: "/"    Result: "/"
Left: "_"    Right: "["    Result: "["
```

### Rule 3: Hierarchy
Class order: `|` > `/\\` > `[]` > `{}` > `()`; higher class survives.
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
Diagonal pairs form X patterns.
```
Left: "/"    Right: "\\"   Result: "X"
Left: ">"    Right: "<"    Result: "X"
Left: "\\"   Right: "/"    Result: "Y"  (reverse order creates Y)
```

### Rule 6: Hardblank
Two hardblanks merge into one (replaced with space at final output).
```
Left: "$"    Right: "$"    Result: "$"  (if $ is hardblank)
```

### Universal Smushing Fallback
If no controlled rule matches but smushing is allowed:
- Take right if left is space
- Take left if right is space
- Otherwise **do not smush** (keep kerning distance)
- **Hardblank collisions never universal‑smush**

---

## 6) Overlap Selection Algorithm

For each glyph boundary, choose the **maximum** overlap such that **every** overlapped column is valid by either a controlled rule or the universal rule.

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
  Column 1: '/' + '\\' → 'X' (Rule 5: Big-X) ✓
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

* Default to the font’s `PrintDirection` (0 = LTR, 1 = RTL). Allow user override via `WithPrintDirection`.
* For RTL, compose glyphs right‑to‑left; rule precedence remains unchanged for per‑column decisions.

---

## 9) Test Matrix (Minimum)

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

## 10) Known Deviations

* *(None yet.)* Add entries here if the implementation intentionally diverges from FIGfont behavior; include rationale and tests.

---

## 11) PR Review Checklist (Plain Bullets)

* Header parsing matches this page (unit tests included)
* OldLayout → Layout normalization covered by tests
* FullLayout bit mapping and validation rules enforced (unit/property tests)
* Controlled smushing precedence verified with targeted glyph‑pair tests
* Goldens regenerated with `tools/generate_goldens.sh` and committed
* CI compares Figgo output byte‑for‑byte to goldens (fails on drift)
* No hardblank leakage in final output; replacement verified
* RTL (`PrintDirection=1`) behavior verified (font default + override)
* Unknown‑rune policy exercised (replacement char configurable)
* Docs updated (this page + PRD + fonts licensing)

---
