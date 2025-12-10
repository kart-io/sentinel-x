#!/bin/bash
# Sonic JSON Integration - Verification Script
# This script verifies the sonic integration is working correctly

set -e

echo "================================================"
echo "Sonic JSON Integration Verification"
echo "================================================"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

check_pass() {
    echo -e "${GREEN}✓${NC} $1"
}

check_fail() {
    echo -e "${RED}✗${NC} $1"
    exit 1
}

# 1. Check if json package builds
echo "1. Building JSON package..."
if go build ./pkg/utils/json 2>&1 > /dev/null; then
    check_pass "JSON package builds successfully"
else
    check_fail "JSON package failed to build"
fi

# 2. Run unit tests
echo ""
echo "2. Running unit tests..."
if go test ./pkg/utils/json -v > /tmp/test_output.txt 2>&1; then
    TESTS_PASSED=$(grep -c "PASS:" /tmp/test_output.txt || true)
    check_pass "All unit tests passed ($TESTS_PASSED tests)"
else
    check_fail "Unit tests failed"
fi

# 3. Check HTTP transport integration
echo ""
echo "3. Verifying HTTP transport integration..."
if go build ./pkg/infra/server/transport/http 2>&1 > /dev/null; then
    check_pass "HTTP transport builds with sonic integration"
else
    check_fail "HTTP transport failed to build"
fi

# 4. Run HTTP transport tests
echo ""
echo "4. Running HTTP transport tests..."
if go test ./pkg/infra/server/transport/http > /dev/null 2>&1; then
    check_pass "HTTP transport tests passed"
else
    check_fail "HTTP transport tests failed"
fi

# 5. Check if sonic is being used
echo ""
echo "5. Checking sonic activation..."
if go run example/json/main.go 2>&1 | grep -q "high-performance sonic"; then
    check_pass "Sonic is active and working"
else
    check_fail "Sonic is not active"
fi

# 6. Run quick benchmark
echo ""
echo "6. Running performance benchmark..."
BENCH_OUTPUT=$(go test -bench=BenchmarkRoundTripAPIResponse -benchtime=500ms -run=^$ ./pkg/utils/json 2>&1)

SONIC_TIME=$(echo "$BENCH_OUTPUT" | grep "Sonic-" | awk '{print $3}')
STDLIB_TIME=$(echo "$BENCH_OUTPUT" | grep "Stdlib-" | awk '{print $3}')

if [ ! -z "$SONIC_TIME" ] && [ ! -z "$STDLIB_TIME" ]; then
    check_pass "Benchmark completed"
    echo "   Sonic:  $SONIC_TIME"
    echo "   Stdlib: $STDLIB_TIME"
else
    check_fail "Benchmark failed to run"
fi

# 7. Verify files exist
echo ""
echo "7. Verifying all required files..."
FILES=(
    "pkg/utils/json/json.go"
    "pkg/utils/json/json_test.go"
    "pkg/utils/json/benchmark_test.go"
    "pkg/utils/json/README.md"
    "pkg/utils/json/PERFORMANCE.md"
    "example/json/main.go"
    "SONIC_INTEGRATION.md"
)

for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        check_pass "$file exists"
    else
        check_fail "$file is missing"
    fi
done

echo ""
echo "================================================"
echo -e "${GREEN}All verification checks passed!${NC}"
echo "================================================"
echo ""
echo "The sonic integration is working correctly and ready for deployment."
echo ""
echo "Next steps:"
echo "1. Review documentation in pkg/utils/json/README.md"
echo "2. Review performance report in pkg/utils/json/PERFORMANCE.md"
echo "3. Deploy to staging environment"
echo "4. Monitor metrics for 24-48 hours"
echo "5. Deploy to production"
echo ""
