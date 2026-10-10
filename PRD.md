# Git Stack PRD

## Problem Statement

Users who work with stacked branches need a tool to keep dependent branches aligned after rebasing a parent branch. Manual rebasing is slow, error-prone, and gets worse when merge commits are involved. The current Git workflow also makes it hard to see whether a branch is synchronized with the structure it is supposed to follow.

## Solution

Build a CLI tool that reads a root-level `.git-stack` file, shows the declared branch cascade and synchronization state, and performs top-down cascade from the current branch through its declared descendants. The tool must handle branches with merge commits, use the first parent as the mainline, and render a graph that shows declared branches and whether each one is synchronized.

## Source of Truth

The repository root contains a `.git-stack` file.

Format:

```text
branch1=master
branch2=branch1
branch3=branch1
branch4=branch2,branch3
```

Rules:
- One line per branch.
- Blank lines are allowed.
- Lines starting with `#` are comments.
- Left side is the branch name.
- Right side is a comma-separated list of parent branches.
- The file defines intent.
- If the current Git history diverges from the declared structure, `cascade` corrects it by rebasing.
- For branches with multiple parents, the first parent is the mainline.

## User Stories

1. As a developer, I want to declare branch dependencies in a simple file, so that the cascade is explicit and reproducible.
2. As a developer, I want the tool to treat the current branch as the sync starting point, so that I can update the subtree I am working on.
3. As a developer, I want to run sync from the main branch to update everything, so that I can propagate a base change through the whole cascade.
4. As a developer, I want the tool to preserve merge branches during rebasing, so that integration points are not lost.
5. As a developer, I want to inspect branch state before running sync, so that I can decide when to apply changes.
6. As a developer, I want a graph view that shows declared branches and their sync state, so that I can understand the cascade quickly.
7. As a developer, I want top-down propagation, so that parents are updated before children.
8. As a developer, I want the tool to skip branches already merged, so that it does not waste time rebasing branches that are already aligned.
9. As a developer, I want sync to stop on conflicts and leave the repository in standard Git rebase state, so that I can resolve conflicts normally.
10. As a developer, I want the file order to define child processing order, so that the intended sequence is deterministic.

## Branch States

The `graph` command shows one of four states for each branch:

- `Merged`: the parent and child point to the same commit.
- `Sync`: the parent's commits are the commits immediately before the child's tip commit.
- `Partial-sync`: the child is ahead in a way that indicates the parent has more than one commit and the declared relationship is mostly aligned.
- `Unsync`: the child's parent commit is not present in any declared parent relationship.

State calculation rules:
- The tool computes the state from the declared relationships and the current Git history.
- If a branch has multiple parents, the most conservative state wins.
- `Merged` only applies when the declared branch and its parent are fully aligned at the same tip.
- `Unsync` overrides other states when any declared parent relationship fails.


## Implementation Decisions

- Implement the CLI in Go.
- Use Git as the source of truth for repository state and rebasing behavior.
- Parse `.git-stack` as a simple key-value file with comma-separated values.
- Use ancestry and declared relationships to derive branch state.
- Keep graph rendering separate from sync execution.
- Prefer a terminal-friendly graph output first.
- Support merge-aware rebasing using Git's native merge-preserving behavior where possible.
- Keep the execution model conservative: dry-run first, explicit confirmation for history rewriting.
- Treat the current branch as the default sync root.

## Testing Decisions

- Test branch parsing independently from Git execution.
- Test graph output against known fixture repositories.
- Test `Sync`, `Partial-sync`, `Unsync`, and `Merged` state calculation with real commit graphs.
- Test top-down propagation order.
- Test merge-commit cases with two or more parents.
- Test dry-run output and conflict-stop behavior.
- Test that `graph` does not mutate history.
- Test that `cascade` only rewrites branches reachable from the current branch subtree.

## Out of Scope

- Web UI.
- GUI branch editor.
- Automatic conflict resolution.
- PR provider integrations.
- Server-side synchronization.
- Background daemons.
- Full history rewriting across unrelated repositories.

## Further Notes

The project should stay explicit and deterministic. The `.git-stack` file defines intent, the graph shows reality, and `cascade` reconciles the two from the current branch downward.

## CLI Surface

Proposed commands:

- `git-stack parent [name]` — create or update the current branch's parent entry in `.git-stack` using the current branch name and one or more comma-separated parent names.
- `git-stack graph [columns]` — show branch states and cascade structure, with optional column filtering.
- `git-stack cascade [--apply|--script]` — dry-run the cascade starting from the current branch or print a shell script for it.

Behavior:
- `graph` is read-only.
- `graph` uses ASCII output in the terminal and can colorize statuses when the terminal supports it.
- `graph` uses state labels plus colors.
- `graph` shows every declared parent relation for merge branches.
- `graph` orders output topologically.
- `graph` is rooted at the current branch subtree and does not add a special current-branch indicator.
- `graph` renders each line like `ab12cd3 [Sync] branch-name Commit title`.
- `graph` includes the short commit hash, the state, the branch name, and the commit subject.
- `cascade` starts from the current branch.
- `cascade` traverses descendants top-down.
- `cascade` skips branches in `Merged` state.
- `cascade` stops on the first conflict and leaves Git in rebase state.
- `cascade` is dry-run by default; execution requires an explicit `--apply` flag.
- `cascade --apply` uses the same line style as `graph`.
- `cascade --apply` replaces the commit subject field with the planned action for that branch.
- `cascade --apply` renders the action as `rebase onto <target>` for rebase steps and `skip merged` for merged branches.
- Dry-run output shows one line per planned branch action with the target base.
- The declaration order in `.git-stack` defines child ordering.
- `parent [name]` updates the root `.git-stack` file rather than inferring config from hidden state.
- `parent [name]` preserves existing comments and blank lines when rewriting the file.
- `parent [name]` replaces the existing parent list for the current branch when one already exists.
