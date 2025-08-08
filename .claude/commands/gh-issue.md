---
description: Implement a GitHub issue for the Figgo project following TDD practices with comprehensive testing and documentation
---

# Implementing GitHub Issue #$ARGUMENTS

I'll implement GitHub issue **#$ARGUMENTS** for the Figgo project following Test-Driven Development (TDD) practices. Let me start by analyzing the issue and creating a comprehensive implementation plan.

## Step 1: Issue Analysis

First, let me fetch and analyze the GitHub issue details.

I'll use `gh issue view $ARGUMENTS` to get:
- Issue title and description
- Labels and assignees
- Any linked PRs or related issues
- Acceptance criteria from the issue body
- Discussion comments for additional context

## Step 2: Verify Prerequisites

Before starting, I'll verify:
1. Working directory is clean (no uncommitted changes)
2. We're on the main branch and up to date
3. No existing branch for this issue
4. Go development environment is properly set up

I'll run:
- `git status` - to check for uncommitted changes
- `git branch --show-current` - to verify current branch
- `git pull origin main` - to ensure we're up to date

## Step 3: Create Feature Branch

I'll create a descriptive feature branch for this issue:
- Pattern: `feat/issue-$ARGUMENTS-brief-description`
- Example: `git checkout -b feat/issue-$ARGUMENTS-font-parser`
- The branch name will be based on the issue title

## Step 4: Implementation Planning

Based on the issue analysis and PRD (docs/prd.md), I'll:
1. Review the PRD sections relevant to this issue
2. Identify all components that need to be created/modified
3. List the test cases needed per spec-compliance.md
4. Check if golden tests need to be generated
5. Note any unclear requirements that need clarification

**Questions for you before proceeding:**
- Are there specific FIGfont features or edge cases to prioritize?
- Should I follow any existing patterns from similar Go FIGlet libraries?
- Any performance requirements beyond the PRD targets (<50µs, <4 allocs)?
- Should I generate golden tests for this feature?

## Step 5: Test-First Development

### 5.1 Write Test Cases
Following the Figgo testing strategy:
- Unit tests for parser, renderer, layout engine as applicable
- Golden tests comparing against C figlet output
- Property-based tests for smushing rules if relevant
- Fuzzing for parser robustness
- Race condition tests for concurrent rendering

Test files will follow Go conventions:
- `*_test.go` in the same package
- `testdata/` for fixtures
- Golden files in `testdata/goldens/`

### 5.2 Generate Golden Tests
If this issue involves rendering:
```bash
./tools/generate-goldens.sh
```

### 5.3 Run Tests (Expect Failures)
```bash
go test -v ./...
```

### 5.4 Implement Minimal Code
Write just enough code to make tests pass, following the API design from the PRD:
- Immutable Font structs
- Stateless rendering functions
- Clean error handling (no panics)
- Options pattern for configuration

### 5.5 Refactor
Once tests pass, optimize for:
- Performance targets from PRD
- Memory efficiency (pooling if needed)
- Code clarity and Go idioms

## Step 6: Code Documentation

All exported types and functions will include comprehensive godoc:

```go
// Font represents an immutable FIGfont that can be safely shared across goroutines.
// 
// Font data is loaded once and never modified, making it safe for concurrent use
// without locking.
//
// Example:
//   font, err := figgo.ParseFont(reader)
//   if err != nil {
//       log.Fatal(err)
//   }
//   output, err := figgo.Render("Hello", font)
type Font struct {
    // ... fields per PRD section 5
}

// Render converts text to ASCII art using the specified font and options.
//
// The font parameter must not be nil. Text can contain any ASCII characters
// (32-126); unknown runes are replaced with '?' by default.
//
// Options can modify layout behavior:
//   - WithLayout: override font's default fitting mode
//   - WithPrintDirection: override font's print direction (0=LTR, 1=RTL)
//
// Example:
//   output, err := figgo.Render("Hello", font, 
//       figgo.WithLayout(figgo.FitSmushing | figgo.RuleEqualChar))
func Render(text string, f *Font, opts ...Option) (string, error) {
    // implementation
}
```

## Step 7: Quality Assurance

### 7.1 Format Code
```bash
go fmt ./...
gofmt -s -w .
```

### 7.2 Run Linting
```bash
go vet ./...
staticcheck ./...  # if available
```

### 7.3 Run Tests with Coverage
```bash
go test -v -race -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # view coverage report
```

### 7.4 Benchmark Performance
If implementing rendering or performance-critical code:
```bash
go test -bench=. -benchmem ./...
```

Verify against PRD targets:
- p50 < 50µs for "The quick brown fox"
- < 4 allocs/op with pooling

### 7.5 Check Module Health
```bash
go mod tidy
go mod verify
```

## Step 8: Update Documentation

### 8.1 Update Relevant Docs
Depending on the issue:
- `docs/spec-compliance.md` - if implementing FIGfont spec features
- `docs/fonts.md` - if adding font support
- `README.md` - if adding user-facing features
- `docs/prd.md` - mark completed deliverables

### 8.2 Update Issue
Add implementation notes to the GitHub issue:
- Actual approach taken
- Any deviations from original plan
- Performance measurements
- Test coverage achieved

## Step 9: Commit Changes

### 9.1 Stage Changes
```bash
git add -A
```

### 9.2 Create Descriptive Commit
Following conventional commit format:

**Format Rules:**
- Type: feat, fix, test, docs, refactor, perf, chore
- Scope: optional, specific component
- Message: imperative mood, lowercase, no period
- Length: ~50 chars for subject line

**Examples for Figgo:**
```
feat: add FIGfont v2 parser with layout normalization
fix: handle hardblanks correctly in smushing rules
test: add golden tests for standard font rendering
perf: optimize glyph lookup with preprocessing
docs: update spec-compliance with rule examples
```

**I'll propose a commit message and wait for your confirmation.**

## Step 10: Final Validation

### 10.1 Verify Issue Requirements
Check each requirement from the GitHub issue:
- [ ] Requirement 1: [verification]
- [ ] Requirement 2: [verification]
- [ ] All acceptance criteria met

### 10.2 Verify Spec Compliance
If implementing FIGfont features:
- [ ] Follows spec-compliance.md guidelines
- [ ] Golden tests match C figlet output
- [ ] Layout normalization correct
- [ ] Smushing rules in right precedence

### 10.3 Run Final Checks
```bash
go test -v ./...
go fmt ./...
go vet ./...
```

## Step 11: Integration Strategy

Based on the scope of changes, I'll recommend the best integration approach:

### Analysis Criteria
I'll evaluate:
- **Lines changed**: < 100 lines → direct merge, > 500 lines → PR
- **Files affected**: 1-2 files → direct merge, 5+ files → PR
- **Complexity**: Simple fixes → direct merge, new features → PR
- **Risk**: Low risk → direct merge, breaking changes → PR
- **Testing needs**: Well-tested → direct merge, needs review → PR

### Option A: Direct Merge to Main (Recommended for small changes)
**Best for**: Bug fixes, small features, documentation updates, single-file changes

```bash
# Ensure all tests pass
go test -v ./...

# Commit changes
git add -A
git commit -m "approved message"

# Merge to main
git checkout main
git pull origin main
git merge --no-ff feat/issue-$ARGUMENTS-description
git push origin main

# Clean up
git branch -d feat/issue-$ARGUMENTS-description
```

**Pros**: Fast integration, no PR overhead, immediate availability
**Cons**: No review process, no CI checks before merge

### Option B: Create Pull Request (Recommended for larger changes)
**Best for**: New features, multi-file changes, API changes, complex logic

```bash
# Push branch
git push -u origin feat/issue-$ARGUMENTS-description

# Create PR
gh pr create \
  --title "feat: implement #$ARGUMENTS - brief description" \
  --body "Closes #$ARGUMENTS

## Summary
Brief description of implementation

## Changes
- Key change 1
- Key change 2

## Testing
- Test coverage: X%
- Golden tests: passing
- Benchmarks: meeting PRD targets

## Checklist
- [ ] Tests pass
- [ ] Documentation updated
- [ ] Spec compliant
- [ ] Performance targets met"
```

**Pros**: Review opportunity, CI validation, discussion thread
**Cons**: Slower integration, requires PR management

### My Recommendation
After analyzing the changes, I'll recommend:
- **Option A (Direct Merge)** if: changes are small, well-tested, and low-risk
- **Option B (Pull Request)** if: changes are substantial, need review, or affect core functionality

**I'll present my recommendation with reasoning and let you make the final decision.**

## Step 12: Post-Integration Cleanup

After integration (either method):
1. Delete local feature branch: `git branch -d feat/issue-$ARGUMENTS-description`
2. Delete remote branch (if PR): `git push origin --delete feat/issue-$ARGUMENTS-description`
3. Verify issue auto-closes (via "Closes #N" in commit/PR)
4. Note any follow-up issues identified

## Progress Tracking

Throughout this process, I'll use the TodoWrite tool to track implementation progress:
- Fetch and analyze issue
- Create branch
- Write tests
- Implement features
- Update documentation
- Create PR

---

**Ready to implement GitHub issue #$ARGUMENTS for Figgo!**

Please confirm you want me to proceed with fetching and analyzing the issue.
