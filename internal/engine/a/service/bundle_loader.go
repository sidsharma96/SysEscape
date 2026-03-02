package service

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

type EngineABundle struct {
	Scenario   BundleScenario
	Actions    []BundleAction
	Signals    BundleSignals
	WinChecks  []WinCheck
	Simulation SimulationSpec
}

type BundleScenario struct {
	Topology       []map[string]any `yaml:"topology"`
	InitialMetrics map[string]any   `yaml:"initialMetrics"`
	Config         map[string]any   `yaml:"config"`
}

type BundleAction struct {
	Key         string `yaml:"key"`
	Description string `yaml:"description"`
}

type bundleActions struct {
	Actions []BundleAction `yaml:"actions"`
}

type BundleSignals struct {
	Metrics     []string `yaml:"metrics"`
	LogPatterns []string `yaml:"logPatterns"`
	TraceSpans  []string `yaml:"traceSpans"`
}

type bundleSimulation struct {
	Simulation struct {
		TickIntervalMS int                              `yaml:"tick_interval_ms"`
		DurationTicks  int                              `yaml:"duration_ticks"`
		Events         []bundleTimedEvent               `yaml:"events"`
		ActionEffects  map[string]bundleActionEffectSet `yaml:"action_effects"`
	} `yaml:"simulation"`
}

type bundleTimedEvent struct {
	AtTick  int            `yaml:"at_tick"`
	Effects []bundleEffect `yaml:"effects"`
}

type bundleActionEffectSet struct {
	Effects []bundleEffect `yaml:"effects"`
}

type bundleEffect struct {
	Metric string  `yaml:"metric"`
	Target float64 `yaml:"target"`
	Rate   float64 `yaml:"rate"`
}

func LoadEngineABundleFromTar(bundleTar []byte) (*EngineABundle, error) {
	required := map[string][]byte{
		"engineA/scenario.yaml":   nil,
		"engineA/actions.yaml":    nil,
		"engineA/signals.yaml":    nil,
		"engineA/win_checks.yaml": nil,
		"engineA/simulation.yaml": nil,
	}

	tr := tar.NewReader(bytes.NewReader(bundleTar))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read bundle tar: %w", err)
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		name := strings.TrimPrefix(path.Clean(h.Name), "./")
		if _, ok := required[name]; !ok {
			continue
		}
		content, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		required[name] = content
	}

	for name, content := range required {
		if content == nil {
			return nil, fmt.Errorf("missing required bundle file: %s", name)
		}
	}

	var scenario BundleScenario
	if err := yaml.Unmarshal(required["engineA/scenario.yaml"], &scenario); err != nil {
		return nil, fmt.Errorf("parse engineA/scenario.yaml: %w", err)
	}
	var actions bundleActions
	if err := yaml.Unmarshal(required["engineA/actions.yaml"], &actions); err != nil {
		return nil, fmt.Errorf("parse engineA/actions.yaml: %w", err)
	}
	var signals BundleSignals
	if err := yaml.Unmarshal(required["engineA/signals.yaml"], &signals); err != nil {
		return nil, fmt.Errorf("parse engineA/signals.yaml: %w", err)
	}
	var winChecks struct {
		Checks []WinCheck `yaml:"checks"`
	}
	if err := yaml.Unmarshal(required["engineA/win_checks.yaml"], &winChecks); err != nil {
		return nil, fmt.Errorf("parse engineA/win_checks.yaml: %w", err)
	}
	var simulation bundleSimulation
	if err := yaml.Unmarshal(required["engineA/simulation.yaml"], &simulation); err != nil {
		return nil, fmt.Errorf("parse engineA/simulation.yaml: %w", err)
	}

	initialMetrics, err := toFloatMetrics(scenario.InitialMetrics)
	if err != nil {
		return nil, fmt.Errorf("parse engineA/scenario.yaml initialMetrics: %w", err)
	}

	spec := SimulationSpec{
		InitialMetrics: initialMetrics,
		Events:         make([]TimedEvent, 0, len(simulation.Simulation.Events)),
		ActionEffects:  make(map[string]ActionEffectSet, len(simulation.Simulation.ActionEffects)),
		WinChecks:      winChecks.Checks,
	}
	for _, ev := range simulation.Simulation.Events {
		effects := make([]Effect, 0, len(ev.Effects))
		for _, eff := range ev.Effects {
			effects = append(effects, Effect(eff))
		}
		spec.Events = append(spec.Events, TimedEvent{AtTick: ev.AtTick, Effects: effects})
	}
	for key, actionSet := range simulation.Simulation.ActionEffects {
		effects := make([]Effect, 0, len(actionSet.Effects))
		for _, eff := range actionSet.Effects {
			effects = append(effects, Effect(eff))
		}
		spec.ActionEffects[key] = ActionEffectSet{Effects: effects}
	}

	return &EngineABundle{
		Scenario:   scenario,
		Actions:    actions.Actions,
		Signals:    signals,
		WinChecks:  winChecks.Checks,
		Simulation: spec,
	}, nil
}

func toFloatMetrics(src map[string]any) (map[string]float64, error) {
	out := make(map[string]float64, len(src))
	for key, raw := range src {
		switch v := raw.(type) {
		case int:
			out[key] = float64(v)
		case float64:
			out[key] = v
		default:
			return nil, fmt.Errorf("metric %s must be numeric, got %T", key, raw)
		}
	}
	return out, nil
}
