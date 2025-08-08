# Golden Test Fixtures

Generated test fixtures for Figgo compliance testing.

## Generation Info

- **Generated:** 2025-08-08T15:44:53Z
- **Figlet Version:** /usr/share/figlet
- **Script:** tools/generate-goldens.sh

## Files

| Font | Layout | Input | File | SHA256 |
|------|--------|-------|------|--------|
| standard | full-width | `Hello, World!...` | [Hello_World.md](standard/full-width/Hello_World.md) | 10bde795... |
| standard | full-width | `FIGgo 1.0...` | [FIGgo_1_0.md](standard/full-width/FIGgo_1_0.md) | ba048157... |
| standard | full-width | `|/\[]{}()<>...` | [.md](standard/full-width/.md) | 6e5532f6... |
| standard | full-width | `The quick brown fox ...` | [The_quick_brown_fox_jumps_over_the_lazy_dog.md](standard/full-width/The_quick_brown_fox_jumps_over_the_lazy_dog.md) | 48dd761a... |
| standard | full-width | `\"\"...` | [.md](standard/full-width/.md) | db98b29d... |
| standard | full-width | `\" \"...` | [.md](standard/full-width/.md) | a0874936... |
| standard | full-width | `a...` | [a.md](standard/full-width/a.md) | f4847b67... |
| standard | full-width | `   ...` | [.md](standard/full-width/.md) | d2d5fcdb... |
| standard | full-width | `$$$$...` | [.md](standard/full-width/.md) | ed5338af... |
| standard | full-width | `!@#$%^&*()_+-=[]{}:;...` | [.md](standard/full-width/.md) | 84a47972... |
| standard | kerning | `Hello, World!...` | [Hello_World.md](standard/kerning/Hello_World.md) | aeb6ccf2... |
| standard | kerning | `FIGgo 1.0...` | [FIGgo_1_0.md](standard/kerning/FIGgo_1_0.md) | e472aeb9... |
| standard | kerning | `|/\[]{}()<>...` | [.md](standard/kerning/.md) | ca0bc897... |
| standard | kerning | `The quick brown fox ...` | [The_quick_brown_fox_jumps_over_the_lazy_dog.md](standard/kerning/The_quick_brown_fox_jumps_over_the_lazy_dog.md) | e610ddb3... |
| standard | kerning | `\"\"...` | [.md](standard/kerning/.md) | 1fe62e88... |
| standard | kerning | `\" \"...` | [.md](standard/kerning/.md) | a0874936... |
| standard | kerning | `a...` | [a.md](standard/kerning/a.md) | f4847b67... |
| standard | kerning | `   ...` | [.md](standard/kerning/.md) | d2d5fcdb... |
| standard | kerning | `$$$$...` | [.md](standard/kerning/.md) | b4d726e1... |
| standard | kerning | `!@#$%^&*()_+-=[]{}:;...` | [.md](standard/kerning/.md) | 6f2d24a3... |

## Verification

To verify golden files haven't been modified:

```bash
cd tests/goldens
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
