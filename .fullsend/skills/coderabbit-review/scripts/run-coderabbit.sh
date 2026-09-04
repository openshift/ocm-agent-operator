#!/usr/bin/env bash
# Collect CodeRabbit findings for a PR and emit them as JSON on stdout.
#
# Modes:
#   comment (default, S3) - read CodeRabbit's existing PR review comments via
#                           `gh`. Sandbox-safe: uses only the read-only GitHub
#                           client already available to the review agent. No
#                           CodeRabbit credentials and no egress to
#                           coderabbit.ai.
#   cli (S2)              - run the CodeRabbit CLI for a fresh review. RUNNER
#                           ONLY. This will FAIL inside the FullSend review
#                           sandbox: no `coderabbit` binary, no `curl`,
#                           `api.coderabbit.ai` is not in the network allowlist,
#                           and CODERABBIT_API_KEY is not injected. Intended to
#                           run in a trusted pre-script whose output is copied
#                           into the sandbox via `host_files`.
#
# Usage:
#   scripts/run-coderabbit.sh <PR_NUMBER>
#   CODERABBIT_MODE=cli scripts/run-coderabbit.sh <PR_NUMBER>
set -euo pipefail

REPO="${CODERABBIT_REPO:-openshift/ocm-agent-operator}"
MODE="${CODERABBIT_MODE:-comment}"
BOT="${CODERABBIT_BOT:-coderabbitai}"

die() { echo "error: $*" >&2; exit 1; }

PR="${1:-}"
[ -n "$PR" ] || die "usage: run-coderabbit.sh <PR_NUMBER>"

case "$MODE" in
  comment)
    command -v gh >/dev/null || die "gh CLI not found"
    command -v jq >/dev/null || die "jq not found"
    # Inline review comments authored by CodeRabbit.
    review=$(gh api --paginate "repos/${REPO}/pulls/${PR}/comments" \
      --jq "[.[] | select(.user.login | test(\"${BOT}\"; \"i\")) |
             {source: \"coderabbit\", path: .path, line: (.line // .original_line),
              body: .body, url: .html_url}]")
    # Top-level PR (issue) comments authored by CodeRabbit.
    issue=$(gh api --paginate "repos/${REPO}/issues/${PR}/comments" \
      --jq "[.[] | select(.user.login | test(\"${BOT}\"; \"i\")) |
             {source: \"coderabbit\", path: null, line: null,
              body: .body, url: .html_url}]")
    jq -s 'add' <(printf '%s' "$review") <(printf '%s' "$issue")
    ;;
  cli)
    # RUNNER ONLY - see header. Blocked inside the review sandbox.
    command -v coderabbit >/dev/null || die "coderabbit CLI not found (runner-only; use MODE=comment in-sandbox)"
    [ -n "${CODERABBIT_API_KEY:-}" ] || die "CODERABBIT_API_KEY not set (never inject this into the sandbox)"
    # --plain keeps output parseable; adjust flags to the installed CLI version.
    coderabbit review --plain --pr "$PR"
    ;;
  *)
    die "unknown CODERABBIT_MODE: $MODE (expected 'comment' or 'cli')"
    ;;
esac
