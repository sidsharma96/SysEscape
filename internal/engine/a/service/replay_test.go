package service

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReplayParityAndGolden(t *testing.T) {
	s := loadSpec(t)
	live := NewEngine(s)
	must(t, live.ApplyAction("enable_singleflight", strptr("r1")))
	must(t, live.Tick())
	must(t, live.ApplyAction("enable_stale_while_revalidate", strptr("r2")))
	must(t, live.Tick())
	must(t, live.Tick())
	must(t, live.Tick())
	r1, err := Replay(s, live.Log())
	if err != nil || !reflect.DeepEqual(live.Snapshot(), r1.Snapshot()) {
		t.Fatalf("replay parity failed err=%v", err)
	}
	var g struct {
		Log      []LogEntry `json:"log"`
		Expected Snapshot   `json:"expected"`
	}
	b, err := os.ReadFile(filepath.Join("testdata", "golden_replay.json"))
	if err != nil || json.Unmarshal(b, &g) != nil {
		t.Fatalf("load golden failed")
	}
	r2, err := Replay(s, g.Log)
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range g.Expected.Metrics {
		if got := r2.Snapshot().Metrics[k]; math.Abs(got-want) > 1e-9 {
			t.Fatalf("metric %s got=%v want=%v", k, got, want)
		}
	}
	if got := r2.Snapshot(); got.Tick != g.Expected.Tick || got.Won != g.Expected.Won {
		t.Fatalf("snapshot mismatch got=%+v want=%+v", got, g.Expected)
	}
}
