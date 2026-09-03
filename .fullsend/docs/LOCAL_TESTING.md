# Local Testing Guide

How to test the FullSend agent implementation locally.

## Quick Start

```bash
# Run all local tests
.fullsend/scripts/test-local.sh

# Test CodeRabbit script against PR #316
.fullsend/scripts/test-local.sh 316

# Verbose output
VERBOSE=1 .fullsend/scripts/test-local.sh 316
```

## Prerequisites

### Required
- ✅ `gh` (GitHub CLI) - Already installed
- ✅ `jq` (JSON processor) - Already installed
- ✅ `python3` (for YAML validation) - System default

### Optional
- `fullsend` CLI (for full agent execution)

## Testing Levels

### Level 1: Configuration Validation ✅ (Works Now)

Validate YAML syntax and configuration structure:

```bash
# Validate all YAML files
python3 -c "import yaml; yaml.safe_load(open('.fullsend/config.yaml'))"
python3 -c "import yaml; yaml.safe_load(open('.fullsend/harness/review.yaml'))"

# Run validation script
.fullsend/scripts/test-local.sh
```

**What it checks:**
- ✓ YAML syntax is valid
- ✓ Required config fields present
- ✓ Review agent is registered
- ✓ Directory structure correct
- ✓ Symlinks point to right locations

### Level 2: Skill Script Testing ✅ (Works Now)

Test the CodeRabbit skill script directly:

```bash
# 1. Authenticate with GitHub (one-time setup)
gh auth login

# 2. Test against a real PR
.fullsend/skills/coderabbit-review/scripts/run-coderabbit.sh 316

# Expected output: JSON array of CodeRabbit findings
```

**Test different scenarios:**

```bash
# PR with CodeRabbit comments
.fullsend/skills/coderabbit-review/scripts/run-coderabbit.sh 316

# PR without CodeRabbit comments (should return empty array)
.fullsend/skills/coderabbit-review/scripts/run-coderabbit.sh 1

# Pretty-print JSON
.fullsend/skills/coderabbit-review/scripts/run-coderabbit.sh 316 | jq '.'

# Count findings
.fullsend/skills/coderabbit-review/scripts/run-coderabbit.sh 316 | jq '. | length'

# Filter by file
.fullsend/skills/coderabbit-review/scripts/run-coderabbit.sh 316 | jq '.[] | select(.path == "Makefile")'
```

### Level 3: Integration Testing ⚠️ (Requires FullSend CLI)

**Option A: Install FullSend CLI** (if available)

```bash
# Install fullsend CLI
pip install fullsend  # or your installation method

# Validate harness
fullsend harness validate .fullsend/harness/review.yaml

# List agents
fullsend agent list

# Run review agent locally (requires sandbox setup)
fullsend run review --local
```

**Option B: Test via GitHub PR** (recommended)

1. Push your changes to a branch
2. Create a pull request
3. Trigger the review agent:
   ```
   Comment on the PR: /fs-review
   ```
4. Monitor GitHub Actions for the agent run

### Level 4: End-to-End Testing (GitHub Only)

Test the full workflow in the actual environment:

```bash
# 1. Create a test branch
git checkout -b test/fullsend-review

# 2. Make a small change
echo "# Test" >> .fullsend/docs/test.md
git add .fullsend/docs/test.md
git commit -m "test: trigger fullsend review"

# 3. Push and create PR
git push -u origin test/fullsend-review
gh pr create --title "Test FullSend Review" --body "Testing Pattern B implementation"

# 4. Trigger review on the PR
gh pr comment <PR_NUMBER> --body "/fs-review"

# 5. Monitor the run
gh run watch
gh run view --log

# 6. Check the review findings
gh pr view <PR_NUMBER> --comments
```

## Validation Checklist

Before pushing to production, verify:

- [ ] YAML syntax valid (`.fullsend/config.yaml`, harness files)
- [ ] Review agent registered in config
- [ ] Symlink `.claude/skills` → `../.fullsend/skills` works
- [ ] CodeRabbit script is executable
- [ ] CodeRabbit script returns valid JSON
- [ ] Harness uses `base:` composition
- [ ] Harness includes SHA256 integrity check
- [ ] Skills reference is correct: `skills/coderabbit-review`
- [ ] Documentation updated (`AGENTS.md`, skill READMEs)
- [ ] Pre-commit hooks pass

## Troubleshooting

### "GitHub CLI not authenticated"

```bash
gh auth login
# Follow prompts to authenticate
```

### "Script not executable"

```bash
chmod +x .fullsend/skills/coderabbit-review/scripts/run-coderabbit.sh
```

### "Symlink broken"

```bash
rm .claude/skills
ln -s ../.fullsend/skills .claude/skills
```

### "YAML syntax error"

```bash
# Validate with detailed error
python3 -c "import yaml; yaml.safe_load(open('.fullsend/harness/review.yaml'))"
```

### "CodeRabbit script returns empty"

Possible causes:
- PR has no CodeRabbit comments yet
- CodeRabbit hasn't finished reviewing
- Wrong PR number or repository

## What You CAN'T Test Locally

These require the full FullSend runtime in GitHub Actions:

- ❌ Sandbox execution
- ❌ Harness composition (base: merging)
- ❌ Agent orchestration
- ❌ Finding synthesis
- ❌ PR comment posting
- ❌ Integration with /fs-fix workflow

For these, use the GitHub PR testing approach (Level 4).

## CI/CD Validation

The CI will run:
- Tekton pipelines (`.tekton/`)
- Prek hooks (formatting, security, linting)
- FullSend agent (when triggered)

Monitor in:
- **GitHub Actions**: `.github/workflows/fullsend.yaml`
- **Prow CI**: https://prow.ci.openshift.org/

## Quick Reference

```bash
# Validate everything
.fullsend/scripts/test-local.sh

# Test CodeRabbit against PR
.fullsend/skills/coderabbit-review/scripts/run-coderabbit.sh <PR_NUMBER>

# Trigger review on GitHub
gh pr comment <PR_NUMBER> --body "/fs-review"

# Monitor GitHub Actions
gh run watch

# View logs
gh run view --log
```

## See Also

- [FullSend Documentation](https://github.com/fullsend-ai/fullsend)
- [AGENTS.md](../../AGENTS.md) - Agent conventions
- [.fullsend/config.yaml](../config.yaml) - Configuration
- [.fullsend/harness/review.yaml](../harness/review.yaml) - Review agent
