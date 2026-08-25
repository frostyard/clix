#!/bin/sh
# Enforce a total statement-coverage floor on a Go coverage profile.
# Ported from frostyard/updex (scripts/check-coverage.sh); clix floors the
# library package at 95.0% (`make coverage-check`).
#
# Usage: scripts/check-coverage.sh [profile] [minimum-percent]
#
#   profile          coverage profile written by `go test -coverprofile`
#                    (default: coverage.out)
#   minimum-percent  required total statement coverage
#                    (default: $COVERAGE_MIN, or 95.0 when unset)
#
# The script runs `go tool cover -func=<profile>`, reads the final `total:`
# line, prints the observed and required percentages, and exits 1 when the
# observed value is below the floor. It also inspects every non-total
# function row and fails, naming each one, when any function has 0.0%
# statement coverage -- an aggregate floor above the required percentage
# does not by itself prove every production function was exercised. Any
# other failure (missing profile, unparsable output) also exits non-zero.
set -eu

PROFILE="${1:-coverage.out}"
MIN="${2:-${COVERAGE_MIN:-95.0}}"

if [ ! -f "$PROFILE" ]; then
    echo "check-coverage: profile not found: $PROFILE" >&2
    exit 2
fi

case "$MIN" in
    ''|*[!0-9.]*|.|*.*.*)
        echo "check-coverage: invalid minimum percentage: '$MIN'" >&2
        exit 2
        ;;
esac

FUNC_OUTPUT="$(go tool cover -func="$PROFILE")"

TOTAL_LINE="$(printf '%s\n' "$FUNC_OUTPUT" | awk '$1 == "total:" { line = $0 } END { print line }')"
if [ -z "$TOTAL_LINE" ]; then
    echo "check-coverage: no 'total:' line in 'go tool cover -func=$PROFILE' output" >&2
    exit 2
fi

OBSERVED="$(printf '%s\n' "$TOTAL_LINE" | awk '{ sub(/%$/, "", $NF); print $NF }')"
case "$OBSERVED" in
    ''|*[!0-9.]*)
        echo "check-coverage: could not parse coverage from: $TOTAL_LINE" >&2
        exit 2
        ;;
esac

echo "check-coverage: observed total statement coverage ${OBSERVED}% (required >= ${MIN}%)"

# Compute the comparison through command substitution so an awk failure aborts
# under `set -e` instead of falling through to OK.
BELOW="$(awk -v observed="$OBSERVED" -v required="$MIN" 'BEGIN { print (observed + 0 < required + 0) ? "yes" : "no" }')"
case "$BELOW" in
    yes)
        echo "check-coverage: FAIL: total coverage ${OBSERVED}% is below the required floor of ${MIN}%" >&2
        exit 1
        ;;
    no) ;;
    *)
        echo "check-coverage: could not compare ${OBSERVED} with ${MIN}" >&2
        exit 2
        ;;
esac

UNCOVERED_FUNCS="$(printf '%s\n' "$FUNC_OUTPUT" | awk '$1 != "total:" && $NF == "0.0%" { print }')"
if [ -n "$UNCOVERED_FUNCS" ]; then
    echo "check-coverage: FAIL: the following function(s) have 0.0% statement coverage:" >&2
    printf '%s\n' "$UNCOVERED_FUNCS" | while IFS= read -r line; do
        echo "  $line" >&2
    done
    exit 1
fi

echo "check-coverage: OK"
