package config

import (
	"io/ioutil"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Model represents a single model entry in the registry.
type Model struct {
	ID            string             `yaml:"id" json:"id"`
	Name          string             `yaml:"name" json:"name"`
	Capabilities  map[string]float64 `yaml:"capabilities" json:"capabilities"`
	CostInput     float64            `yaml:"cost_input" json:"cost_input"`
	CostOutput    float64            `yaml:"cost_output" json:"cost_output"`
	ContextWindow int                `yaml:"context_window" json:"context_window"`
	LatencyMs     int                `yaml:"latency_ms" json:"latency_ms"`
	Tags          []string           `yaml:"tags" json:"tags"`
}

// ModelsConfig is used to unmarshal the YAML configuration file.
type ModelsConfig struct {
	Models []Model `yaml:"models" json:"models"`
}

// LoadModels loads model definitions from a YAML file.  It returns a slice
// of Model structs or an error.  The path may be absolute or relative; it
// resolves relative paths with filepath.Clean.
func LoadModels(path string) ([]Model, error) {
	p := filepath.Clean(path)
	data, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var cfg ModelsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	// Ensure capabilities maps are not nil
	for i := range cfg.Models {
		if cfg.Models[i].Capabilities == nil {
			cfg.Models[i].Capabilities = make(map[string]float64)
		}
	}
	return cfg.Models, nil
}

// ConfigStore provides thread‑safe access to the loaded models.  It uses
// a mutex to guard access to the underlying slice.  For frequent
// readers/writers an atomic.Value could be used instead, but a mutex is
// sufficient here.
type ConfigStore struct {
	mu     sync.RWMutex
	models []Model
}

// NewConfigStore creates an empty ConfigStore.
func NewConfigStore() *ConfigStore {
	return &ConfigStore{}
}

// SetModels replaces the current model list.  It makes a copy of the
// provided slice to avoid potential modifications by callers.
func (s *ConfigStore) SetModels(models []Model) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Deep copy the provided models so that callers cannot mutate the
	// store's internal state after SetModels returns.
	newModels := make([]Model, len(models))
	for i, m := range models {
		// Copy the struct value first.
		nm := m

		// Deep copy the capabilities map if present.
		if m.Capabilities != nil {
			nm.Capabilities = make(map[string]float64, len(m.Capabilities))
			for k, v := range m.Capabilities {
				nm.Capabilities[k] = v
			}
		}

		// Deep copy the tags slice.
		if m.Tags != nil {
			nm.Tags = append([]string(nil), m.Tags...)
		}

		newModels[i] = nm
	}

	s.models = newModels
}

// GetModels returns a copy of the current models slice.  Callers must not
// modify the returned slice or its contents.
func (s *ConfigStore) GetModels() []Model {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a deep copy so that callers cannot mutate the store's
	// internal models through the returned slice.
	out := make([]Model, len(s.models))
	for i, m := range s.models {
		nm := m

		if m.Capabilities != nil {
			nm.Capabilities = make(map[string]float64, len(m.Capabilities))
			for k, v := range m.Capabilities {
				nm.Capabilities[k] = v
			}
		}

		if m.Tags != nil {
			nm.Tags = append([]string(nil), m.Tags...)
		}

		out[i] = nm
	}
	return out
}
