# Claude Code Configuration for ocm-agent-operator

This directory contains Claude Code configuration, skills, agents, and hooks for the ocm-agent-operator repository.

## Directory Structure

```
.claude/
├── README.md                          # This file
├── settings.json                      # Claude Code configuration (permissions, hooks, env vars)
│
├── hooks/                             # Event-driven automation scripts
│   ├── user-prompt-submit            # Runs when user submits a prompt to Claude
│   └── before-commit                 # Runs before Claude creates a git commit
│
├── skills/                            # Custom slash commands
│   ├── full-review.md                # /full-review - Run all review agents
│   ├── run-tests.md                  # /run-tests - Execute test suite
│   └── update-crds.md                # /update-crds - Regenerate CRDs/manifests
│
└── agents/                            # Specialized review agents
    ├── security-reviewer.md          # @security-reviewer - Security analysis
    ├── test-reviewer.md              # @test-reviewer - Test quality review
    ├── error-handling-reviewer.md    # @error-handling-reviewer - Error handling patterns
    ├── concurrency-reviewer.md       # @concurrency-reviewer - Race conditions, goroutines
    └── performance-reviewer.md       # @performance-reviewer - Performance analysis
```

## Quick Reference

### Skills (Slash Commands)

| Command | Description | Usage |
|---------|-------------|-------|
| `/full-review` | Run all review agents in parallel | `/full-review` |
| `/run-tests` | Execute lint + tests + race detection | `/run-tests` |
| `/update-crds` | Regenerate CRDs after API changes | `/update-crds` |

### Agents (@ mentions)

| Agent | Focus | Model | Usage |
|-------|-------|-------|-------|
| `@security-reviewer` | OWASP Top 10, RBAC, secrets | Opus | `@security-reviewer` |
| `@test-reviewer` | Coverage, test quality | Sonnet | `@test-reviewer` |
| `@error-handling-reviewer` | Error patterns, logging | Sonnet | `@error-handling-reviewer` |
| `@concurrency-reviewer` | Races, goroutines, deadlocks | Opus | `@concurrency-reviewer` |
| `@performance-reviewer` | Allocations, API efficiency | Sonnet | `@performance-reviewer` |

### Hooks

| Hook | Trigger | Purpose |
|------|---------|---------|
| `user-prompt-submit` | When submitting prompt | Remind about testing |
| `before-commit` | Before git commit | Run lint + tests |

## Configuration Files

### settings.json

Defines:
- **Permissions**: Auto-approved Bash commands (make, go, git, etc.)
- **Hooks**: Event-driven scripts
- **Environment variables**: GOLANGCI_LINT_CONFIG, etc.

### Hooks

Hooks are executable shell scripts that run at specific events:

**user-prompt-submit**:
- Checks for Go/YAML file changes
- Reminds to run tests
- Non-blocking (always exits 0)

**before-commit**:
- Runs `make lint`
- Runs `make go-test`
- Blocks commit if failures (exits 1)

**Making hooks executable**:
```bash
chmod +x .claude/hooks/*
```

## Creating Custom Skills

1. Create a markdown file in `.claude/skills/`:

```markdown
---
name: my-skill
description: Short description
---

# My Skill

Instructions for Claude on what to do when this skill is invoked.

## Steps
1. Do this
2. Then this
```

2. Use it: `/my-skill`

## Creating Custom Agents

1. Create a markdown file in `.claude/agents/`:

```markdown
---
name: my-agent
description: What this agent reviews
model: sonnet  # or opus
isolation: worktree  # runs in isolated git worktree
---

Review instructions here.
Focus on specific patterns, anti-patterns, etc.
```

2. Use it: `@my-agent`

## Permissions

The `settings.json` file defines auto-approved commands. Commands not in the allow list will prompt for user approval.

**Current allowed commands**:
- `Read:*`, `Glob:*`, `Grep:*` - File reading
- `Bash:make *` - All make targets
- `Bash:go *` - Go commands
- `Bash:git *` - Git read operations
- `Bash:kubectl get/describe/explain` - Kubernetes read operations
- `Bash:gh *` - GitHub CLI

**To add new permissions**, edit `settings.json`:

```json
{
  "permissions": {
    "allow": [
      "Bash:your-command *"
    ]
  }
}
```

## Environment Variables

Set in `settings.json`:

```json
{
  "env": {
    "VAR_NAME": "value"
  }
}
```

These are available to hooks and Claude's Bash tool calls.

## Integration with Boilerplate (SREP-4410)

This `.claude/` directory and its contents are managed by [openshift/boilerplate](https://github.com/openshift/boilerplate).

**Updating from boilerplate**:
```bash
make boilerplate-update
```

This will sync:
- Common Claude Code configuration
- Standard hooks and skills
- Review agent templates

**Local customizations**: Keep repo-specific customizations separate from boilerplate-managed sections.

## Troubleshooting

### Hooks not running
- Ensure hooks are executable: `chmod +x .claude/hooks/*`
- Check hooks have correct shebang: `#!/bin/bash`
- Verify settings.json references hooks correctly

### Skills not found
- Skills must be in `.claude/skills/`
- Must have valid frontmatter with `name:` field
- Use exact name from frontmatter: `/full-review` not `/fullreview`

### Agents failing
- Ensure git worktree can be created (needs disk space)
- Check agent model is valid: `sonnet`, `opus`, or `haiku`
- Verify branch is checked out (not detached HEAD)

### Permission prompts
- Add frequently used commands to `settings.json` allow list
- Use wildcards: `Bash:kubectl get*` allows all get subcommands

## Resources

- [Complete Claude Code Guide](../docs/CLAUDE_CODE_GUIDE.md) - Comprehensive usage documentation
- [Claude Code Documentation](https://docs.anthropic.com/claude-code)
- [SREP-4410 Epic](https://issues.redhat.com/browse/SREP-4410) - Claude integration for operators

---

**For detailed usage instructions**, see [docs/CLAUDE_CODE_GUIDE.md](../docs/CLAUDE_CODE_GUIDE.md)
