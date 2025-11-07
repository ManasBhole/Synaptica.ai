package predictor

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
)

type Artifact struct {
	Model struct {
		Type         string   `json:"type"`
		Algorithm    string   `json:"algorithm"`
		FeatureNames []string `json:"feature_names"`
		Weights      struct {
			Bias         float64   `json:"bias"`
			Coefficients []float64 `json:"coefficients"`
		} `json:"weights"`
	} `json:"model"`
}

type Predictor struct {
	dir   string
	cache map[string]cachedArtifact
	mu    sync.RWMutex
}

type cachedArtifact struct {
	artifact Artifact
	modTime  int64
}

func NewPredictor(dir string) *Predictor {
	return &Predictor{
		dir:   dir,
		cache: make(map[string]cachedArtifact),
	}
}

func (p *Predictor) Predict(model string, features map[string]float64) (float64, error) {
	artifact, err := p.loadArtifact(model)
	if err != nil {
		return 0, err
	}
	if len(artifact.Model.FeatureNames) == 0 {
		return 0, fmt.Errorf("artifact missing feature names")
	}
	sample := make([]float64, len(artifact.Model.FeatureNames))
	for idx, name := range artifact.Model.FeatureNames {
		value, ok := features[name]
		if !ok {
			return 0, fmt.Errorf("missing feature %s", name)
		}
		sample[idx] = value
	}
	sum := artifact.Model.Weights.Bias
	for i, coeff := range artifact.Model.Weights.Coefficients {
		sum += coeff * sample[i]
	}
	return sigmoid(sum), nil
}

func (p *Predictor) loadArtifact(model string) (Artifact, error) {
	latest := filepath.Join(p.dir, fmt.Sprintf("%s_latest.json", model))
	info, err := os.Stat(latest)
	if err != nil {
		return Artifact{}, err
	}
	mod := info.ModTime().UnixNano()

	p.mu.RLock()
	cached, ok := p.cache[model]
	p.mu.RUnlock()
	if ok && cached.modTime == mod {
		return cached.artifact, nil
	}

	content, err := os.ReadFile(latest)
	if err != nil {
		return Artifact{}, err
	}
	var artifact Artifact
	if err := json.Unmarshal(content, &artifact); err != nil {
		return Artifact{}, err
	}
	p.mu.Lock()
	p.cache[model] = cachedArtifact{artifact: artifact, modTime: mod}
	p.mu.Unlock()
	return artifact, nil
}

func sigmoid(x float64) float64 {
	return 1 / (1 + math.Exp(-x))
}
