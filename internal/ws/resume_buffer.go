package ws

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Delta struct {
	Seq     int             `json:"seq"`
	Payload json.RawMessage `json:"payload"`
}

type ResumeBuffer struct {
	mu      sync.Mutex
	cap     int
	items   []Delta
	lastSeq int
}

func NewResumeBuffer(capacity int) *ResumeBuffer {
	if capacity <= 0 {
		capacity = 1
	}
	return &ResumeBuffer{cap: capacity, items: make([]Delta, 0, capacity)}
}

func (b *ResumeBuffer) Add(delta Delta) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if delta.Seq <= b.lastSeq {
		return fmt.Errorf("non-monotonic seq")
	}
	if len(b.items) == b.cap {
		copy(b.items, b.items[1:])
		b.items[len(b.items)-1] = delta
	} else {
		b.items = append(b.items, delta)
	}
	b.lastSeq = delta.Seq
	return nil
}

func (b *ResumeBuffer) ResumeFrom(lastSeq int) ([]Delta, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) == 0 {
		return nil, false
	}
	if lastSeq >= b.lastSeq {
		return []Delta{}, false
	}
	earliest := b.items[0].Seq
	if lastSeq < earliest-1 {
		return nil, true
	}

	start := 0
	for i, d := range b.items {
		if d.Seq > lastSeq {
			start = i
			break
		}
	}
	out := make([]Delta, len(b.items[start:]))
	copy(out, b.items[start:])
	return out, false
}
