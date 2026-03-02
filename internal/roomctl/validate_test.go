package roomctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidate_ValidEngineARoom(t *testing.T) {
	r := setupRoom(t, "cache-stampede", validMeta("cache-stampede"), true)
	if err := ValidateRoomDir(r); err != nil {
		t.Fatalf("ValidateRoomDir() error = %v", err)
	}
}

func TestValidate_MissingMetadata(t *testing.T) {
	r := setupRoom(t, "cache-stampede", "", true)
	mustWrite(t, filepath.Join(r, "engineA", "scenario.yaml"), "topology: []\n")
	err := ValidateRoomDir(r)
	if err == nil || !strings.Contains(err.Error(), "metadata.yaml") {
		t.Fatalf("expected metadata.yaml error, got %v", err)
	}
}

func TestValidate_SlugMismatch(t *testing.T) {
	r := setupRoom(t, "cache-stampede", validMeta("wrong-slug"), true)
	err := ValidateRoomDir(r)
	if err == nil || !strings.Contains(err.Error(), "slug") {
		t.Fatalf("expected slug error, got %v", err)
	}
}

func TestValidate_MissingEngineDir(t *testing.T) {
	r := setupRoom(t, "cache-stampede", validMeta("cache-stampede"), false)
	err := ValidateRoomDir(r)
	if err == nil || !strings.Contains(err.Error(), "engineA") {
		t.Fatalf("expected engineA error, got %v", err)
	}
}

func TestValidate_InvalidDifficulty(t *testing.T) {
	r := setupRoom(t, "cache-stampede", strings.Replace(validMeta("cache-stampede"), "difficulty: L1", "difficulty: L9", 1), true)
	err := ValidateRoomDir(r)
	if err == nil || !strings.Contains(err.Error(), "difficulty") {
		t.Fatalf("expected difficulty error, got %v", err)
	}
}

func TestValidate_MissingRequiredFields(t *testing.T) {
	r := setupRoom(t, "cache-stampede", strings.Replace(validMeta("cache-stampede"), "title: Cache Stampede", "title: ''", 1), true)
	err := ValidateRoomDir(r)
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("expected title error, got %v", err)
	}
}

func TestValidate_MissingSimulationYAML(t *testing.T) {
	r := setupRoom(t, "cache-stampede", validMeta("cache-stampede"), true)
	if err := os.Remove(filepath.Join(r, "engineA", "simulation.yaml")); err != nil {
		t.Fatalf("remove simulation.yaml: %v", err)
	}
	err := ValidateRoomDir(r)
	if err == nil || !strings.Contains(err.Error(), "simulation.yaml") {
		t.Fatalf("expected simulation.yaml error, got %v", err)
	}
}

func TestValidate_ActionSimulationKeyMismatch(t *testing.T) {
	r := setupRoom(t, "cache-stampede", validMeta("cache-stampede"), true)
	mustWrite(t, filepath.Join(r, "engineA", "actions.yaml"), "actions:\n  - key: enable_singleflight\n  - key: add_jitter_to_ttl\n")
	mustWrite(t, filepath.Join(r, "engineA", "simulation.yaml"), "simulation:\n  tick_interval_ms: 1000\n  duration_ticks: 300\n  events: []\n  action_effects:\n    enable_singleflight:\n      effects: []\n")
	err := ValidateRoomDir(r)
	if err == nil || !strings.Contains(err.Error(), "action key mismatch") {
		t.Fatalf("expected action key mismatch error, got %v", err)
	}
}

func setupRoom(t *testing.T, dirName, meta string, withEngine bool) string {
	t.Helper()
	r := filepath.Join(t.TempDir(), dirName)
	mustMkdir(t, r)
	if meta != "" {
		mustWrite(t, filepath.Join(r, "metadata.yaml"), meta)
	}
	if withEngine {
		e := filepath.Join(r, "engineA")
		mustMkdir(t, e)
		mustWrite(t, filepath.Join(e, "scenario.yaml"), "topology: []\n")
		mustWrite(t, filepath.Join(e, "actions.yaml"), "actions: []\n")
		mustWrite(t, filepath.Join(e, "signals.yaml"), "metrics: []\n")
		mustWrite(t, filepath.Join(e, "win_checks.yaml"), "checks: []\n")
		mustWrite(t, filepath.Join(e, "simulation.yaml"), "simulation:\n  tick_interval_ms: 1000\n  duration_ticks: 300\n  events: []\n  action_effects: {}\n")
	}
	return r
}

func validMeta(slug string) string {
	return "slug: " + slug + "\n" +
		"title: Cache Stampede\n" +
		"district: distributed-systems\n" +
		"difficulty: L1\n" +
		"engine: A\n" +
		"description: test\n" +
		"estimatedMinutes: 15\n" +
		"tags: [caching]\n"
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
}

func mustWrite(t *testing.T, p, s string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}
