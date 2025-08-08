# Figgo — Fonts & Licensing

This page documents which fonts Figgo ships with (if any), how to add more, and how we handle licensing/attribution.

> **Policy:** Keep the bundled set **minimal** and **permissively licensed**. Prefer linking to external font packs or a separate repo to avoid license churn in the main module.

---

## Bundled fonts (proposed minimal set)

* `standard.flf` — widely distributed; confirm license in source and include attribution if required.
* `slant.flf` — widely distributed; confirm license in source and include attribution if required.

*(If any font requires attribution, list it explicitly below and include the text verbatim.)*

---

## Attributions

Create `fonts/ATTRIBUTIONS.md` if bundling fonts that require attribution. Each entry should include:

* **Name** and short description
* **Author** and year (if known)
* **License** (SPDX identifier if possible)
* **Source URL** (where the `.flf` was obtained)

Example entry:

```md
### standard.flf
- Author: <author-if-known>
- License: <license>
- Source: <url>
```

---

## Adding fonts locally (for development/tests)

1. Place `.flf` files under `fonts/` (do **not** commit unless license is permissive and attribution is included).
2. Update this page with license details and add an entry to `fonts/ATTRIBUTIONS.md` if needed.
3. Run the test suite to verify header parsing and rendering.

---

## Non-goals (MVP)

* `.flc` control files
* Compressed/zipped font ingestion

---

## Notes

* The **hardblank** is font-specific; verify your renderer replaces it with space at the end of composition.
* Some community fonts have inconsistent headers; the parser should fail with `ErrBadFontFormat` if key fields are missing.
