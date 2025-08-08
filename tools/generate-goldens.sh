#!/usr/bin/env bash
set -euo pipefail

# Generate golden fixtures by calling the C `figlet` binary.
# Usage:
#   FIGLET=/path/to/figlet FIGLET_ARGS="-w 120" ./tools/generate_goldens.sh
# or just:
#   ./tools/generate_goldens.sh
# Requires: figlet (C implementation) in $PATH unless FIGLET is set.

FIGLET="${FIGLET:-figlet}"
FIGLET_ARGS=${FIGLET_ARGS:-}
OUT_DIR=${OUT_DIR:-tests/goldens}
FONTS=(${FONTS:-standard slant small big})

# Sample strings (ASCII only, quoted to preserve characters)
readarray -t SAMPLES < <(cat <<'EOS'
Hello, World!
FIGgo 1.0
|/\[]{}()<>
The quick brown fox jumps over the lazy dog 0123456789 ~!@#$%^&*()-_=+[]{};:'",.<>?/\|
EOS
)

# Ensure dependencies
if ! command -v "$FIGLET" >/dev/null 2>&1; then
  echo "error: could not find figlet: set FIGLET=/path/to/figlet or install it" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

slugify() {
  # Replace non-alnum with _, collapse repeats
  printf '%s' "$1" | tr -c 'A-Za-z0-9_' '_' | sed -E 's/_+/_/g;s/^_+//;s/_+$//'
}

for font in "${FONTS[@]}"; do
  for sample in "${SAMPLES[@]}"; do
    slug=$(slugify "$sample")
    out="$OUT_DIR/${font}.${slug}.txt"
    printf 'Generating %s\n' "$out"
    # Use printf to avoid echo interpreting backslashes
    printf '%s\n' "$sample" | "$FIGLET" -f "$font" $FIGLET_ARGS > "$out"
  done
done

cat <<'EON'
Done. Committed goldens should be treated as **fixtures**:
- Do not hand-edit; regenerate via this script when fonts/inputs change.
- Tests should compare Figgo output byte-for-byte against these files.
EON
