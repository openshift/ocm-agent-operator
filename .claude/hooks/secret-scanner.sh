#!/usr/bin/env bash
#
# Secret Scanner Hook for OCM Agent Operator
# Inspired by: Railgun, aitmpl.com security hooks, claude-code-redaction-hooks
#
# Scans file content for secrets before Write/Edit operations
# Blocks operations containing secrets with specific remediation guidance
#

set -euo pipefail

FILE_PATH="${1:-}"
CONTENT="${2:-}"
NEW_STRING="${3:-}"

# Combine all content to scan
SCAN_TEXT="${CONTENT}${NEW_STRING}"

# Exit if no content to scan
if [[ -z "$SCAN_TEXT" ]]; then
  exit 0
fi

# =============================================================================
# SECRET PATTERNS
# =============================================================================

declare -A SECRET_PATTERNS=(
  # AWS Credentials
  ["aws-access-key"]='(A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}'
  ["aws-secret-key"]='aws_secret_access_key[[:space:]]*[:=][[:space:]]*["\047]?[A-Za-z0-9/+=]{40}["\047]?'

  # GitHub Tokens
  ["github-token"]='(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{36,}'
  ["github-fine-grained"]='github_pat_[A-Za-z0-9_]{82}'

  # Generic API Keys
  ["api-key"]='api[_-]?key[[:space:]]*[:=][[:space:]]*["\047][A-Za-z0-9_\-]{16,}["\047]'
  ["secret-key"]='secret[_-]?key[[:space:]]*[:=][[:space:]]*["\047][A-Za-z0-9_\-]{16,}["\047]'

  # Private Keys (using literal spaces instead of \s)
  ["private-key-pem"]='-----BEGIN[[:space:]]+(RSA[[:space:]]+)?PRIVATE KEY-----'
  ["ssh-private-key"]='-----BEGIN[[:space:]]+OPENSSH PRIVATE KEY-----'

  # Passwords
  ["password-assignment"]='password[[:space:]]*[:=][[:space:]]*["\047][^"\047]{8,}["\047]'

  # Database Connection Strings
  ["mongodb-uri"]='mongodb(\+srv)?://[^:]+:[^@]+@'
  ["postgres-uri"]='postgres(ql)?://[^:]+:[^@]+@'
  ["mysql-uri"]='mysql://[^:]+:[^@]+@'

  # OpenShift/Kubernetes
  ["kubeconfig-token"]='token:[[:space:]]*[A-Za-z0-9_\-\.]{20,}'
  ["openshift-pull-secret"]='(\.dockerconfigjson|auths).*[A-Za-z0-9+/]{30,}={0,2}'

  # OCM Specific
  ["ocm-token"]='ocm[_-]?token[[:space:]]*[:=][[:space:]]*["\047]?[A-Za-z0-9_\-\.]{20,}["\047]?'
  ["ocm-refresh-token"]='refresh[_-]?token[[:space:]]*[:=][[:space:]]*["\047]?[A-Za-z0-9_\-\.]{20,}["\047]?'

  # High Entropy Strings (potential secrets)
  ["high-entropy"]='["\047][A-Za-z0-9+/=]{64,}["\047]'
)

# =============================================================================
# ALLOWLIST PATTERNS
# =============================================================================

# These patterns indicate false positives or test data
ALLOWLIST_PATTERNS=(
  'example'
  'test[_-]?(secret|token|key|password)'
  'fake[_-]?(secret|token|key|password)'
  'dummy[_-]?(secret|token|key|password)'
  'placeholder'
  'AKIAIOSFODNN7EXAMPLE'  # AWS documentation example
  'your[_-]?(secret|token|key|password)[_-]?here'
  'TODO:'
  'FIXME:'
  '\$\{.*\}'  # Variable substitution
  '\{\{.*\}\}'  # Template variables
)

# =============================================================================
# FILE PATH EXCEPTIONS
# =============================================================================

# Skip scanning for these file patterns
if [[ "$FILE_PATH" =~ ^(test/|.*_test\.go|.*\.md|.*/fixtures/) ]]; then
  exit 0
fi

# =============================================================================
# ALLOWLIST CHECK
# =============================================================================

is_allowlisted() {
  local text="$1"
  for pattern in "${ALLOWLIST_PATTERNS[@]}"; do
    if echo "$text" | grep -qiE "$pattern"; then
      return 0
    fi
  done
  return 1
}

# =============================================================================
# SECRET SCANNING
# =============================================================================

SECRETS_FOUND=()

for secret_type in "${!SECRET_PATTERNS[@]}"; do
  pattern="${SECRET_PATTERNS[$secret_type]}"

  # Search for pattern (suppress grep errors for complex patterns)
  if echo "$SCAN_TEXT" | grep -qE "$pattern" 2>/dev/null; then
    # Extract matches
    matches=$(echo "$SCAN_TEXT" | grep -oE "$pattern" 2>/dev/null | head -5)

    # Check each match against allowlist
    while IFS= read -r match; do
      if ! is_allowlisted "$match"; then
        SECRETS_FOUND+=("$secret_type: $match")
      fi
    done <<< "$matches"
  fi
done

# =============================================================================
# BLOCK IF SECRETS FOUND
# =============================================================================

if [ ${#SECRETS_FOUND[@]} -gt 0 ]; then
  echo "❌ BLOCKED: Secrets detected in file: $FILE_PATH"
  echo ""
  echo "Found ${#SECRETS_FOUND[@]} potential secret(s):"
  echo ""

  for secret in "${SECRETS_FOUND[@]}"; do
    echo "  - $secret"
  done

  echo ""
  echo "🔐 REMEDIATION:"
  echo ""
  echo "1. Use environment variables:"
  echo "   token := os.Getenv(\"OCM_TOKEN\")"
  echo ""
  echo "2. Use Kubernetes Secrets:"
  echo "   secretRef:"
  echo "     name: ocm-agent-secret"
  echo "     key: token"
  echo ""
  echo "3. If this is test data, add to allowlist:"
  echo "   - Use 'test_', 'fake_', or 'example_' prefix"
  echo "   - Or move to test/fixtures/"
  echo ""
  echo "4. If this is a false positive:"
  echo "   - Add pattern to .gitleaks.toml allowlist"
  echo "   - Document why it's safe"
  echo ""
  echo "❌ NEVER commit real secrets to git"
  echo ""

  exit 2  # Exit code 2 blocks the operation
fi

# All checks passed
exit 0
