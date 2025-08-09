---
description: Implement a GitHub issue for the Figgo project following TDD practices with comprehensive testing and documentation
---

# Implementing GitHub Issue #$ARGUMENTS

I'll implement GitHub issue **#$ARGUMENTS** for the Figgo project following Test-Driven Development (TDD) practices.

## ⚠️ CRITICAL RULES - MUST FOLLOW

1. **Commit Messages**: Max 50 chars subject, 2-4 line body, NO watermarks/signatures/emoji
2. **Use TodoWrite**: Track ALL major steps to give visibility
3. **Be Concise**: Avoid verbose explanations during implementation
4. **No Documentation Creation**: Unless explicitly requested in the issue
5. **Ask for Integration Strategy**: ALWAYS get confirmation before merging or creating PR
6. **Follow CLAUDE.md**: All rules in CLAUDE.md apply here too

## Step 1: Issue Analysis

I'll fetch and analyze the GitHub issue details using `gh issue view $ARGUMENTS`.

## Step 2: Verify Prerequisites

I'll verify:
1. Working directory is clean
2. We're on the main branch and up to date
3. No existing branch for this issue

```bash
git status
git branch --show-current
git pull origin main
```

## Step 3: Create Feature Branch

Create branch pattern: `feat/issue-$ARGUMENTS-brief-description`

## Step 4: Implementation Planning

I'll review the issue requirements and identify:
- Components to create/modify
- Test cases needed
- Any unclear requirements

**I'll proceed with implementation unless you have specific requirements to add.**

## Step 5: Test-First Development

### 5.1 Write Test Cases
- Unit tests in `*_test.go`
- Golden tests if rendering involved
- Race condition tests for concurrency

### 5.2 Run Tests (Expect Failures)
```bash
go test -v ./...
```

### 5.3 Implement Minimal Code
Write just enough code to make tests pass.

### 5.4 Refactor
Optimize for performance and clarity once tests pass.

## Step 6: Code Documentation

Write concise godoc comments for exported types and functions.
Follow Go conventions without excessive examples.

## Step 7: Quality Assurance

```bash
just fmt        # Format code
just lint       # Run linting
go test -v -race ./...  # Test with race detection
```

For performance-critical code:
```bash
go test -bench=. -benchmem ./...
```

## Step 8: Update Documentation

Only update docs mentioned in the issue:
- `docs/spec-compliance.md` - if FIGfont spec features
- `README.md` - if user-facing features
- NO new documentation files unless requested

## Step 9: Commit Changes

### 9.1 Stage Changes
```bash
git add -A
```

### 9.2 Commit with CONCISE Message

**STOP! Check commit message rules:**
- Subject ≤ 50 characters
- Format: `type: brief description - closes #N`
- NO watermarks, NO signatures, NO emoji
- Body: 2-4 lines MAX if needed

Example:
```
feat: implement full-width rendering - closes #9

Add renderFullWidth function for no-overlap composition.
Support RTL direction and hardblank replacement.
```

**I'll create a concise commit and show you before executing.**

## Step 10: Final Validation

### 10.1 Verify Requirements
Check all acceptance criteria from the issue are met.

### 10.2 Run Final Tests
```bash
go test -v ./...
just lint
```

## Step 11: Integration Decision ⚠️ REQUIRES YOUR APPROVAL

I'll analyze the changes and show you:
- Lines changed
- Files affected
- Complexity assessment

**Then I'll recommend either:**
- **Direct merge**: <100 lines, low risk, well-tested
- **Pull Request**: >500 lines, new features, needs review

**I WILL WAIT FOR YOUR DECISION before proceeding.**

### Option A: Direct Merge to Main
```bash
git checkout main
git pull origin main
git merge --no-ff feat/issue-$ARGUMENTS-description
git push origin main
git branch -d feat/issue-$ARGUMENTS-description
```

### Option B: Create Pull Request
```bash
git push -u origin feat/issue-$ARGUMENTS-description

gh pr create \
  --title "feat: implement #$ARGUMENTS - brief description" \
  --body "Closes #$ARGUMENTS

## Summary
[1-2 lines describing what was implemented]

## Changes
- [Key change 1]
- [Key change 2]

## Testing
- All tests passing
- Coverage: X%"
```

**NO WATERMARKS in PR body.**

## Step 12: Post-Integration Cleanup

After your approved integration method:
1. Delete local feature branch
2. Delete remote branch if PR
3. Verify issue auto-closes

## Before Starting - CHECKLIST:
- [ ] Will use TodoWrite to track progress
- [ ] Commit messages: ≤50 chars, no watermarks
- [ ] Will be concise in outputs
- [ ] Will ask for integration strategy approval
- [ ] Will follow all CLAUDE.md rules

---

**Ready to implement GitHub issue #$ARGUMENTS for Figgo!**

I'll start by fetching and analyzing the issue.