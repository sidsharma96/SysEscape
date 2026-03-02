package service

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEngineABundleFromTar_Success(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "cache_stampede_bundle.tar"))
	if err != nil {
		t.Fatalf("ReadFile(bundle): %v", err)
	}

	got, err := LoadEngineABundleFromTar(b)
	if err != nil {
		t.Fatalf("LoadEngineABundleFromTar() error = %v", err)
	}

	if len(got.Actions) != 4 {
		t.Fatalf("actions len = %d, want 4", len(got.Actions))
	}
	if got.Simulation.InitialMetrics["cache_hit_rate"] != 0.97 {
		t.Fatalf("initial metric cache_hit_rate = %v, want 0.97", got.Simulation.InitialMetrics["cache_hit_rate"])
	}
	if _, ok := got.Simulation.ActionEffects["enable_singleflight"]; !ok {
		t.Fatal("missing action effect for enable_singleflight")
	}
	if len(got.Simulation.WinChecks) != 3 {
		t.Fatalf("win checks len = %d, want 3", len(got.Simulation.WinChecks))
	}
}

func TestLoadEngineABundleFromTar_MissingSimulationYAML(t *testing.T) {
	_, err := LoadEngineABundleFromTar(minimalBundleTar(t, ""))
	if err == nil || !strings.Contains(err.Error(), "simulation.yaml") {
		t.Fatalf("expected simulation.yaml error, got %v", err)
	}
}

func TestLoadEngineABundleFromTar_InvalidSimulationYAML(t *testing.T) {
	_, err := LoadEngineABundleFromTar(minimalBundleTar(t, "simulation:\n  tick_interval_ms: [broken\n"))
	if err == nil || !strings.Contains(err.Error(), "simulation.yaml") {
		t.Fatalf("expected simulation.yaml parse error, got %v", err)
	}
}

func minimalBundleTar(t *testing.T, simulationYAML string) []byte {
	t.Helper()
	files := map[string]string{
		"engineA/scenario.yaml":   "topology: []\ninitialMetrics:\n  cache_hit_rate: 0.97\n  db_connections: 20\nconfig: {}\n",
		"engineA/actions.yaml":    "actions:\n  - key: enable_singleflight\n",
		"engineA/signals.yaml":    "metrics: []\nlogPatterns: []\ntraceSpans: []\n",
		"engineA/win_checks.yaml": "checks:\n  - metric: cache_hit_rate\n    op: \">\"\n    value: 0.95\n",
	}
	if simulationYAML != "" {
		files["engineA/simulation.yaml"] = simulationYAML
	}
	return tarFromFiles(t, files)
}

func tarFromFiles(t *testing.T, files map[string]string) []byte {
	t.Helper()
	buf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buf)
	for name, raw := range files {
		content := []byte(raw)
		if err := tw.WriteHeader(&tar.Header{
			Name:     name,
			Typeflag: tar.TypeReg,
			Mode:     0o644,
			Size:     int64(len(content)),
		}); err != nil {
			t.Fatalf("WriteHeader(%s): %v", name, err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("Write(%s): %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("Close tar: %v", err)
	}
	return buf.Bytes()
}
