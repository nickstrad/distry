package problems

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("problem not found")

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

type Manifest struct {
	Slug       string    `yaml:"slug"`
	Title      string    `yaml:"title"`
	Difficulty string    `yaml:"difficulty"`
	Language   string    `yaml:"language"`
	Tags       []string  `yaml:"tags"`
	Order      int       `yaml:"order"`
	Entrypoint string    `yaml:"entrypoint"`
	Templates  []string  `yaml:"templates"`
	Runs       RunConfig `yaml:"runs"`
}

type RunConfig struct {
	Seeds          []int `json:"seeds" yaml:"seeds"`
	TimeoutSeconds int   `json:"timeout_seconds" yaml:"timeout_seconds"`
}

type Problem struct {
	Slug          string            `json:"slug"`
	Title         string            `json:"title"`
	Difficulty    string            `json:"difficulty"`
	Language      string            `json:"language"`
	Tags          []string          `json:"tags"`
	Order         int               `json:"order"`
	Entrypoint    string            `json:"entrypoint"`
	DescriptionMD string            `json:"description_md"`
	Templates     map[string]string `json:"templates"`
	RunConfig     RunConfig         `json:"run_config"`
}

type Summary struct {
	Slug       string   `json:"slug"`
	Title      string   `json:"title"`
	Difficulty string   `json:"difficulty"`
	Tags       []string `json:"tags"`
	Order      int      `json:"order"`
	Solved     bool     `json:"solved,omitempty"`
}

type Repo interface {
	Upsert(context.Context, Problem) error
	List(context.Context) ([]Summary, error)
	Get(context.Context, string) (Problem, error)
}

type SolvedLister interface {
	ListSolved(context.Context, string) (map[string]bool, error)
}
