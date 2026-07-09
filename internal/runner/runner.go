package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"distry/internal/problems"
	"distry/internal/submissions"
	"distry/pkg/simtest"
)

type GoRunner struct {
	repoRoot string
	timeout  time.Duration
}

type buildResult struct {
	output string
	err    error
}

func NewGoRunner(repoRoot string) *GoRunner {
	return &GoRunner{repoRoot: repoRoot, timeout: 60 * time.Second}
}

func (r *GoRunner) Compile(ctx context.Context, ws submissions.Workspace) (submissions.CompileResult, error) {
	dir, err := r.prepare(ws)
	if err != nil {
		return submissions.CompileResult{}, err
	}
	defer os.RemoveAll(dir)

	result := build(ctx, dir, r.timeout)
	return submissions.CompileResult{Output: result.output}, result.err
}

func (r *GoRunner) RunSeed(ctx context.Context, ws submissions.Workspace, seed int64, opts submissions.RunSeedOptions) (simtest.Report, error) {
	dir, err := r.prepare(ws)
	if err != nil {
		return simtest.Report{}, err
	}
	defer os.RemoveAll(dir)
	if result := build(ctx, dir, r.timeout); result.err != nil {
		return simtest.Report{}, errors.New(result.output)
	}
	timeout := seedTimeout(ws.Problem)
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := []string{"-seed", fmt.Sprint(seed)}
	if opts.FullTrace {
		args = append(args, "-full-trace")
	}
	cmd := exec.CommandContext(runCtx, filepath.Join(dir, "run"), args...)
	cmd.Dir = dir
	cmd.Env = runnerEnv()
	output, err := cmd.CombinedOutput()
	if runCtx.Err() != nil {
		return simtest.Report{}, fmt.Errorf("seed %d timed out", seed)
	}
	if err != nil {
		return simtest.Report{}, fmt.Errorf("seed %d failed: %s", seed, cleanOutput(string(output), dir))
	}
	var report simtest.Report
	if err := json.Unmarshal(output, &report); err != nil {
		return simtest.Report{}, fmt.Errorf("seed %d emitted invalid report JSON: %w", seed, err)
	}
	return report, nil
}

func (r *GoRunner) prepare(ws submissions.Workspace) (string, error) {
	if ws.Problem.Language != "go" {
		return "", fmt.Errorf("unsupported language %q", ws.Problem.Language)
	}
	dir, err := os.MkdirTemp("", "distry-submission-*")
	if err != nil {
		return "", fmt.Errorf("create submission workspace: %w", err)
	}
	if err := writeWorkspace(dir, r.repoRoot, ws.Problem, ws.Files); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func writeWorkspace(dir, repoRoot string, problem problems.Problem, files map[string]string) error {
	if err := os.MkdirAll(filepath.Join(dir, "solution"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "harness"), 0o755); err != nil {
		return err
	}
	goMod := fmt.Sprintf("module submission\n\ngo 1.26.4\n\nrequire distry v0.0.0\n\nreplace distry => %s\n", filepath.ToSlash(repoRoot))
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		return fmt.Errorf("write go.mod: %w", err)
	}
	for name, contents := range files {
		if err := safeWrite(filepath.Join(dir, "solution"), name, contents); err != nil {
			return err
		}
	}
	harnessRoot := filepath.Join(repoRoot, "problems", problem.Slug, "harness")
	entries, err := os.ReadDir(harnessRoot)
	if err != nil {
		return fmt.Errorf("read harness files: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(harnessRoot, entry.Name()))
		if err != nil {
			return fmt.Errorf("read harness file %q: %w", entry.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(dir, "harness", entry.Name()), contents, 0o644); err != nil {
			return fmt.Errorf("write harness file %q: %w", entry.Name(), err)
		}
	}
	mainGo := `package main

import (
	"encoding/json"
	"flag"
	"os"
	"reflect"

	"distry/pkg/sim"
	"submission/harness"
	solution "submission/solution"
)

func main() {
	seed := flag.Int64("seed", 1, "seed")
	fullTrace := flag.Bool("full-trace", false, "include full trace")
	flag.Parse()
	newNode := func(deps harness.Deps) sim.Node {
		return solution.New(adaptDeps(deps))
	}
	report := harness.Run(*seed, newNode, harness.RunOptions{FullTrace: *fullTrace})
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		panic(err)
	}
}

func adaptDeps(deps harness.Deps) solution.Deps {
	var out solution.Deps
	src := reflect.ValueOf(deps)
	dst := reflect.ValueOf(&out).Elem()
	for i := 0; i < dst.NumField(); i++ {
		dstField := dst.Field(i)
		srcField := src.FieldByName(dst.Type().Field(i).Name)
		if srcField.IsValid() && srcField.Type().AssignableTo(dstField.Type()) {
			dstField.Set(srcField)
		}
	}
	return out
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0o644); err != nil {
		return fmt.Errorf("write runner shim: %w", err)
	}
	return nil
}

func safeWrite(root, name, contents string) error {
	clean := filepath.Clean(name)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean) {
		return fmt.Errorf("unsafe submission filename %q", name)
	}
	path := filepath.Join(root, clean)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("write submission file %q: %w", name, err)
	}
	return nil
}

func build(ctx context.Context, dir string, timeout time.Duration) buildResult {
	buildCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(buildCtx, "go", "build", "-o", "run", ".")
	cmd.Dir = dir
	cmd.Env = runnerEnv()
	output, err := cmd.CombinedOutput()
	clean := cleanOutput(string(output), dir)
	if buildCtx.Err() != nil {
		return buildResult{output: strings.TrimSpace(clean + "\ncompile timed out"), err: buildCtx.Err()}
	}
	return buildResult{output: clean, err: err}
}

func seedTimeout(problem problems.Problem) time.Duration {
	timeout := time.Duration(problem.RunConfig.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		return 10 * time.Second
	}
	return timeout
}

func runnerEnv() []string {
	env := os.Environ()
	env = append(env, "CGO_ENABLED=0", "GOMEMLIMIT=256MiB", "GOPROXY=off", "GOSUMDB=off", "GOFLAGS=-mod=mod")
	return env
}

func cleanOutput(output, workspace string) string {
	output = strings.ReplaceAll(output, workspace, "$WORKSPACE")
	return strings.TrimSpace(output)
}
