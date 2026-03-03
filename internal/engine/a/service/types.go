package service

import "errors"

var (
	ErrUnknownAction    = errors.New("unknown action")
	ErrRunCompleted     = errors.New("run completed")
	ErrInvalidReplayLog = errors.New("invalid replay log")
)

const (
	ActionTypePlayer = "player"
	ActionTypeTick   = "tick"
)

type Effect struct {
	Metric string  `json:"metric"`
	Target float64 `json:"target"`
	Rate   float64 `json:"rate"`
}

type TimedEvent struct {
	AtTick  int      `json:"at_tick"`
	Effects []Effect `json:"effects"`
}

type ActionEffectSet struct {
	Effects []Effect `json:"effects"`
}

type WinCheck struct {
	Metric string  `json:"metric"`
	Op     string  `json:"op"`
	Value  float64 `json:"value"`
}

type SimulationSpec struct {
	TickIntervalMS int                        `json:"tick_interval_ms"`
	DurationTicks  int                        `json:"duration_ticks"`
	Seed           int64                      `json:"seed"`
	InitialMetrics map[string]float64         `json:"initial_metrics"`
	Events         []TimedEvent               `json:"events"`
	ActionEffects  map[string]ActionEffectSet `json:"action_effects"`
	WinChecks      []WinCheck                 `json:"win_checks"`
}

type LogEntry struct {
	Seq             int     `json:"seq"`
	ActionType      string  `json:"action_type"`
	ActionKey       *string `json:"action_key,omitempty"`
	ClientRequestID *string `json:"client_request_id,omitempty"`
}

type Snapshot struct {
	Tick    int                `json:"tick"`
	Won     bool               `json:"won"`
	Metrics map[string]float64 `json:"metrics"`
}

type Engine struct {
	spec          SimulationSpec
	metrics       map[string]float64
	tick          int
	won           bool
	nextSeq       int
	log           []LogEntry
	activeActions []string
}
