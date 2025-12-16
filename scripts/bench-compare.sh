#!/usr/bin/env bash
set -euo pipefail

# Benchmark comparison script
# Usage: ./scripts/bench-compare.sh [baseline_file]
#
# Compares current benchmark results against a baseline.
# Exit codes:
#   0 - No significant regressions
#   1 - Significant regressions detected (>25%)
#   2 - Minor regressions detected (>10%)

BASELINE_FILE="${1:-benchmarks/baseline.txt}"
THRESHOLD_MINOR=10
THRESHOLD_MAJOR=25

if [[ ! -f "$BASELINE_FILE" ]]; then
    echo "Error: Baseline file not found: $BASELINE_FILE"
    echo "Run: go test -bench=. -benchmem ./... > benchmarks/baseline.txt"
    exit 1
fi

# Check if benchstat is available
if ! command -v benchstat &> /dev/null; then
    echo "Warning: benchstat not installed, using simple comparison"
    echo "Install with: go install golang.org/x/perf/cmd/benchstat@latest"

    # Simple comparison: just run and show side-by-side
    echo ""
    echo "=== Current Benchmark Results ==="
    go test -bench=. -benchmem ./... 2>/dev/null
    echo ""
    echo "=== Baseline (from $BASELINE_FILE) ==="
    cat "$BASELINE_FILE"
    exit 0
fi

# Run current benchmarks
CURRENT_FILE=$(mktemp)
trap "rm -f $CURRENT_FILE" EXIT

echo "Running benchmarks..."
go test -bench=. -benchmem -count=3 ./... 2>/dev/null > "$CURRENT_FILE"

echo ""
echo "=== Benchmark Comparison ==="
benchstat "$BASELINE_FILE" "$CURRENT_FILE"

# Parse benchstat output for regressions
# Look for lines with significant slowdowns
REGRESSIONS=$(benchstat "$BASELINE_FILE" "$CURRENT_FILE" 2>/dev/null | grep -E '\+[0-9]+\.[0-9]+%' | grep -v '~' || true)

if [[ -n "$REGRESSIONS" ]]; then
    echo ""
    echo "=== Potential Regressions ==="
    echo "$REGRESSIONS"

    # Check for major regressions (>25%)
    MAJOR=$(echo "$REGRESSIONS" | grep -E '\+[2-9][5-9]\.[0-9]+%|\+[3-9][0-9]\.[0-9]+%|\+[0-9]{3,}\.[0-9]+%' || true)
    if [[ -n "$MAJOR" ]]; then
        echo ""
        echo "ERROR: Major performance regressions detected (>${THRESHOLD_MAJOR}%)"
        exit 1
    fi

    # Check for minor regressions (>10%)
    MINOR=$(echo "$REGRESSIONS" | grep -E '\+[1-9][0-9]\.[0-9]+%' || true)
    if [[ -n "$MINOR" ]]; then
        echo ""
        echo "WARNING: Minor performance regressions detected (>${THRESHOLD_MINOR}%)"
        exit 2
    fi
fi

echo ""
echo "No significant regressions detected."
exit 0
