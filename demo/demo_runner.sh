#!/usr/bin/env bash
# Run demo tests. Each demo prints trace output for a particular
# scenario. Exits 0 regardless of test results -- the output IS the demo.

set -u
cd "$(dirname "$0")"

# ANSI colors when stdout is a tty.
if [[ -t 1 ]]; then
	BOLD=$'\e[1m'; BLU=$'\e[34m'; RST=$'\e[0m'
else
	BOLD=''; BLU=''; RST=''
fi

# Strip go test trailer noise and indent the rest for visual grouping.
strip_and_indent() {
	awk -v ind='  ' '
		/^(FAIL|PASS)$/                 { next }
		/^(FAIL|ok)[ \t]+[^[:space:]]/  { next }
		/^=== RUN/                      { next }
		/^--- (PASS|FAIL):/             { next }
		{ printed=1; print ind $0 }
		END { if (!printed) print ind "(no output)" }
	'
}

run_one() {
	local name=$1
	printf '\n%s======== %s ========%s\n\n' "$BOLD" "$name" "$RST"
	printf '%s%sDEMO%s (%s)\n' "$BOLD" "$BLU" "$RST" "$name"
	go test -v -tags demo -count=1 -run "^${name}\$" . 2>&1 | strip_and_indent || true
}

# Collect TestDemo* names from demos_test.go.
suffixes=$(
	grep -hEo '^func TestDemo[A-Za-z0-9_]+' demos_test.go \
		| sed -E 's/^func TestDemo//' \
		| sort -u
)

for s in $suffixes; do
	run_one "TestDemo${s}"
done
