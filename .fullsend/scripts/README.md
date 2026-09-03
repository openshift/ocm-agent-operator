# Scripts

Utility scripts for FullSend agents and workflows.

## Purpose

This directory contains executable scripts that support agent workflows,
automation, and operational tasks.

## Contents

Examples of what to place here:
- **Pre/post scripts**: Run before/after agent execution
- **Utility functions**: Common operations used by multiple agents
- **Integration helpers**: Scripts to interact with external systems
- **Data processing**: Transform, validate, or enrich data
- **CI/CD helpers**: Build, deploy, test automation

## Usage

Scripts should be:
- **Executable**: `chmod +x script.sh`
- **Self-documenting**: Include usage in header comments
- **Idempotent**: Safe to run multiple times
- **Fail-fast**: Exit on errors (`set -euo pipefail` for bash)

## Naming Convention

- Use descriptive names: `fetch-prow-logs.sh`, `validate-rbac.py`
- Include extension: `.sh`, `.py`, `.js`
- Prefix with purpose: `pre-review.sh`, `post-deploy.sh`
