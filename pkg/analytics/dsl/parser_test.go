package dsl

import "testing"

func TestParseBasicQuery(t *testing.T) {
	query, err := Parse("SELECT patient_id, resource_type WHERE concept = 'blood-glucose' LIMIT 50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(query.SelectFields) != 2 {
		t.Fatalf("expected 2 select fields, got %d", len(query.SelectFields))
	}
	if len(query.Filters) != 1 {
		t.Fatalf("expected 1 filter, got %d", len(query.Filters))
	}
	if query.Limit != 50 {
		t.Fatalf("expected limit 50, got %d", query.Limit)
	}
}

func TestParseRequiresSelect(t *testing.T) {
	_, err := Parse("WHERE concept = 'risk'")
	if err == nil {
		t.Fatal("expected error for missing select")
	}
}
