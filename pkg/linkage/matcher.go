package linkage

import (
	"strings"

	"github.com/google/uuid"
)

type MatchResult struct {
	MasterID string
	Score    float64
	Method   string
}

type Matcher struct {
	deterministicKeys []string
	threshold         float64
}

func NewMatcher(keys []string, threshold float64) *Matcher {
	var normalized []string
	for _, k := range keys {
		if trimmed := strings.TrimSpace(strings.ToLower(k)); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	if threshold <= 0 {
		threshold = 0.85
	}
	return &Matcher{deterministicKeys: normalized, threshold: threshold}
}

func (m *Matcher) DeterministicKey(record map[string]interface{}) string {
	for _, key := range m.deterministicKeys {
		if value, ok := record[key]; ok {
			if str := getString(value); str != "" {
				return strings.ToLower(key) + ":" + strings.ToLower(str)
			}
		}
	}
	return ""
}

func (m *Matcher) Probabilistic(masterCandidates []PatientLink, record map[string]interface{}) MatchResult {
	bestScore := 0.0
	bestMaster := ""
	targetID := getString(record["patient_id"])
	for _, candidate := range masterCandidates {
		score := jaroWinkler(candidate.PatientID, targetID)
		if score > bestScore {
			bestScore = score
			bestMaster = candidate.MasterID
		}
	}

	if bestScore >= m.threshold {
		return MatchResult{MasterID: bestMaster, Score: bestScore, Method: "probabilistic"}
	}
	return MatchResult{MasterID: uuid.New().String(), Score: 1.0, Method: "new"}
}

func getString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	default:
		return ""
	}
}

func jaroWinkler(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	if s1 == "" || s2 == "" {
		return 0
	}

	matchDistance := max(len(s1), len(s2))/2 - 1
	if matchDistance < 0 {
		matchDistance = 0
	}

	s1Matches := make([]bool, len(s1))
	s2Matches := make([]bool, len(s2))

	matches := 0
	transpositions := 0

	for i := range s1 {
		start := max(0, i-matchDistance)
		end := min(i+matchDistance+1, len(s2))
		for j := start; j < end; j++ {
			if s2Matches[j] || s1[i] != s2[j] {
				continue
			}
			s1Matches[i] = true
			s2Matches[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0
	}

	k := 0
	for i := range s1 {
		if !s1Matches[i] {
			continue
		}
		for ; !s2Matches[k]; k++ {
		}
		if s1[i] != s2[k] {
			transpositions++
		}
		k++
	}

	transpositions /= 2

	jaro := (float64(matches)/float64(len(s1)) + float64(matches)/float64(len(s2)) + float64(matches-transpositions)/float64(matches)) / 3

	prefix := 0
	for i := 0; i < min(4, min(len(s1), len(s2))); i++ {
		if s1[i] == s2[i] {
			prefix++
		} else {
			break
		}
	}

	return jaro + float64(prefix)*0.1*(1-jaro)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
