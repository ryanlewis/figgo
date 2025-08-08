# Figgo — Spec Compliance Guide

This document is the **source of truth** for how Figgo interprets **FIGfont v2** and maps font header data to Figgo’s runtime `Layout` flags and rendering behavior. When a PR claims “spec compliant,” reviewers should validate it against this page.

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
* Glyph blocks for ASCII 32–126 (MVP scope)

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

1. **Equal character** — identical non‑space; keep that character.
2. **Underscore** — `_` + border chars (`|/\\[]{}()<>`) → the border char.
3. **Hierarchy** — class order `|` > `/\\` > `[]` > `{}` > `()`; higher class survives.
4. **Opposite pairs** — `[]`, `{}`, `()` → `|`.
5. **Big‑X** — `/\\` → `X`, `><` → `X`.
6. **Hardblank** — two hardblanks smush to one hardblank.

If no controlled rule matches for a candidate column but smushing is allowed, fall back to **universal smushing**: take right if left is space, left if right is space; otherwise **do not smush** that column (keep kerning distance). **Hardblank collisions never universal‑smush.**

---

## 6) Overlap Selection Algorithm

For each glyph boundary, choose the **maximum** overlap such that **every** overlapped column is valid by either a controlled rule or the universal rule. If any overlapped column would violate the rules, reduce the overlap; if no valid overlap remains, fall back to the **kerning distance**.

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

  * `"Hello, World!"`
  * `"FIGgo 1.0"`
  * `"|/\\[]{}()<>"` (rule‑trigger set)
  * A long ASCII line (> 120 chars)

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
