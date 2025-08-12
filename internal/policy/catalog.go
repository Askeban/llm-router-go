package policy

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Model struct {
	Name           string   `yaml:"name"`
	Strengths      []string `yaml:"strengths"`
	MaxInputTokens int      `yaml:"max_input_tokens"`
	CostTier       string   `yaml:"cost_tier"`
	LatencyTier    string   `yaml:"latency_tier"`
	Languages      []string `yaml:"languages"`
}

type Catalog struct {
	Models []Model `yaml:"models"`
}

func LoadCatalog(path string) (Catalog, error) {
	var c Catalog
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	return c, yaml.Unmarshal(b, &c)
}
