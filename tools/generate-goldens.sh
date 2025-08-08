#!/bin/sh
set -eu

# Generate golden fixtures as markdown files with metadata
# Usage:
#   FIGLET=/path/to/figlet ./tools/generate-goldens.sh
#   FONTS="standard slant" LAYOUTS="full kern smush" ./tools/generate-goldens.sh
# Requires: figlet (C implementation) in PATH unless FIGLET is set

FIGLET="${FIGLET:-figlet}"
OUT_DIR="${OUT_DIR:-testdata/goldens}"
FONTS="${FONTS:-standard slant small big}"
LAYOUTS="${LAYOUTS:-full kern smush}"
INDEX_FILE="${OUT_DIR}/index.md"

# Sample strings (ASCII only, one per line)
SAMPLES=$(cat <<'EOS'
Hello, World!
FIGgo 1.0
|/\[]{}()<>
The quick brown fox jumps over the lazy dog
""
" "
a
   
$$$$
!@#$%^&*()_+-=[]{}:;'",.<>?/\|
EOS
)

# Ensure dependencies
if ! command -v "$FIGLET" >/dev/null 2>&1; then
  echo "error: could not find figlet: set FIGLET=/path/to/figlet or install it" >&2
  exit 1
fi

# Check figlet version (use -I 5 for version info)
FIGLET_VERSION=$("$FIGLET" -I 5 2>/dev/null | head -1 || echo "unknown")
echo "Using figlet version: $FIGLET_VERSION"

# Define hash function for portability
hash_sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum
  else
    shasum -a 256
  fi
}

# Create output directory
mkdir -p "$OUT_DIR"

# Initialize index file
cat > "$INDEX_FILE" << 'EOF'
# Golden Test Fixtures

Generated test fixtures for Figgo compliance testing.

## Generation Info

EOF

printf '%s\n' "- **Generated:** $(date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date)" >> "$INDEX_FILE"
printf '%s\n' "- **Figlet Version:** $FIGLET_VERSION" >> "$INDEX_FILE"
printf '%s\n' "- **Script:** tools/generate-goldens.sh" >> "$INDEX_FILE"
printf '\n' >> "$INDEX_FILE"

cat >> "$INDEX_FILE" << 'EOF'
## Files

| Font | Layout | Input | File | SHA256 |
|------|--------|-------|------|--------|
EOF

slugify() {
  # Replace non-alnum with _, handle empty string
  if [ -z "$1" ]; then
    printf "empty"
  else
    printf '%s' "$1" | tr -c 'A-Za-z0-9' '_' | sed 's/^_*//;s/_*$//;s/__*/_/g'
  fi
}

get_layout_args() {
  case "$1" in
    full)  echo "" ;;
    kern)  echo "-k" ;;
    smush) echo "-S" ;;
    *)     echo "" ;;
  esac
}

get_layout_name() {
  case "$1" in
    full)  echo "full-width" ;;
    kern)  echo "kerning" ;;
    smush) echo "smushing" ;;
    *)     echo "$1" ;;
  esac
}

# Process each combination
for font in $FONTS; do
  # Check if font is available
  if ! printf "test" | "$FIGLET" -f "$font" >/dev/null 2>&1; then
    echo "Warning: Font '$font' not available, skipping" >&2
    continue
  fi
  
  for layout in $LAYOUTS; do
    layout_args=$(get_layout_args "$layout")
    layout_name=$(get_layout_name "$layout")
    
    # Create font/layout directory
    mkdir -p "$OUT_DIR/$font/$layout_name"
    
    # Process each sample
    printf '%s\n' "$SAMPLES" | while IFS= read -r sample; do
      slug=$(slugify "$sample")
      out_file="$OUT_DIR/$font/$layout_name/${slug}.md"
      
      # Skip if sample is empty
      if [ "$slug" = "empty" ] || [ -z "$slug" ]; then
        continue
      fi
      
      printf 'Generating %s/%s/%s.md\n' "$font" "$layout_name" "$slug"
      
      # Generate the ASCII art
      if ! art_output=$(printf '%s' "$sample" | "$FIGLET" -f "$font" $layout_args 2>/dev/null); then
        echo "Warning: figlet failed for font=$font layout=$layout_name sample=$slug" >&2
        continue
      fi
      
      # Calculate checksum of the art output
      checksum=$(printf '%s' "$art_output" | hash_sha256 | awk '{print $1}')
      
      # Escape sample for YAML (handle quotes and special chars)
      escaped_sample=$(printf '%s' "$sample" | sed 's/"/\\"/g')
      
      # Create markdown file with front matter
      cat > "$out_file" << EOF
---
font: $font
layout: $layout_name
figlet_args: "$layout_args"
input: "$escaped_sample"
generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date)
figlet_version: "$FIGLET_VERSION"
checksum_sha256: "$checksum"
---

~~~
$art_output
~~~
EOF
      
      # Add entry to index (escape pipe characters for markdown tables)
      short_checksum=$(printf '%s' "$checksum" | cut -c1-8)
      short_sample=$(printf '%s' "$escaped_sample" | cut -c1-20 | sed 's/|/\\|/g')
      printf '| %s | %s | `%s...` | [%s.md](%s/%s/%s.md) | %s... |\n' \
        "$font" \
        "$layout_name" \
        "$short_sample" \
        "$slug" \
        "$font" \
        "$layout_name" \
        "$slug" \
        "$short_checksum" >> "$INDEX_FILE"
    done
  done
done

# Generate checksums file
printf '\nGenerating checksums file...\n'
find "$OUT_DIR" -name "*.md" -type f ! -name "index.md" -print0 | \
  xargs -0 -I {} sh -c 'hash_sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum "$1"
    else
      shasum -a 256 "$1"
    fi
  }; hash_sha256 "$1"' _ {} | \
  sed "s|$OUT_DIR/||" | LC_ALL=C sort > "$OUT_DIR/checksums.txt"

# Add checksums info to index
cat >> "$INDEX_FILE" << 'EOF'

## Verification

To verify golden files haven't been modified:

```bash
cd testdata/goldens
sha256sum -c checksums.txt
```

## Regeneration

To regenerate all golden files:

```bash
./tools/generate-goldens.sh
```

To regenerate specific combinations:

```bash
FONTS="standard" LAYOUTS="smush" ./tools/generate-goldens.sh
```

---

*Note: These files are auto-generated. Do not edit manually.*
EOF

cat << 'EON'

✅ Generation complete!

Files created in: testdata/goldens/
- Individual golden files: {font}/{layout}/{input}.md
- Index file: index.md
- Checksums: checksums.txt

Each .md file contains:
- YAML front matter with metadata
- The ASCII art output in a code block

To verify integrity:
  cd testdata/goldens && sha256sum -c checksums.txt

Tests should parse the front matter and compare Figgo output
against the ASCII art in the code block.
EON