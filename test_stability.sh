#!/bin/bash

# Test Stability Script
# Runs tests 10 times with cache clearing to ensure stability

echo "🧪 Starting test stability check..."
echo "This will run tests 10 times with cache clearing"
echo "================================================"

PASSED=0
FAILED=0

for i in {1..10}; do
    echo ""
    echo "🔄 Run $i/10"
    echo "Clearing test cache..."
    go clean -testcache
    
    echo "Running integration tests..."
    if task test-integration > /dev/null 2>&1; then
        echo "✅ Run $i: PASSED"
        ((PASSED++))
    else
        echo "❌ Run $i: FAILED"
        ((FAILED++))
        echo "Full output for failed run:"
        task test-integration
    fi
done

echo ""
echo "================================================"
echo "📊 Results Summary:"
echo "✅ Passed: $PASSED/10"
echo "❌ Failed: $FAILED/10"

if [ $FAILED -eq 0 ]; then
    echo "🎉 All tests passed! System is stable."
    exit 0
else
    echo "⚠️  Some tests failed. System needs attention."
    exit 1
fi
