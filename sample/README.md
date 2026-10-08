# Sample scenarios

This folder contains bash scripts that create small Git repositories with branch and merge shapes used to test stacked-branch behavior.

## Files

- `lib.sh` — shared helpers
- `run-all.sh` — runs every sample and verifies that `.git-stack` exists in each generated repo
- `configs/` — scenario-specific `.git-stack` files copied into each fixture repo
- `01-linear-branches-and-merges.sh` — 3 main commits, branches, and merge commits
- `02-merge-two-updated-branches.sh` — two updated branches merged into an integration branch
- `03-stale-branch-conflict.sh` — stale branch update that ends in a merge conflict
- `04-merge-two-stale-parents.sh` — two stale parents merged on top of a branch update
- `05-rebase-merge-two-parents.sh` — merge-aware rebase of a branch with two parents

## Usage

Run a script directly:

```bash
./sample/01-linear-branches-and-merges.sh
```

Each script writes its repo into a matching directory under `sample/out/`.

Pass a custom output directory as the first argument:

```bash
./sample/03-stale-branch-conflict.sh /tmp/git-stack-sample
```

## State file

Every commit updates `state.txt` with a small payload:

```text
branch=<name>
index=<n>
```

Merges rewrite the file with the joined branch name and reset the index to `1`.
