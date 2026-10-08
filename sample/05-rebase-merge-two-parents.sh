#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "$script_dir/lib.sh"

out_dir="${1:-$script_dir/out/05-rebase-merge-two-parents}"
repo_dir="$out_dir/repo"

init_repo "$repo_dir"
copy_stack_config "$script_dir/configs/05-rebase-merge-two-parents.git-stack" "$repo_dir"

commit_state "$repo_dir" main 1 "main: commit 1"
commit_state "$repo_dir" main 2 "main: commit 2"
commit_state "$repo_dir" main 3 "main: commit 3"

base_commit="$(git -C "$repo_dir" rev-parse main~1)"
git -C "$repo_dir" checkout -b left "$base_commit" >/dev/null
git -C "$repo_dir" checkout -b right "$base_commit" >/dev/null

git -C "$repo_dir" checkout left >/dev/null
commit_state "$repo_dir" left 1 "left: commit 1"
commit_state "$repo_dir" left 2 "left: commit 2"

git -C "$repo_dir" checkout right >/dev/null
commit_state "$repo_dir" right 1 "right: commit 1"
commit_state "$repo_dir" right 2 "right: commit 2"

git -C "$repo_dir" checkout -b integration left >/dev/null
merge_state "$repo_dir" "integration+right" 1 right "merge right into integration"

git -C "$repo_dir" checkout main >/dev/null
commit_state "$repo_dir" main 4 "main: commit 4"

git -C "$repo_dir" checkout integration >/dev/null
if git -C "$repo_dir" rebase --rebase-merges --onto main "$base_commit" >/tmp/git-stack-sample-rebase.log 2>&1; then
  cat /tmp/git-stack-sample-rebase.log
  show_summary "$repo_dir"
  printf '\nscenario created at: %s\n' "$repo_dir"
  exit 0
fi

cat /tmp/git-stack-sample-rebase.log
printf '\nrebase left the repo in progress\n' >&2
git -C "$repo_dir" status --short --branch
