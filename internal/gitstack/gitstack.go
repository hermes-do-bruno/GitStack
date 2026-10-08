package gitstack

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const ConfigFilename = ".git-stack"

type App struct {
	Stdout io.Writer
	Stderr io.Writer
}

var Version = "dev"

type BranchState string

const (
	StateMerged      BranchState = "Merged"
	StateSync        BranchState = "Sync"
	StatePartialSync BranchState = "Partial-sync"
	StateUnsync      BranchState = "Unsync"
)

type ConfigLineKind string

const (
	LineBlank   ConfigLineKind = "blank"
	LineComment ConfigLineKind = "comment"
	LineEntry   ConfigLineKind = "entry"
)

type ConfigLine struct {
	Kind    ConfigLineKind
	Raw     string
	Branch  string
	Parents []string
}

type ConfigFile struct {
	Lines        []ConfigLine
	EntryIndexes map[string]int
}

type branchPlan struct {
	Branch       string
	Hash         string
	State        BranchState
	Action       string
	TargetRef    string
	UpstreamHash string
}

type cascadeStepError struct {
	kind   string
	branch string
	target string
	output string
	err    error
}

func (e *cascadeStepError) Error() string {
	msg := fmt.Sprintf("git-stack: %s failed for %s onto %s: %v", e.kind, e.branch, e.target, e.err)
	if trimmed := strings.TrimSpace(e.output); trimmed != "" {
		msg += "\n" + trimmed
		if strings.Contains(trimmed, "CONFLICT (") || strings.Contains(trimmed, "Could not apply") {
			msg += "\ngit-stack: conflict detected; resolve it and then run `git rebase --continue` or `git rebase --abort`"
		}
	}
	return msg
}

func (e *cascadeStepError) Unwrap() error { return e.err }

func NewApp() *App {
	return &App{Stdout: os.Stdout, Stderr: os.Stderr}
}

func (a *App) Execute(args []string) error {
	if len(args) == 0 {
		return a.help()
	}

	switch args[0] {
	case "parent":
		if hasHelpFlag(args[1:]) {
			return a.parentHelp()
		}
		return a.runParent(args[1:])
	case "graph":
		if hasHelpFlag(args[1:]) {
			return a.graphHelp()
		}
		return a.runGraph()
	case "cascade":
		if hasHelpFlag(args[1:]) {
			return a.cascadeHelp()
		}
		return a.runCascade(args[1:])
	case "completion":
		if hasHelpFlag(args[1:]) {
			return a.completionHelp()
		}
		return a.runCompletion(args[1:])
	case "version", "--version", "-v":
		if hasHelpFlag(args[1:]) {
			return a.versionHelp()
		}
		return a.version()
	case "help", "--help", "-h":
		return a.help()
	default:
		return cliErrorf("unknown command %q", args[0])
	}
}

func (a *App) help() error {
	_, err := fmt.Fprintln(a.Stdout, helpText())
	return err
}

func (a *App) version() error {
	_, err := fmt.Fprintln(a.Stdout, Version)
	return err
}

func (a *App) versionHelp() error {
	_, err := fmt.Fprintln(a.Stdout, "usage: git-stack version")
	return err
}

func (a *App) parentHelp() error {
	_, err := fmt.Fprintln(a.Stdout, "usage: git-stack parent <parent> [<parent>...]")
	return err
}

func (a *App) graphHelp() error {
	_, err := fmt.Fprintln(a.Stdout, "usage: git-stack graph")
	return err
}

func (a *App) cascadeHelp() error {
	_, err := fmt.Fprintln(a.Stdout, "usage: git-stack cascade [--apply]")
	return err
}

func (a *App) completionHelp() error {
	_, err := fmt.Fprintln(a.Stdout, "usage: git-stack completion <bash|zsh>")
	return err
}

func helpText() string {
	return strings.TrimSpace(`git-stack - stacked-branch workflow tool

Usage:
  git-stack <command> [args]

Commands:
  git-stack parent <parent> [<parent>...]  Set the current branch parent(s) in .git-stack
  git-stack graph                          Show the branch graph and sync state
  git-stack cascade [--apply]              Plan or apply the cascade from the current branch
  git-stack completion <bash|zsh>          Print shell completion script
  git-stack version                        Show the CLI version
  git-stack help                           Show this help
  git-stack --help                         Show this help
  git-stack -h                             Show this help

Examples:
  git-stack parent master
  git-stack graph
  git-stack cascade --apply`)
}

func bashCompletionScript() string {
	return strings.TrimSpace(`_git_stack_completion() {
  local cur refs
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"

  if [ "${COMP_CWORD}" -eq 1 ]; then
    COMPREPLY=( $(compgen -W "parent graph cascade completion version help --help -h --version -v" -- "$cur") )
    return 0
  fi

  case "${COMP_WORDS[1]}" in
    parent)
      refs="$(git for-each-ref --format='%(refname:short)' refs/heads refs/remotes 2>/dev/null)"
      COMPREPLY=( $(compgen -W "$refs --help -h" -- "$cur") )
      ;;
    cascade)
      COMPREPLY=( $(compgen -W "--apply --help -h" -- "$cur") )
      ;;
    completion)
      COMPREPLY=( $(compgen -W "bash zsh --help -h" -- "$cur") )
      ;;
    *)
      COMPREPLY=( $(compgen -W "--help -h" -- "$cur") )
      ;;
  esac
}
complete -F _git_stack_completion git-stack`)
}

func zshCompletionScript() string {
	return strings.TrimSpace(`#compdef git-stack

_git_stack_completion() {
  local state
  local -a branches
  typeset -A opt_args

  _arguments -C \
    '1:command:(parent graph cascade completion version help)' \
    '*::arg:->args'

  case "$state" in
    args)
      case "$words[2]" in
        parent)
          branches=("${(@f)$(git for-each-ref --format='%(refname:short)' refs/heads refs/remotes 2>/dev/null)}")
          compadd -- $branches
          ;;
        cascade)
          _arguments '--apply[apply the cascade]' '--help[show help]' '-h[show help]'
          ;;
        completion)
          _arguments '1:shell:(bash zsh)'
          ;;
      esac
      ;;
  esac
}
compdef _git_stack_completion git-stack`)
}

func (a *App) runCompletion(args []string) error {
	if len(args) != 1 {
		return cliErrorf("usage: git-stack completion <bash|zsh>")
	}
	var script string
	switch args[0] {
	case "bash":
		script = bashCompletionScript()
	case "zsh":
		script = zshCompletionScript()
	default:
		return cliErrorf("unsupported completion shell %q", args[0])
	}
	_, err := fmt.Fprintln(a.Stdout, script)
	return err
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "help", "--help", "-h":
			return true
		}
	}
	return false
}

func (a *App) usage() error {
	return a.help()
}

func (a *App) runParent(args []string) error {
	if len(args) == 0 {
		return cliErrorf("usage: git-stack parent <parent> [<parent>...]")
	}

	root, err := repoRoot()
	if err != nil {
		return err
	}
	current, err := currentBranch(root)
	if err != nil {
		return err
	}
	cfgPath := filepath.Join(root, ConfigFilename)
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		return err
	}
	if cfg.EntryIndexes == nil {
		cfg.EntryIndexes = make(map[string]int)
	}
	cfg.Upsert(current, args)
	if err := cfg.Save(cfgPath); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.Stdout, "%s=%s\n", current, strings.Join(args, ","))
	return err
}

func (a *App) runGraph() error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	current, err := currentBranch(root)
	if err != nil {
		return err
	}
	cfg, err := loadConfig(filepath.Join(root, ConfigFilename))
	if err != nil {
		return err
	}
	order, err := cfg.Subtree(current)
	if err != nil {
		return err
	}
	if len(order) == 0 {
		return cliErrorf("branch %q is not declared in %s", current, ConfigFilename)
	}

	cache := map[string]string{}
	for _, branch := range order {
		hash, err := refHash(root, branch, cache)
		if err != nil {
			return err
		}
		title, err := commitTitle(root, branch)
		if err != nil {
			return err
		}
		state, err := branchState(root, branch, cfg.Parents(branch), cache)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(a.Stdout, "%s [%s] %s %s\n", shortHash(hash), state, branch, title)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *App) runCascade(args []string) error {
	apply := false
	if len(args) > 0 {
		if args[0] == "--apply" {
			apply = true
			args = args[1:]
		}
	}
	if len(args) > 0 {
		return cliErrorf("usage: git-stack cascade [--apply]")
	}

	root, err := repoRoot()
	if err != nil {
		return err
	}
	current, err := currentBranch(root)
	if err != nil {
		return err
	}
	cfg, err := loadConfig(filepath.Join(root, ConfigFilename))
	if err != nil {
		return err
	}
	order, err := cfg.Subtree(current)
	if err != nil {
		return err
	}
	if len(order) == 0 {
		return cliErrorf("branch %q is not declared in %s", current, ConfigFilename)
	}

	allRefs := cfg.RelevantRefs(order)
	original := map[string]string{}
	for _, ref := range allRefs {
		hash, err := refHash(root, ref, original)
		if err != nil {
			return err
		}
		original[ref] = hash
	}
	currentHashes := cloneMap(original)

	plans := make([]branchPlan, 0, len(order))
	for _, branch := range order {
		parents := cfg.Parents(branch)
		branchHash, err := refHash(root, branch, currentHashes)
		if err != nil {
			return err
		}
		state, err := branchState(root, branch, parents, currentHashes)
		if err != nil {
			return err
		}
		if len(parents) == 0 {
			plans = append(plans, branchPlan{Branch: branch, Hash: branchHash, State: state, Action: "skip (no parents)"})
			continue
		}
		if state == StateMerged {
			plans = append(plans, branchPlan{Branch: branch, Hash: branchHash, State: state, Action: "skip merged"})
			continue
		}
		targetRef := parents[0]
		targetHash, err := refHash(root, targetRef, currentHashes)
		if err != nil {
			return err
		}
		upstreamHash, ok := original[targetRef]
		if !ok {
			return cliErrorf("missing original hash for parent %q", targetRef)
		}
		plans = append(plans, branchPlan{Branch: branch, Hash: branchHash, State: state, Action: fmt.Sprintf("rebase onto %s", shortHash(targetHash)), TargetRef: targetRef, UpstreamHash: upstreamHash})
	}

	for _, plan := range plans {
		if _, err := fmt.Fprintf(a.Stdout, "%s [%s] %s %s\n", shortHash(plan.Hash), plan.State, plan.Branch, plan.Action); err != nil {
			return err
		}
	}

	if apply {
		for _, plan := range plans {
			if plan.Action == "skip merged" || plan.Action == "skip (no parents)" {
				continue
			}
			targetHash, err := refHash(root, plan.TargetRef, currentHashes)
			if err != nil {
				return err
			}
			if err := rebaseBranch(root, plan.Branch, plan.TargetRef, targetHash, plan.UpstreamHash, len(cfg.Parents(plan.Branch)) > 1); err != nil {
				return err
			}
			delete(currentHashes, plan.Branch)
			updated, err := refHash(root, plan.Branch, currentHashes)
			if err != nil {
				return err
			}
			currentHashes[plan.Branch] = updated
		}
	}
	return nil
}

func repoRoot() (string, error) {
	out, err := runGit("", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func currentBranch(root string) (string, error) {
	out, err := runGit(root, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(out)
	if branch == "" {
		return "", cliErrorf("detached HEAD is not supported")
	}
	return branch, nil
}

func refHash(root, ref string, cache map[string]string) (string, error) {
	if cache != nil {
		if hash, ok := cache[ref]; ok && hash != "" {
			return hash, nil
		}
	}
	out, err := runGit(root, "rev-parse", ref)
	if err != nil {
		return "", err
	}
	hash := strings.TrimSpace(out)
	if cache != nil {
		cache[ref] = hash
	}
	return hash, nil
}

func commitTitle(root, ref string) (string, error) {
	out, err := runGit(root, "log", "-1", "--format=%s", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func firstParentHash(root, ref string) (string, error) {
	out, err := runGit(root, "rev-parse", ref+"^")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func isAncestor(root, ancestor, descendant string) bool {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", ancestor, descendant)
	cmd.Dir = root
	return cmd.Run() == nil
}

func branchState(root, branch string, parents []string, cache map[string]string) (BranchState, error) {
	childHash, err := refHash(root, branch, cache)
	if err != nil {
		return "", err
	}
	if len(parents) == 0 {
		return StateUnsync, nil
	}

	state := StateMerged
	for _, parent := range parents {
		parentHash, err := refHash(root, parent, cache)
		if err != nil {
			return "", err
		}
		rel := relationState(root, childHash, parentHash, branch)
		state = worseState(state, rel)
	}
	return state, nil
}

func relationState(root, childHash, parentHash, branch string) BranchState {
	if childHash == parentHash {
		return StateMerged
	}
	firstParent, err := firstParentHash(root, branch)
	if err == nil && firstParent == parentHash {
		return StateSync
	}
	if isAncestor(root, parentHash, childHash) {
		return StatePartialSync
	}
	return StateUnsync
}

func worseState(a, b BranchState) BranchState {
	order := func(s BranchState) int {
		switch s {
		case StateMerged:
			return 0
		case StateSync:
			return 1
		case StatePartialSync:
			return 2
		case StateUnsync:
			return 3
		default:
			return 3
		}
	}
	if order(b) > order(a) {
		return b
	}
	return a
}

func shortHash(hash string) string {
	if len(hash) <= 7 {
		return hash
	}
	return hash[:7]
}

func rebaseBranch(root, branch, targetRef, ontoHash, upstreamRef string, mergeAware bool) error {
	cmd := exec.Command("git", "rebase", "--rebase-merges", "--onto", ontoHash, upstreamRef, branch)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	kind := "rebase"
	if mergeAware {
		kind = "merge-aware rebase"
	}
	return &cascadeStepError{
		kind:   kind,
		branch: branch,
		target: targetRef,
		output: strings.TrimSpace(string(output)),
		err:    err,
	}
}

func runGit(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if root != "" {
		cmd.Dir = root
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", cliErrorf("git %s failed: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func loadConfig(path string) (*ConfigFile, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &ConfigFile{EntryIndexes: map[string]int{}}, nil
		}
		return nil, err
	}
	defer f.Close()
	cfg := &ConfigFile{EntryIndexes: map[string]int{}}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		switch {
		case strings.TrimSpace(line) == "":
			cfg.Lines = append(cfg.Lines, ConfigLine{Kind: LineBlank, Raw: line})
		case strings.HasPrefix(strings.TrimSpace(line), "#"):
			cfg.Lines = append(cfg.Lines, ConfigLine{Kind: LineComment, Raw: line})
		default:
			branch, parents, ok := strings.Cut(line, "=")
			if !ok {
				return nil, cliErrorf("invalid line in %s: %q", path, line)
			}
			branch = strings.TrimSpace(branch)
			if branch == "" {
				return nil, cliErrorf("invalid branch name in %s: %q", path, line)
			}
			parentsList := parseCSVList(parents)
			cfg.EntryIndexes[branch] = len(cfg.Lines)
			cfg.Lines = append(cfg.Lines, ConfigLine{Kind: LineEntry, Raw: line, Branch: branch, Parents: parentsList})
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func parseCSVList(text string) []string {
	parts := strings.Split(text, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (c *ConfigFile) Parents(branch string) []string {
	for _, line := range c.Lines {
		if line.Kind == LineEntry && line.Branch == branch {
			return append([]string(nil), line.Parents...)
		}
	}
	return nil
}

func (c *ConfigFile) Upsert(branch string, parents []string) {
	line := ConfigLine{Kind: LineEntry, Branch: branch, Parents: append([]string(nil), parents...)}
	if c.EntryIndexes == nil {
		c.EntryIndexes = map[string]int{}
	}
	if idx, ok := c.EntryIndexes[branch]; ok {
		c.Lines[idx] = line
		return
	}
	c.EntryIndexes[branch] = len(c.Lines)
	c.Lines = append(c.Lines, line)
}

func (c *ConfigFile) Save(path string) error {
	var b strings.Builder
	for i, line := range c.Lines {
		switch line.Kind {
		case LineBlank:
			b.WriteString("\n")
		case LineComment:
			b.WriteString(line.Raw)
			b.WriteString("\n")
		case LineEntry:
			b.WriteString(line.Branch)
			b.WriteString("=")
			b.WriteString(strings.Join(line.Parents, ","))
			b.WriteString("\n")
		default:
			return cliErrorf("unsupported line kind at index %d", i)
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func (c *ConfigFile) Subtree(root string) ([]string, error) {
	if len(c.Lines) == 0 {
		return nil, nil
	}
	children := c.childrenMap()
	reachable := map[string]bool{}
	var collect func(string)
	collect = func(branch string) {
		if reachable[branch] {
			return
		}
		reachable[branch] = true
		for _, child := range children[branch] {
			collect(child)
		}
	}
	if c.containsBranch(root) {
		collect(root)
	} else {
		for _, child := range children[root] {
			collect(child)
		}
	}
	if len(reachable) == 0 {
		return nil, nil
	}

	indegree := map[string]int{}
	for branch := range reachable {
		indegree[branch] = 0
	}
	for branch := range reachable {
		for _, parent := range c.Parents(branch) {
			if reachable[parent] {
				indegree[branch]++
			}
		}
	}

	queue := make([]string, 0, len(reachable))
	pushZero := func(branch string) {
		queue = append(queue, branch)
		sort.SliceStable(queue, func(i, j int) bool {
			return c.orderOf(queue[i]) < c.orderOf(queue[j])
		})
	}
	for branch, degree := range indegree {
		if degree == 0 {
			pushZero(branch)
		}
	}

	order := make([]string, 0, len(reachable))
	for len(queue) > 0 {
		branch := queue[0]
		queue = queue[1:]
		order = append(order, branch)
		for _, child := range children[branch] {
			if !reachable[child] {
				continue
			}
			indegree[child]--
			if indegree[child] == 0 {
				pushZero(child)
			}
		}
	}
	return order, nil
}

func (c *ConfigFile) RelevantRefs(order []string) []string {
	set := map[string]struct{}{}
	for _, branch := range order {
		set[branch] = struct{}{}
		for _, parent := range c.Parents(branch) {
			set[parent] = struct{}{}
		}
	}
	refs := make([]string, 0, len(set))
	for ref := range set {
		refs = append(refs, ref)
	}
	return refs
}

func (c *ConfigFile) childrenMap() map[string][]string {
	children := map[string][]string{}
	for _, line := range c.Lines {
		if line.Kind != LineEntry {
			continue
		}
		for _, parent := range line.Parents {
			children[parent] = append(children[parent], line.Branch)
		}
	}
	return children
}

func (c *ConfigFile) containsBranch(branch string) bool {
	_, ok := c.EntryIndexes[branch]
	return ok
}

func (c *ConfigFile) orderOf(branch string) int {
	if idx, ok := c.EntryIndexes[branch]; ok {
		return idx
	}
	return len(c.Lines)
}

func cloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cliErrorf(format string, args ...any) error {
	return fmt.Errorf("git-stack: "+format, args...)
}
