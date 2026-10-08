#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "$script_dir/lib.sh"

out_dir="${1:-$script_dir/out/01-linear-branches-and-merges}"
repo_dir="$out_dir/repo"

init_repo "$repo_dir"
copy_stack_config "$script_dir/configs/01-linear-branches-and-merges.git-stack" "$repo_dir"

commit_state "$repo_dir" main 1 "main: commit 1"
commit_state "$repo_dir" main 2 "main: commit 2"
commit_state "$repo_dir" main 3 "main: commit 3"

base_commit="$(git -C "$repo_dir" rev-parse main~1)"
git -C "$repo_dir" checkout -b feature-a "$base_commit" >/dev/null
commit_state "$repo_dir" feature-a 1 "feature-a: commit 1"
commit_state "$repo_dir" feature-a 2 "feature-a: commit 2"

git -C "$repo_dir" branch feature-b "$base_commit"
git -C "$repo_dir" checkout feature-b >/dev/null
commit_state "$repo_dir" feature-b 1 "feature-b: commit 1"

git -C "$repo_dir" checkout main >/dev/null
merge_state "$repo_dir" "main+feature-a" 1 feature-a "merge feature-a into main"
merge_state "$repo_dir" "main+feature-a+feature-b" 1 feature-b "merge feature-b into main"

show_summary "$repo_dir"
printf '\nscenario created at: %s\n' "$repo_dir"
