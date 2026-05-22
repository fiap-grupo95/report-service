package repository

import (
	"testing"
	"time"
)

func TestDocToDomain(t *testing.T) {
	createdAt := time.Date(2026, 5, 21, 10, 30, 0, 0, time.UTC)
	doc := &reportDocument{
		ID:          "report-1",
		ProcessID:   "process-1",
		Components:  []string{"api", "database"},
		Risks:       []string{"public bucket"},
		Recs:        []string{"restrict access"},
		RawResponse: "raw-response",
		CreatedAt:   createdAt,
	}

	report := docToDomain(doc)

	if report.ID != doc.ID {
		t.Errorf("expected id %s, got %s", doc.ID, report.ID)
	}
	if report.ProcessID != doc.ProcessID {
		t.Errorf("expected process id %s, got %s", doc.ProcessID, report.ProcessID)
	}
	if report.RawResponse != doc.RawResponse {
		t.Errorf("expected raw response %s, got %s", doc.RawResponse, report.RawResponse)
	}
	if !report.CreatedAt.Equal(createdAt) {
		t.Errorf("expected created at %s, got %s", createdAt, report.CreatedAt)
	}
	assertStringSlicesEqual(t, report.Analysis.Components, doc.Components)
	assertStringSlicesEqual(t, report.Analysis.Risks, doc.Risks)
	assertStringSlicesEqual(t, report.Analysis.Recommendations, doc.Recs)
}

func assertStringSlicesEqual(t *testing.T, got, expected []string) {
	t.Helper()

	if len(got) != len(expected) {
		t.Fatalf("expected length %d, got %d", len(expected), len(got))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("expected item %d to be %q, got %q", i, expected[i], got[i])
		}
	}
}
