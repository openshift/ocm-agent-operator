# ROSAENG-62419 — CodeRabbit × FullSend review integration: spike findings

Spike to investigate and prototype extending the FullSend `/fs-review` agent to
support CodeRabbit as a code review source for `openshift/ocm-agent-operator`,
and to recommend an integration pattern. Parent epic: ROSAENG-62415.

## TL;DR

- **Complementary, not replacement.** CodeRabbit should be an *extra finding
  source* the review orchestrator synthesises — it must not replace the built-in
  `pr-review` / `code-review` skills, which own protected-path checks,
  intent/coherence, the challenger pass, and the schema-valid verdict the
  post-script consumes.
- **Prototype = Pattern A + S3.** A novel-named `coderabbit-review` repo skill,
  wired via `AGENTS.md`, that ingests CodeRabbit's existing GitHub PR comments
  through the read-only `gh` client already in the sandbox.
- **Production default = Pattern B + S2.** A config-registered derived review
  harness (`base:` composition) that keeps every built-in skill and adds
  `coderabbit-review`, with the CodeRabbit CLI run on the trusted runner and its
  output injected via `host_files`.
- **The in-sandbox CodeRabbit CLI is blocked** and should not be attempted for
  the spike (see Invocation designs / Constraints in part 2).

## Acceptance criteria — status

| Criterion | Status |
|---|---|
| Document most viable integration pattern (A/B/C) | Done — see below |
| Prototype a `coderabbit-review` skill in `.agents/skills/` | Done — `SKILL.md` + `scripts/run-coderabbit.sh` |
| Validate skill is discoverable by the review agent | Done — novel name, symlink resolves, referenced from `AGENTS.md` |
| Confirm CodeRabbit invocation in the sandbox | Done — in-sandbox CLI blocked; `gh` ingest (S3) works |
| Document sandbox / network / auth constraints | Done — see part 2 |
| Recommendation on proceeding | Done — proceed, complementary; see part 2 |
| Effort estimate for production | Done — see part 2 |

## Integration patterns

| Pattern | What it is | Viable for spike? | Viable as "default"? |
|---|---|---|---|
| **A — Parallel skill** | `.agents/skills/coderabbit-review/` with a unique name, referenced by `AGENTS.md` | **Yes** — proves discovery + ingest | **No** — the review agent won't treat it as the default; in-sandbox CLI fails |
| **B — Agent registration** | Thin `harness/review.yaml` with `base:` + the extra skill, registered as `review` in `.fullsend/config.yaml` | Partial — needs more than a `SKILL.md` | **Yes** — config-registered agents win on name collision, so this becomes the default review agent |
| **C — BYOA replacement** | A new custom agent that replaces FullSend review | Overkill | **No** — you lose the built-in dimensions, schema, labels, and `/fs-fix` loop |

**Verdict:** Pattern **A** for this spike; Pattern **B** for a production default.
Pattern **C** only makes sense if the goal is to retire FullSend review entirely
and let CodeRabbit own the PR comment — which fights the ADLC design, since the
review verdict is the gate the rest of the pipeline consumes.

> "Default review agent" in FullSend terms means the registered `review` role
> still runs and CodeRabbit is a mandatory *extra input* to synthesis — not that
> CodeRabbit replaces `/fs-review`.

## Invocation designs

Three ways to get CodeRabbit findings into the review; the skill supports the
two that don't require sandbox changes.

- **S1 — CLI inside the sandbox.** Needs a custom image containing the
  `coderabbit` binary, an openshell profile allowing `api.coderabbit.ai`, and
  `CODERABBIT_API_KEY` injected via `env.sandbox`. **Highest risk** — the API
  key lands next to `GH_TOKEN` in the sandbox. **Not for the spike;** it is an
  upstream FullSend change (provider profile + policy), out of scope.
- **S2 — CLI on the trusted runner (pre-script).** A pre-script runs *outside*
  the sandbox: install the CLI, `coderabbit review --api-key "$CODERABBIT_API_KEY"`,
  write JSON, and copy it in via `host_files` to
  `/sandbox/workspace/coderabbit-findings.json`. The key never enters the
  sandbox. Requires Pattern B (pre-script + `host_files` + `env.runner` come via
  `base:`). **Best if fresh CLI findings are required.**
- **S3 — Ingest existing CodeRabbit GitHub reviews via `gh`.** This repo already
  has `.coderabbit.yaml`, so the CodeRabbit GitHub App already reviews PRs. The
  review agent already has read-only `gh`. `scripts/run-coderabbit.sh` pulls
  CodeRabbit's PR comments and maps them in — no extra network, binary, or
  secret. **Best first experiment.** Risks: a race (FullSend may finish before
  CodeRabbit posts) and duplicate human-visible comments if both bots post.

**Spike prototypes A + S3.** Production implements B + S2 (if CLI-quality
findings are needed), and should mute CodeRabbit's own PR comments so humans do
not get two reviews.

## Discoverability — discovery ≠ use

- Skills live in `.agents/skills/`; `.claude/skills` is symlinked to it. Skill
  precedence is Personal (`CLAUDE_CONFIG_DIR/skills/`, FullSend built-ins) >
  Project (`.claude/skills/`, repo). A repo skill named the same as a built-in
  (`code-review`, `pr-review`, …) is **shadowed**. We use the novel name
  `coderabbit-review`, so it is always available.
- **Discovery is not invocation.** The review agent is a fixed orchestrator; a
  discoverable skill is not run automatically. To make it run:
  - **Spike:** reference it by name from `AGENTS.md` (done) — the lightest lever.
  - **Production:** add it to the derived harness `skills:` list (Pattern B) for
    reliable, config-driven invocation.
- This repo's migration (move `prow-ci` under `.agents/skills/`, add the
  symlink) is already complete, so the symlink resolves and the skill is
  discoverable today.

## Constraints (sandbox / network / auth)

Asserted from FullSend's review policy (`policies/github/review.yaml`, upstream —
not vendored in this repo); confirm against the installed CLI before production.

| Constraint | Effect |
|---|---|
| Network allowlist | Only `api.anthropic.com`, `*.googleapis.com`, `api.github.com`, `github.com`. No `api.coderabbit.ai` / `cli.coderabbit.ai` |
| Binary allowlist | `claude`, `node`, `gh` only. No `coderabbit`, no `curl` |
| Secrets | No `CODERABBIT_API_KEY` injected; a CodeRabbit *Agentic* API key would be needed on the runner, never committed |
| Repo access | `readonly_repo: true` |
| `allowed_remote_resources` | Config allows only `fullsend-ai/fullsend` + `fullsend-ai/agents` (so a Pattern B `base:` on `fullsend-ai/agents` is permitted) |
| Image | `fullsend-code` does not ship the CodeRabbit CLI |
| Timeout | Review harness ~20 min; CLI + dimensions may need a bump |
| Result schema | `additionalProperties: false` — extra fields on the result are rejected; the skill must not emit the result JSON |

Net effect: **skill discoverable = yes; in-sandbox CodeRabbit CLI = no** without
an upstream harness/policy change. S3 (`gh` ingest) is the sandbox-safe path.

## Pattern B specifics (for production)

- Register a derived `review` in `.fullsend/config.yaml` under `agents:` with a
  `harness/review.yaml` using `base:` (pinned SHA + `#sha256=` integrity hash)
  and `coderabbit-review` added to `skills:` (merged by basename).
- Config-registered agents win on name collision → this becomes the default
  review agent while inheriting all base scripts, policies, `host_files`, plugins.
- **Not inherited from `base:`:** `allowed_remote_resources`, `allow_runtime_fetch`,
  `max_runtime_fetches` — the child must redeclare them.
- Confirm the installed FullSend version supports config `agents:` + `base:`
  skill merge-by-basename before implementing (this repo is pinned to v0.32.0).

## Recommendation

**Proceed — complementary, not replacement.**

- **Spike (this card):** Pattern A skill + `AGENTS.md` wiring + document that the
  in-sandbox CLI is blocked + optional S3 ingest on a test PR. Do **not** try to
  make CodeRabbit the default via a `customized/skills/code-review/` overlay
  (deprecated per ADR-0064).
- **Production:** Pattern B derived review harness that keeps all built-in skills
  and adds `coderabbit-review`; S2 runner pre-script if CLI-quality findings are
  required, otherwise keep S3.
- Do **not** do Pattern C unless product explicitly wants to retire FullSend
  review.

## Effort estimate (production-ready)

| Slice | Estimate |
|---|---|
| Spike: skill + discovery proof + constraints write-up (this card) | 3–5 days |
| S3 ingest + synthesis/severity mapping + tests on a real PR | +3–4 days |
| Pattern B harness (`base:` + `agents:` pin + secret + S2 pre-script) | +1–1.5 weeks |
| Dedup vs CodeRabbit App comments, severity mapping, timeout, failure modes | +3–5 days |
| Pattern C full replacement (only if required) | 4–6 weeks, high regression risk |

Recommended path total: **~2.5–4 weeks after the spike**, mostly harness /
secrets / CI — not `SKILL.md` prose.

## Open / unverified items

- v0.32.0 pin: confirm config `agents:` + `base:` skill merge support before B.
- Confirm CodeRabbit already posts on `openshift/ocm-agent-operator` PRs (this
  checkout is a fork); if so, plan to mute its comments to avoid double reviews.
- The review sandbox policy above is from upstream FullSend, not vendored here —
  re-verify against the installed package.
- Parent epic ROSAENG-62415 not pulled (Atlassian MCP not authenticated).

## Out of scope

Full production implementation, FullSend core changes, a
`customized/skills/code-review` override, and integration with agents other than
review.
