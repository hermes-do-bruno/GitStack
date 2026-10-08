#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
out_root="${1:-$script_dir/out}"

rm -rf "$out_root"
mkdir -p "$out_root"

scripts=(
  "$script_dir/01-linear-branches-and-merges.sh"
  "$script_dir/02-merge-two-updated-branches.sh"
  "$script_dir/03-stale-branch-conflict.sh"
  "$script_dir/04-merge-two-stale-parents.sh"
  "$script_dir/05-rebase-merge-two-parents.sh"
)

for script in "${scripts[@]}"; do
  name="$(basename "$script" .sh)"
  target="$out_root/$name"
  printf '==> %s\n' "$name"
  "$script" "$target" >/dev/null
  test -f "$target/repo/.git-stack"
done

printf 'all sample scenarios created in %s\n' "$out_root"
