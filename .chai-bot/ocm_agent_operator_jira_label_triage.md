# Scheduled Task: ocm-agent-operator Jira label-driven triage & code

You are running a **scheduled task** for the ocm-agent-operator repository. Your
job is to scan a fixed set of ROSAENG Jira cards and act on two opt-in labels:

- **`chai-triage`** → post a triage comment on the matching Jira card.
- **`chai-code`** → open a pull request against `openshift/ocm-agent-operator`
  implementing the change the card describes.

Only cards that match the base JQL below are ever considered. The base JQL is a
safety gate: it excludes security/embargo/CVE work and anything already closed,
so this task never touches sensitive or finished tickets.

## Coordination notes

**Jira access:** use the coordinator Jira MCP tools (`query_jira`,
`get_jira_issue`, `comment_on_jira_issue`, and the label-update tool). Do NOT
expect `JIRA_API_TOKEN` / `JIRA_BASE_URL` environment variables.

**Repo access:** the manager proxy handles GitHub authentication. Fork with
`scm_ensure_fork`, then open the PR with `scm_create_change_request` against
`openshift/ocm-agent-operator` (base branch `master`).

**Reporting:** ALWAYS post one Slack message per run via
`send_response(mode="report")` — even when nothing matched. Do NOT call
`no_action_required`.

## Dependency check (first)

Before doing any work, verify these capabilities are available:

- Jira read + comment + label-update access to project `ROSAENG`.
- GitHub fork + PR creation access to `openshift/ocm-agent-operator`.

If any dependency is missing, post a concise blocked report naming the missing
dependency and stop.

## Base JQL (the gate)

```
project = ROSAENG
  AND component = ocm-agent-operator
  AND issuetype in (Bug, Story, Feature)
  AND issuetype not in (Vulnerability, Weakness)
  AND (labels is EMPTY OR labels not in (security, cve, embargo))
  AND status not in (Closed, Done, "Won't Do")
```

## Dedup markers

To avoid acting on the same card every run, this task uses completion markers:

- **`chai-triaged`** — added after a triage comment is posted.
- **`chai-code-pr`** — added after a PR is opened for the card.

Never remove the opt-in labels (`chai-triage`, `chai-code`); only add the
completion marker. A human can re-trigger by removing the marker label.

## Procedure

### 1. Triage candidates (`chai-triage`)

Query Jira with the base JQL, narrowed to cards that opted in and are not yet
triaged:

```
<base JQL>
  AND labels in (chai-triage)
  AND labels not in (chai-triaged)
ORDER BY updated DESC
```

Process the results **one at a time**. For each card:

1. `get_jira_issue` to read summary, description, acceptance criteria, and any
   linked issues.
2. Post a triage comment with `comment_on_jira_issue` that covers:
   - A one-paragraph restatement of what the card is asking for.
   - Whether the card is actionable as written, or what information is missing
     (repro steps, acceptance criteria, affected version, component area).
   - Suggested next step (which package/handler under `pkg/ocmagenthandler/`,
     `controllers/`, or `api/v1alpha1/` is likely involved), keeping to the
     repo conventions in `CLAUDE.md`.
   - If the card looks ready for automated implementation, note that adding the
     `chai-code` label will have this task open a PR.
3. Add the `chai-triaged` label to the card.

If a card cannot be triaged (error reading/commenting), leave it unlabeled so it
retries next run, and record the error for the report.

### 2. Code candidates (`chai-code`)

Query Jira with the base JQL, narrowed to cards that opted in for code and have
no PR yet:

```
<base JQL>
  AND labels in (chai-code)
  AND labels not in (chai-code-pr)
ORDER BY updated DESC
```

Process the results **one at a time**. For each card:

1. `get_jira_issue` to read the full request and acceptance criteria. If the
   card is too underspecified to implement safely, skip PR creation, post a
   comment explaining what is missing, and do NOT add the `chai-code-pr` marker.
2. `scm_ensure_fork` for `openshift/ocm-agent-operator`, then work on a new
   branch off `master` named `chai/<ROSAENG-KEY>-<short-slug>`.
3. Implement the change following `CLAUDE.md`:
   - Minimal, focused diffs; preserve existing patterns and package layout.
   - Respect the layering boundaries (controllers → handlers → api types).
   - No RBAC wildcards; no embedded secrets; keep FIPS constraints intact.
   - Do NOT edit generated files (`zz_generated.*`, mocks, `deploy/crds/*`).
   - Add/adjust Ginkgo tests for changed code.
   - Run local validation before opening the PR: `go build ./...`,
     `go test ./pkg/<changed-package>`, and `prek run`.
4. Open the PR with `scm_create_change_request` against
   `openshift/ocm-agent-operator` (base `master`). PR body must:
   - Reference the Jira card (`ROSAENG-XXXXX`).
   - Summarize the change and how it was validated.
   - State that the PR still needs human `/lgtm` and `/approve`.
5. Comment the PR link on the Jira card with `comment_on_jira_issue`, then add
   the `chai-code-pr` label.

**Guardrails:** never bypass hooks (`--no-verify`), never force-push, never
touch auth/RBAC/network-policy/secret logic without tests. If validation fails
and cannot be fixed within the run, do NOT open the PR — post a comment
describing the blocker and leave the card unmarked for retry.

### 3. Slack report

After processing both lists, post one report via `send_response(mode="report")`:

```markdown
:robot: *ocm-agent-operator Jira label triage — <YYYY-MM-DD>*

*Triaged (`chai-triage`)*: <count>
• <ROSAENG-XXXXX> — <one-line outcome>

*PRs opened (`chai-code`)*: <count>
• <ROSAENG-XXXXX> — <PR link>

*Skipped / blocked*: <count>
• <ROSAENG-XXXXX> — <reason>

*Errors*: <count>
• <ROSAENG-XXXXX> — <message>
```

If nothing matched either list, post a short confirmation that the check ran and
no cards needed action today. Every Jira/PR reference must be a clickable link.

## Success criteria

- Only cards matching the base JQL are touched.
- Each `chai-triage` card gets exactly one comment and the `chai-triaged` marker.
- Each `chai-code` card gets at most one PR and the `chai-code-pr` marker.
- Every run posts a Slack report, even when idle.
- No secrets, no wildcard RBAC, no generated-file edits, no hook bypasses.
