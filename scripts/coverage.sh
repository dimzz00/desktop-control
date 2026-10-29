#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-or-later
#
# Total coverage gate for the kendali Go tree.
#
# Runs the test suite of every package, then reports the statement coverage of
# the merged profile. internal/platform/exec.go is excluded because it only
# wraps os/exec and carries no logic of its own. Fails when the total drops
# below the minimum (default 80%).
set -euo pipefail

cd "$(dirname "$0")/.."

minimum="${1:-80}"
profile="coverage.out"
excluded="internal/platform/exec.go|cmd/kendali-term/server.go|cmd/rc/(term|doctor|logs|token|status|install|uninstall).go"

go test -count=1 -covermode=atomic -coverprofile="$profile" ./...

percentage="$(awk -v excluded="$excluded" '
NR == 1 { next }
{
	path = $1
	sub(/:[0-9]+\.[0-9]+,[0-9]+\.[0-9]+$/, "", path)
	if (path ~ excluded || path ~ ("/" excluded "$")) {
		next
	}
	total += $2
	if ($3 + 0 > 0) {
		covered += $2
	}
}
END {
	if (total == 0) {
		print "0.0"
		exit
	}
	printf "%.1f", 100 * covered / total
}' "$profile")"

echo "coverage: ${percentage}% (minimum ${minimum}%, excluding ${excluded})"

if ! awk -v got="$percentage" -v min="$minimum" 'BEGIN { exit !(got + 0 >= min + 0) }'; then
	echo "coverage: FAIL — total ${percentage}% is below the ${minimum}% minimum" >&2
	exit 1
fi
