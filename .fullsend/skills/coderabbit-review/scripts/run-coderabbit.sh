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

# Emit empty JSON array and exit (for non-fatal failures in comment mode)
emit_empty() {
  echo "[]"
  exit 0
}

REPO="${CODERABBIT_REPO:-openshift/ocm-agent-operator}"
MODE="${CODERABBIT_MODE:-comment}"
BOT="${CODERABBIT_BOT:-coderabbitai}"

die() { echo "error: $*" >&2; exit 1; }

PR="${1:-}"
[ -n "$PR" ] || die "usage: run-coderabbit.sh <PR_NUMBER>"

case "$MODE" in
  comment)
    # Sandbox-compatible: uses only gh and node (both allowlisted).
    # Failures are non-fatal - emit empty array if CodeRabbit data unavailable.
    command -v gh >/dev/null || { echo "warning: gh CLI not found" >&2; emit_empty; }
    command -v node >/dev/null || { echo "warning: node not found" >&2; emit_empty; }

    # Get current PR head commit to filter stale comments.
    head_sha=$(gh api "repos/${REPO}/pulls/${PR}" --jq '.head.sha' 2>/dev/null || true)
    if [ -z "$head_sha" ]; then
      echo "warning: failed to get PR head commit" >&2
      emit_empty
    fi

    # Fetch review comments and issue comments (non-fatal on failure).
    review_raw=$(gh api --paginate "repos/${REPO}/pulls/${PR}/comments" 2>/dev/null || echo "[]")
    issue_raw=$(gh api --paginate "repos/${REPO}/issues/${PR}/comments" 2>/dev/null || echo "[]")

    # Use node to filter and transform comments (sandbox-compatible).
    node -e "
      const bot = process.argv[1].toLowerCase();
      const headSha = process.argv[2];
      const reviewRaw = JSON.parse(process.argv[3]);
      const issueRaw = JSON.parse(process.argv[4]);

      // Filter review comments: exact bot login, current commit only.
      const review = reviewRaw
        .filter(c => c.user?.login?.toLowerCase() === bot)
        .filter(c => c.commit_id === headSha)
        .map(c => ({
          source: 'coderabbit',
          path: c.path,
          line: c.line ?? c.original_line,
          body: c.body,
          url: c.html_url,
          commit_id: c.commit_id
        }));

      // Filter issue comments: exact bot login.
      const issue = issueRaw
        .filter(c => c.user?.login?.toLowerCase() === bot)
        .map(c => ({
          source: 'coderabbit',
          path: null,
          line: null,
          body: c.body,
          url: c.html_url
        }));

      console.log(JSON.stringify([...review, ...issue]));
    " "$BOT" "$head_sha" "$review_raw" "$issue_raw"
    ;;
  cli)
    # RUNNER ONLY - see header. Blocked inside the review sandbox.
    # CODERABBIT_API_KEY must be set as a CI secret and exposed to this pre-script.
    # See SKILL.md and review.yaml for CI setup instructions.
    command -v coderabbit >/dev/null || die "coderabbit CLI not found (runner-only; use MODE=comment in-sandbox)"
    command -v node >/dev/null || die "node not found (needed to normalize CLI output)"
    [ -n "${CODERABBIT_API_KEY:-}" ] || die "CODERABBIT_API_KEY not set (must be configured as CI secret; see SKILL.md)"

    # Get CLI output (try JSON format first, fall back to plain text parsing if needed).
    # Adjust flags based on your installed CLI version's capabilities.
    cli_output=$(coderabbit review --format json --pr "$PR" --api-key "$CODERABBIT_API_KEY" 2>/dev/null \
                 || coderabbit review --plain --pr "$PR" --api-key "$CODERABBIT_API_KEY" 2>&1 \
                 || die "coderabbit CLI failed")

    # Normalize CLI output to {source, path, line, body, url}[] format.
    node -e "
      const raw = process.argv[1];
      let findings = [];

      try {
        // Try parsing as JSON first
        const parsed = JSON.parse(raw);
        findings = (Array.isArray(parsed) ? parsed : [parsed])
          .filter(f => f.file && f.line && f.message)
          .map(f => ({
            source: 'coderabbit',
            path: f.file || f.path || null,
            line: f.line || null,
            body: f.message || f.body || f.comment || '',
            url: f.url || null
          }));
      } catch (e) {
        // Plain text fallback: emit empty array (CLI output not parseable)
        console.error('warning: CLI output not in expected JSON format', e.message);
        findings = [];
      }

      console.log(JSON.stringify(findings));
    " "$cli_output"
    ;;
  *)
    die "unknown CODERABBIT_MODE: $MODE (expected 'comment' or 'cli')"
    ;;
esac