package imap

import (
	"testing"
	"time"
)

func TestSearchCriteriaAndSize(t *testing.T) {
	since := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		a, b                    SearchCriteria
		wantLarger, wantSmaller int64
	}{
		{SearchCriteria{Smaller: 100}, SearchCriteria{Since: since}, 0, 100},
		{SearchCriteria{Since: since}, SearchCriteria{Smaller: 100}, 0, 100},
		{SearchCriteria{Smaller: 100}, SearchCriteria{Smaller: 50}, 0, 50},
		{SearchCriteria{Smaller: 50}, SearchCriteria{Smaller: 100}, 0, 50},
		{SearchCriteria{Larger: 100}, SearchCriteria{Since: since}, 100, 0},
		{SearchCriteria{Larger: 50}, SearchCriteria{Larger: 100}, 100, 0},
	}
	for _, tc := range tests {
		c := tc.a
		c.And(&tc.b)
		if c.Larger != tc.wantLarger || c.Smaller != tc.wantSmaller {
			t.Errorf("%+v AND %+v: Larger = %v, Smaller = %v, want %v, %v",
				tc.a, tc.b, c.Larger, c.Smaller, tc.wantLarger, tc.wantSmaller)
		}
	}
}
