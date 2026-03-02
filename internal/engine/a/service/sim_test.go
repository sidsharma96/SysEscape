package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDeterministicScript(t *testing.T) {
	s := loadSpec(t)
	a, b := NewEngine(s), NewEngine(s)
	for _, e := range []*Engine{a, b} {
		must(t, e.ApplyAction("enable_singleflight", strptr("r1")))
		must(t, e.Tick())
		must(t, e.ApplyAction("enable_stale_while_revalidate", strptr("r2")))
		must(t, e.Tick())
		must(t, e.Tick())
	}
	if !reflect.DeepEqual(a.Snapshot(), b.Snapshot()) || !reflect.DeepEqual(a.Log(), b.Log()) {
		t.Fatalf("non-deterministic result")
	}
}

func loadSpec(t *testing.T) SimulationSpec {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "simulation_fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s SimulationSpec
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	return s
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func strptr(s string) *string { return &s }
