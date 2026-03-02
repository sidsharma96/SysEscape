package ws

import (
	"encoding/json"
	"testing"
)

func TestResumeBufferReplayAndSnapshotFallback(t *testing.T) {
	b := NewResumeBuffer(3)
	for i := 1; i <= 5; i++ {
		if err := b.Add(Delta{Seq: i, Payload: json.RawMessage(`{"ops":[]}`)}); err != nil {
			t.Fatalf("Add(%d) err=%v", i, err)
		}
	}

	deltas, snapshotRequired := b.ResumeFrom(2)
	if snapshotRequired {
		t.Fatalf("ResumeFrom(2) snapshotRequired=true")
	}
	if len(deltas) != 3 || deltas[0].Seq != 3 || deltas[2].Seq != 5 {
		t.Fatalf("ResumeFrom(2) deltas=%+v", deltas)
	}

	_, snapshotRequired = b.ResumeFrom(1)
	if !snapshotRequired {
		t.Fatalf("ResumeFrom(1) snapshotRequired=false")
	}

	deltas, snapshotRequired = b.ResumeFrom(5)
	if snapshotRequired || len(deltas) != 0 {
		t.Fatalf("ResumeFrom(5) deltas=%+v snapshotRequired=%v", deltas, snapshotRequired)
	}
}

func TestResumeBufferRejectsNonMonotonicSeq(t *testing.T) {
	b := NewResumeBuffer(2)
	if err := b.Add(Delta{Seq: 1}); err != nil {
		t.Fatalf("Add(1) err=%v", err)
	}
	if err := b.Add(Delta{Seq: 1}); err == nil {
		t.Fatalf("Add(non-monotonic) err=nil")
	}
}
