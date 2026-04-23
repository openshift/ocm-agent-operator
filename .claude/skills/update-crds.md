---
name: update-crds
description: Update CRDs after modifying API types
---

# Update CRDs

This skill regenerates Kubernetes manifests and CRDs after modifying API types.

## Usage

```
/update-crds
```

## When to Use

Run this skill whenever you:
- Modify types in `api/v*/`
- Add/remove fields to Custom Resources
- Change kubebuilder markers/comments
- Update validation rules
- Change RBAC permissions (kubebuilder:rbac comments)

## What Gets Updated

### 1. Generate Manifests
```bash
make manifests
```

Updates:
- `config/crd/bases/*.yaml` - Custom Resource Definitions
- `config/rbac/role.yaml` - RBAC permissions
- `config/webhook/*.yaml` - Webhook configurations (if webhooks exist)

### 2. Generate Code
```bash
make generate
```

Updates:
- `zz_generated.deepcopy.go` - DeepCopy methods
- Client code generation
- Runtime.Object implementations

## Typical Workflow

### 1. Modify API Types
```go
// api/v1alpha1/myresource_types.go

type MyResourceSpec struct {
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinLength=1
    Name string `json:"name"`
    
    // NEW FIELD
    // +kubebuilder:validation:Optional
    // +kubebuilder:default="default-value"
    NewField string `json:"newField,omitempty"`
}
```

### 2. Update Kubebuilder Markers
Common markers:
```go
// Validation
// +kubebuilder:validation:Required
// +kubebuilder:validation:Optional
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:Maximum=100
// +kubebuilder:validation:Pattern="^[a-z0-9-]+$"
// +kubebuilder:validation:Enum=value1;value2;value3

// Defaults
// +kubebuilder:default="default-value"
// +kubebuilder:default=true

// CRD-level markers (on type)
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`

// RBAC markers (on Reconciler)
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;create;update
```

### 3. Run Update
```bash
/update-crds
```

### 4. Verify Changes
```bash
git diff config/
```

Review:
- CRD schema changes in `config/crd/bases/`
- RBAC changes in `config/rbac/role.yaml`
- Generated code in `zz_generated.deepcopy.go`

### 5. Test Changes
```bash
# Deploy CRDs to test cluster
make install

# Create test resource
kubectl apply -f config/samples/

# Verify validation works
kubectl describe <resource-name>
```

## Common Scenarios

### Adding a New Field

1. Add field to type with kubebuilder markers:
```go
// +kubebuilder:validation:Optional
NewField string `json:"newField,omitempty"`
```

2. Run `/update-crds`
3. Check CRD updated: `git diff config/crd/bases/`

### Adding RBAC Permissions

1. Add marker to reconciler:
```go
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;create;update
func (r *MyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
```

2. Run `/update-crds`
3. Check role updated: `git diff config/rbac/role.yaml`

### Changing Validation Rules

1. Update markers:
```go
// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
Name string `json:"name"`
```

2. Run `/update-crds`
3. Test validation works:
```bash
make install
kubectl apply -f - <<EOF
apiVersion: v1alpha1
kind: MyResource
metadata:
  name: INVALID-NAME  # Should fail validation
spec:
  name: "invalid name"
EOF
```

### Adding Status Subresource

1. Ensure marker on type:
```go
// +kubebuilder:subresource:status
type MyResource struct {
```

2. Run `/update-crds`
3. Verify subresource in CRD:
```bash
grep -A 5 "subresources:" config/crd/bases/*.yaml
```

## Troubleshooting

### "make manifests" fails

Check:
- Go modules are up to date: `go mod tidy`
- controller-gen is installed: `make ensure`
- Syntax errors in API types

### CRD not updated

Ensure:
- Kubebuilder markers are on correct lines (directly above field/type)
- Markers start with `// +kubebuilder:` (note the `+`)
- No syntax errors in markers

### RBAC not updated

Check:
- Markers are on `Reconcile()` function or reconciler struct
- Correct format: `// +kubebuilder:rbac:groups=...,resources=...,verbs=...`
- Commas separate key=value pairs, semicolons separate list items

### Generated code has errors

Run:
```bash
make generate
go fmt ./...
```

## Files Modified

After running this skill, expect changes in:

```
config/
├── crd/
│   └── bases/
│       └── *.yaml          # CRD definitions
├── rbac/
│   └── role.yaml           # RBAC permissions
└── webhook/
    └── *.yaml              # Webhook configs (if applicable)

api/
└── v1alpha1/
    └── zz_generated.deepcopy.go  # Generated code
```

## Best Practices

1. **Always run after API changes** - Don't manually edit generated files
2. **Review diffs carefully** - Ensure only expected changes
3. **Test validation** - Apply sample resources to verify rules work
4. **Version CRDs properly** - Use v1alpha1, v1beta1, v1 appropriately
5. **Document fields** - Add godoc comments for user-facing documentation

## Integration with CI

This command runs automatically in CI:
- Konflux pipelines check manifests are up-to-date
- PR checks fail if you forget to run this

**Tip**: Add to pre-commit hook to never forget!

## Quick Reference

```bash
# Full update
make manifests && make generate

# Install CRDs to cluster
make install

# Uninstall CRDs from cluster
make uninstall

# View CRD schema
kubectl get crd <crd-name> -o yaml

# Check CRD validation
kubectl explain <resource-kind>.spec
```
