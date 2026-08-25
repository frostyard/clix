#!/bin/sh
# Exercise scripts/check-coverage.sh against fixture coverage profiles.
#
# Usage: scripts/test-coverage-check.sh
#
# Builds synthetic Go coverage profiles in a temporary directory -- one
# whose total statement coverage is above a floor, one below it, and one
# whose aggregate is above the floor but leaves a function entirely
# uncovered -- and asserts check-coverage.sh's exit code and diagnostics
# for each. Exits 0 when every assertion holds.
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CHECK="$SCRIPT_DIR/check-coverage.sh"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# `go tool cover -func` reads absolute file paths from the profile directly, so
# a plain Go source file outside any module is enough to anchor the blocks.
SRC="$TMP/fixture.go"
cat > "$SRC" <<'GO'
package fixture

func covered() int {
	a := 1
	b := 2
	c := 3
	d := 4
	e := 5
	f := 6
	g := 7
	h := 8
	i := 9
	return a + b + c + d + e + f + g + h + i
}
GO

# Ten statements inside covered(): lines 4-13.
# Above an 80.0 floor: 9 of 10 statements executed (90.0%).
cat > "$TMP/above.out" <<EOF_PROFILE
mode: atomic
$SRC:3.20,13.2 9 1
$SRC:13.2,14.2 1 0
EOF_PROFILE

# Below an 80.0 floor: 5 of 10 statements executed (50.0%).
cat > "$TMP/below.out" <<EOF_PROFILE
mode: atomic
$SRC:3.20,8.2 5 1
$SRC:8.2,14.2 5 0
EOF_PROFILE

# A second fixture, kept separate from SRC so `go tool cover -func` does not
# also attribute its unreferenced uncovered() to the above/below profiles:
# covered() is fully executed (10/10) but uncovered() is entirely
# unexecuted, so the aggregate is 10/11 (90.9%) -- above an 80.0 floor --
# while uncovered() itself reports 0.0% function coverage.
ZERO_FUNC_SRC="$TMP/zero_func_fixture.go"
cat > "$ZERO_FUNC_SRC" <<'GO'
package fixture

func covered() int {
	a := 1
	b := 2
	c := 3
	d := 4
	e := 5
	f := 6
	g := 7
	h := 8
	i := 9
	return a + b + c + d + e + f + g + h + i
}

func uncovered() int {
	return 0
}
GO

cat > "$TMP/zero-func.out" <<EOF_PROFILE
mode: atomic
$ZERO_FUNC_SRC:3.20,14.1 10 1
$ZERO_FUNC_SRC:16.22,18.1 1 0
EOF_PROFILE

failures=0

echo "--- expect exit 0: coverage above floor"
if "$CHECK" "$TMP/above.out" 80.0; then
    echo "PASS: above-floor profile accepted"
else
    echo "FAIL: above-floor profile rejected (exit $?)"
    failures=$((failures + 1))
fi

echo "--- expect exit 1: coverage below floor"
if "$CHECK" "$TMP/below.out" 80.0; then
    echo "FAIL: below-floor profile accepted"
    failures=$((failures + 1))
else
    status=$?
    if [ "$status" -eq 1 ]; then
        echo "PASS: below-floor profile rejected with exit 1"
    else
        echo "FAIL: below-floor profile rejected with unexpected exit $status"
        failures=$((failures + 1))
    fi
fi

echo "--- expect exit 0: COVERAGE_MIN env override lowers the floor"
if COVERAGE_MIN=40 "$CHECK" "$TMP/below.out"; then
    echo "PASS: COVERAGE_MIN override honoured"
else
    echo "FAIL: COVERAGE_MIN override ignored (exit $?)"
    failures=$((failures + 1))
fi

echo "--- expect exit 1: an explicit floor argument outranks a lower COVERAGE_MIN"
if COVERAGE_MIN=40 "$CHECK" "$TMP/below.out" 80.0; then
    echo "FAIL: explicit 80.0 floor relaxed by COVERAGE_MIN=40 (below-floor profile accepted)"
    failures=$((failures + 1))
else
    status=$?
    if [ "$status" -eq 1 ]; then
        echo "PASS: explicit floor argument took precedence over COVERAGE_MIN"
    else
        echo "FAIL: explicit floor rejected below-floor profile with unexpected exit $status"
        failures=$((failures + 1))
    fi
fi

echo "--- expect exit 1: aggregate above floor but a function has 0.0% coverage"
if "$CHECK" "$TMP/zero-func.out" 80.0 2>"$TMP/zero-func.err"; then
    echo "FAIL: profile with an uncovered function accepted despite the aggregate floor being met"
    failures=$((failures + 1))
else
    status=$?
    if [ "$status" -eq 1 ]; then
        if grep -q "uncovered" "$TMP/zero-func.err"; then
            echo "PASS: profile with an uncovered function rejected with exit 1 naming 'uncovered'"
        else
            echo "FAIL: check-coverage rejected the profile but did not name the uncovered function"
            failures=$((failures + 1))
        fi
    else
        echo "FAIL: profile with an uncovered function rejected with unexpected exit $status"
        failures=$((failures + 1))
    fi
fi

if [ "$failures" -ne 0 ]; then
    echo "test-coverage-check: $failures assertion(s) failed" >&2
    exit 1
fi
echo "test-coverage-check: all assertions passed"
