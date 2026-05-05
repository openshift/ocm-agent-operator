# Claude Code Usage Guide for ocm-agent-operator

This guide explains how to use Claude Code effectively with this operator repository.

## Table of Contents

- [Getting Started](#getting-started)
- [Available Skills](#available-skills)
- [Review Agents](#review-agents)
- [Hooks](#hooks)
- [Common Workflows](#common-workflows)
- [Tips & Best Practices](#tips--best-practices)

## Getting Started

### What is Claude Code?

Claude Code is an AI-powered development assistant that helps with:
- Code review and quality analysis
- Writing and refactoring code
- Running tests and debugging
- Understanding codebases
- Following repository conventions

### Installation

Claude Code is available as:
- **CLI**: `claude` command in terminal
- **Desktop App**: Mac/Windows application
- **Web App**: https://claude.ai/code
- **IDE Extensions**: VS Code, JetBrains

### First Time Setup

1. Navigate to the repository:
```bash
cd ~/rh-projects/ROSA-730/ocm-agent-operator
```

2. Start Claude Code:
```bash
claude
# or use the desktop/web app
```

3. Claude automatically loads:
   - `CLAUDE.md` - Repository-specific guidance
   - `.claude/settings.json` - Permissions and hooks
   - `.claude/skills/` - Custom commands
   - `.claude/agents/` - Specialized review agents

## Available Skills

Skills are custom slash commands for common tasks. Type `/` to see all available skills.

### `/full-review`

Run comprehensive code review with security, linting, test coverage, and error handling analysis.

**Usage:**
```
/full-review
```

**What it runs:**
- Security review (OWASP Top 10, RBAC, secrets)
- Linting via `make lint` (golangci-lint)
- Test coverage analysis (gaps, quality)
- Error handling review (wrapping, logging)

**When to use:**
- Before creating a PR
- After making significant changes
- When reviewing someone else's code

**Output:** Severity-organized report from all review agents

---

### `/run-tests`

Execute the full test suite including linting, unit tests, and race detection.

**Usage:**
```
/run-tests
/run-tests with race detection
/run-tests quick
```

**What it runs:**
- `make lint` - golangci-lint checks
- `make test` - Unit tests with coverage
- `go test -race` - Race condition detection
- `make go-build` - Build verification

---

### `/update-crds`

Regenerate CRDs and manifests after modifying API types.

**Usage:**
```
/update-crds
```

**When to use:**
- After changing `api/v1alpha1/*.go` files
- After adding/modifying kubebuilder markers
- After changing RBAC requirements

**What it does:**
- Runs `make manifests` - Updates CRDs and RBAC
- Runs `make generate` - Updates generated code

---

## Review Agents

Agents are specialized AI assistants for focused code analysis. They run in isolated worktrees in the background.

### Available Agents

| Agent | Focus Area | Usage |
|-------|-----------|-------|
| `@security-reviewer` | OWASP Top 10, RBAC, secrets | `@security-reviewer` |
| `@lint-reviewer` | Code quality using golangci-lint | `@lint-reviewer` |
| `@test-reviewer` | Coverage, test quality | `@test-reviewer` |
| `@error-handling-reviewer` | Error handling patterns | `@error-handling-reviewer` |

### Manual Code Quality Checks

For performance and concurrency analysis, use the existing tooling:

| Tool | Focus Area | Command |
|------|-----------|---------|
| Race detector | Concurrency issues | `go test -race ./...` |
| Profiling | Performance bottlenecks | `go test -cpuprofile=cpu.prof -bench=.` |

### Running Individual Agents

You can invoke agents individually for focused reviews:

```
@security-reviewer
```

Or with specific instructions:

```
@security-reviewer focus on RBAC permissions in the new controller
```

### Agent Review Scope

All agents review **only the diff** between your branch and `master`:
```bash
git diff master...HEAD
```

They won't report issues in unchanged code.

## Hooks

Hooks are automated scripts that run during Claude Code workflows.

### `user-prompt-submit`

**Triggers:** When you submit a prompt to Claude  
**Purpose:** Reminds you to run tests after code changes

**Example output:**
```
📝 Code changes detected. After changes, remember to:
   • make lint  - Run golangci-lint
   • make test  - Run unit tests
```

### `before-commit`

**Triggers:** Before Claude creates a git commit  
**Purpose:** Runs validation to catch issues early

**What it runs:**
1. `make lint` - Linting must pass
2. `make test` - Tests must pass

**Note:** This runs *in addition* to git pre-commit hooks (if configured).

## Common Workflows

### Workflow 1: Implementing a New Feature

```
1. You: "Add a new field 'retryCount' to MyResource spec with validation"

2. Claude: [Modifies api/v1alpha1/myresource_types.go]

3. You: "/update-crds"
   Claude: [Runs make manifests && make generate]

4. You: "Now update the controller to use this field"
   Claude: [Modifies controller logic]

5. You: "/run-tests"
   Claude: [Runs lint + tests]

6. You: "Create a commit"
   Claude: [before-commit hook runs tests, then creates commit]
```

### Workflow 2: Pre-PR Code Review

```bash
# On your feature branch
git checkout feature/my-awesome-feature

# Start Claude Code
claude
```

```
You: "/full-review"

Claude: [Launches all 5 review agents in parallel]
        
        ## Full Review Report
        
        ### Security Review
        ✅ No critical issues found
        
        ### Performance Review
        🔴 1 high severity: Inefficient API calls in reconcile loop
        [Details and fix suggestions]
        
        ### Test Review
        🔴 2 high severity: Missing test coverage for error paths
        [Details and suggested tests]
        
        ### Error Handling Review
        ⚠️  3 medium: Missing error wrapping
        [Details]
        
        ### Concurrency Review
        ✅ No issues found
        
        ---
        Summary: 3 high, 3 medium issues found
        Estimated fix time: 1-2 hours
```

```
You: "Fix the high severity issues"

Claude: [Makes fixes]

You: "/run-tests"  # Verify fixes work

You: "/full-review"  # Re-review to confirm
```

### Workflow 3: Debugging Test Failures

```
You: "Tests are failing in TestReconcile"

Claude: [Reads test file, analyzes failure]

You: "Run the test with -v flag"

Claude: [Runs go test -v ./controllers/... -run TestReconcile]

You: "Fix the assertion"

Claude: [Updates test]

You: "/run-tests"  # Verify fix
```

### Workflow 4: Understanding Code

```
You: "Explain how the reconciliation loop works"

Claude: [Reads controller code, explains flow with references]

You: "Show me where we update the status"

Claude: [Shows specific lines with file:line references]

You: "What happens if the resource is being deleted?"

Claude: [Explains finalizer logic with code examples]
```

### Workflow 5: Creating a PR

```
You: "I'm ready to create a PR for this feature"

Claude: [Runs full review first]

You: "Create a PR"

Claude: 
   1. Runs git status, git diff
   2. Checks all commits
   3. Drafts PR title and description
   4. Pushes branch
   5. Creates PR via gh CLI
   6. Returns PR URL
```

## Tips & Best Practices

### DO

✅ **Run `/full-review` before creating PRs**  
Catches issues early, reduces review cycles

✅ **Use hooks to automate validation**  
Less manual work, fewer mistakes

✅ **Let agents run in background**  
Continue working while agents analyze code

✅ **Review agent findings critically**  
Agents are smart but not perfect - use judgment

✅ **Provide context in prompts**  
"Add rate limiting to the reconciler" is better than "add rate limiting"

✅ **Use specific agent for focused review**  
`@security-reviewer` is faster than full review if you only changed auth code

### DON'T

❌ **Don't ignore Critical/High findings**  
These often indicate real bugs

❌ **Don't bypass hooks**  
They exist to maintain quality

❌ **Don't commit without testing**  
Hooks will catch this, but test earlier

❌ **Don't assume agents know your intent**  
Provide context about business logic and constraints

❌ **Don't forget to re-review after fixes**  
Verify your changes actually addressed the issues

### Performance Tips

**Fast feedback:**
```
/run-tests quick  # Skips slow integration tests
```

**Parallel agent review:**
```
/full-review  # All agents run in parallel, faster than sequential
```

**Cache informer:**  
Agents use git worktrees, so your working directory stays clean

### Keyboard Shortcuts (if using desktop/IDE)

- `Ctrl+L` - Clear conversation
- `Ctrl+K` - New conversation
- `↑` - Previous prompt
- `Tab` - Autocomplete skill names

### Getting Help

**Within Claude Code:**
```
/help
```

**Repository-specific help:**
```
Read docs/CLAUDE_CODE_GUIDE.md  # This file
```

**Feedback & Issues:**
- General Claude Code: https://github.com/anthropics/claude-code/issues
- This repo's setup: Create issue in ocm-agent-operator repo

## Advanced Usage

### Custom Agents

You can create custom agents in `.claude/agents/`:

```markdown
---
name: my-custom-reviewer
description: Reviews XYZ aspects
model: sonnet
---

Review instructions here...
```

Then use with: `@my-custom-reviewer`

### Custom Skills

Create custom skills in `.claude/skills/`:

```markdown
---
name: my-skill
description: Does something useful
---

Skill instructions here...
```

Then use with: `/my-skill`

### Permissions

Adjust in `.claude/settings.json`:

```json
{
  "permissions": {
    "allow": [
      "Read:*",
      "Bash:make *",
      "Bash:go test *"
    ]
  }
}
```

### Environment Variables

Set operator-specific env vars in `.claude/settings.json`:

```json
{
  "env": {
    "KUBECONFIG": "/path/to/kubeconfig",
    "OPERATOR_NAMESPACE": "test-namespace"
  }
}
```

## Troubleshooting

### "Permission denied for Bash tool"

Add to `.claude/settings.json`:
```json
{
  "permissions": {
    "allow": ["Bash:your-command *"]
  }
}
```

### "Agent failed to run"

Check:
- Git is installed and repo has commits
- Branch is checked out (not detached HEAD)
- Enough disk space for worktree

### "Hooks are not running"

Verify:
- Hooks are executable: `chmod +x .claude/hooks/*`
- Hooks have correct shebang: `#!/bin/bash`
- Settings.json references hooks correctly

### "Skill not found"

- Skill files must be in `.claude/skills/`
- Must have frontmatter with `name:` field
- Use exact name: `/full-review` not `/fullreview`

## Resources

- [Claude Code Documentation](https://docs.anthropic.com/claude-code)
- [OSD Operator Guide](https://github.com/openshift/ops-sop/blob/master/operators/README.md)
- [Boilerplate Documentation](https://github.com/openshift/boilerplate)
- [SREP-4410: Claude Integration Epic](https://issues.redhat.com/browse/SREP-4410)

---

**Questions?** Ask Claude! It has context about this entire setup.
