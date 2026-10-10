package gitstack

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCascadeApplyRebasesStackedRepo(t *testing.T) {
	repo := t.TempDir()
	mustGit(t, repo, "init", "-b", "master")
	mustGit(t, repo, "config", "user.name", "test")
	mustGit(t, repo, "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repo, "base.txt"), "base\n")
	mustGit(t, repo, "add", "base.txt")
	mustGit(t, repo, "commit", "-m", "base")

	mustGit(t, repo, "checkout", "-b", "branch1")
	writeFile(t, filepath.Join(repo, "branch1.txt"), "branch1\n")
	mustGit(t, repo, "add", "branch1.txt")
	mustGit(t, repo, "commit", "-m", "branch1")
	branch1Before := gitRevParse(t, repo, "branch1")

	mustGit(t, repo, "checkout", "-b", "branch2")
	writeFile(t, filepath.Join(repo, "branch2.txt"), "branch2\n")
	mustGit(t, repo, "add", "branch2.txt")
	mustGit(t, repo, "commit", "-m", "branch2")
	branch2Before := gitRevParse(t, repo, "branch2")

	mustGit(t, repo, "checkout", "-b", "branch3", "branch1")
	writeFile(t, filepath.Join(repo, "branch3.txt"), "branch3\n")
	mustGit(t, repo, "add", "branch3.txt")
	mustGit(t, repo, "commit", "-m", "branch3")
	branch3Before := gitRevParse(t, repo, "branch3")

	mustGit(t, repo, "checkout", "-b", "branch4", "branch2")
	mustGit(t, repo, "merge", "--no-ff", "branch3", "-m", "merge branch3 into branch4")
	branch4Before := gitRevParse(t, repo, "branch4")

	mustGit(t, repo, "checkout", "master")
	writeFile(t, filepath.Join(repo, "master.txt"), "master-update\n")
	mustGit(t, repo, "add", "master.txt")
	mustGit(t, repo, "commit", "-m", "master update")

	masterAfter := gitRevParse(t, repo, "master")

	writeFile(t, filepath.Join(repo, ConfigFilename), strings.Join([]string{
		"branch1=master",
		"branch2=branch1",
		"branch3=branch1",
		"branch4=branch2,branch3",
		"",
	}, "\n"))

	app := &App{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	cwd, _ := os.Getwd()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	mustGit(t, repo, "checkout", "branch1")

	if err := app.Execute([]string{"cascade", "--apply"}); err != nil {
		t.Fatalf("cascade --apply failed: %v", err)
	}

	branch1After := gitRevParse(t, repo, "branch1")
	branch2After := gitRevParse(t, repo, "branch2")
	branch3After := gitRevParse(t, repo, "branch3")
	branch4After := gitRevParse(t, repo, "branch4")

	if branch1After == branch1Before {
		t.Fatalf("branch1 did not change after rebase")
	}
	if branch2After == branch2Before {
		t.Fatalf("branch2 did not change after rebase")
	}
	if branch3After == branch3Before {
		t.Fatalf("branch3 did not change after rebase")
	}
	if branch4After == branch4Before {
		t.Fatalf("branch4 did not change after rebase")
	}

	assertRefParent(t, repo, "branch1", masterAfter)
	assertRefParent(t, repo, "branch2", branch1After)
	assertRefParent(t, repo, "branch3", branch1After)
	assertRefParent(t, repo, "branch4", branch2After)
}

func TestCascadeApplyRebasesBranchesWithMultipleCommits(t *testing.T) {
	repo := t.TempDir()
	mustGit(t, repo, "init", "-b", "master")
	mustGit(t, repo, "config", "user.name", "test")
	mustGit(t, repo, "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repo, "base.txt"), "base\n")
	mustGit(t, repo, "add", "base.txt")
	mustGit(t, repo, "commit", "-m", "base")

	mustGit(t, repo, "checkout", "-b", "branch1")
	writeFile(t, filepath.Join(repo, "branch1.txt"), "branch1-a\n")
	mustGit(t, repo, "add", "branch1.txt")
	mustGit(t, repo, "commit", "-m", "branch1 a")
	writeFile(t, filepath.Join(repo, "branch1.txt"), "branch1-b\n")
	mustGit(t, repo, "commit", "-am", "branch1 b")
	branch1Before := gitRevParse(t, repo, "branch1")

	mustGit(t, repo, "checkout", "-b", "branch2")
	writeFile(t, filepath.Join(repo, "branch2.txt"), "branch2\n")
	mustGit(t, repo, "add", "branch2.txt")
	mustGit(t, repo, "commit", "-m", "branch2")
	branch2Before := gitRevParse(t, repo, "branch2")

	mustGit(t, repo, "checkout", "master")
	writeFile(t, filepath.Join(repo, "master.txt"), "master-update\n")
	mustGit(t, repo, "add", "master.txt")
	mustGit(t, repo, "commit", "-m", "master update")

	masterAfter := gitRevParse(t, repo, "master")

	writeFile(t, filepath.Join(repo, ConfigFilename), strings.Join([]string{
		"branch1=master",
		"branch2=branch1",
		"",
	}, "\n"))

	app := &App{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	cwd, _ := os.Getwd()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	mustGit(t, repo, "checkout", "branch1")

	if err := app.Execute([]string{"cascade", "--apply"}); err != nil {
		t.Fatalf("cascade --apply failed: %v", err)
	}

	branch1After := gitRevParse(t, repo, "branch1")
	branch2After := gitRevParse(t, repo, "branch2")

	if branch1After == branch1Before {
		t.Fatalf("branch1 did not change after rebase")
	}
	if branch2After == branch2Before {
		t.Fatalf("branch2 did not change after rebase")
	}

	assertRefParent(t, repo, "branch2", branch1After)

	cmd := exec.Command("git", "rev-list", "--count", masterAfter+"..branch1")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-list --count failed: %v\n%s", err, string(out))
	}
	if strings.TrimSpace(string(out)) != "2" {
		t.Fatalf("expected branch1 to keep 2 commits after rebase, got %s", strings.TrimSpace(string(out)))
	}
}

func mustGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func gitRevParse(t *testing.T, repo, ref string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", ref)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse %s failed: %v\n%s", ref, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func assertRefParent(t *testing.T, repo, ref, expected string) {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", ref+"^")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse %s^ failed: %v\n%s", ref, err, string(out))
	}
	got := strings.TrimSpace(string(out))
	if got != expected {
		t.Fatalf("parent mismatch for %s: want %s got %s", ref, expected, got)
	}
}

func TestCascadeApplySurfacesRebaseConflicts(t *testing.T) {
	repo := t.TempDir()
	mustGit(t, repo, "init", "-b", "master")
	mustGit(t, repo, "config", "user.name", "test")
	mustGit(t, repo, "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repo, "f.txt"), "base\n")
	mustGit(t, repo, "add", "f.txt")
	mustGit(t, repo, "commit", "-m", "base")

	mustGit(t, repo, "checkout", "-b", "branch1")
	writeFile(t, filepath.Join(repo, "f.txt"), "branch1\n")
	mustGit(t, repo, "commit", "-am", "branch1")

	mustGit(t, repo, "checkout", "master")
	writeFile(t, filepath.Join(repo, "f.txt"), "master-change\n")
	mustGit(t, repo, "commit", "-am", "master change")

	writeFile(t, filepath.Join(repo, ConfigFilename), "branch1=master\n")

	app := &App{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	cwd, _ := os.Getwd()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	err := app.Execute([]string{"cascade", "--apply"})
	if err == nil {
		t.Fatal("expected cascade to fail")
	}
	msg := err.Error()
	for _, want := range []string{"rebase failed for branch1 onto master", "CONFLICT (content): Merge conflict in f.txt", "Could not apply", "conflict detected; resolve it and then run `git rebase --continue` or `git rebase --abort`"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error message missing %q:\n%s", want, msg)
		}
	}
}

func TestCascadePlanShowsThreeRefRebase(t *testing.T) {
	repo := t.TempDir()
	mustGit(t, repo, "init", "-b", "master")
	mustGit(t, repo, "config", "user.name", "test")
	mustGit(t, repo, "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repo, "base.txt"), "base\n")
	mustGit(t, repo, "add", "base.txt")
	mustGit(t, repo, "commit", "-m", "base")

	mustGit(t, repo, "checkout", "-b", "branch1")
	writeFile(t, filepath.Join(repo, "branch1.txt"), "branch1\n")
	mustGit(t, repo, "add", "branch1.txt")
	mustGit(t, repo, "commit", "-m", "branch1")

	mustGit(t, repo, "checkout", "master")
	writeFile(t, filepath.Join(repo, "master.txt"), "master-update\n")
	mustGit(t, repo, "add", "master.txt")
	mustGit(t, repo, "commit", "-m", "master update")

	masterAfter := gitRevParse(t, repo, "master")

	writeFile(t, filepath.Join(repo, ConfigFilename), strings.Join([]string{
		"branch1=master",
		"",
	}, "\n"))

	app := &App{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	cwd, _ := os.Getwd()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	mustGit(t, repo, "checkout", "branch1")

	if err := app.Execute([]string{"cascade"}); err != nil {
		t.Fatalf("cascade dry-run failed: %v", err)
	}

	output := app.Stdout.(*bytes.Buffer).String()
	want := "git rebase " + masterAfter[:7] + " branch1 --onto master"
	if !strings.Contains(output, want) {
		t.Fatalf("dry-run output missing three-ref rebase command %q:\n%s", want, output)
	}
}

func TestCascadeScriptPrintsShellCommands(t *testing.T) {
	repo := t.TempDir()
	mustGit(t, repo, "init", "-b", "master")
	mustGit(t, repo, "config", "user.name", "test")
	mustGit(t, repo, "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repo, "base.txt"), "base\n")
	mustGit(t, repo, "add", "base.txt")
	mustGit(t, repo, "commit", "-m", "base")

	mustGit(t, repo, "checkout", "-b", "branch1")
	writeFile(t, filepath.Join(repo, "branch1.txt"), "branch1\n")
	mustGit(t, repo, "add", "branch1.txt")
	mustGit(t, repo, "commit", "-m", "branch1")

	mustGit(t, repo, "checkout", "master")
	writeFile(t, filepath.Join(repo, "master.txt"), "master-update\n")
	mustGit(t, repo, "add", "master.txt")
	mustGit(t, repo, "commit", "-m", "master update")

	masterAfter := gitRevParse(t, repo, "master")

	writeFile(t, filepath.Join(repo, ConfigFilename), strings.Join([]string{
		"branch1=master",
		"",
	}, "\n"))

	app := &App{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	cwd, _ := os.Getwd()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	mustGit(t, repo, "checkout", "branch1")

	if err := app.Execute([]string{"cascade", "--script"}); err != nil {
		t.Fatalf("cascade --script failed: %v", err)
	}

	output := app.Stdout.(*bytes.Buffer).String()
	want := "git rebase " + masterAfter[:7] + " branch1 --onto master"
	for _, snippet := range []string{"#!/usr/bin/env sh", "set -eu", want} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("script output missing %q:\n%s", snippet, output)
		}
	}
	if strings.Contains(output, "[Sync]") || strings.Contains(output, "git-stack cascade") {
		t.Fatalf("script output should contain shell commands, not plan lines:\n%s", output)
	}
}

func TestRebasePlanActionUsesOntoLastForMergeAware(t *testing.T) {
	got := rebasePlanAction("onto", "abcdef1234567890", "branch1", true)
	want := "git rebase --rebase-merges abcdef1 branch1 --onto onto"
	if got != want {
		t.Fatalf("merge-aware plan mismatch: want %q got %q", want, got)
	}
}

func TestHelpShowsCommandsAndParameters(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := &App{Stdout: &stdout, Stderr: &stderr}

	if err := app.Execute([]string{"--help"}); err != nil {
		t.Fatalf("Execute(--help) failed: %v", err)
	}

	output := stdout.String() + stderr.String()
	for _, want := range []string{
		"git-stack parent <parent> [<parent>...]  Set the current branch parent(s) in .git-stack",
		"git-stack graph [columns]                Show the branch graph and sync state, with optional column filtering",
		"git-stack cascade [--apply|--script]     Plan, print a script, or apply the cascade from the current branch",
		"git-stack completion <bash|zsh>          Print shell completion script",
		"git-stack version                        Show the CLI version",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("top-level help missing %q:\n%s", want, output)
		}
	}

	cases := map[string]string{
		"parent":  "usage: git-stack parent <parent> [<parent>...]",
		"graph":   "usage: git-stack graph",
		"cascade": "usage: git-stack cascade [--apply|--script]",
		"version": "usage: git-stack version",
	}
	for command, want := range cases {
		stdout.Reset()
		stderr.Reset()
		if err := app.Execute([]string{command, "--help"}); err != nil {
			t.Fatalf("Execute(%s --help) failed: %v", command, err)
		}
		output := stdout.String() + stderr.String()
		if !strings.Contains(output, want) {
			t.Fatalf("%s help missing %q:\n%s", command, want, output)
		}
	}
}

func TestGraphCommandPresentsMixedLayout(t *testing.T) {
	repo := t.TempDir()
	mustGit(t, repo, "init", "-b", "master")
	mustGit(t, repo, "config", "user.name", "test")
	mustGit(t, repo, "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repo, "root.txt"), "root\n")
	mustGit(t, repo, "add", "root.txt")
	mustGit(t, repo, "commit", "-m", "root commit")
	masterHash := gitRevParse(t, repo, "master")

	mustGit(t, repo, "checkout", "-b", "branch1")
	writeFile(t, filepath.Join(repo, "branch1.txt"), "branch1\n")
	mustGit(t, repo, "add", "branch1.txt")
	mustGit(t, repo, "commit", "-m", "branch one")
	branch1Hash := gitRevParse(t, repo, "branch1")

	mustGit(t, repo, "checkout", "-b", "branch2")
	writeFile(t, filepath.Join(repo, "branch2.txt"), "branch2\n")
	mustGit(t, repo, "add", "branch2.txt")
	mustGit(t, repo, "commit", "-m", "branch two")
	branch2Hash := gitRevParse(t, repo, "branch2")

	mustGit(t, repo, "checkout", "-b", "branch3", "branch1")
	writeFile(t, filepath.Join(repo, "branch3.txt"), "branch3\n")
	mustGit(t, repo, "add", "branch3.txt")
	mustGit(t, repo, "commit", "-m", "branch three")
	branch3Hash := gitRevParse(t, repo, "branch3")

	mustGit(t, repo, "checkout", "-b", "branch4", "branch2")
	mustGit(t, repo, "merge", "--no-ff", "branch3", "-m", "merge branch3 into branch4")
	branch4Hash := gitRevParse(t, repo, "branch4")

	writeFile(t, filepath.Join(repo, ConfigFilename), strings.Join([]string{
		"master=",
		"branch1=master",
		"branch2=branch1",
		"branch3=branch1",
		"branch4=branch2,branch3",
		"",
	}, "\n"))

	app := &App{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	cwd, _ := os.Getwd()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	mustGit(t, repo, "checkout", "master")

	if err := app.Execute([]string{"graph"}); err != nil {
		t.Fatalf("graph failed: %v", err)
	}

	raw := app.Stdout.(*bytes.Buffer).String()
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	if len(lines) != 8 {
		t.Fatalf("expected 8 lines, got %d:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	if lines[0] != "Graph:" {
		t.Fatalf("missing graph title: %q", lines[0])
	}
	header := lines[1]
	if !strings.Contains(header, "Chart") || !strings.Contains(header, "Branch") || !strings.Contains(header, "State") || !strings.Contains(header, "Hash") || !strings.Contains(header, "Title") || !strings.Contains(header, "Parents") {
		t.Fatalf("header missing columns:\n%s", header)
	}
	branchCol := strings.Index(header, "Branch")
	stateCol := strings.Index(header, "State")
	hashCol := strings.Index(header, "Hash")
	titleCol := strings.Index(header, "Title")
	parentsCol := strings.Index(header, "Parents")
	runeIndex := func(s, sub string) int {
		byteIndex := strings.Index(s, sub)
		if byteIndex < 0 {
			return -1
		}
		return len([]rune(s[:byteIndex]))
	}
	if lines[2] == "" {
		t.Fatal("missing separator")
	}
	if len([]rune(lines[2])) != len([]rune(header)) {
		t.Fatalf("separator length mismatch: header=%d separator=%d", len([]rune(header)), len([]rune(lines[2])))
	}
	for _, line := range lines[3:] {
		if len([]rune(line)) != len([]rune(header)) {
			t.Fatalf("row length mismatch: want %d got %d\n%s", len([]rune(header)), len([]rune(line)), line)
		}
	}

	rows := []struct {
		line    string
		chart   string
		branch  string
		state   string
		hash    string
		title   string
		parents string
	}{
		{lines[3], "●", "master", "Unsync", masterHash[:7], "root commit", "-"},
		{lines[4], "└──", "branch1", "Sync", branch1Hash[:7], "branch one", "master"},
		{lines[5], "    ├──", "branch2", "Sync", branch2Hash[:7], "branch two", "branch1"},
		{lines[6], "    │   └──", "branch4", "Partial-sync", branch4Hash[:7], "merge branch3 into branch4", "branch2, branch3"},
		{lines[7], "    └──", "branch3", "Sync", branch3Hash[:7], "branch three", "branch1"},
	}
	for _, row := range rows {
		if !strings.HasPrefix(row.line, row.chart) {
			t.Fatalf("chart column mismatch for %q:\n%s", row.branch, row.line)
		}
		if runeIndex(row.line, row.branch) != branchCol {
			t.Fatalf("branch column mismatch for %q:\n%s", row.branch, row.line)
		}
		if runeIndex(row.line, row.state) != stateCol {
			t.Fatalf("state column mismatch for %q:\n%s", row.branch, row.line)
		}
		if runeIndex(row.line, row.hash) != hashCol {
			t.Fatalf("hash column mismatch for %q:\n%s", row.branch, row.line)
		}
		if runeIndex(row.line, row.title) != titleCol {
			t.Fatalf("title column mismatch for %q:\n%s", row.branch, row.line)
		}
		if runeIndex(row.line, row.parents) != parentsCol {
			t.Fatalf("parents column mismatch for %q:\n%s", row.branch, row.line)
		}
	}
}

func TestGraphCommandFiltersColumns(t *testing.T) {
	repo := t.TempDir()
	mustGit(t, repo, "init", "-b", "master")
	mustGit(t, repo, "config", "user.name", "test")
	mustGit(t, repo, "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repo, "root.txt"), "root\n")
	mustGit(t, repo, "add", "root.txt")
	mustGit(t, repo, "commit", "-m", "root commit")

	mustGit(t, repo, "checkout", "-b", "branch1")
	writeFile(t, filepath.Join(repo, "branch1.txt"), "branch1\n")
	mustGit(t, repo, "add", "branch1.txt")
	mustGit(t, repo, "commit", "-m", "branch one")

	writeFile(t, filepath.Join(repo, ConfigFilename), strings.Join([]string{
		"master=",
		"branch1=master",
		"",
	}, "\n"))

	app := &App{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	cwd, _ := os.Getwd()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	mustGit(t, repo, "checkout", "master")

	if err := app.Execute([]string{"graph", "branch,state"}); err != nil {
		t.Fatalf("graph failed: %v", err)
	}

	output := app.Stdout.(*bytes.Buffer).String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d:\n%s", len(lines), output)
	}
	if lines[0] != "Graph:" {
		t.Fatalf("missing graph title: %q", lines[0])
	}
	header := lines[1]
	if !strings.Contains(header, "Branch") || !strings.Contains(header, "State") {
		t.Fatalf("header missing selected columns:\n%s", header)
	}
	for _, unwanted := range []string{"Chart", "Hash", "Title", "Parents"} {
		if strings.Contains(header, unwanted) {
			t.Fatalf("header contains unexpected column %q:\n%s", unwanted, header)
		}
	}
}

func TestGraphCommandUsesColorsWhenForced(t *testing.T) {
	repo := t.TempDir()
	mustGit(t, repo, "init", "-b", "master")
	mustGit(t, repo, "config", "user.name", "test")
	mustGit(t, repo, "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repo, "root.txt"), "root\n")
	mustGit(t, repo, "add", "root.txt")
	mustGit(t, repo, "commit", "-m", "root commit")

	mustGit(t, repo, "checkout", "-b", "branch1")
	writeFile(t, filepath.Join(repo, "branch1.txt"), "branch1\n")
	mustGit(t, repo, "add", "branch1.txt")
	mustGit(t, repo, "commit", "-m", "branch one")

	writeFile(t, filepath.Join(repo, ConfigFilename), strings.Join([]string{
		"master=",
		"branch1=master",
		"",
	}, "\n"))

	app := &App{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	cwd, _ := os.Getwd()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	t.Setenv("FORCE_COLOR", "1")

	mustGit(t, repo, "checkout", "master")

	if err := app.Execute([]string{"graph"}); err != nil {
		t.Fatalf("graph failed: %v", err)
	}

	output := app.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, "\x1b[") {
		t.Fatalf("expected ANSI colors in graph output:\n%s", output)
	}
	for _, want := range []string{"Graph:", "Chart", "Branch", "State", "Hash", "Title", "Parents", "master", "branch1"} {
		if !strings.Contains(output, want) {
			t.Fatalf("colored output missing %q:\n%s", want, output)
		}
	}
}

func TestCompletionCommand(t *testing.T) {
	var stdout bytes.Buffer
	app := &App{Stdout: &stdout, Stderr: &bytes.Buffer{}}

	if err := app.Execute([]string{"completion", "bash"}); err != nil {
		t.Fatalf("Execute(completion bash) failed: %v", err)
	}
	bash := stdout.String()
	for _, want := range []string{"complete -F _git_stack_completion git-stack", "compgen -W \"parent graph cascade completion version help --help -h --version -v\"", "--apply --script --help -h"} {
		if !strings.Contains(bash, want) {
			t.Fatalf("bash completion missing %q:\n%s", want, bash)
		}
	}

	stdout.Reset()
	if err := app.Execute([]string{"completion", "zsh"}); err != nil {
		t.Fatalf("Execute(completion zsh) failed: %v", err)
	}
	zsh := stdout.String()
	for _, want := range []string{"#compdef git-stack", "compdef _git_stack_completion git-stack", "--script[print a shell script]"} {
		if !strings.Contains(zsh, want) {
			t.Fatalf("zsh completion missing %q:\n%s", want, zsh)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	var stdout bytes.Buffer
	app := &App{Stdout: &stdout, Stderr: &bytes.Buffer{}}

	if err := app.Execute([]string{"version"}); err != nil {
		t.Fatalf("Execute(version) failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		t.Fatal("version output is empty")
	}
	if output != Version {
		t.Fatalf("version output mismatch: want %q got %q", Version, output)
	}
}

func TestCascadeStepErrorFormatting(t *testing.T) {
	err := (&cascadeStepError{
		kind:   "merge-aware rebase",
		branch: "branch4",
		target: "branch2",
		output: "CONFLICT (content): Merge conflict in f.txt",
		err:    exec.ErrNotFound,
	}).Error()
	for _, want := range []string{"merge-aware rebase failed for branch4 onto branch2", "CONFLICT (content): Merge conflict in f.txt"} {
		if !strings.Contains(err, want) {
			t.Fatalf("formatted error missing %q: %s", want, err)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
