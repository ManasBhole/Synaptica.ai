package terminology

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Concept struct {
	Display string `yaml:"display" json:"display"`
	SNOMED  string `yaml:"snomed" json:"snomed"`
	LOINC   string `yaml:"loinc" json:"loinc"`
	ICD10   string `yaml:"icd10" json:"icd10"`
}

type Catalog struct {
	Concepts map[string]Concept `yaml:"concepts" json:"concepts"`
}

func Load(path string) (Catalog, error) {
	if path == "" {
		return DefaultCatalog(), nil
	}
	content, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return DefaultCatalog(), err
	}
	var cat Catalog
	if err := yaml.Unmarshal(content, &cat); err != nil {
		return Catalog{}, err
	}
	if len(cat.Concepts) == 0 {
		return Catalog{}, fmt.Errorf("terminology catalog empty")
	}
	return cat, nil
}

func (c Catalog) Lookup(key string) (Concept, bool) {
	if c.Concepts == nil {
		return Concept{}, false
	}
	concept, ok := c.Concepts[strings.ToLower(key)]
	if ok {
		return concept, true
	}
	for k, v := range c.Concepts {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return Concept{}, false
}

func DefaultCatalog() Catalog {
	return Catalog{Concepts: map[string]Concept{
		"blood-glucose": {
			Display: "Blood Glucose",
			SNOMED:  "271062007",
			LOINC:   "2339-0",
			ICD10:   "R73.9",
		},
		"blood-pressure": {
			Display: "Blood Pressure",
			SNOMED:  "75367002",
			LOINC:   "85354-9",
			ICD10:   "I10",
		},
	}}
}
