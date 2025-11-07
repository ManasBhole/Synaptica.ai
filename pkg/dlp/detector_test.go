package dlp

import "testing"

func TestDetectorDetectsPatterns(t *testing.T) {
	rules := DefaultRules()
	detector, err := NewDetector(rules)
	if err != nil {
		t.Fatalf("failed to create detector: %v", err)
	}

	data := map[string]interface{}{
		"note":   "Patient John Doe SSN 123-45-6789 email john@example.com",
		"nested": map[string]interface{}{"phone": "(555) 123-4567"},
	}

	result := detector.Detect(data)
	if !result.Detected {
		t.Fatal("expected PHI detection")
	}
	if len(result.PHITypes) < 2 {
		t.Fatalf("expected at least two PHI types, got %v", result.PHITypes)
	}

	sanitized := detector.Sanitize(data)
	note := sanitized["note"].(string)
	if note == data["note"].(string) {
		t.Fatal("expected sanitized note to differ from original")
	}
}
