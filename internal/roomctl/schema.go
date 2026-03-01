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

type ActionList struct {
	Actions []map[string]any `yaml:"actions"`
}

type Signals struct {
	Metrics     []string `yaml:"metrics"`
	LogPatterns []string `yaml:"logPatterns"`
	TraceSpans  []string `yaml:"traceSpans"`
}

type WinChecks struct {
	Checks []map[string]any `yaml:"checks"`
}
