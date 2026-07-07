package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Config struct {
	Main          string
	Develop       string
	FeaturePrefix string
	ReleasePrefix string
	HotfixPrefix  string
	Remote        string
}

type Git struct {
	stdout  io.Writer
	stderr  io.Writer
	verbose bool
	dryRun  bool
}

type CLI struct {
	git *Git
}

const usage = `kitflow - small GitFlow-style workflow helper

Usage:
  kitflow [--verbose] [--dry-run] init [--main main] [--develop develop]
  kitflow [--verbose] [--dry-run] feature start <name> [<base>]
  kitflow [--verbose] [--dry-run] feature finish [name]
  kitflow [--verbose] [--dry-run] feature publish [name]
  kitflow [--verbose] [--dry-run] feature track <name>
  kitflow [--verbose] [--dry-run] feature list
  kitflow [--verbose] [--dry-run] release start <version> [<base>]
  kitflow [--verbose] [--dry-run] release finish <version>
  kitflow [--verbose] [--dry-run] release publish [version]
  kitflow [--verbose] [--dry-run] release track <version>
  kitflow [--verbose] [--dry-run] release list
  kitflow [--verbose] [--dry-run] hotfix start <version> [<base>]
  kitflow [--verbose] [--dry-run] hotfix finish <version>
  kitflow [--verbose] [--dry-run] hotfix publish [version]
  kitflow [--verbose] [--dry-run] hotfix list
  kitflow status

Commands:
  init             Store local kitflow config and create develop if possible
  feature start    Create feature/<name> from develop (or <base>)
  feature finish   Merge feature/<name> into develop and delete it
  feature publish  Push feature/<name> and set upstream tracking
  feature track    Create a local feature/<name> tracking the remote
  release start    Create release/<version> from develop (or <base>)
  release finish   Merge release into main and develop, tag it, delete it
  hotfix start     Create hotfix/<version> from main (or <base>)
  hotfix finish    Merge hotfix into main and develop, tag it, delete it

Remote defaults to origin (override with git config kitflow.remote).
`

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout, stderr io.Writer) int {
	global := flag.NewFlagSet("kitflow", flag.ContinueOnError)
	global.SetOutput(stderr)
	verbose := global.Bool("verbose", false, "print git commands")
	dryRun := global.Bool("dry-run", false, "print write operations without running them")

	if err := global.Parse(args); err != nil {
		return 2
	}

	rest := global.Args()
	if len(rest) == 0 || rest[0] == "help" || rest[0] == "--help" || rest[0] == "-h" {
		fmt.Fprint(stdout, usage)
		return 0
	}

	cli := CLI{git: &Git{stdout: stdout, stderr: stderr, verbose: *verbose, dryRun: *dryRun}}
	if err := cli.run(rest); err != nil {
		fmt.Fprintf(stderr, "kitflow: %v\n", err)
		return 1
	}
	return 0
}

func (c CLI) run(args []string) error {
	switch args[0] {
	case "init":
		return c.init(args[1:])
	case "feature":
		return c.branchCommand("feature", args[1:])
	case "release":
		return c.branchCommand("release", args[1:])
	case "hotfix":
		return c.branchCommand("hotfix", args[1:])
	case "status":
		return c.status(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func (c CLI) init(args []string) error {
	if err := c.git.ensureRepo(); err != nil {
		return err
	}

	cfg := c.defaultConfig()
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(c.git.stderr)
	mainBranch := fs.String("main", cfg.Main, "production branch")
	developBranch := fs.String("develop", cfg.Develop, "integration branch")
	featurePrefix := fs.String("feature-prefix", cfg.FeaturePrefix, "feature branch prefix")
	releasePrefix := fs.String("release-prefix", cfg.ReleasePrefix, "release branch prefix")
	hotfixPrefix := fs.String("hotfix-prefix", cfg.HotfixPrefix, "hotfix branch prefix")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("init does not take positional arguments")
	}

	cfg = Config{
		Main:          *mainBranch,
		Develop:       *developBranch,
		FeaturePrefix: *featurePrefix,
		ReleasePrefix: *releasePrefix,
		HotfixPrefix:  *hotfixPrefix,
	}
	if err := c.validateConfig(cfg); err != nil {
		return err
	}
	if err := c.saveConfig(cfg); err != nil {
		return err
	}

	createdDevelop := false
	if c.git.hasCommits() && !c.git.branchExists(cfg.Develop) && c.git.branchExists(cfg.Main) {
		if err := c.git.run("branch", cfg.Develop, cfg.Main); err != nil {
			return err
		}
		createdDevelop = true
	}

	if createdDevelop {
		fmt.Fprintf(c.git.stdout, "Initialized kitflow: main=%s develop=%s\n", cfg.Main, cfg.Develop)
		return nil
	}
	fmt.Fprintf(c.git.stdout, "Initialized kitflow config: main=%s develop=%s\n", cfg.Main, cfg.Develop)
	return nil
}

func (c CLI) branchCommand(kind string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%s requires start, finish, or list", kind)
	}

	switch args[0] {
	case "start":
		if len(args) < 2 || len(args) > 3 {
			return fmt.Errorf("usage: kitflow %s start <name> [<base>]", kind)
		}
		base := ""
		if len(args) == 3 {
			base = args[2]
		}
		return c.start(kind, args[1], base)
	case "finish":
		if len(args) > 2 {
			return fmt.Errorf("usage: kitflow %s finish [name]", kind)
		}
		name := ""
		if len(args) == 2 {
			name = args[1]
		}
		return c.finish(kind, name)
	case "publish":
		if len(args) > 2 {
			return fmt.Errorf("usage: kitflow %s publish [name]", kind)
		}
		name := ""
		if len(args) == 2 {
			name = args[1]
		}
		return c.publish(kind, name)
	case "track":
		if len(args) != 2 {
			return fmt.Errorf("usage: kitflow %s track <name>", kind)
		}
		return c.track(kind, args[1])
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("usage: kitflow %s list", kind)
		}
		return c.list(kind)
	default:
		return fmt.Errorf("unknown %s action %q", kind, args[0])
	}
}

func (c CLI) start(kind, name, baseOverride string) error {
	if err := c.git.ensureRepo(); err != nil {
		return err
	}
	if err := c.git.ensureClean(); err != nil {
		return err
	}
	cfg := c.loadConfig()
	base, prefix, err := c.flowParts(cfg, kind)
	if err != nil {
		return err
	}
	if baseOverride != "" {
		base = baseOverride
	}
	branch := fullBranchName(prefix, name)
	if err := c.git.validateBranch(branch); err != nil {
		return err
	}
	if !c.git.branchExists(base) {
		return fmt.Errorf("base branch %q does not exist", base)
	}
	if c.git.branchExists(branch) {
		return fmt.Errorf("branch %q already exists", branch)
	}
	if err := c.git.run("checkout", base); err != nil {
		return err
	}
	if err := c.git.run("checkout", "-b", branch); err != nil {
		return err
	}
	fmt.Fprintf(c.git.stdout, "Started %s %s\n", kind, branch)
	return nil
}

func (c CLI) finish(kind, name string) error {
	if err := c.git.ensureRepo(); err != nil {
		return err
	}
	if err := c.git.ensureClean(); err != nil {
		return err
	}
	cfg := c.loadConfig()
	base, prefix, err := c.flowParts(cfg, kind)
	if err != nil {
		return err
	}
	branch, err := c.branchToFinish(prefix, name)
	if err != nil {
		return err
	}
	if !c.git.branchExists(branch) {
		return fmt.Errorf("branch %q does not exist", branch)
	}

	switch kind {
	case "feature":
		if err := c.git.run("checkout", base); err != nil {
			return err
		}
		if err := c.git.run("merge", "--no-ff", "--no-edit", branch); err != nil {
			return err
		}
		if err := c.git.run("branch", "-d", branch); err != nil {
			return err
		}
	case "release", "hotfix":
		version := strings.TrimPrefix(branch, prefix)
		if err := c.git.validateTag(version); err != nil {
			return err
		}
		label := "Release"
		if kind == "hotfix" {
			label = "Hotfix"
		}
		if err := c.finishVersioned(cfg, branch, version, label); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown flow kind %q", kind)
	}

	fmt.Fprintf(c.git.stdout, "Finished %s %s\n", kind, branch)
	return nil
}

func (c CLI) finishVersioned(cfg Config, branch, version, label string) error {
	if !c.git.branchExists(cfg.Main) {
		return fmt.Errorf("main branch %q does not exist", cfg.Main)
	}
	if !c.git.branchExists(cfg.Develop) {
		return fmt.Errorf("develop branch %q does not exist", cfg.Develop)
	}
	if c.git.tagExists(version) {
		return fmt.Errorf("tag %q already exists", version)
	}
	if err := c.git.run("checkout", cfg.Main); err != nil {
		return err
	}
	if err := c.git.run("merge", "--no-ff", "--no-edit", branch); err != nil {
		return err
	}
	if err := c.git.run("tag", "-a", version, "-m", label+" "+version); err != nil {
		return err
	}
	if err := c.git.run("checkout", cfg.Develop); err != nil {
		return err
	}
	if err := c.git.run("merge", "--no-ff", "--no-edit", branch); err != nil {
		return err
	}
	if err := c.git.run("branch", "-d", branch); err != nil {
		return err
	}
	return nil
}

func (c CLI) list(kind string) error {
	if err := c.git.ensureRepo(); err != nil {
		return err
	}
	cfg := c.loadConfig()
	_, prefix, err := c.flowParts(cfg, kind)
	if err != nil {
		return err
	}
	out, err := c.git.output("for-each-ref", "--format=%(refname:short)", "refs/heads/"+prefix)
	if err != nil {
		return err
	}
	fmt.Fprint(c.git.stdout, out)
	return nil
}

func (c CLI) publish(kind, name string) error {
	if err := c.git.ensureRepo(); err != nil {
		return err
	}
	cfg := c.loadConfig()
	_, prefix, err := c.flowParts(cfg, kind)
	if err != nil {
		return err
	}
	branch, err := c.branchToFinish(prefix, name)
	if err != nil {
		return err
	}
	if !c.git.branchExists(branch) {
		return fmt.Errorf("branch %q does not exist", branch)
	}
	remote := cfg.Remote
	if err := c.git.run("push", "--set-upstream", remote, branch); err != nil {
		return err
	}
	fmt.Fprintf(c.git.stdout, "Published %s %s to %s\n", kind, branch, remote)
	return nil
}

func (c CLI) track(kind, name string) error {
	if err := c.git.ensureRepo(); err != nil {
		return err
	}
	if err := c.git.ensureClean(); err != nil {
		return err
	}
	cfg := c.loadConfig()
	_, prefix, err := c.flowParts(cfg, kind)
	if err != nil {
		return err
	}
	branch := fullBranchName(prefix, name)
	if err := c.git.validateBranch(branch); err != nil {
		return err
	}
	if c.git.branchExists(branch) {
		return fmt.Errorf("branch %q already exists", branch)
	}
	remote := cfg.Remote
	if err := c.git.run("checkout", "-b", branch, "--track", remote+"/"+branch); err != nil {
		return err
	}
	fmt.Fprintf(c.git.stdout, "Tracking %s %s from %s\n", kind, branch, remote)
	return nil
}

func (c CLI) status(args []string) error {
	if len(args) != 0 {
		return errors.New("status does not take arguments")
	}
	if err := c.git.ensureRepo(); err != nil {
		return err
	}
	cfg := c.loadConfig()
	branch, err := c.git.currentBranch()
	if err != nil {
		branch = "(detached)"
	}
	fmt.Fprintf(c.git.stdout, "main=%s\ndevelop=%s\nfeaturePrefix=%s\nreleasePrefix=%s\nhotfixPrefix=%s\ncurrent=%s\n", cfg.Main, cfg.Develop, cfg.FeaturePrefix, cfg.ReleasePrefix, cfg.HotfixPrefix, branch)
	return nil
}

func (c CLI) defaultConfig() Config {
	main := "main"
	if !c.git.branchExists("main") && c.git.branchExists("master") {
		main = "master"
	}
	return Config{
		Main:          main,
		Develop:       "develop",
		FeaturePrefix: "feature/",
		ReleasePrefix: "release/",
		HotfixPrefix:  "hotfix/",
		Remote:        "origin",
	}
}

func (c CLI) loadConfig() Config {
	cfg := c.defaultConfig()
	cfg.Main = c.configOrDefault("kitflow.branch.main", cfg.Main)
	cfg.Develop = c.configOrDefault("kitflow.branch.develop", cfg.Develop)
	cfg.FeaturePrefix = c.configOrDefault("kitflow.prefix.feature", cfg.FeaturePrefix)
	cfg.ReleasePrefix = c.configOrDefault("kitflow.prefix.release", cfg.ReleasePrefix)
	cfg.HotfixPrefix = c.configOrDefault("kitflow.prefix.hotfix", cfg.HotfixPrefix)
	cfg.Remote = c.configOrDefault("kitflow.remote", cfg.Remote)
	return cfg
}

func (c CLI) configOrDefault(key, fallback string) string {
	value, err := c.git.output("config", "--get", key)
	if err != nil {
		return fallback
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func (c CLI) saveConfig(cfg Config) error {
	pairs := [][2]string{
		{"kitflow.branch.main", cfg.Main},
		{"kitflow.branch.develop", cfg.Develop},
		{"kitflow.prefix.feature", cfg.FeaturePrefix},
		{"kitflow.prefix.release", cfg.ReleasePrefix},
		{"kitflow.prefix.hotfix", cfg.HotfixPrefix},
	}
	for _, pair := range pairs {
		if err := c.git.run("config", "--local", pair[0], pair[1]); err != nil {
			return err
		}
	}
	return nil
}

func (c CLI) validateConfig(cfg Config) error {
	if err := c.git.validateBranch(cfg.Main); err != nil {
		return fmt.Errorf("invalid main branch: %w", err)
	}
	if err := c.git.validateBranch(cfg.Develop); err != nil {
		return fmt.Errorf("invalid develop branch: %w", err)
	}
	for label, prefix := range map[string]string{"feature": cfg.FeaturePrefix, "release": cfg.ReleasePrefix, "hotfix": cfg.HotfixPrefix} {
		if prefix == "" {
			return fmt.Errorf("%s prefix cannot be empty", label)
		}
		if err := c.git.validateBranch(prefix + "example"); err != nil {
			return fmt.Errorf("invalid %s prefix: %w", label, err)
		}
	}
	return nil
}

func (c CLI) flowParts(cfg Config, kind string) (string, string, error) {
	switch kind {
	case "feature":
		return cfg.Develop, cfg.FeaturePrefix, nil
	case "release":
		return cfg.Develop, cfg.ReleasePrefix, nil
	case "hotfix":
		return cfg.Main, cfg.HotfixPrefix, nil
	default:
		return "", "", fmt.Errorf("unknown flow kind %q", kind)
	}
}

func (c CLI) branchToFinish(prefix, name string) (string, error) {
	if name != "" {
		return fullBranchName(prefix, name), nil
	}
	current, err := c.git.currentBranch()
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(current, prefix) {
		return "", fmt.Errorf("current branch %q does not have prefix %q", current, prefix)
	}
	return current, nil
}

func fullBranchName(prefix, name string) string {
	if strings.HasPrefix(name, prefix) {
		return name
	}
	return prefix + name
}

func (g *Git) ensureRepo() error {
	out, err := g.output("rev-parse", "--is-inside-work-tree")
	if err != nil {
		return errors.New("not inside a git work tree")
	}
	if strings.TrimSpace(out) != "true" {
		return errors.New("not inside a git work tree")
	}
	return nil
}

func (g *Git) ensureClean() error {
	out, err := g.output("status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "" {
		return errors.New("working tree is not clean")
	}
	return nil
}

func (g *Git) hasCommits() bool {
	return g.quiet("rev-parse", "--verify", "HEAD")
}

func (g *Git) branchExists(branch string) bool {
	return g.quiet("show-ref", "--verify", "--quiet", "refs/heads/"+branch)
}

func (g *Git) tagExists(tag string) bool {
	return g.quiet("show-ref", "--verify", "--quiet", "refs/tags/"+tag)
}

func (g *Git) validateBranch(branch string) error {
	_, err := g.output("check-ref-format", "--branch", branch)
	if err != nil {
		return fmt.Errorf("%q is not a valid branch name", branch)
	}
	return nil
}

func (g *Git) validateTag(tag string) error {
	_, err := g.output("check-ref-format", "refs/tags/"+tag)
	if err != nil {
		return fmt.Errorf("%q is not a valid tag name", tag)
	}
	return nil
}

func (g *Git) currentBranch() (string, error) {
	out, err := g.output("branch", "--show-current")
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(out)
	if branch == "" {
		return "", errors.New("detached HEAD")
	}
	return branch, nil
}

func (g *Git) run(args ...string) error {
	if g.verbose || g.dryRun {
		fmt.Fprintf(g.stderr, "git %s\n", joinArgs(args))
	}
	if g.dryRun {
		return nil
	}
	cmd := exec.Command("git", args...)
	cmd.Stdout = g.stdout
	cmd.Stderr = g.stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s failed: %w", joinArgs(args), err)
	}
	return nil
}

func (g *Git) output(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("git %s failed: %s", joinArgs(args), message)
	}
	return string(out), nil
}

func (g *Git) quiet(args ...string) bool {
	cmd := exec.Command("git", args...)
	return cmd.Run() == nil
}

func joinArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		if arg == "" || strings.ContainsAny(arg, " \t\n\"'") {
			quoted[i] = strconv.Quote(arg)
		} else {
			quoted[i] = arg
		}
	}
	return strings.Join(quoted, " ")
}
