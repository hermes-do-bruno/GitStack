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
