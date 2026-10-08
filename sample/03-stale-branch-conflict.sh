#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "$script_dir/lib.sh"

out_dir="${1:-$script_dir/out/03-stale-branch-conflict}"
repo_dir="$out_dir/repo"

init_repo "$repo_dir"
copy_stack_config "$script_dir/configs/03-stale-branch-conflict.git-stack" "$repo_dir"

commit_state "$repo_dir" main 1 "main: commit 1"
commit_state "$repo_dir" main 2 "main: commit 2"
commit_state "$repo_dir" main 3 "main: commit 3"

base_commit="$(git -C "$repo_dir" rev-parse main~2)"
git -C "$repo_dir" branch legacy "$base_commit"

git -C "$repo_dir" checkout legacy >/dev/null
commit_state "$repo_dir" legacy 1 "legacy: commit 1"

git -C "$repo_dir" checkout main >/dev/null
commit_state "$repo_dir" main 4 "main: commit 4"

legacy_tip="$(git -C "$repo_dir" rev-parse legacy)"
git -C "$repo_dir" branch topic "$legacy_tip"
git -C "$repo_dir" checkout topic >/dev/null
commit_state "$repo_dir" topic 1 "topic: commit 1"

if git -C "$repo_dir" merge --no-ff main >/dev/null 2>&1; then
  printf 'expected a conflict, but merge succeeded\n' >&2
  exit 1
fi

printf 'conflict created in %s\n' "$repo_dir"
git -C "$repo_dir" status --short --branch
