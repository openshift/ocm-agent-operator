---
name: full-review
description: Run all review agents in parallel for comprehensive code quality analysis
---

Run a comprehensive code review by launching all available review agents in parallel to analyze changes between the current branch and master.

## Process

1. **Verify branch state**
   - Check we're on a feature branch (not master)
   - Get the diff between master and current branch: `git diff master...HEAD`
   - Pass the complete diff output to each agent so they know exactly which files and lines changed

2. **Launch all review agents in parallel**
   
   Spawn each of the following agents in the background without worktree isolation. These agents are read-only reviewers and do not need isolated worktrees. IMPORTANT: Include the diff output in each agent's prompt and instruct them to ONLY review files and lines that appear in the diff — they must not explore or report on files outside the changeset:

   - @security-reviewer - OWASP Top 10, RBAC, secrets management, input validation
   - @lint-reviewer - Code quality, style, golangci-lint findings
   - @test-reviewer - Test coverage gaps, test quality, Ginkgo/Gomega patterns
   - @error-handling-reviewer - Error wrapping, logging, failure modes

   Additional focus areas if specified: $ARGUMENTS

3. **Compile findings**
   
   Once all agents complete, compile their findings into a single summary organized by severity:
   
   1. **Critical** - must fix before merge
   2. **High** - should fix before merge  
   3. **Medium** - consider fixing
   4. **Low** - optional improvements

   Include file:line references for every finding.

## Rules

- Do not run this on the master branch
- All agents should run in parallel for efficiency
- Each agent should only analyze files that appear in the diff
- Provide a consolidated report at the end, not individual agent reports
