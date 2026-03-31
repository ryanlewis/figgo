# Debug Module — Production Hygiene

## Approach: Nil-Check Fast Path (Status Quo, Verified)

The debug module uses **nil-receiver checks** for zero-cost when disabled.
Build tags were evaluated but rejected — the nil-check approach is simpler,
keeps a single build configuration, and has verified negligible overhead.

## How It Works

```
SetEnabled(false)        ← default
    ↓
NewSession(sink) → nil   ← no allocation
    ↓
Emit("phase", "event", data)
    if s == nil { return } ← ~1.6 ns, 0 allocs
```

All 15+ emission points in the renderer are guarded by `if state.debug != nil`.
When debug is disabled (the default), the session is nil and every Emit call
returns immediately after a single pointer comparison.

## Benchmark Evidence

```
BenchmarkEmitDisabled    731M iterations    1.6 ns/op    0 B/op    0 allocs/op
BenchmarkEmitEnabled     1.2M iterations    1024 ns/op   984 B/op  2 allocs/op
```

With 15 emission points per render and ~1.6 ns each, the total debug overhead
in a disabled render is ~24 ns — under 0.05% of a typical 50-80µs render.

## Why Not Build Tags?

| Criterion | Nil-Checks | Build Tags |
|-----------|-----------|------------|
| Overhead when disabled | ~24 ns/render | 0 ns/render |
| Build complexity | Single binary | Two configurations |
| CI coverage | One path | Must test both |
| Risk of drift | None | Tag-excluded code may rot |
| Debuggability | Always available | Must rebuild |

The 24 ns overhead is negligible. Build tags add maintenance cost for no
practical gain and risk untested code paths.

## Guarantees

- `debug.Enabled()` uses `sync/atomic.LoadUint32` — lock-free
- `NewSession()` returns nil when disabled — no heap allocation
- `Session.Emit()` is nil-safe — single branch, perfectly predicted by CPU
- Event structs are only allocated when debug is active
- No debug I/O occurs when disabled
