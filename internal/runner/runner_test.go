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
	Trace []map[string]int `+"`json:\"trace,omitempty\"`"+`
}
`)
	writeTestFile(t, repoRoot, "pkg/sim/sim.go", `package sim

type Node interface{}
`)
	writeTestFile(t, repoRoot, "problems/toy/harness/harness.go", `package harness

import (
	"distry/pkg/sim"
	"distry/pkg/simtest"
)

type Deps struct {
	OK bool
	Unused string
}

type RunOptions struct {
	FullTrace bool
}

type NewNodeFunc func(Deps) sim.Node

func Run(seed int64, newNode NewNodeFunc, opts ...RunOptions) *simtest.Report {
	passed, _ := newNode(Deps{OK: true, Unused: "ignored"}).(bool)
	report := &simtest.Report{Version: simtest.ReportVersion, Seed: seed, Passed: passed}
	if len(opts) > 0 && opts[0].FullTrace {
		report.Trace = []map[string]int{{"seq": 1}}
	}
	return report
}
`)
	r := NewGoRunner(repoRoot)
	ws := submissions.Workspace{
		Problem: problems.Problem{
			Slug:      "toy",
			Language:  "go",
			RunConfig: problems.RunConfig{TimeoutSeconds: 5},
		},
		Files: map[string]string{"solution.go": "package solution\n\ntype Deps struct { OK bool }\n\nfunc New(deps Deps) any { return deps.OK }\n"},
	}

	compile, err := r.Compile(context.Background(), ws)
	if err != nil {
		t.Fatalf("compile failed: %v\n%s", err, compile.Output)
	}
	report, err := r.RunSeed(context.Background(), ws, 42, submissions.RunSeedOptions{FullTrace: true})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || report.Seed != 42 || len(report.Trace) == 0 {
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
