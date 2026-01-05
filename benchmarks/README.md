# Figgo Performance Benchmarks

This document tracks performance targets, baselines, and regression guardrails for the Figgo library.

## Performance Targets (from PRD)

The following performance targets guide optimization efforts:

| Metric | Target | Status | Notes |
|--------|--------|--------|-------|
| **p50 Render Time** | <50µs | 🟡 Close (1.2x) | Actual: 57-61µs for "The quick brown fox" (19 chars) |
| **Allocations** | <4 allocs/op | 🔴 Needs Work (5-12x) | Actual: 22-50 allocs/op (varies by text length) |
| **Throughput** | ~1M merges/sec | 🟡 Partial (30%) | Actual: 100K-640K merges/sec (stretch goal) |

**Test Platform:** Intel(R) N150, Linux, Go 1.23

## Baseline Measurements

### Quick Brown Fox Benchmark (PRD Target Text)

Text: "The quick brown fox" (19 characters)

| Layout Mode | Time (ns/op) | Time (µs) | Allocs | Memory (B/op) |
|-------------|--------------|-----------|--------|---------------|
| Full Width  | 73,042       | 73.0 µs   | 50     | 248,122       |
| Kerning     | 80,825       | 80.8 µs   | 50     | 248,116       |
| Smushing    | 94,669       | 94.7 µs   | 50     | 248,107       |
| RenderTo    | 99,997       | 100.0 µs  | 48     | 246,911       |

**Analysis:**
- Render time is ~1.5-2x the PRD target of <50µs
- All modes perform within same order of magnitude
- Smushing is slowest (more complex logic)
- RenderTo has slightly fewer allocations

### Allocation Breakdown by Text Length

| Text Length | Example | Full Width | Kerning | Smushing |
|-------------|---------|------------|---------|----------|
| 5 chars | "Hello" | 22 allocs | 22 allocs | 22 allocs |
| 11 chars | "Hello World" | 34 allocs | 34 allocs | 34 allocs |
| 19 chars | "The quick brown fox" | 50 allocs | 50 allocs | 50 allocs |
| 43 chars | "The quick brown fox jumps over..." | 98-99 allocs | 98-99 allocs | 98-99 allocs |

**Pattern:** Allocations scale roughly linearly with text length (~2.3 allocs per character on average).

**Target Gap:** Currently 5-12x above the <4 allocs/op target. Main optimization opportunity.

### Throughput Measurements

| Text Type | Layout | Ops/sec | Merges/sec | Time (µs) |
|-----------|--------|---------|------------|-----------|
| Short (5 chars) | Full Width | 20,694 | 103,469 | 48.3 |
| Short (5 chars) | Kerning | 20,793 | 103,966 | 48.1 |
| Short (5 chars) | Smushing | 19,835 | 99,174 | 50.4 |
| Medium (19 chars) | Full Width | 17,490 | 332,316 | 57.2 |
| Medium (19 chars) | Kerning | 16,228 | 308,335 | 61.6 |
| Medium (19 chars) | Smushing | 16,535 | 314,165 | 60.5 |
| Long (43 chars) | Full Width | 14,953 | 642,993 | 66.9 |
| Long (43 chars) | Kerning | 13,370 | 574,908 | 74.8 |
| Long (43 chars) | Smushing | 12,704 | 546,261 | 78.7 |

**Analysis:**
- Throughput increases with longer text (more merges per operation)
- Medium text smushing: **314K merges/sec** (~30% of 1M stretch goal)
- Long text full width: **643K merges/sec** (closest to stretch goal)

### Font Cache Performance

| Scenario | Time (ns/op) | Speedup | Allocs | Memory |
|----------|--------------|---------|--------|---------|
| Load without cache | 20,825 | 1x | 69 | 25,127 B |
| Load with cache (hit) | 53.57 | **389x faster** | 0 | 0 B |
| Parse bytes without cache | 10,761 | 1x | 67 | 24,909 B |
| Parse bytes with cache (hit) | 367.4 | **29x faster** | 3 | 208 B |

**Analysis:** Font caching provides massive performance gains. Cache hits are essentially free.

## Running Benchmarks

### Core Performance Suite

Run the comprehensive performance benchmarks targeting PRD metrics:

```bash
# All performance benchmarks
go test -bench=^Benchmark -benchmem -run=^$ .

# Specific PRD targets
go test -bench=BenchmarkPRDTargets -benchmem .

# Allocation analysis
go test -bench=BenchmarkAllocationTarget -benchmem .

# Throughput measurements
go test -bench=BenchmarkThroughputTarget -benchmem .

# Multiple fonts comparison
go test -bench=BenchmarkMultipleFonts -benchmem .

# Concurrent rendering
go test -bench=BenchmarkConcurrentRendering -benchmem .
```

### Existing Benchmark Suites

```bash
# Renderer benchmarks (internal/renderer)
go test -bench=. -benchmem ./internal/renderer/

# Parser benchmarks (internal/parser)
go test -bench=. -benchmem ./internal/parser/

# Font cache benchmarks
go test -bench=BenchmarkFontCache -benchmem .

# Full benchmark suite
just bench
```

## Performance Regression Detection

### Warning Thresholds (Informational Only)

These thresholds trigger warnings but **do not block CI**:

| Metric | Baseline | Warning (+%) | Notes |
|--------|----------|--------------|-------|
| Quick Brown Fox render time | 73µs | +20% (>88µs) | For full width mode |
| Allocations (short text) | 22 | +50% (>33) | For 5-char strings |
| Cache hit time | 54ns | +100% (>108ns) | Should stay near zero |
| Throughput (medium smushing) | 314K/sec | -20% (<251K/sec) | Character merges |

### How to Measure

1. **Before making changes:**
   ```bash
   go test -bench=BenchmarkPRDTargets -benchmem . > baseline.txt
   ```

2. **After making changes:**
   ```bash
   go test -bench=BenchmarkPRDTargets -benchmem . > current.txt
   ```

3. **Compare:**
   ```bash
   # Install benchstat if needed: go install golang.org/x/perf/cmd/benchstat@latest
   benchstat baseline.txt current.txt
   ```

## Optimization Opportunities

Based on baseline measurements, priority optimization areas:

### 🔴 High Priority: Reduce Allocations

**Current:** 22-50 allocs/op
**Target:** <4 allocs/op
**Gap:** 5-12x

**Ideas:**
- Pool more render buffers and intermediate allocations
- Pre-allocate result strings based on estimated size
- Reduce string concatenations during rendering
- Review escape analysis to keep more allocations on stack

### 🟡 Medium Priority: Improve Render Time

**Current:** 73-95µs for medium text
**Target:** <50µs
**Gap:** 1.5-2x

**Ideas:**
- Optimize hot path in glyph merging
- Reduce bounds checks in tight loops
- Consider SIMD for character comparison (future)
- Profile to identify specific bottlenecks

### 🟢 Low Priority: Increase Throughput

**Current:** 314K merges/sec (medium smushing)
**Target:** 1M merges/sec (stretch goal)
**Gap:** 3x

**Ideas:**
- Same optimizations as render time
- Parallel rendering for very long text (future)
- This is a stretch goal, not required for MVP

## CI Integration

To add benchmark tracking to CI, update `.github/workflows/test.yml`:

```yaml
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Run Benchmarks
        run: |
          go test -bench=BenchmarkPRDTargets -benchmem . | tee benchmark-results.txt

      - name: Upload Results
        uses: actions/upload-artifact@v4
        with:
          name: benchmark-results
          path: benchmark-results.txt
```

**Note:** This job is **informational only** and should not block PRs. Use it to track trends over time.

## Benchmark Maintenance

- **Update baselines** when intentional performance changes are made
- **Review trends** monthly to detect gradual regressions
- **Re-measure** on different hardware/OS to understand variance
- **Document** any optimizations and their measured impact

## References

- **PRD Performance Section:** `docs/prd.md` (Performance & Optimization)
- **Issue #34:** Performance Guardrails (Basic)
- **Benchmark Code:**
  - `performance_bench_test.go` - PRD-targeted benchmarks
  - `internal/renderer/render_bench_test.go` - Renderer internals
  - `internal/parser/parser_bench_test.go` - Parser performance
  - `font_cache_bench_test.go` - Cache performance
