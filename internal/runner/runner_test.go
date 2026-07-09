package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"distry/internal/problems"
	"distry/internal/submissions"
)

func TestGoRunnerCompilesAndRunsSeed(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestFile(t, repoRoot, "go.mod", "module distry\n\ngo 1.26.4\n")
	writeTestFile(t, repoRoot, "pkg/simtest/simtest.go", `package simtest

const ReportVersion = 1

type Report struct {
	Version int `+"`json:\"v\"`"+`
	Seed int64 `+"`json:\"seed\"`"+`
	Passed bool `+"`json:\"passed\"`"+`
}
`)
	writeTestFile(t, repoRoot, "problems/toy/harness/harness.go", `package harness

import (
	"distry/pkg/simtest"
	"submission/solution"
)

func Run(seed int64) *simtest.Report {
	return &simtest.Report{Version: simtest.ReportVersion, Seed: seed, Passed: solution.OK()}
}
`)
	r := NewGoRunner(repoRoot)
	ws := submissions.Workspace{
		Problem: problems.Problem{
			Slug:      "toy",
			Language:  "go",
			RunConfig: problems.RunConfig{TimeoutSeconds: 5},
		},
		Files: map[string]string{"solution.go": "package solution\n\nfunc OK() bool { return true }\n"},
	}

	compile, err := r.Compile(context.Background(), ws)
	if err != nil {
		t.Fatalf("compile failed: %v\n%s", err, compile.Output)
	}
	report, err := r.RunSeed(context.Background(), ws, 42)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || report.Seed != 42 {
		t.Fatalf("unexpected report %+v", report)
	}
}

func TestGoRunnerCleansCompileOutput(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestFile(t, repoRoot, "go.mod", "module distry\n\ngo 1.26.4\n")
	writeTestFile(t, repoRoot, "problems/toy/harness/harness.go", "package harness\n")
	r := NewGoRunner(repoRoot)
	ws := submissions.Workspace{
		Problem: problems.Problem{Slug: "toy", Language: "go"},
		Files:   map[string]string{"solution.go": "package solution\n\nfunc Broken( {\n"},
	}

	compile, err := r.Compile(context.Background(), ws)
	if err == nil {
		t.Fatal("expected compile error")
	}
	if strings.Contains(compile.Output, repoRoot) {
		t.Fatalf("compile output leaked repo path: %s", compile.Output)
	}
}

func writeTestFile(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
