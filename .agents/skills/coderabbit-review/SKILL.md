---
name: coderabbit-review
description: >-
  Ingest CodeRabbit findings for the current PR and map them into FullSend
  review findings for synthesis. Use during /fs-review after the built-in
  dimension sub-agents return, as a complementary (not replacement) source.
---

# CodeRabbit Review Skill

This skill adds CodeRabbit's AI review as an **extra finding source** for the
FullSend review agent. It does **not** replace `code-review` / `pr-review` —
those own protected-path checks, intent/coherence, the challenger pass, and the
JSON the post-script consumes. This skill only *gathers* CodeRabbit findings and
*maps* them into the same finding shape so the orchestrator can synthesise them.

> Naming: this skill is intentionally named `coderabbit-review` (a novel name).
> A repo skill named `code-review` or `pr-review` would be shadowed by the
> built-in and never invoked (Personal > Project precedence).

## When to use

- During `/fs-review` on a pull request, after the built-in review dimensions
  have produced their findings, to fold in CodeRabbit's findings.
- Not for local pre-push (`code-review`) unless CodeRabbit results are already
  available for the branch.

## Sources (in priority order)

The CodeRabbit CLI **cannot** run inside the review sandbox (no `coderabbit`
binary, no `curl`, `api.coderabbit.ai` is not in the network allowlist, and no
`CODERABBIT_API_KEY` is injected). So this skill never invokes the CLI directly.
Instead it reads findings that already exist:

1. **Injected file (S2, preferred for production):** if
   `/sandbox/workspace/coderabbit-findings.json` exists, read it. A runner-side
   pre-script produced it outside the sandbox; the API key never enters the
   sandbox.
2. **GitHub ingest (S3, spike default):** otherwise run
   `scripts/run-coderabbit.sh <PR_NUMBER>`, which uses the read-only `gh` client
   already available to the review agent to pull CodeRabbit's existing PR review
   comments. No extra network, binary, or secret.

If neither source yields findings, emit a short informational note and continue.
**Do not fail the whole review because CodeRabbit was unavailable.**

## Step 1: Gather CodeRabbit findings

```bash
# Prefer the injected file; fall back to GitHub ingest.
if [ -f /sandbox/workspace/coderabbit-findings.json ]; then
  cat /sandbox/workspace/coderabbit-findings.json
else
  scripts/run-coderabbit.sh "$PR_NUMBER"
fi
```

The script emits a JSON array of `{source, path, line, body, url}` objects.

## Step 2: Map into FullSend review findings

For each CodeRabbit item, produce a finding object with these fields:

- `severity` — map CodeRabbit's severity to the review scale; default to a
  low/`info` severity when CodeRabbit does not state one.
- `category` — prefix with `coderabbit-` (e.g. `coderabbit-correctness`,
  `coderabbit-style`) so the challenger / synthesiser can dedupe against the
  built-in dimensions.
- `file`, `line` — from the CodeRabbit comment (`null` for PR-level comments).
- `description` — CodeRabbit's finding text.
- `remediation` — CodeRabbit's suggested change, if present.
- `url` — link back to the CodeRabbit comment for human traceability.

## Step 3: Respect repo path exclusions

`.coderabbit.yaml` already excludes `boilerplate/**`, `hack/**`, `vendor/**`,
`**/testdata/**`, and generated `**/zz_generated.*.go`. Drop any CodeRabbit
finding whose `path` matches those globs so the review does not re-flag
generated or vendored code.

## Constraints

- **Do not** write `agent-result.json` / `review-result.json` — `pr-review` is
  the sole producer of the schema-valid verdict. This skill only contributes
  findings for synthesis.
- **Never** log, echo, or commit `CODERABBIT_API_KEY` or any token.
- Treat CodeRabbit findings as advisory input, not gate decisions — the
  built-in dimensions and challenger remain authoritative.

## Related

- `scripts/run-coderabbit.sh` — GitHub ingest (S3) and the runner-only CLI (S2)
  mode, with sandbox caveats documented inline.
- `FINDINGS.md` — spike write-up: integration patterns, invocation designs,
  sandbox constraints, recommendation, and effort estimate.
