# Performance Guardrails Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Establish baseline performance metrics, create documentation structure, add missing benchmark coverage, and implement CI regression detection for the figgo library.

**Architecture:** Three-phase approach: (1) validate existing benchmarks meet PRD targets and create documentation, (2) add missing benchmark coverage for smushing rules and E2E pipeline, (3) implement CI-based regression detection with baseline comparison.

**Tech Stack:** Go 1.22+, `go test -bench`, GitHub Actions, shell scripting for comparison tooling.

---

## Context

**PRD Performance Targets (§10):**

- p50 render latency: < 50µs for "The quick brown fox" with standard.flf
- Allocations: < 4 allocs/op with pooling enabled
- Throughput: ~1M glyph merges/sec (stretch goal)

**Existing Infrastructure:**

- 4 benchmark files already exist (parser, pool, cache, renderer)
- CI runs benchmarks on PRs but doesn't compare results
- `just bench` command available

**Gaps to Fill:**

- No `benchmarks/` directory with documentation (PRD §11 deliverable)
- No baseline results stored
- No regression detection
- Missing per-rule smushing benchmarks

---

## Phase 1: Validate & Document Existing Benchmarks

### Task 1: Create Benchmarks Directory Structure

**Files:**

- Create: `benchmarks/README.md`
- Create: `benchmarks/TARGETS.md`

**Step 1: Create benchmarks directory**

```bash
mkdir -p benchmarks
```

**Step 2: Create README.md with interpretation guide**

Create `benchmarks/README.md`:

````markdown
# Figgo Benchmarks

This directory contains performance benchmarks and documentation for the figgo library.

## Running Benchmarks

```bash
# Run all benchmarks
just bench

# Run specific benchmark
go test -bench=BenchmarkRenderOptimized -benchmem ./internal/renderer/

# Run with CPU profiling
go test -bench=BenchmarkRenderOptimized -cpuprofile=cpu.prof ./internal/renderer/

# Run with memory profiling
go test -bench=BenchmarkRenderOptimized -memprofile=mem.prof ./internal/renderer/
```
````

## Benchmark Files

| File                                     | Package  | Focus                        |
| ---------------------------------------- | -------- | ---------------------------- |
| `internal/parser/parser_bench_test.go`   | parser   | Font parsing, header parsing |
| `internal/parser/pool_bench_test.go`     | parser   | Memory pooling effectiveness |
| `internal/renderer/render_bench_test.go` | renderer | Rendering performance        |
| `font_cache_bench_test.go`               | figgo    | LRU cache performance        |
| `prd_bench_test.go`                      | figgo    | PRD target validation        |

## Interpreting Results

### Key Metrics

- **ns/op**: Nanoseconds per operation (lower is better)
- **B/op**: Bytes allocated per operation (lower is better)
- **allocs/op**: Number of allocations per operation (target: < 4)

### Expected Ranges

| Benchmark                    | Expected ns/op | Expected allocs/op |
| ---------------------------- | -------------- | ------------------ |
| PRD Target (quick brown fox) | < 50,000       | < 4                |
| Simple render (HELLO)        | < 20,000       | < 4                |
| Long text render             | < 100,000      | < 10               |

### Variance

Benchmark results can vary by 5-15% between runs. For reliable comparisons:

1. Run benchmarks multiple times (`-count=5`)
2. Use `benchstat` to compare results
3. Ensure machine is idle during benchmarking

## Updating Baselines

After intentional performance changes:

```bash
just bench > benchmarks/baseline.txt
git add benchmarks/baseline.txt
git commit -m "perf: update benchmark baselines"
```

````

**Step 3: Create TARGETS.md documenting PRD targets**

Create `benchmarks/TARGETS.md`:

```markdown
# Performance Targets

These targets are defined in the PRD (§10) and measured on Go 1.22, 8-core dev machine.

## Primary Targets

| Metric | Target | Benchmark | Status |
|--------|--------|-----------|--------|
| p50 render latency | < 50µs | `BenchmarkPRDTarget_QuickBrownFox` | TBD |
| Allocations per op | < 4 | `BenchmarkPRDTarget_Allocations` | TBD |
| Glyph merge throughput | ~1M/sec | `BenchmarkPRDTarget_MergeThruput` | Stretch |

## Test Case

The primary benchmark uses:
- **Text:** "The quick brown fox"
- **Font:** standard.flf
- **Layout:** Default smushing from font

## Measurement Notes

- Targets are indicative, not hard SLAs
- Measured in single-threaded mode
- Memory pooling must be enabled
- Results may vary by hardware

## Optimisation Strategy (from PRD)

1. Precompute per-glyph left/right trim widths
2. Use `strings.Builder` + `[]byte` scratch buffers from `sync.Pool`
3. Avoid allocations in hot loops
4. No locking on Font (read-only)
````

**Step 4: Commit the documentation structure**

```bash
git add benchmarks/
git commit -m "docs: add benchmarks directory structure - closes #34 partial"
```

---

### Task 2: Create PRD Target Validation Benchmarks

**Files:**

- Create: `prd_bench_test.go`

**Step 1: Write the PRD target benchmark tests**

Create `prd_bench_test.go`:

```go
package figgo

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkPRDTarget_QuickBrownFox validates the PRD p50 latency target.
// Target: < 50µs for "The quick brown fox" with standard.flf
func BenchmarkPRDTarget_QuickBrownFox(b *testing.B) {
	fontPath := filepath.Join("fonts", "standard.flf")
	f, err := os.Open(fontPath)
	if err != nil {
		b.Fatalf("failed to open font: %v", err)
	}
	defer f.Close()

	font, err := ParseFont(f)
	if err != nil {
		b.Fatalf("failed to parse font: %v", err)
	}

	text := "The quick brown fox"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := Render(text, font)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPRDTarget_Allocations validates the allocation target.
// Target: < 4 allocs/op with pooling enabled
func BenchmarkPRDTarget_Allocations(b *testing.B) {
	fontPath := filepath.Join("fonts", "standard.flf")
	f, err := os.Open(fontPath)
	if err != nil {
		b.Fatalf("failed to open font: %v", err)
	}
	defer f.Close()

	font, err := ParseFont(f)
	if err != nil {
		b.Fatalf("failed to parse font: %v", err)
	}

	text := "The quick brown fox"
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		err := RenderTo(&buf, text, font)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPRDTarget_MergeThruput measures glyph merge throughput.
// Stretch target: ~1M glyph merges/sec with smushing enabled
func BenchmarkPRDTarget_MergeThruput(b *testing.B) {
	fontPath := filepath.Join("fonts", "standard.flf")
	f, err := os.Open(fontPath)
	if err != nil {
		b.Fatalf("failed to open font: %v", err)
	}
	defer f.Close()

	font, err := ParseFont(f)
	if err != nil {
		b.Fatalf("failed to parse font: %v", err)
	}

	// Use text with many characters to measure merge throughput
	text := "The quick brown fox jumps over the lazy dog"
	charCount := len(text) - 1 // merges = characters - 1

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := Render(text, font)
		if err != nil {
			b.Fatal(err)
		}
	}

	// Report merges per second
	mergesPerOp := float64(charCount)
	opsPerSec := float64(b.N) / b.Elapsed().Seconds()
	mergesPerSec := mergesPerOp * opsPerSec
	b.ReportMetric(mergesPerSec, "merges/sec")
}

// BenchmarkPRDTarget_AllFonts tests PRD target across all bundled fonts
func BenchmarkPRDTarget_AllFonts(b *testing.B) {
	fonts := []string{"standard.flf", "slant.flf", "small.flf", "big.flf"}
	text := "The quick brown fox"

	for _, fontName := range fonts {
		b.Run(fontName, func(b *testing.B) {
			fontPath := filepath.Join("fonts", fontName)
			f, err := os.Open(fontPath)
			if err != nil {
				b.Skipf("font not found: %s", fontPath)
				return
			}
			defer f.Close()

			font, err := ParseFont(f)
			if err != nil {
				b.Fatalf("failed to parse font: %v", err)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = Render(text, font)
			}
		})
	}
}
```

**Step 2: Run the benchmarks to verify they work**

```bash
go test -bench=BenchmarkPRDTarget -benchmem .
```

Expected output format:

```
BenchmarkPRDTarget_QuickBrownFox-8         XXXXX         XXXXX ns/op         XXX B/op         X allocs/op
BenchmarkPRDTarget_Allocations-8           XXXXX         XXXXX ns/op         XXX B/op         X allocs/op
BenchmarkPRDTarget_MergeThruput-8          XXXXX         XXXXX ns/op         XXX B/op         X allocs/op    XXXXXX merges/sec
```

**Step 3: Commit the PRD target benchmarks**

```bash
git add prd_bench_test.go
git commit -m "test: add PRD target validation benchmarks"
```

---

### Task 3: Capture Initial Baseline

**Files:**

- Create: `benchmarks/baseline.txt`

**Step 1: Run full benchmark suite and capture results**

```bash
go test -bench=. -benchmem -count=3 ./... 2>/dev/null | tee benchmarks/baseline.txt
```

**Step 2: Verify baseline file was created**

```bash
head -50 benchmarks/baseline.txt
```

**Step 3: Commit the baseline**

```bash
git add benchmarks/baseline.txt
git commit -m "perf: capture initial benchmark baseline"
```

---

## Phase 2: Add Missing Benchmark Coverage

### Task 4: Add Per-Rule Smushing Benchmarks

**Files:**

- Modify: `internal/renderer/smushing_test.go`

**Step 1: Read the existing smushing test file**

Read `internal/renderer/smushing_test.go` to understand the current structure.

**Step 2: Add benchmarks for each smushing rule**

Append to `internal/renderer/smushing_test.go`:

```go
// Benchmark smushing rules individually to identify performance characteristics

func createSmushBenchState(smushMode int) *renderState {
	state := &renderState{
		smushMode:         smushMode,
		hardblank:         '$',
		previousCharWidth: 7,
		currentCharWidth:  7,
		right2left:        0,
	}
	return state
}

// BenchmarkSmush_Universal benchmarks universal smushing (no rules set)
func BenchmarkSmush_Universal(b *testing.B) {
	state := createSmushBenchState(SMSmush) // Smushing enabled, no rules

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.smush('A', 'B')
		_ = state.smush('|', '/')
		_ = state.smush('[', ']')
	}
}

// BenchmarkSmush_EqualChar benchmarks Rule 1: Equal character smushing
func BenchmarkSmush_EqualChar(b *testing.B) {
	state := createSmushBenchState(SMSmush | SMEqual)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.smush('|', '|')
		_ = state.smush('/', '/')
		_ = state.smush('A', 'A')
	}
}

// BenchmarkSmush_Underscore benchmarks Rule 2: Underscore smushing
func BenchmarkSmush_Underscore(b *testing.B) {
	state := createSmushBenchState(SMSmush | SMLowline)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.smush('_', '|')
		_ = state.smush('_', '/')
		_ = state.smush('[', '_')
	}
}

// BenchmarkSmush_Hierarchy benchmarks Rule 3: Hierarchy smushing
// This is the most complex rule with multiple level checks
func BenchmarkSmush_Hierarchy(b *testing.B) {
	state := createSmushBenchState(SMSmush | SMHierarchy)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.smush('|', '/')
		_ = state.smush('[', '{')
		_ = state.smush('(', '<')
		_ = state.smush('/', '[')
		_ = state.smush('{', '(')
	}
}

// BenchmarkSmush_OppositePair benchmarks Rule 4: Opposite pair smushing
func BenchmarkSmush_OppositePair(b *testing.B) {
	state := createSmushBenchState(SMSmush | SMPair)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.smush('[', ']')
		_ = state.smush('{', '}')
		_ = state.smush('(', ')')
	}
}

// BenchmarkSmush_BigX benchmarks Rule 5: Big X smushing
func BenchmarkSmush_BigX(b *testing.B) {
	state := createSmushBenchState(SMSmush | SMBigX)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.smush('/', '\\')
		_ = state.smush('\\', '/')
		_ = state.smush('>', '<')
	}
}

// BenchmarkSmush_Hardblank benchmarks Rule 6: Hardblank smushing
func BenchmarkSmush_Hardblank(b *testing.B) {
	state := createSmushBenchState(SMSmush | SMHardblank)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.smush('$', '$')
		_ = state.smush('$', 'A')
		_ = state.smush('A', '$')
	}
}

// BenchmarkSmush_AllRules benchmarks all rules enabled together
func BenchmarkSmush_AllRules(b *testing.B) {
	state := createSmushBenchState(SMSmush | SMEqual | SMLowline | SMHierarchy | SMPair | SMBigX | SMHardblank)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.smush('|', '|')   // Equal
		_ = state.smush('_', '|')   // Underscore
		_ = state.smush('|', '/')   // Hierarchy
		_ = state.smush('[', ']')   // Pair
		_ = state.smush('/', '\\')  // BigX
		_ = state.smush('$', '$')   // Hardblank
	}
}

// BenchmarkSmush_NoMatch benchmarks the case where no rules match
func BenchmarkSmush_NoMatch(b *testing.B) {
	state := createSmushBenchState(SMSmush | SMEqual | SMLowline | SMHierarchy | SMPair | SMBigX | SMHardblank)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.smush('A', 'B') // No rule matches
		_ = state.smush('X', 'Y')
		_ = state.smush('1', '2')
	}
}
```

**Step 3: Verify the benchmarks compile and run**

```bash
go test -bench=BenchmarkSmush -benchmem ./internal/renderer/
```

**Step 4: Commit the smushing benchmarks**

```bash
git add internal/renderer/smushing_test.go
git commit -m "test: add per-rule smushing benchmarks"
```

---

### Task 5: Add End-to-End Pipeline Benchmarks

**Files:**

- Modify: `prd_bench_test.go`

**Step 1: Add E2E pipeline benchmarks**

Append to `prd_bench_test.go`:

```go
// BenchmarkE2E_ParseAndRender measures the full pipeline without caching
func BenchmarkE2E_ParseAndRender(b *testing.B) {
	fontPath := filepath.Join("fonts", "standard.flf")
	text := "Hello World"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		f, err := os.Open(fontPath)
		if err != nil {
			b.Fatal(err)
		}

		font, err := ParseFont(f)
		f.Close()
		if err != nil {
			b.Fatal(err)
		}

		_, err = Render(text, font)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkE2E_CachedRender measures rendering with pre-parsed font
func BenchmarkE2E_CachedRender(b *testing.B) {
	fontPath := filepath.Join("fonts", "standard.flf")
	f, err := os.Open(fontPath)
	if err != nil {
		b.Fatalf("failed to open font: %v", err)
	}

	font, err := ParseFont(f)
	f.Close()
	if err != nil {
		b.Fatalf("failed to parse font: %v", err)
	}

	text := "Hello World"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err = Render(text, font)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkE2E_FontCacheHit measures performance with font cache
func BenchmarkE2E_FontCacheHit(b *testing.B) {
	cache := NewFontCache(10)
	fontPath := filepath.Join("fonts", "standard.flf")

	// Pre-warm the cache
	_, err := cache.LoadFont(fontPath)
	if err != nil {
		b.Fatalf("failed to load font: %v", err)
	}

	text := "Hello World"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		font, err := cache.LoadFont(fontPath)
		if err != nil {
			b.Fatal(err)
		}

		_, err = Render(text, font)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkE2E_LayoutModes compares performance across layout modes
func BenchmarkE2E_LayoutModes(b *testing.B) {
	fontPath := filepath.Join("fonts", "standard.flf")
	f, err := os.Open(fontPath)
	if err != nil {
		b.Fatalf("failed to open font: %v", err)
	}

	font, err := ParseFont(f)
	f.Close()
	if err != nil {
		b.Fatalf("failed to parse font: %v", err)
	}

	text := "The quick brown fox"

	b.Run("FullWidth", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = Render(text, font, WithLayout(LayoutFullWidth))
		}
	})

	b.Run("Kerning", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = Render(text, font, WithLayout(LayoutKerning))
		}
	})

	b.Run("Smushing", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = Render(text, font, WithLayout(LayoutSmushing))
		}
	})
}
```

**Step 2: Verify the benchmarks compile and run**

```bash
go test -bench=BenchmarkE2E -benchmem .
```

**Step 3: Commit the E2E benchmarks**

```bash
git add prd_bench_test.go
git commit -m "test: add end-to-end pipeline benchmarks"
```

---

## Phase 3: CI Regression Detection

### Task 6: Create Benchmark Comparison Script

**Files:**

- Create: `scripts/bench-compare.sh`

**Step 1: Create scripts directory**

```bash
mkdir -p scripts
```

**Step 2: Create the comparison script**

Create `scripts/bench-compare.sh`:

```bash
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
```

**Step 3: Make the script executable**

```bash
chmod +x scripts/bench-compare.sh
```

**Step 4: Commit the comparison script**

```bash
git add scripts/
git commit -m "feat: add benchmark comparison script"
```

---

### Task 7: Add Justfile Commands

**Files:**

- Modify: `Justfile`

**Step 1: Read the current Justfile**

Read `Justfile` to find the right location for new commands.

**Step 2: Add benchmark comparison commands**

Append to `Justfile` after the `bench` target:

```just
# Compare benchmarks against baseline
bench-compare:
    ./scripts/bench-compare.sh

# Update benchmark baseline (run after intentional performance changes)
bench-update:
    go test -bench=. -benchmem -count=3 ./... 2>/dev/null > benchmarks/baseline.txt
    @echo "Baseline updated: benchmarks/baseline.txt"

# Run benchmarks with CPU profiling
bench-profile:
    go test -bench=BenchmarkPRDTarget -cpuprofile=cpu.prof .
    @echo "CPU profile written to cpu.prof"
    @echo "View with: go tool pprof -http=:8080 cpu.prof"
```

**Step 3: Verify the new commands work**

```bash
just --list | grep bench
```

Expected output:

```
bench           # Run benchmarks
bench-compare   # Compare benchmarks against baseline
bench-update    # Update benchmark baseline
bench-profile   # Run benchmarks with CPU profiling
```

**Step 4: Commit the Justfile changes**

```bash
git add Justfile
git commit -m "feat: add benchmark comparison commands to Justfile"
```

---

### Task 8: Update CI Workflow for Regression Detection

**Files:**

- Modify: `.github/workflows/ci.yml`

**Step 1: Read the current CI workflow**

Read `.github/workflows/ci.yml` to understand the current structure.

**Step 2: Update the benchmark job**

Replace the benchmark job section (around lines 122-142) with:

````yaml
benchmark:
  name: Benchmark
  runs-on: ubuntu-latest
  if: github.event_name == 'pull_request'
  steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Setup Go
      uses: actions/setup-go@v5
      with:
        go-version: stable

    - name: Install benchstat
      run: go install golang.org/x/perf/cmd/benchstat@latest

    - name: Run benchmarks
      run: go test -bench=. -benchmem -count=3 ./... 2>/dev/null | tee current.txt

    - name: Compare with baseline
      id: compare
      continue-on-error: true
      run: |
        if [[ -f benchmarks/baseline.txt ]]; then
          echo "## Benchmark Comparison" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY
          echo '```' >> $GITHUB_STEP_SUMMARY
          benchstat benchmarks/baseline.txt current.txt >> $GITHUB_STEP_SUMMARY 2>&1 || true
          echo '```' >> $GITHUB_STEP_SUMMARY

          # Check for regressions
          REGRESSIONS=$(benchstat benchmarks/baseline.txt current.txt 2>/dev/null | grep -E '\+[2-9][0-9]\.[0-9]+%|\+[1-9][0-9]{2,}\.[0-9]+%' || true)
          if [[ -n "$REGRESSIONS" ]]; then
            echo "" >> $GITHUB_STEP_SUMMARY
            echo "### Warning: Potential Regressions" >> $GITHUB_STEP_SUMMARY
            echo '```' >> $GITHUB_STEP_SUMMARY
            echo "$REGRESSIONS" >> $GITHUB_STEP_SUMMARY
            echo '```' >> $GITHUB_STEP_SUMMARY
            echo "regression=true" >> $GITHUB_OUTPUT
          fi
        else
          echo "No baseline found, skipping comparison" >> $GITHUB_STEP_SUMMARY
        fi

    - name: Upload benchmark results
      uses: actions/upload-artifact@v4
      with:
        name: benchmark-results
        path: current.txt

    - name: Warn on regression
      if: steps.compare.outputs.regression == 'true'
      run: |
        echo "::warning::Performance regression detected. Review benchmark comparison in job summary."
````

**Step 3: Commit the CI changes**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add benchmark regression detection"
```

---

### Task 9: Update TARGETS.md with Validation Results

**Files:**

- Modify: `benchmarks/TARGETS.md`

**Step 1: Run PRD target benchmarks and capture results**

```bash
go test -bench=BenchmarkPRDTarget -benchmem . 2>/dev/null
```

**Step 2: Update TARGETS.md with actual results**

Update the Status column in `benchmarks/TARGETS.md` with the actual measurements.

**Step 3: Commit the updated targets**

```bash
git add benchmarks/TARGETS.md
git commit -m "docs: update benchmark targets with validation results"
```

---

### Task 10: Final Integration and Cleanup

**Files:**

- Modify: `benchmarks/baseline.txt` (regenerate)

**Step 1: Run full test suite to verify everything works**

```bash
just ci
```

**Step 2: Regenerate baseline with all new benchmarks**

```bash
just bench-update
```

**Step 3: Run comparison to verify it works**

```bash
just bench-compare
```

Expected output: "No significant regressions detected."

**Step 4: Create final commit**

```bash
git add .
git commit -m "feat: complete performance guardrails implementation - closes #34"
```

---

## Verification Checklist

Before marking #34 as complete:

- [ ] `benchmarks/README.md` exists with interpretation guide
- [ ] `benchmarks/TARGETS.md` documents PRD targets
- [ ] `benchmarks/baseline.txt` contains current baseline
- [ ] `prd_bench_test.go` validates PRD targets
- [ ] Per-rule smushing benchmarks added to `smushing_test.go`
- [ ] E2E pipeline benchmarks work
- [ ] `just bench-compare` runs successfully
- [ ] CI benchmark job includes regression detection
- [ ] All tests pass: `just ci`
