# AGENTS.md

Conventions for anyone — human or agent — working in this repository.

The full behavioral ruleset lives in [CLAUDE.md](./CLAUDE.md) (build/lint/test
commands, RBAC and FIPS constraints, layering boundaries). This file adds the
few things FullSend agents should know beyond that.

## Repo skills

Repo skills live in `.fullsend/skills/` (with `.claude/skills` symlinked to it).
They extend the built-in FullSend skills; they do not replace them.

## Code review

When reviewing a pull request, also consult the **`coderabbit-review`** skill
(`.fullsend/skills/coderabbit-review/`) as a complementary finding source:

- Treat CodeRabbit findings as **advisory input** to synthesise alongside the
  built-in review dimensions — not as a replacement for `pr-review`/`code-review`
  and not as the gate decision.
- The built-in dimensions and challenger remain authoritative for the verdict.
- Skip CodeRabbit findings on paths excluded by `.coderabbit.yaml`
  (`boilerplate/**`, `hack/**`, `vendor/**`, `**/testdata/**`, generated
  `**/zz_generated.*.go`) so generated and vendored code is not re-flagged.
- If CodeRabbit results are unavailable, note it and continue — do not fail the
  review.

Do not commit `CODERABBIT_API_KEY` or any token; the review sandbox has no
CodeRabbit credentials by design.
