package service

import (
	"math"
	"sort"
)

func NewEngine(spec SimulationSpec) *Engine {
	m := make(map[string]float64, len(spec.InitialMetrics))
	for k, v := range spec.InitialMetrics {
		m[k] = v
	}
	return &Engine{spec: spec, metrics: m, nextSeq: 1}
}

func (e *Engine) Tick() error {
	if e.won {
		return ErrRunCompleted
	}
	e.appendLog(ActionTypeTick, nil, nil)
	keys := make([]string, 0, len(e.metrics))
	for k := range e.metrics {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, metric := range keys {
		if eff, ok := e.resolveEffect(metric); ok {
			e.metrics[metric] = stepToward(e.metrics[metric], eff.Target, eff.Rate)
		}
	}
	e.tick++
	e.won = winSatisfied(e.metrics, e.spec.WinChecks)
	return nil
}

func (e *Engine) Snapshot() Snapshot {
	m := map[string]float64{}
	for k, v := range e.metrics {
		m[k] = v
	}
	return Snapshot{Tick: e.tick, Won: e.won, Metrics: m}
}

func (e *Engine) Log() []LogEntry { out := make([]LogEntry, len(e.log)); copy(out, e.log); return out }

func (e *Engine) resolveEffect(metric string) (Effect, bool) {
	var chosen Effect
	ok := false
	for _, ev := range e.spec.Events {
		if ev.AtTick > e.tick {
			continue
		}
		for _, eff := range ev.Effects {
			if eff.Metric == metric {
				chosen, ok = eff, true
			}
		}
	}
	for _, actionKey := range e.activeActions {
		for _, eff := range e.spec.ActionEffects[actionKey].Effects {
			if eff.Metric == metric {
				chosen, ok = eff, true
			}
		}
	}
	return chosen, ok
}

func (e *Engine) appendLog(actionType string, actionKey, clientRequestID *string) {
	e.log = append(e.log, LogEntry{Seq: e.nextSeq, ActionType: actionType, ActionKey: clone(actionKey), ClientRequestID: clone(clientRequestID)})
	e.nextSeq++
}

func stepToward(current, target, rate float64) float64 {
	if rate < 0 {
		rate = -rate
	}
	delta := target - current
	if math.Abs(delta) <= rate {
		return target
	}
	if delta > 0 {
		return current + rate
	}
	return current - rate
}

func clone(in *string) *string {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}
