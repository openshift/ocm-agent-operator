#!/usr/bin/env bash
# Local testing script for FullSend agent implementation
#
# This script validates the FullSend configuration and tests components
# that can run locally without the full FullSend runtime.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

echo "🧪 FullSend Local Testing"
echo "========================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

pass() { echo -e "${GREEN}✓${NC} $*"; }
fail() { echo -e "${RED}✗${NC} $*"; }
warn() { echo -e "${YELLOW}⚠${NC} $*"; }

ERRORS=0

# Test 1: Validate YAML syntax
echo "1️⃣  Validating YAML syntax..."
for file in .fullsend/config.yaml .fullsend/harness/*.yaml .fullsend/policies/*.yaml; do
    if [ -f "$file" ]; then
        if python3 -c "import yaml; yaml.safe_load(open('$file'))" 2>/dev/null; then
            pass "$file"
        else
            fail "$file - invalid YAML"
            ERRORS=$((ERRORS + 1))
        fi
    fi
done
echo ""

# Test 2: Validate config structure
echo "2️⃣  Validating config.yaml structure..."
if python3 -c "
import yaml, sys
config = yaml.safe_load(open('.fullsend/config.yaml'))
required = ['version', 'roles', 'agents']
for key in required:
    if key not in config:
        print(f'Missing required key: {key}')
        sys.exit(1)
if 'review' not in [a['name'] for a in config.get('agents', [])]:
    print('Review agent not registered')
    sys.exit(1)
print('Config structure valid')
" 2>/dev/null; then
    pass "Config structure valid"
else
    fail "Config structure invalid"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# Test 3: Verify skills directory
echo "3️⃣  Verifying skills directory structure..."
if [ -d ".fullsend/skills" ]; then
    pass ".fullsend/skills/ exists"
else
    fail ".fullsend/skills/ missing"
    ERRORS=$((ERRORS + 1))
fi

if [ -L ".claude/skills" ]; then
    target=$(readlink .claude/skills)
    if [ "$target" = "../.fullsend/skills" ]; then
        pass ".claude/skills symlink correct"
    else
        fail ".claude/skills symlink points to: $target (expected: ../.fullsend/skills)"
        ERRORS=$((ERRORS + 1))
    fi
else
    fail ".claude/skills symlink missing"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# Test 4: Verify skill files
echo "4️⃣  Verifying skill files..."
for skill in coderabbit-review prow-ci; do
    if [ -f ".fullsend/skills/$skill/SKILL.md" ]; then
        pass "$skill/SKILL.md exists"
    else
        fail "$skill/SKILL.md missing"
        ERRORS=$((ERRORS + 1))
    fi
done
echo ""

# Test 5: Test CodeRabbit script
echo "5️⃣  Testing CodeRabbit skill script..."
script=".fullsend/skills/coderabbit-review/scripts/run-coderabbit.sh"
if [ -x "$script" ]; then
    pass "$script is executable"
else
    fail "$script not executable"
    ERRORS=$((ERRORS + 1))
fi

# Check if gh is authenticated
if gh auth status >/dev/null 2>&1; then
    pass "GitHub CLI authenticated"

    # Test against a real PR (optional - requires PR number)
    if [ "${1:-}" != "" ]; then
        PR_NUMBER="$1"
        echo ""
        echo "   Testing against PR #$PR_NUMBER..."
        if OUTPUT=$("$script" "$PR_NUMBER" 2>&1); then
            FINDING_COUNT=$(echo "$OUTPUT" | jq '. | length' 2>/dev/null || echo "0")
            pass "Script executed successfully ($FINDING_COUNT findings)"
            if [ "${VERBOSE:-}" = "1" ]; then
                echo "$OUTPUT" | jq '.' 2>/dev/null || echo "$OUTPUT"
            fi
        else
            fail "Script failed: $OUTPUT"
            ERRORS=$((ERRORS + 1))
        fi
    fi
else
    warn "GitHub CLI not authenticated (run: gh auth login)"
    echo "   Cannot test CodeRabbit script without authentication"
fi
echo ""

# Test 6: Verify harness composition
echo "6️⃣  Verifying review harness..."
if [ -f ".fullsend/harness/review.yaml" ]; then
    pass "review.yaml exists"

    # Check for base: field
    if grep -q "^base:" .fullsend/harness/review.yaml; then
        pass "Uses base: composition"
    else
        fail "Missing base: composition"
        ERRORS=$((ERRORS + 1))
    fi

    # Check for skills: field
    if grep -q "^skills:" .fullsend/harness/review.yaml; then
        pass "Defines skills"
    else
        fail "Missing skills definition"
        ERRORS=$((ERRORS + 1))
    fi

    # Check SHA256 integrity
    if grep -q "#sha256=" .fullsend/harness/review.yaml; then
        pass "Has SHA256 integrity check"
    else
        warn "Missing SHA256 integrity check"
    fi
else
    fail "review.yaml missing"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# Test 7: Verify directory structure
echo "7️⃣  Verifying .fullsend directory structure..."
for dir in harness policies skills knowledge scripts docs; do
    if [ -d ".fullsend/$dir" ]; then
        pass ".fullsend/$dir/ exists"
    else
        warn ".fullsend/$dir/ missing (may be optional)"
    fi
done
echo ""

# Summary
echo "========================="
if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Commit changes: git commit -m 'feat: implement FullSend Pattern B with CodeRabbit'"
    echo "  2. Push to GitHub: git push"
    echo "  3. Test on a PR: comment '/fs-review' on a pull request"
    exit 0
else
    echo -e "${RED}✗ $ERRORS test(s) failed${NC}"
    exit 1
fi
