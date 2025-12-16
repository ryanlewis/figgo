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
