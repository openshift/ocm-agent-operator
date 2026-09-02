#!/usr/bin/env bash
#
# pre-boilerplate.sh — runs on the trusted runner BEFORE the sandbox is created.
#
# Runs `make boilerplate-update` in the target repo checkout. If the update
# produced no changes, neutral-skips (exit 78) so the sandbox is never created.
# Otherwise it records the diff and the old/new boilerplate SHAs into a file the
# harness mounts into the sandbox for the agent to review.
#
# Contract:
#   TARGET_REPO_DIR         — target repository checkout (set by the runner)
#   RUNNER_TEMP             — scratch dir on the runner
#   FULLSEND_PRESCRIPT_OUTPUT (optional) — key=value output file for the runner
set -euo pipefail

REPO_DIR="${TARGET_REPO_DIR:-.}"
cd "${REPO_DIR}"

COMMIT_FILE="boilerplate/_data/last-boilerplate-commit"
OLD_SHA="$(cat "${COMMIT_FILE}" 2>/dev/null || echo "unknown")"

echo "Running 'make boilerplate-update' in ${REPO_DIR} (from ${OLD_SHA})..."
make boilerplate-update

# Nothing changed -> neutral skip; the last non-empty stdout line is the reason.
if [ -z "$(git status --porcelain)" ]; then
  if [ -n "${FULLSEND_PRESCRIPT_OUTPUT:-}" ]; then
    {
      echo "skipped=true"
      echo "reason=boilerplate already up to date (${OLD_SHA})"
    } >> "${FULLSEND_PRESCRIPT_OUTPUT}"
  fi
  echo "Boilerplate already up to date; nothing to do."
  exit 78
fi

NEW_SHA="$(cat "${COMMIT_FILE}" 2>/dev/null || echo "unknown")"
DIFF_OUT="${RUNNER_TEMP:-/tmp}/boilerplate-diff.txt"
{
  echo "boilerplate_from=${OLD_SHA}"
  echo "boilerplate_to=${NEW_SHA}"
  echo "---"
  git --no-pager diff
} > "${DIFF_OUT}"

echo "Boilerplate update produced changes (${OLD_SHA} -> ${NEW_SHA})."
echo "Diff recorded at ${DIFF_OUT}."
