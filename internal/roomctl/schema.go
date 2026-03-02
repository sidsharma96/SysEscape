package roomctl

type Metadata struct {
	Slug             string   `yaml:"slug"`
	Title            string   `yaml:"title"`
	District         string   `yaml:"district"`
	Difficulty       string   `yaml:"difficulty"`
	Engine           string   `yaml:"engine"`
	Description      string   `yaml:"description"`
	EstimatedMinutes int      `yaml:"estimatedMinutes"`
	Tags             []string `yaml:"tags"`
}

type Scenario struct {
	Topology       []map[string]any `yaml:"topology"`
	InitialMetrics map[string]any   `yaml:"initialMetrics"`
	Config         map[string]any   `yaml:"config"`
}

type Action struct {
	Key         string `yaml:"key"`
	Description string `yaml:"description"`
}

type ActionList struct {
	Actions []Action `yaml:"actions"`
}

type Signals struct {
	Metrics     []string `yaml:"metrics"`
	LogPatterns []string `yaml:"logPatterns"`
	TraceSpans  []string `yaml:"traceSpans"`
}

type WinChecks struct {
	Checks []map[string]any `yaml:"checks"`
}

type SimulationFile struct {
	Simulation Simulation `yaml:"simulation"`
}

type Simulation struct {
	TickIntervalMS int                         `yaml:"tick_interval_ms"`
	DurationTicks  int                         `yaml:"duration_ticks"`
	Events         []SimulationEvent           `yaml:"events"`
	ActionEffects  map[string]SimulationAction `yaml:"action_effects"`
}

type SimulationEvent struct {
	AtTick      int                `yaml:"at_tick"`
	Description string             `yaml:"description"`
	Effects     []SimulationEffect `yaml:"effects"`
}

type SimulationAction struct {
	Effects []SimulationEffect `yaml:"effects"`
}

type SimulationEffect struct {
	Metric string  `yaml:"metric"`
	Target float64 `yaml:"target"`
	Rate   float64 `yaml:"rate"`
}
