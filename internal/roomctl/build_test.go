package roomctl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuild_ProducesTarAndManifest(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	roomDir := filepath.Join("rooms", "cache-stampede")
	writeEngineARoomForBuild(t, roomDir)

	res, err := BuildRoom(roomDir, 7)
	if err != nil {
		t.Fatalf("BuildRoom() error = %v", err)
	}
	if _, err := os.Stat(res.BundlePath); err != nil {
		t.Fatalf("bundle.tar missing: %v", err)
	}
	if _, err := os.Stat(res.ManifestPath); err != nil {
		t.Fatalf("manifest.json missing: %v", err)
	}
}

func TestBuild_DeterministicHash(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	roomDir := filepath.Join("rooms", "cache-stampede")
	writeEngineARoomForBuild(t, roomDir)

	res1, err := BuildRoom(roomDir, 1)
	if err != nil {
		t.Fatalf("first BuildRoom() error = %v", err)
	}
	bundle1, err := os.ReadFile(res1.BundlePath)
	if err != nil {
		t.Fatalf("ReadFile(first bundle): %v", err)
	}

	res2, err := BuildRoom(roomDir, 1)
	if err != nil {
		t.Fatalf("second BuildRoom() error = %v", err)
	}
	bundle2, err := os.ReadFile(res2.BundlePath)
	if err != nil {
		t.Fatalf("ReadFile(second bundle): %v", err)
	}

	h1 := sha256.Sum256(bundle1)
	h2 := sha256.Sum256(bundle2)
	if hex.EncodeToString(h1[:]) != hex.EncodeToString(h2[:]) {
		t.Fatalf("hash mismatch: %s != %s", hex.EncodeToString(h1[:]), hex.EncodeToString(h2[:]))
	}
	if string(bundle1) != string(bundle2) {
		t.Fatal("bundle bytes differ across identical builds")
	}
}

func TestBuild_ManifestContainsExpectedFields(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	roomDir := filepath.Join("rooms", "cache-stampede")
	writeEngineARoomForBuild(t, roomDir)

	res, err := BuildRoom(roomDir, 42)
	if err != nil {
		t.Fatalf("BuildRoom() error = %v", err)
	}

	b, err := os.ReadFile(res.ManifestPath)
	if err != nil {
		t.Fatalf("ReadFile(manifest): %v", err)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("Unmarshal(manifest): %v", err)
	}

	if m.SchemaVersion != 1 || m.Slug != "cache-stampede" || m.Engine != "A" || m.Version != 42 {
		t.Fatalf("unexpected manifest core fields: %+v", m)
	}
	if m.BundleHashSha256 == "" || m.BuiltAt == "" {
		t.Fatalf("manifest hash/builtAt empty: %+v", m)
	}
	if _, err := time.Parse(time.RFC3339, m.BuiltAt); err != nil {
		t.Fatalf("BuiltAt not RFC3339: %v", err)
	}
	if len(m.Files) == 0 {
		t.Fatal("manifest files empty")
	}
}

func TestBuild_FailsOnInvalidRoom(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	roomDir := filepath.Join("rooms", "broken-room")
	mustMkdir(t, roomDir)

	if _, err := BuildRoom(roomDir, 1); err == nil {
		t.Fatal("expected BuildRoom to fail for invalid room")
	}
}

func TestBuild_LeakCheckStub(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	roomDir := filepath.Join("rooms", "engine-b-room")
	mustMkdir(t, filepath.Join(roomDir, "engineB", "workspace", "judge", "hidden_tests"))
	mustWrite(t, filepath.Join(roomDir, "metadata.yaml"), "slug: engine-b-room\ntitle: Engine B\ndistrict: distributed-systems\ndifficulty: L1\nengine: B\ndescription: test\nestimatedMinutes: 10\ntags: [b]\n")
	mustWrite(t, filepath.Join(roomDir, "engineB", "workspace", "judge", "hidden_tests", "secret.txt"), "do not leak\n")

	if _, err := BuildRoom(roomDir, 3); err == nil {
		t.Fatal("expected leak check to fail")
	}
}

func writeEngineARoomForBuild(t *testing.T, roomDir string) {
	t.Helper()
	mustMkdir(t, filepath.Join(roomDir, "engineA"))
	mustWrite(t, filepath.Join(roomDir, "metadata.yaml"), "slug: cache-stampede\ntitle: Cache Stampede\ndistrict: distributed-systems\ndifficulty: L1\nengine: A\ndescription: test\nestimatedMinutes: 15\ntags: [caching]\n")
	mustWrite(t, filepath.Join(roomDir, "engineA", "scenario.yaml"), "topology: []\n")
	mustWrite(t, filepath.Join(roomDir, "engineA", "actions.yaml"), "actions: []\n")
	mustWrite(t, filepath.Join(roomDir, "engineA", "signals.yaml"), "metrics: []\n")
	mustWrite(t, filepath.Join(roomDir, "engineA", "win_checks.yaml"), "checks: []\n")
}
