# `.chai-bot/`

Repo-local configuration for [chai-bot](https://gitlab.cee.redhat.com/) (the
ship-help-bot). Following the established convention (see `openshift-online/rosa-e2e`,
`openshift-online/rosa-hyperfleet`, `app-sre/automata`), the **prompt content for
a scheduled task lives in the target repo** under `.chai-bot/`, and ship-help-bot
references it via a `%include` of the raw GitHub URL.

## Files

| File | Purpose |
|------|---------|
| `ocm_agent_operator_jira_label_triage.md` | The scheduled-task prompt chai-bot runs. This is the behaviour. |
| `schedulejob.yaml` | Source-of-truth record of the schedule (cron, channel, grants) that must be mirrored into ship-help-bot. |

## What the scheduled task does

On its cron, it queries a fixed set of ROSAENG Jira cards (the "base JQL" — a
safety gate that excludes security/embargo/CVE and closed work) and acts on two
opt-in labels:

- **`chai-triage`** → chai-bot posts a triage comment on the Jira card, then
  marks it `chai-triaged`.
- **`chai-code`** → chai-bot opens a PR against `openshift/ocm-agent-operator`
  implementing the card, then marks it `chai-code-pr` and comments the PR link.

PRs always require human `/lgtm` and `/approve`.

### Base JQL

```
project = ROSAENG
  AND component = ocm-agent-operator
  AND issuetype in (Bug, Story, Feature)
  AND issuetype not in (Vulnerability, Weakness)
  AND (labels is EMPTY OR labels not in (security, cve, embargo))
  AND status not in (Closed, Done, "Won't Do")
```

## How it is wired into ship-help-bot

The prompt here is not run by anything until it is registered in ship-help-bot.
Two changes are required there (see `schedulejob.yaml` for the exact values):

1. A `scheduled:` entry on the `rosa_engineering_public` persona in
   `config/groups/rosa/personas/rosa_engineering_public.yaml`, plus a
   `ocm_agent_operator_jira_triage_managers` user-group under
   `workspaces.internal.user_groups`.
2. A thin prompt file
   `ship_help_bot/shared/instructions/scheduled/ocm_agent_operator_jira_label_triage.md`
   whose only content is:

   ```
   %include(https://raw.githubusercontent.com/openshift/ocm-agent-operator/refs/heads/master/.chai-bot/ocm_agent_operator_jira_label_triage.md)
   ```

Because the prompt is resolved fresh on every run, edits to
`ocm_agent_operator_jira_label_triage.md` in this repo take effect on the next
scheduled run without any ship-help-bot change.
