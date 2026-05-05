---
name: update-crds
description: Regenerate CRD manifests and deepcopy code after API changes
---

Regenerate Custom Resource Definition (CRD) manifests and associated code after making changes to API types in `api/v1alpha1/`.

## Process

1. **Run code generation**
   ```bash
   make generate
   ```
   This regenerates:
   - Kubernetes deepcopy methods
   - OpenAPI schemas
   - Mock interfaces for testing

2. **Update manifests**
   ```bash
   make manifests
   ```
   This regenerates CRD YAML files in `config/crd/bases/` based on API type definitions.

3. **Verify changes**
   ```bash
   git status
   ```
   Show what files were modified by the code generation.

## When to use

Run this skill whenever you:
- Modify API types in `api/v1alpha1/*.go`
- Add new fields to custom resources
- Change validation rules or field types
- Add or modify kubebuilder markers

## Output

Report:
- ✅ Code generation successful
- ✅ Manifest generation successful
- List of modified files

If either step fails, report the error details.

Additional context: $ARGUMENTS
