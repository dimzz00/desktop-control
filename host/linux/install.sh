#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

archive=""
dry_run=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --archive) archive="$2"; shift 2 ;;
    --dry-run) dry_run=1; shift ;;
    --url) echo "url mode not implemented in this bootstrap" >&2; exit 2 ;;
    *) echo "unknown flag: $1" >&2; exit 2 ;;
  esac
done

if [[ -z "$archive" ]]; then
  echo "install.sh: --archive <path> is required" >&2
  exit 2
fi
if [[ ! -f "$archive" ]]; then
  echo "install.sh: archive not found: $archive" >&2
  exit 2
fi

if [[ "$dry_run" -eq 0 && "$(id -u)" -ne 0 ]]; then
  echo "install.sh: must run as root (use sudo)" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

if [[ "$dry_run" -eq 1 ]]; then
  echo "[dry-run] extract $archive -> /opt/kendali/releases/<ver>"
  echo "[dry-run] /opt/kendali/current/rc install --config /etc/kendali/rc.yaml"
  exit 0
fi

tar -xzf "$archive" -C "$tmpdir"
version="$(cat "$tmpdir/VERSION" 2>/dev/null || echo "0.1.0")"
release_dir="/opt/kendali/releases/$version"
mkdir -p "$release_dir"
cp -a "$tmpdir"/. "$release_dir"/
ln -sfn "$release_dir" /opt/kendali/current

mkdir -p /etc/kendali /var/log/kendali
/opt/kendali/current/rc install --config /etc/kendali/rc.yaml
echo "install: done ($version)"
