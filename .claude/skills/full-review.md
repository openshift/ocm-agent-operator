---
name: full-review
description: Run all review agents in parallel to analyze code changes
---

# Full Code Review

This skill launches all available review agents in parallel to comprehensively analyze code changes on the current branch versus `master`.

## Implementation

When this skill is invoked, execute the following steps:

1. **Launch review agents in parallel**:
   - `@security-reviewer` - Security vulnerabilities and RBAC issues
   - `@lint-reviewer` - Code quality using golangci-lint
   - `@test-reviewer` - Test coverage gaps and quality issues
   - `@error-handling-reviewer` - Error handling patterns and logging

2. **Wait for all agents to complete** (they run in background).

3. **Consolidate results** from all reviews into a single severity-organized report.

4. **Present summary** with actionable recommendations and total issue count by severity.

## Usage

```
/full-review
```

Optional: Provide focus areas or specific concerns to guide the review:
```
/full-review focus on error handling and concurrency
```

## What Gets Reviewed

The review agents analyze changes using `git diff master...HEAD`:

1. **Security Review** - OWASP Top 10, Kubernetes RBAC, secrets management
2. **Linting Review** - Go code quality using golangci-lint from boilerplate
3. **Test Review** - Coverage gaps, test quality, Ginkgo/Gomega patterns
4. **Error Handling Review** - Error wrapping, logging, reconciliation errors

## Output

Results are compiled into a single report organized by severity:
- **Critical**: Must fix before merging
- **High**: Should fix before merging
- **Medium**: Consider fixing or document why not
- **Low**: Nice to have improvements

## When to Use

### Before Creating a PR
```bash
git checkout your-feature-branch
/full-review
```

### Reviewing Someone Else's PR
```bash
git fetch origin
git checkout origin/their-branch
/full-review
```

### After Addressing Review Comments
```bash
# Make fixes based on review
git add .
git commit -m "Address review feedback"
/full-review  # Re-run to verify fixes
```

## Individual Agent Usage

You can also run individual agents for focused review:

```
@security-reviewer
@lint-reviewer
@test-reviewer
@error-handling-reviewer
```

Or run linting directly:
```bash
make lint
```

## Pre-requisites

- Must be on a branch (not detached HEAD)
- Should have commits different from `master`
- Clean working directory recommended (but not required)

## Review Workflow

1. **Checkout your branch**: `git checkout feature-branch`
2. **Run full review**: `/full-review`
3. **Wait for results**: Agents run in parallel (1-3 minutes typically)
4. **Review findings**: Organized by severity
5. **Address issues**: Fix critical and high severity items
6. **Re-review if needed**: Run `/full-review` again to verify fixes
7. **Create PR**: Once review is clean

## Tips

- Run review **before** creating PR to catch issues early
- Address **Critical** and **High** severity findings before requesting human review
- **Medium** findings are judgment calls - use your discretion
- **Low** findings are nice-to-haves, can be addressed in follow-up
- Re-run after significant changes to ensure no regressions

## Example Output Structure

```
## Full Review Report

### Security Review
- ✅ No critical security issues found
- ⚠️  1 medium finding: [details]

### Linting Review
- 🔴 2 high severity: Unused variables, ineffective assignments
- ⚠️  3 medium findings: [details]

### Test Review
- 🔴 3 high severity: Missing coverage for new controller logic
- ⚠️  5 medium findings: [details]

### Error Handling Review
- ✅ No critical issues
- ℹ️  2 low severity: Improve error messages

---

## Summary
- Total findings: 0 Critical, 5 High, 9 Medium, 2 Low
- Must address before merge: 5 issues
- Recommended time to fix: ~2-3 hours
```
