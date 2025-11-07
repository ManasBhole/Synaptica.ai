package dlp

import (
	"errors"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Rule struct {
	Name     string `yaml:"name" json:"name"`
	Type     string `yaml:"type" json:"type"`
	Pattern  string `yaml:"pattern" json:"pattern"`
	Mask     string `yaml:"mask" json:"mask"`
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Severity string `yaml:"severity" json:"severity"`
}

type RulesConfig struct {
	Rules []Rule `yaml:"rules" json:"rules"`
}

func LoadRules(path string) (RulesConfig, error) {
	if path == "" {
		return DefaultRules(), nil
	}
	content, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return DefaultRules(), err
	}

	var cfg RulesConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return RulesConfig{}, err
	}

	if len(cfg.Rules) == 0 {
		return RulesConfig{}, errors.New("no DLP rules configured")
	}

	return cfg, nil
}

func DefaultRules() RulesConfig {
	return RulesConfig{Rules: []Rule{
		{Name: "SSN", Type: "ssn", Pattern: `\b\d{3}-\d{2}-\d{4}\b`, Mask: "***-**-****", Enabled: true, Severity: "high"},
		{Name: "DOB", Type: "dob", Pattern: `\b\d{1,2}/\d{1,2}/\d{4}\b`, Mask: "##/##/####", Enabled: true, Severity: "medium"},
		{Name: "Email", Type: "email", Pattern: `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, Mask: "***@***", Enabled: true, Severity: "medium"},
		{Name: "Phone", Type: "phone", Pattern: `\b\d{3}-\d{3}-\d{4}\b|\b\(\d{3}\)\s?\d{3}-\d{4}\b`, Mask: "(***) ***-****", Enabled: true, Severity: "medium"},
	}}
}
