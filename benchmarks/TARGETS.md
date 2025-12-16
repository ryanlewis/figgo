# Performance Targets

These targets are defined in the PRD (§10) and measured on Go 1.22, 8-core dev machine.

## Primary Targets

| Metric                 | Target  | Benchmark                          | Status  |
| ---------------------- | ------- | ---------------------------------- | ------- |
| p50 render latency     | < 50µs  | `BenchmarkPRDTarget_QuickBrownFox` | TBD     |
| Allocations per op     | < 4     | `BenchmarkPRDTarget_Allocations`   | TBD     |
| Glyph merge throughput | ~1M/sec | `BenchmarkPRDTarget_MergeThruput`  | Stretch |

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
