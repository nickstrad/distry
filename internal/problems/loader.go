package problems

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

var validDifficulties = map[string]struct{}{
	string(DifficultyEasy):   {},
	string(DifficultyMedium): {},
	string(DifficultyHard):   {},
}

func LoadDir(fsys fs.FS) ([]Problem, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("read problems dir: %w", err)
	}

	seen := make(map[string]struct{})
	loaded := make([]Problem, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		problem, err := loadProblem(fsys, entry.Name())
		if err != nil {
			return nil, err
		}
		if _, ok := seen[problem.Slug]; ok {
			return nil, fmt.Errorf("problem %q: duplicate slug", problem.Slug)
		}
		seen[problem.Slug] = struct{}{}
		loaded = append(loaded, problem)
	}

	return loaded, nil
}

func loadProblem(fsys fs.FS, dir string) (Problem, error) {
	var manifest Manifest
	manifestBytes, err := fs.ReadFile(fsys, path.Join(dir, "manifest.yaml"))
	if err != nil {
		return Problem{}, fmt.Errorf("problem %q: read manifest: %w", dir, err)
	}
	if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
		return Problem{}, fmt.Errorf("problem %q: parse manifest: %w", dir, err)
	}
	if err := validateManifest(dir, manifest); err != nil {
		return Problem{}, err
	}

	description, err := fs.ReadFile(fsys, path.Join(dir, "description.md"))
	if err != nil {
		return Problem{}, fmt.Errorf("problem %q: read description.md: %w", manifest.Slug, err)
	}

	templates := make(map[string]string, len(manifest.Templates))
	for _, name := range manifest.Templates {
		contents, err := fs.ReadFile(fsys, path.Join(dir, "template", name))
		if err != nil {
			return Problem{}, fmt.Errorf("problem %q: read template %q: %w", manifest.Slug, name, err)
		}
		templates[name] = string(contents)
	}

	return Problem{
		Slug:          manifest.Slug,
		Title:         manifest.Title,
		Difficulty:    manifest.Difficulty,
		Language:      manifest.Language,
		Tags:          manifest.Tags,
		Order:         manifest.Order,
		Entrypoint:    manifest.Entrypoint,
		DescriptionMD: string(description),
		Templates:     templates,
		RunConfig:     manifest.Runs,
	}, nil
}

func validateManifest(dir string, manifest Manifest) error {
	label := manifest.Slug
	if label == "" {
		label = dir
	}

	if strings.TrimSpace(manifest.Slug) == "" {
		return fmt.Errorf("problem %q: slug is required", dir)
	}
	if strings.TrimSpace(manifest.Title) == "" {
		return fmt.Errorf("problem %q: title is required", label)
	}
	if _, ok := validDifficulties[manifest.Difficulty]; !ok {
		return fmt.Errorf("problem %q: invalid difficulty %q", label, manifest.Difficulty)
	}
	if strings.TrimSpace(manifest.Language) == "" {
		return fmt.Errorf("problem %q: language is required", label)
	}
	if len(manifest.Templates) == 0 {
		return fmt.Errorf("problem %q: at least one template is required", label)
	}
	if strings.TrimSpace(manifest.Entrypoint) == "" {
		return fmt.Errorf("problem %q: entrypoint is required", label)
	}
	if !contains(manifest.Templates, manifest.Entrypoint) {
		return fmt.Errorf("problem %q: entrypoint must be listed in templates", label)
	}
	if len(manifest.Runs.Seeds) == 0 {
		return fmt.Errorf("problem %q: at least one seed is required", label)
	}
	if manifest.Runs.TimeoutSeconds <= 0 {
		return fmt.Errorf("problem %q: timeout_seconds must be positive", label)
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
