package travelpayouts_data

import (
	"encoding/json"
	"testing"
)

// MonthIndexForTest exposes parsed month price index for tests.
type MonthIndexForTest struct {
	byDate map[string]dayPrice
}

func (m MonthIndexForTest) CoverageDays() int {
	return len(m.byDate)
}

// ParseCalendarResponseForTest parses calendar JSON for external tests.
func ParseCalendarResponseForTest(raw []byte) (MonthIndexForTest, error) {
	idx, err := parseCalendarResponse(raw)
	if err != nil {
		return MonthIndexForTest{}, err
	}
	return MonthIndexForTest{byDate: idx.ByDate}, nil
}

// ParseMonthMatrixResponseForTest parses month-matrix JSON for external tests.
func ParseMonthMatrixResponseForTest(raw []byte) (MonthIndexForTest, error) {
	idx, err := parseMonthMatrixResponse(raw)
	if err != nil {
		return MonthIndexForTest{}, err
	}
	return MonthIndexForTest{byDate: idx.ByDate}, nil
}

func TestParseCalendarAndMatrix_Internal(t *testing.T) {
	cal := `{"success":true,"data":{"2026-09-17":{"price":169,"airline":"AI","transfers":1}}}`
	matrix := `{"success":true,"data":[{"depart_date":"2026-09-18","value":172,"number_of_changes":1}]}`
	cidx, err := parseCalendarResponse([]byte(cal))
	if err != nil {
		t.Fatal(err)
	}
	midx, err := parseMonthMatrixResponse([]byte(matrix))
	if err != nil {
		t.Fatal(err)
	}
	idx := mergeMonthIndexes(cidx, midx)
	if idx.coverageDays() != 2 {
		t.Fatalf("days=%d", idx.coverageDays())
	}
	cheap, ok := idx.cheapestInWindow("2026-09-18", 1, "")
	if !ok || cheap.Date != "2026-09-17" {
		t.Fatalf("cheapest=%+v", cheap)
	}
	nearest, ok := idx.nearestInWindow("2026-09-18", 1, "")
	if !ok || nearest.Date != "2026-09-18" {
		t.Fatalf("nearest=%+v", nearest)
	}
	_, ok = idx.nearestInWindow("2026-09-28", 3, "2026-09-28")
	if ok {
		t.Fatal("fixture has no prices on/after 2026-09-28")
	}
	_, ok = idx.nearestInWindow("2026-09-18", 1, "2026-09-17")
	if !ok {
		t.Fatal("expected 2026-09-18 on/after hub arrival")
	}
	_ = json.Valid([]byte(cal))
}
