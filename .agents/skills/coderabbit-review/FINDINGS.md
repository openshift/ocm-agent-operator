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
