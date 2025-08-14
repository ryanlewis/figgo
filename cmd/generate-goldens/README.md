# Golden Test Generator

This tool generates golden test files for FIGlet compatibility testing.

## Usage

### Basic usage:
```bash
go run ./cmd/generate-goldens
```

### With custom options:
```bash
go run ./cmd/generate-goldens \
  -fonts "standard slant small big" \
  -layouts "default full kern smush" \
  -out testdata/goldens
```

### Available flags:
- `-fonts`: Space-separated list of fonts to test (default: "standard slant small big")
- `-layouts`: Space-separated list of layouts to test (default: "default full kern smush")
- `-out`: Output directory for golden files (default: "testdata/goldens")
- `-figlet`: Path to figlet binary (default: "figlet")
- `-fontdir`: Font directory for figlet (optional)
- `-strict`: Exit on any warning (default: false)
- `-index`: Path to index file, empty to skip (optional)

## Test Samples

The generator uses the following test samples by default:
- Basic text: "Hello, World!", "FIGgo 1.0"
- Special characters: `|/\[]{}()<>`, `!@#$%^&*()_+-=[]{}:;'",.<>?/\|`
- Edge cases: single space, three spaces, empty string
- Full alphabets: uppercase, lowercase, digits
- Long text: "The quick brown fox jumps over the lazy dog"

## Output Format

Each golden file is a Markdown file with:
1. YAML front matter containing metadata
2. The ASCII art output in a code block

Example:
```yaml
---
font: standard
layout: default
sample: "Hello, World!"
figlet_version: FIGlet Copyright
font_info: "FIGlet Copyright (C) 1991-2012..."
layout_info: "20205"
print_direction: 0
generated: "2025-08-14"
generator: generate-goldens
figlet_args: ""
checksum_sha256: "..."
---

```text
 _   _      _ _         __        __         _     _ _ 
| | | | ___| | | ___    \ \      / /__  _ __| | __| | |
| |_| |/ _ \ | |/ _ \    \ \ /\ / / _ \| '__| |/ _` | |
|  _  |  __/ | | (_) |    \ V  V / (_) | |  | | (_| |_|
|_| |_|\___|_|_|\___/      \_/\_/ \___/|_|  |_|\__,_(_)
```

## Requirements

- Go 1.22 or later
- `figlet` command-line tool installed and in PATH
- FIGlet fonts available (standard, slant, small, big)