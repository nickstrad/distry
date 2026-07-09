package problems

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadDirHappyPath(t *testing.T) {
	got, err := LoadDir(validFS())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(got))
	}

	problem := got[0]
	if problem.Slug != "perfect-link" {
		t.Fatalf("expected slug perfect-link, got %q", problem.Slug)
	}
	if problem.DescriptionMD != "# Perfect Link\n" {
		t.Fatalf("unexpected description %q", problem.DescriptionMD)
	}
	if problem.Templates["solution.go"] != "package solution\n" {
		t.Fatalf("unexpected template contents %q", problem.Templates["solution.go"])
	}
	if problem.RunConfig.Seeds[0] != 1 || problem.RunConfig.TimeoutSeconds != 30 {
		t.Fatalf("unexpected run config %+v", problem.RunConfig)
	}
}

func TestLoadDirValidatesDuplicateSlugs(t *testing.T) {
	fsys := validFS()
	fsys["copy/manifest.yaml"] = &fstest.MapFile{Data: []byte(validManifest())}
	fsys["copy/description.md"] = &fstest.MapFile{Data: []byte("# Copy\n")}
	fsys["copy/template/solution.go"] = &fstest.MapFile{Data: []byte("package solution\n")}

	_, err := LoadDir(fsys)
	assertErrContains(t, err, "duplicate slug")
}

func TestLoadDirValidatesTemplateExists(t *testing.T) {
	fsys := validFS()
	delete(fsys, "perfect-link/template/solution.go")

	_, err := LoadDir(fsys)
	assertErrContains(t, err, "read template")
}

func TestLoadDirValidatesEntrypointListed(t *testing.T) {
	fsys := validFS()
	fsys["perfect-link/manifest.yaml"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(validManifest(), "entrypoint: solution.go", "entrypoint: main.go"))}

	_, err := LoadDir(fsys)
	assertErrContains(t, err, "entrypoint must be listed")
}

func TestLoadDirValidatesDifficulty(t *testing.T) {
	fsys := validFS()
	fsys["perfect-link/manifest.yaml"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(validManifest(), "difficulty: easy", "difficulty: spicy"))}

	_, err := LoadDir(fsys)
	assertErrContains(t, err, "invalid difficulty")
}

func TestLoadDirValidatesSeeds(t *testing.T) {
	fsys := validFS()
	fsys["perfect-link/manifest.yaml"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(validManifest(), "seeds: [1, 2]", "seeds: []"))}

	_, err := LoadDir(fsys)
	assertErrContains(t, err, "at least one seed")
}

func validFS() fstest.MapFS {
	return fstest.MapFS{
		"perfect-link/manifest.yaml":        {Data: []byte(validManifest())},
		"perfect-link/description.md":       {Data: []byte("# Perfect Link\n")},
		"perfect-link/template/solution.go": {Data: []byte("package solution\n")},
		"perfect-link/harness/harness.go":   {Data: []byte("package harness\n")},
	}
}

func validManifest() string {
	return `slug: perfect-link
title: Perfect Point-to-Point Link
difficulty: easy
language: go
tags: [links, retransmission]
order: 1
entrypoint: solution.go
templates:
  - solution.go
runs:
  seeds: [1, 2]
  timeout_seconds: 30
`
}

func assertErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %q", want, err.Error())
	}
}
