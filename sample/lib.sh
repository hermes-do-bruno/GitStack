#!/usr/bin/env bash
set -euo pipefail

sample_root() {
  cd "$(dirname "${BASH_SOURCE[0]}")" && pwd
}

init_repo() {
  local repo_dir="$1"

  rm -rf "$repo_dir"
  mkdir -p "$repo_dir"
  git init -b main "$repo_dir" >/dev/null
  git -C "$repo_dir" config user.name "Git Stack Sample"
  git -C "$repo_dir" config user.email "sample@example.com"
}

copy_stack_config() {
  local config_src="$1"
  local repo_dir="$2"

  install -m 0644 "$config_src" "$repo_dir/.git-stack"
}

write_state() {
  local repo_dir="$1"
  local branch_name="$2"
  local index="$3"

  cat >"$repo_dir/state.txt" <<EOF
branch=${branch_name}
index=${index}
EOF
}

commit_state() {
  local repo_dir="$1"
  local branch_name="$2"
  local index="$3"
  local message="${4:-commit ${branch_name} ${index}}"

  write_state "$repo_dir" "$branch_name" "$index"
  git -C "$repo_dir" add state.txt
  git -C "$repo_dir" commit -m "$message" >/dev/null
}

merge_state() {
  local repo_dir="$1"
  local branch_name="$2"
  local index="$3"
  local source_ref="$4"
  local message="${5:-merge ${source_ref} into ${branch_name}}"

  if ! git -C "$repo_dir" merge --no-ff --no-commit "$source_ref" >/dev/null 2>&1; then
    :
  fi

  write_state "$repo_dir" "$branch_name" "$index"
  git -C "$repo_dir" add state.txt
  git -C "$repo_dir" commit -m "$message" >/dev/null
}

show_summary() {
  local repo_dir="$1"
  git -C "$repo_dir" log --graph --decorate --oneline --all -- state.txt
}
