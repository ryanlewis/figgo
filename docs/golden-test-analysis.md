# Golden Test Failure Analysis Report

## Date: 2025-08-14

## Executive Summary

Two golden tests are currently failing for the standard font with default layout:
1. `standard/default/c4e3c52d` - Special characters test
2. `standard/default/The_quick_brown_fox_jumps_over_the_lazy_dog` - Long text test

Both failures are related to line wrapping behavior differences between our implementation and the reference figlet.c implementation.

## Test Failures Detail

### Test 1: Special Characters (`c4e3c52d`)

**Input String:** `!@#$%^&*()_+-=[]{}:;'",.<>?/\|`

**Failure Location:** Line 7 (second output line, continuation)

**Expected Output (figlet):**
```
   ____      _ _ _    ____ ___   ____    _ 
```

**Actual Output (figgo):**
```
   ____      ___   _ _    ____ ___   ____    _ 
```

**Specific Issue:** The sequence `:;'` renders as `___` instead of `_ _ _`

### Test 2: Long Text (`The_quick_brown_fox_jumps_over_the_lazy_dog`)

**Input String:** `The quick brown fox jumps over the lazy dog`

**Failure Location:** Line 1

**Expected Output (figlet):**
```
 _____ _                        _      _      _                             
```
(76 characters)

**Actual Output (figgo):**
```
 _____ _                        _      _      _                              
```
(77 characters - extra trailing space)

## Technical Analysis

### Root Cause: Line Wrapping Behavior

The core issue is a difference in how figlet and figgo handle line wrapping when approaching the 80-character line limit:

1. **figlet behavior:**
   - Stops adding characters to a line when the next character would exceed 80 columns
   - For the string `!@#$%^&*()_+-=[]{}:;'...`, figlet stops after `}` and starts a new line with `:;'`
   - This results in `:;'` being rendered at the start of a new line with proper spacing

2. **figgo behavior:**
   - Attempts to fit `:;` on the same line as `[]{}`, resulting in different smushing
   - The characters `:;` are smushed together with `}` differently than when they start a new line

### Code Investigation Points

#### 1. Line Width Calculation (`internal/renderer/renderer.go:28-34`)
```go
if opts.Width != nil && *opts.Width > 0 {
    state.outlineLenLimit = *opts.Width - 1 // -1 to match figlet behavior
} else {
    state.outlineLenLimit = 79 // Default: 80 - 1
}
```
The width limit is correctly set to 79 (80-1), matching figlet's default.

#### 2. Character Addition Check (`internal/renderer/renderer.go:333-336`)
```go
newLength := state.outlineLen + state.currentCharWidth - smushAmount
if newLength > state.outlineLenLimit {
    return false
}
```
The check for whether a character fits appears correct.

#### 3. SmushAmount Calculation (`internal/renderer/smushing.go`)
The smushAmount calculation was thoroughly investigated and appears to be working correctly for individual character pairs. The issue manifests only in the context of line wrapping.

## Exhausted Hypotheses

### Hypothesis 1: SmushAmount Calculation Bug
**Investigation:** Extensive debugging of the `smushAmount()` function, comparing with figlet.c reference implementation.

**Result:** The smushAmount calculation is correct. When `:;` is rendered in isolation or at the start of a line, it produces the correct output `_ _`.

**Evidence:**
- `:;` alone produces correct output
- `:;'` alone produces correct output  
- The issue only occurs when these characters appear after `[]{}`

### Hypothesis 2: LineBoundary Calculation Error
**Investigation:** Traced through the lineBoundary calculation in the LTR processing path.

**Result:** The calculation correctly identifies the rightmost non-space character in the output line.

### Hypothesis 3: Character Width Miscalculation
**Investigation:** Verified that `currentCharWidth` is correctly calculated using rune count of the first glyph row.

**Result:** Character widths are correctly calculated, matching figlet's approach.

### Hypothesis 4: Smushing Rules Implementation
**Investigation:** Reviewed all smushing rules, particularly hierarchy smushing which affects brackets.

**Result:** Individual smushing rules work correctly. The issue is not with the rules themselves.

## Key Findings

1. **Context-Dependent Rendering:** The same character sequence (`:;'`) renders differently depending on whether it starts a new line or continues from previous characters.

2. **Line Break Point:** figlet breaks the line after `}` (at position 78), while figgo attempts to include `:;` on the same line.

3. **Word Boundary Handling:** The trailing space issue in long text suggests differences in how word boundaries are handled during line wrapping.

## Test Verification

Golden test files were verified to be correct by comparing with actual figlet output:
```bash
echo '!@#$%^&*()_+-=[]{}:;'"'"'",.<>?/\|' | figlet -f standard
```

The golden files accurately represent figlet's expected output.

## Recommendations

### Short-term Solutions

1. **Accept Current Behavior:** Document the known differences in line wrapping behavior as acceptable variations from the reference implementation.

2. **Adjust Tests:** Create figgo-specific golden tests that reflect our line wrapping behavior while maintaining correct smushing logic.

### Long-term Solutions

1. **Implement Figlet-Compatible Line Wrapping:** 
   - Study figlet.c's `addchar()` and `printline()` functions
   - Implement word boundary detection matching figlet's approach
   - Add lookahead logic to determine when to break lines

2. **Trailing Space Handling:**
   - Implement proper trimming of trailing spaces when lines are wrapped
   - Review the `flushLine()` function for proper space handling

## Impact Assessment

- **Current Pass Rate:** 10/12 tests passing for standard/default configuration (83.3%)
- **Affected Scenarios:** Only affects rendering when:
  - Text approaches the 80-character line limit
  - Special character sequences span line boundaries
  - Long text with multiple words requires wrapping

## Conclusion

The golden test failures are not due to fundamental flaws in the smushing algorithm but rather differences in line wrapping behavior between figgo and figlet. The core rendering logic is sound, and the issues only manifest in edge cases involving line boundaries.

The decision to fix these issues should weigh the complexity of implementing figlet-exact line wrapping against the practical impact on users, as the current behavior produces readable and correct ASCII art in most cases.