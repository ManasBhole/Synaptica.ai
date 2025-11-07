package dlp

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/synaptica-ai/platform/pkg/common/models"
)

type compiledRule struct {
	rule Rule
	re   *regexp.Regexp
}

type Detector struct {
	rules []compiledRule
}

func NewDetector(cfg RulesConfig) (*Detector, error) {
	var compiled []compiledRule
	for _, rule := range cfg.Rules {
		if !rule.Enabled {
			continue
		}
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, compiledRule{rule: rule, re: re})
	}
	return &Detector{rules: compiled}, nil
}

func (d *Detector) Detect(data map[string]interface{}) models.PHIDetectionResult {
	if d == nil {
		return models.PHIDetectionResult{}
	}

	var positions []models.PHIPosition
	phiTypes := make(map[string]struct{})
	suggestionSet := make(map[string]struct{})

	recurse := func(key string, value interface{}) {}
	recurse = func(key string, value interface{}) {
		switch v := value.(type) {
		case string:
			detectInText(v, d.rules, phiTypes, suggestionSet, &positions)
		case map[string]interface{}:
			for nestedKey, nestedVal := range v {
				recurse(nestedKey, nestedVal)
			}
		case []interface{}:
			for _, nestedVal := range v {
				recurse(key, nestedVal)
			}
		default:
			text := strings.TrimSpace(convertToString(v))
			if text == "" {
				return
			}
			detectInText(text, d.rules, phiTypes, suggestionSet, &positions)
		}
	}

	for key, value := range data {
		recurse(key, value)
	}

	phiList := make([]string, 0, len(phiTypes))
	for t := range phiTypes {
		phiList = append(phiList, t)
	}

	suggestions := make([]string, 0, len(suggestionSet))
	for s := range suggestionSet {
		suggestions = append(suggestions, s)
	}

	result := models.PHIDetectionResult{
		Detected:   len(positions) > 0,
		Confidence: confidenceScore(len(positions)),
		PHITypes:   phiList,
		Positions:  positions,
	}
	if len(suggestions) > 0 {
		result.Suggestions = suggestions
	}

	return result
}

func detectInText(text string, rules []compiledRule, phiTypes map[string]struct{}, suggestions map[string]struct{}, positions *[]models.PHIPosition) {
	for _, rule := range rules {
		matches := rule.re.FindAllStringIndex(text, -1)
		if len(matches) == 0 {
			continue
		}
		phiTypes[rule.rule.Type] = struct{}{}
		suggestions[rule.rule.Mask] = struct{}{}
		for _, match := range matches {
			*positions = append(*positions, models.PHIPosition{
				Start: match[0],
				End:   match[1],
				Type:  rule.rule.Type,
				Value: text[match[0]:match[1]],
			})
		}
	}
}

func (d *Detector) Sanitize(data map[string]interface{}) map[string]interface{} {
	if d == nil {
		return data
	}

	copyMap := make(map[string]interface{}, len(data))
	for key, value := range data {
		copyMap[key] = sanitizeValue(value, d.rules)
	}
	return copyMap
}

func sanitizeValue(value interface{}, rules []compiledRule) interface{} {
	switch v := value.(type) {
	case string:
		masked := v
		for _, rule := range rules {
			masked = rule.re.ReplaceAllString(masked, rule.rule.Mask)
		}
		return masked
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, nested := range v {
			out[k] = sanitizeValue(nested, rules)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, nested := range v {
			out[i] = sanitizeValue(nested, rules)
		}
		return out
	default:
		return value
	}
}

func convertToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", val)
	}
}

func confidenceScore(count int) float64 {
	switch {
	case count == 0:
		return 0
	case count == 1:
		return 0.7
	case count == 2:
		return 0.85
	default:
		return 0.95
	}
}
