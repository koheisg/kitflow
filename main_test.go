package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestFullBranchName(t *testing.T) {
	tests := []struct {
		prefix string
		name   string
		want   string
	}{
		{prefix: "feature/", name: "login", want: "feature/login"},
		{prefix: "feature/", name: "feature/login", want: "feature/login"},
	}

	for _, tt := range tests {
		got := fullBranchName(tt.prefix, tt.name)
		if got != tt.want {
			t.Fatalf("fullBranchName(%q, %q) = %q, want %q", tt.prefix, tt.name, got, tt.want)
		}
	}
}

func TestHelp(t *testing.T) {
	out := new(strings.Builder)
	errOut := new(strings.Builder)
	code := runCLI([]string{"help"}, out, errOut)
	if code != 0 {
		t.Fatalf("runCLI help exited %d", code)
	}
	if !strings.Contains(out.String(), "kitflow") {
		t.Fatalf("help did not contain kitflow: %q", out.String())
	}
}

func TestReleaseAndHotfixTagLabels(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitInit(t, dir)

	runInDir(t, dir, "release", "start", "1.0.0")
	gitCommit(t, dir, "release work")
	runInDir(t, dir, "release", "finish", "1.0.0")
	if msg := tagMessage(t, dir, "1.0.0"); !strings.Contains(msg, "Release 1.0.0") {
		t.Fatalf("release tag message = %q, want to contain %q", msg, "Release 1.0.0")
	}

	runInDir(t, dir, "hotfix", "start", "1.0.1")
	gitCommit(t, dir, "hotfix work")
	runInDir(t, dir, "hotfix", "finish", "1.0.1")
	if msg := tagMessage(t, dir, "1.0.1"); !strings.Contains(msg, "Hotfix 1.0.1") {
		t.Fatalf("hotfix tag message = %q, want to contain %q", msg, "Hotfix 1.0.1")
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	git(t, dir, "init", "-b", "main")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "commit", "--allow-empty", "-m", "init")
	runInDir(t, dir, "init")
}

func gitCommit(t *testing.T, dir, message string) {
	t.Helper()
	git(t, dir, "commit", "--allow-empty", "-m", message)
}

func tagMessage(t *testing.T, dir, tag string) string {
	t.Helper()
	return git(t, dir, "tag", "-n1", tag)
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func runInDir(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(kitflowBin(t), args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("kitflow %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func kitflowBin(t *testing.T) string {
	t.Helper()
	if binOnce == "" {
		binOnce = t.TempDir() + "/kitflow"
		cmd := exec.Command("go", "build", "-o", binOnce, ".")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build kitflow failed: %v\n%s", err, out)
		}
	}
	return binOnce
}

var binOnce string
