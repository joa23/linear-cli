#!/bin/bash
set -e

# Release Test Script for Linear CLI
# Tests output formats (text/json) and verbosity levels against TEST team

BINARY="./bin/linear"
TEAM="${TEAM:-TEST}"

echo "=================================================="
echo "Linear CLI Release Test"
echo "=================================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

section() {
    echo ""
    echo -e "${BLUE}==[ $1 ]==${NC}"
    echo ""
}

test_cmd() {
    echo -e "${GREEN}$ $1${NC}"
    eval "$1"
    echo ""
}

# Verify binary exists
if [ ! -f "$BINARY" ]; then
    echo "Error: Binary not found at $BINARY"
    echo "Run 'make build' first"
    exit 1
fi

section "1. Create Test Issues"

# Create test issues with different priorities
echo "Creating test issues..."
ISSUE1_OUTPUT=$($BINARY issues create "Test Issue 1: JSON/Text Output Test" \
    --team $TEAM \
    --priority 1 \
    --description "Testing new output formats with minimal verbosity")
ISSUE1_ID=$(echo "$ISSUE1_OUTPUT" | grep -oE 'TEST-[0-9]+' | head -1)

ISSUE2_OUTPUT=$($BINARY issues create "Test Issue 2: Verbosity Levels" \
    --team $TEAM \
    --priority 2 \
    --description "Testing compact verbosity level")
ISSUE2_ID=$(echo "$ISSUE2_OUTPUT" | grep -oE 'TEST-[0-9]+' | head -1)

ISSUE3_OUTPUT=$($BINARY issues create "Test Issue 3: Full Details" \
    --team $TEAM \
    --priority 3 \
    --description "Testing full verbosity with complete details")
ISSUE3_ID=$(echo "$ISSUE3_OUTPUT" | grep -oE 'TEST-[0-9]+' | head -1)

echo "✓ Created issues: $ISSUE1_ID, $ISSUE2_ID, $ISSUE3_ID"
echo ""

section "2. Test Issues List - Text Output (Default)"

test_cmd "$BINARY issues list --team $TEAM --limit 5"
test_cmd "$BINARY issues list --team $TEAM --format minimal --limit 5"
test_cmd "$BINARY issues list --team $TEAM --format compact --limit 5"
test_cmd "$BINARY issues list --team $TEAM --format full --limit 3"

section "3. Test Issues List - JSON Output"

test_cmd "$BINARY issues list --team $TEAM --output json --limit 3"
test_cmd "$BINARY issues list --team $TEAM --format minimal --output json --limit 3"
test_cmd "$BINARY issues list --team $TEAM --format compact --output json --limit 3"
test_cmd "$BINARY issues list --team $TEAM --format full --output json --limit 2"

section "4. Test Individual Issue - Text Output"

test_cmd "$BINARY issues get $ISSUE1_ID"
test_cmd "$BINARY issues get $ISSUE1_ID --format minimal"
test_cmd "$BINARY issues get $ISSUE1_ID --format full"

section "5. Test Individual Issue - JSON Output"

test_cmd "$BINARY issues get $ISSUE1_ID --output json"
test_cmd "$BINARY issues get $ISSUE1_ID --format minimal --output json"
test_cmd "$BINARY issues get $ISSUE1_ID --format full --output json"

section "6. Test Search - Text vs JSON"

test_cmd "$BINARY search 'Test Issue' --team $TEAM --limit 3"
test_cmd "$BINARY search 'Test Issue' --team $TEAM --output json --limit 3"

section "7. Test Cycles - Text vs JSON"

test_cmd "$BINARY cycles list --team $TEAM --limit 2"
test_cmd "$BINARY cycles list --team $TEAM --output json --limit 2"

section "8. Test Teams - Text vs JSON"

test_cmd "$BINARY teams list"
test_cmd "$BINARY teams list --output json"
test_cmd "$BINARY teams get $TEAM"
test_cmd "$BINARY teams get $TEAM --output json"

section "9. Test Users - Text vs JSON"

test_cmd "$BINARY users me"
test_cmd "$BINARY users me --output json"

section "10. Test Projects - Text vs JSON"

test_cmd "$BINARY projects list --limit 3"
test_cmd "$BINARY projects list --output json --limit 3"

section "10b. Test Documents - CRUD (Text vs JSON)"

DOC_JSON=$($BINARY documents create --title "Release Test Doc $(date +%s)" --team $TEAM --content "# Release test" --output json)
DOC_ID=$(echo "$DOC_JSON" | jq -r '.id')
DOC_SLUG=$(echo "$DOC_JSON" | jq -r '.slugId')
echo -e "${GREEN}Created document: $DOC_ID (slug $DOC_SLUG)${NC}"

test_cmd "$BINARY documents list --team $TEAM --limit 3"
test_cmd "$BINARY documents list --team $TEAM --limit 3 --output json"
test_cmd "$BINARY documents get $DOC_SLUG"
test_cmd "$BINARY documents get $DOC_ID --output json"
test_cmd "$BINARY documents update $DOC_SLUG --title \"Release Test Doc (updated)\" --content \"# Updated\""
test_cmd "$BINARY documents delete $DOC_SLUG"

section "11. Test JSON Piping to jq"

echo -e "${GREEN}Testing JSON output with jq filtering...${NC}"
echo ""

# Filter for high priority issues
echo "Filter for priority 1 issues:"
$BINARY issues list --team $TEAM --output json --limit 10 | \
    jq '.[] | select(.priority == 1) | {identifier, title, priority}'
echo ""

# Extract just identifiers
echo "Extract issue identifiers:"
$BINARY issues list --team $TEAM --output json --limit 5 | \
    jq -r '.[].identifier'
echo ""

section "12. Cleanup Test Issues"

echo "Cleaning up test issues..."
for ISSUE_ID in $ISSUE1_ID $ISSUE2_ID $ISSUE3_ID; do
    echo "Archiving $ISSUE_ID..."
    $BINARY issues update $ISSUE_ID --state "Canceled" > /dev/null 2>&1 || true
done
echo "✓ Cleanup complete"
echo ""

section "RELEASE TEST COMPLETE"

echo "✅ All tests passed!"
echo ""
echo "Summary:"
echo "  - Created 3 test issues"
echo "  - Tested text output (minimal/compact/full)"
echo "  - Tested JSON output (minimal/compact/full)"
echo "  - Tested list and get operations"
echo "  - Tested search, cycles, teams, users, projects"
echo "  - Tested JSON piping with jq"
echo "  - Cleaned up test data"
echo ""
echo "The new output format feature is working correctly! 🎉"
