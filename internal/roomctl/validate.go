package roomctl

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func ValidateAllRooms(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read rooms root: %w", err)
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		count++
		if err := ValidateRoomDir(filepath.Join(root, e.Name())); err != nil {
			return err
		}
	}
	if count == 0 {
		return fmt.Errorf("no room directories found in %s", root)
	}
	return nil
}

func ValidateRoomDir(roomDir string) error {
	if err := mustDir(roomDir); err != nil {
		return fmt.Errorf("room dir: %w", err)
	}

	metaPath := filepath.Join(roomDir, "metadata.yaml")
	var meta Metadata
	if err := parseYAML(metaPath, &meta); err != nil {
		return fmt.Errorf("metadata.yaml: %w", err)
	}
	if err := validateMetadata(meta, filepath.Base(roomDir)); err != nil {
		return err
	}

	engineDir := "engineA"
	if meta.Engine == "B" {
		engineDir = "engineB"
	}
	enginePath := filepath.Join(roomDir, engineDir)
	if err := mustDir(enginePath); err != nil {
		return err
	}
	if meta.Engine == "A" {
		var actions ActionList
		var simulation SimulationFile
		for _, f := range []struct {
			name string
			dst  any
		}{
			{"scenario.yaml", &Scenario{}},
			{"actions.yaml", &actions},
			{"signals.yaml", &Signals{}},
			{"win_checks.yaml", &WinChecks{}},
			{"simulation.yaml", &simulation},
		} {
			if err := parseYAML(filepath.Join(enginePath, f.name), f.dst); err != nil {
				return fmt.Errorf("%s: %w", f.name, err)
			}
		}
		if err := validateActionSimulationKeyMatch(actions, simulation); err != nil {
			return err
		}
	}
	return nil
}

func validateMetadata(m Metadata, dir string) error {
	required := map[string]string{
		"slug": m.Slug, "title": m.Title, "district": m.District,
		"difficulty": m.Difficulty, "engine": m.Engine, "description": m.Description,
	}
	for k, v := range required {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("missing required field: %s", k)
		}
	}
	if m.EstimatedMinutes <= 0 {
		return fmt.Errorf("missing required field: estimatedMinutes")
	}
	if len(m.Tags) == 0 {
		return fmt.Errorf("missing required field: tags")
	}
	if m.Slug != dir {
		return fmt.Errorf("slug %q must match directory %q", m.Slug, dir)
	}
	if !allowed(m.Difficulty, "L0", "L1", "L2", "L3") {
		return fmt.Errorf("invalid difficulty: %s", m.Difficulty)
	}
	if !allowed(m.Engine, "A", "B") {
		return fmt.Errorf("invalid engine: %s", m.Engine)
	}
	return nil
}

func parseYAML(path string, dst any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, dst)
}

func mustDir(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("expected directory: %s", path)
	}
	return nil
}

func allowed(v string, list ...string) bool {
	for _, s := range list {
		if v == s {
			return true
		}
	}
	return false
}

func validateActionSimulationKeyMatch(actions ActionList, simulation SimulationFile) error {
	actionKeys := map[string]struct{}{}
	for _, action := range actions.Actions {
		key := strings.TrimSpace(action.Key)
		if key == "" {
			continue
		}
		actionKeys[key] = struct{}{}
	}

	simulationKeys := map[string]struct{}{}
	for key := range simulation.Simulation.ActionEffects {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		simulationKeys[trimmed] = struct{}{}
	}

	missingInSimulation := []string{}
	for key := range actionKeys {
		if _, ok := simulationKeys[key]; !ok {
			missingInSimulation = append(missingInSimulation, key)
		}
	}
	missingInActions := []string{}
	for key := range simulationKeys {
		if _, ok := actionKeys[key]; !ok {
			missingInActions = append(missingInActions, key)
		}
	}

	if len(missingInSimulation) == 0 && len(missingInActions) == 0 {
		return nil
	}
	sort.Strings(missingInSimulation)
	sort.Strings(missingInActions)
	return fmt.Errorf("action key mismatch: missing in simulation=%v missing in actions=%v", missingInSimulation, missingInActions)
}
