package models_test

import (
	"testing"

	"uptui/internal/models"
)

func TestParseAcceptedStatusesEmpty(t *testing.T) {
	ranges, err := models.ParseAcceptedStatuses("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ranges != nil {
		t.Errorf("expected nil ranges for empty string, got %v", ranges)
	}
}

func TestParseAcceptedStatusesSingleCode(t *testing.T) {
	ranges, err := models.ParseAcceptedStatuses("401")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 1 || ranges[0] != [2]int{401, 401} {
		t.Errorf("unexpected ranges: %v", ranges)
	}
}

func TestParseAcceptedStatusesRange(t *testing.T) {
	ranges, err := models.ParseAcceptedStatuses("200-299")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 1 || ranges[0] != [2]int{200, 299} {
		t.Errorf("unexpected ranges: %v", ranges)
	}
}

func TestParseAcceptedStatusesMixed(t *testing.T) {
	ranges, err := models.ParseAcceptedStatuses("200-299,401,403")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 3 {
		t.Fatalf("expected 3 ranges, got %d: %v", len(ranges), ranges)
	}
	if ranges[0] != [2]int{200, 299} {
		t.Errorf("ranges[0]: got %v, want [200 299]", ranges[0])
	}
	if ranges[1] != [2]int{401, 401} {
		t.Errorf("ranges[1]: got %v, want [401 401]", ranges[1])
	}
	if ranges[2] != [2]int{403, 403} {
		t.Errorf("ranges[2]: got %v, want [403 403]", ranges[2])
	}
}

func TestParseAcceptedStatusesWhitespace(t *testing.T) {
	ranges, err := models.ParseAcceptedStatuses(" 200-299 , 401 ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 2 {
		t.Errorf("expected 2 ranges, got %d", len(ranges))
	}
}

func TestParseAcceptedStatusesInvalidCode(t *testing.T) {
	_, err := models.ParseAcceptedStatuses("abc")
	if err == nil {
		t.Fatal("expected error for non-numeric code")
	}
}

func TestParseAcceptedStatusesOutOfRange(t *testing.T) {
	_, err := models.ParseAcceptedStatuses("99")
	if err == nil {
		t.Fatal("expected error for code < 100")
	}
	_, err = models.ParseAcceptedStatuses("600")
	if err == nil {
		t.Fatal("expected error for code > 599")
	}
}

func TestParseAcceptedStatusesInvertedRange(t *testing.T) {
	_, err := models.ParseAcceptedStatuses("299-200")
	if err == nil {
		t.Fatal("expected error for lo > hi")
	}
}
