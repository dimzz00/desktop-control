#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-or-later
#
# License gate: every Go source file and shell script in the repository must
# carry the AGPL-3.0-or-later SPDX header at the top of the file. Shell scripts
# keep their shebang on line 1 so they stay directly executable; the SPDX
# header then has to be the very next line.
set -euo pipefail

cd "$(dirname "$0")/.."

expected="SPDX-License-Identifier: AGPL-3.0-or-later"
failed=0

while IFS= read -r -d '' file; do
	first_line="$(head -n 1 "$file")"
	case "$file" in
	*.go)
		want="// ${expected}"
		actual="$first_line"
		;;
	*.sh)
		want="# ${expected}"
		case "$first_line" in
		'#!'*) actual="$(sed -n '2p' "$file")" ;;
		*) actual="$first_line" ;;
		esac
		;;
	*) continue ;;
	esac

	if [ "$actual" != "$want" ]; then
		echo "license: ${file} must contain '${want}' at the top of the file" >&2
		failed=1
	fi
done < <(find . -type f \( -name '*.go' -o -name '*.sh' \) \
	-not -path './.git/*' \
	-not -path './.worktrees/*' \
	-not -path './.superpowers/*' \
	-not -path './.codegraph/*' \
	-print0)

if [ "$failed" -ne 0 ]; then
	echo "license: FAIL — add the AGPL-3.0-or-later SPDX header to the files above" >&2
	exit 1
fi

echo "license: ok"
