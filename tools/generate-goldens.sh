#!/bin/sh
set -eu

# Generate golden fixtures as markdown files with metadata
# Usage:
#   FIGLET=/path/to/figlet ./tools/generate-goldens.sh
#   FONTS="standard slant" LAYOUTS="full kern smush" ./tools/generate-goldens.sh
#   STRICT=1 ./tools/generate-goldens.sh  # Fail on warnings (CI mode)
#   SAMPLES_FILE=custom_samples.txt ./tools/generate-goldens.sh
# Requires: figlet (C implementation) in PATH unless FIGLET is set

# Force consistent locale for deterministic output
export LC_ALL=C

FIGLET="${FIGLET:-figlet}"
OUT_DIR="${OUT_DIR:-testdata/goldens}"
FONTS="${FONTS:-standard slant small big}"
LAYOUTS="${LAYOUTS:-default full kern smush}"
INDEX_FILE="${OUT_DIR}/index.md"
STRICT="${STRICT:-0}"
SAMPLES_FILE="${SAMPLES_FILE:-}"
FONTDIR="${FONTDIR:-}"  # Optional font directory for figlet

# Sample strings (ASCII only, one per line)
if [ -n "$SAMPLES_FILE" ] && [ -f "$SAMPLES_FILE" ]; then
  SAMPLES=$(cat "$SAMPLES_FILE")
else
  # Default samples including edge cases from issue #26
  SAMPLES=$(cat <<'EOS'
Hello, World!
FIGgo 1.0
|/\[]{}()<>
The quick brown fox jumps over the lazy dog
 
a
   
$$$$
!@#$%^&*()_+-=[]{}:;'",.<>?/\|
ABCDEFGHIJKLMNOPQRSTUVWXYZ
abcdefghijklmnopqrstuvwxyz
0123456789
EOS
)
fi

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
  # Replace non-alnum with _, handle special cases
  if [ -z "$1" ]; then
    printf "empty"
  elif [ "$1" = " " ]; then
    printf "space"
  elif [ "$1" = "  " ]; then
    printf "two_spaces"
  elif [ "$1" = "   " ]; then
    printf "three_spaces"
  else
    # General slugification
    slug=$(printf '%s' "$1" | tr -c 'A-Za-z0-9' '_' | sed 's/^_*//;s/_*$//;s/__*/_/g')
    # Handle collision by appending hash if needed
    if [ -z "$slug" ]; then
      # If slug is empty after processing, use hash of original
      slug=$(printf '%s' "$1" | hash_sha256 | cut -c1-8)
    fi
    printf '%s' "$slug"
  fi
}

get_layout_args() {
  case "$1" in
    default) echo "" ;;  # Default mode - no layout arguments
    full)  echo "-W" ;;  # Full width mode
    kern)  echo "-k" ;;  # Kerning mode
    smush) 
      # Detect which smushing flag is supported
      if "$FIGLET" -S test 2>/dev/null | grep -q "test"; then
        echo "-S"  # Standard smushing
      elif "$FIGLET" -m 128 test 2>/dev/null | grep -q "test"; then
        echo "-m 128"  # Alternative smushing flag
      else
        echo "-s"  # Fallback to old smushing
      fi
      ;;
    *)     echo "" ;;
  esac
}

get_layout_name() {
  case "$1" in
    default) echo "default" ;;
    full)  echo "full-width" ;;
    kern)  echo "kerning" ;;
    smush) echo "smushing" ;;
    *)     echo "$1" ;;
  esac
}

# Detect smushing flag support once
SMUSH_FLAG=""
if printf "test" | "$FIGLET" -S 2>/dev/null | grep -q "test"; then
  SMUSH_FLAG="-S"
elif printf "test" | "$FIGLET" -m 128 2>/dev/null | grep -q "test"; then
  SMUSH_FLAG="-m 128"
else
  SMUSH_FLAG="-s"
fi

# Process each combination
for font in $FONTS; do
  # Set font directory if provided
  font_args=""
  if [ -n "$FONTDIR" ]; then
    font_args="-d $FONTDIR"
  fi
  
  # Check if font is available
  if ! printf "test" | "$FIGLET" $font_args -f "$font" >/dev/null 2>&1; then
    msg="Warning: Font '$font' not available, skipping"
    echo "$msg" >&2
    if [ "$STRICT" = "1" ]; then
      echo "ERROR: In strict mode, all fonts must be available" >&2
      exit 1
    fi
    continue
  fi
  
  for layout in $LAYOUTS; do
    # Set layout args based on detected support
    case "$layout" in
      default) layout_args="" ;;
      full)  layout_args="-W" ;;
      kern)  layout_args="-k" ;;
      smush) layout_args="$SMUSH_FLAG" ;;
      *)     layout_args="" ;;
    esac
    layout_name=$(get_layout_name "$layout")
    
    # Create font/layout directory
    mkdir -p "$OUT_DIR/$font/$layout_name"
    
    # Process each sample
    printf '%s\n' "$SAMPLES" | while IFS= read -r sample; do
      slug=$(slugify "$sample")
      out_file="$OUT_DIR/$font/$layout_name/${slug}.md"
      
      # Don't skip empty or space samples - they're important edge cases
      
      printf 'Generating %s/%s/%s.md\n' "$font" "$layout_name" "$slug"
      
      # Get font info and layout info for metadata
      font_info=""
      layout_info=""
      if command -v "$FIGLET" >/dev/null 2>&1; then
        font_info=$("$FIGLET" $font_args -f "$font" -I 0 2>/dev/null | head -1 || echo "")
        layout_info=$("$FIGLET" $font_args -f "$font" -I 1 2>/dev/null | head -1 || echo "")
      fi
      
      # Generate the ASCII art
      if ! art_output=$(printf '%s' "$sample" | "$FIGLET" $font_args -f "$font" $layout_args 2>/dev/null); then
        msg="Warning: figlet failed for font=$font layout=$layout_name sample=$slug"
        echo "$msg" >&2
        if [ "$STRICT" = "1" ]; then
          echo "ERROR: In strict mode, all renders must succeed" >&2
          exit 1
        fi
        continue
      fi
      
      # Calculate checksum of the art output
      checksum=$(printf '%s' "$art_output" | hash_sha256 | awk '{print $1}')
      
      # Escape sample for YAML (handle quotes and special chars)
      escaped_sample=$(printf '%s' "$sample" | sed 's/"/\\"/g')
      
      # Create markdown file with enhanced front matter per issue #26
      cat > "$out_file" << EOF
---
font: $font
layout: $layout
sample: "$escaped_sample"
figlet_version: $FIGLET_VERSION
font_info: "$font_info"
layout_info: "$layout_info"
print_direction: 0
generated: "$(date -u +"%Y-%m-%d" 2>/dev/null || date +"%Y-%m-%d")"
generator: generate-goldens.sh
figlet_args: "$layout_args"
checksum_sha256: "$checksum"
---

\`\`\`text
$art_output
\`\`\`
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