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
	masterBefore := gitRevParse(t, repo, "master")

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

	assertRefParent(t, repo, "branch1", masterBefore)
	assertRefParent(t, repo, "branch2", branch1After)
	assertRefParent(t, repo, "branch3", branch1After)
	assertRefParent(t, repo, "branch4", branch2After)
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
		"git-stack graph                          Show the branch graph and sync state",
		"git-stack cascade [--apply]              Plan or apply the cascade from the current branch",
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
		"cascade": "usage: git-stack cascade [--apply]",
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

func TestGraphCommandFormatsTable(t *testing.T) {
	repo := t.TempDir()
	mustGit(t, repo, "init", "-b", "master")
	mustGit(t, repo, "config", "user.name", "test")
	mustGit(t, repo, "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repo, "root.txt"), "root\n")
	mustGit(t, repo, "add", "root.txt")
	mustGit(t, repo, "commit", "-m", "root commit")
	masterHash := gitRevParse(t, repo, "master")

	mustGit(t, repo, "checkout", "-b", "branch-one")
	writeFile(t, filepath.Join(repo, "one.txt"), "one\n")
	mustGit(t, repo, "add", "one.txt")
	mustGit(t, repo, "commit", "-m", "branch one")
	branchOneHash := gitRevParse(t, repo, "branch-one")

	mustGit(t, repo, "checkout", "-b", "branch-two")
	writeFile(t, filepath.Join(repo, "two.txt"), "two\n")
	mustGit(t, repo, "add", "two.txt")
	mustGit(t, repo, "commit", "-m", "branch two")
	branchTwoHash := gitRevParse(t, repo, "branch-two")

	writeFile(t, filepath.Join(repo, ConfigFilename), strings.Join([]string{
		"master=",
		"branch-one=master",
		"branch-two=branch-one",
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
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d:\n%s", len(lines), strings.Join(lines, "\n"))
	}

	header := lines[0]
	separator := lines[1]
	if !strings.Contains(header, "Hash") || !strings.Contains(header, "State") || !strings.Contains(header, "Branch") || !strings.Contains(header, "Title") {
		t.Fatalf("header missing columns:\n%s", header)
	}
	if len(separator) != len(header) {
		t.Fatalf("separator length mismatch: header=%d separator=%d", len(header), len(separator))
	}
	for _, line := range lines[2:] {
		if len(line) != len(header) {
			t.Fatalf("row length mismatch: want %d got %d\n%s", len(header), len(line), line)
		}
	}

	hashCol := strings.Index(header, "Hash")
	stateCol := strings.Index(header, "State")
	branchCol := strings.Index(header, "Branch")
	titleCol := strings.Index(header, "Title")
	for _, want := range []string{masterHash[:7], branchOneHash[:7], branchTwoHash[:7]} {
		if !strings.Contains(app.Stdout.(*bytes.Buffer).String(), want) {
			t.Fatalf("output missing hash %s:\n%s", want, app.Stdout.(*bytes.Buffer).String())
		}
	}
	for _, row := range []struct {
		line  string
		hash  string
		state string
		branch string
		title string
	}{
		{lines[2], masterHash[:7], "Unsync", "master", "root commit"},
		{lines[3], branchOneHash[:7], "Sync", "branch-one", "branch one"},
		{lines[4], branchTwoHash[:7], "Sync", "branch-two", "branch two"},
	} {
		if strings.Index(row.line, row.hash) != hashCol {
			t.Fatalf("hash column not aligned for %q:\n%s", row.branch, row.line)
		}
		if strings.Index(row.line, row.state) != stateCol {
			t.Fatalf("state column not aligned for %q:\n%s", row.branch, row.line)
		}
		if strings.Index(row.line, row.branch) != branchCol {
			t.Fatalf("branch column not aligned for %q:\n%s", row.branch, row.line)
		}
		if strings.Index(row.line, row.title) != titleCol {
			t.Fatalf("title column not aligned for %q:\n%s", row.branch, row.line)
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
	for _, want := range []string{"complete -F _git_stack_completion git-stack", "compgen -W \"parent graph cascade completion version help --help -h --version -v\""} {
		if !strings.Contains(bash, want) {
			t.Fatalf("bash completion missing %q:\n%s", want, bash)
		}
	}

	stdout.Reset()
	if err := app.Execute([]string{"completion", "zsh"}); err != nil {
		t.Fatalf("Execute(completion zsh) failed: %v", err)
	}
	zsh := stdout.String()
	for _, want := range []string{"#compdef git-stack", "compdef _git_stack_completion git-stack"} {
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
