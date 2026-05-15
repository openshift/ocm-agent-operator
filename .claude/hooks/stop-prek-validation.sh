#!/usr/bin/env bash
set -uo pipefail

HOOK_INPUT=$(cat)

# Allow stop on retry to prevent infinite loops
STOP_HOOK_ACTIVE=$(echo "$HOOK_INPUT" | jq -r '.stop_hook_active // false')
if [[ "$STOP_HOOK_ACTIVE" == "true" ]]; then
  exit 0
fi

# Check if prek is installed — block and nudge instead of silently passing
if ! command -v prek &> /dev/null; then
  jq -n \
    --arg reason "prek is not installed — required for quality checks before stopping.

Install it:
  uv tool install prek      # recommended
  pipx install prek         # alternative
  pip install --user prek   # fallback

Then wire up the git hook: prek install

Retry the action once installed so validation can run." \
    '{"decision": "block", "reason": $reason}'
  exit 0
fi

# Run prek validation (using CI config to skip network-dependent hooks)
PREK_OUTPUT=$(prek run --config hack/prek.ci.toml --all-files 2>&1)
PREK_EXIT=$?

if [[ $PREK_EXIT -eq 0 ]]; then
  exit 0
fi

# Block stop and tell Claude what to fix
jq -n \
  --arg reason "prek validation failed. Fix the issues below, then try again:

$PREK_OUTPUT" \
  '{"decision": "block", "reason": $reason}'
