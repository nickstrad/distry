package config

import "testing"

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PORT", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected missing DATABASE_URL to fail")
	}
}

func TestLoadDefaultsPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %q", cfg.Port)
	}
}

func TestLoadUsesPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("PORT", "9090")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "9090" {
		t.Fatalf("expected port 9090, got %q", cfg.Port)
	}
}
