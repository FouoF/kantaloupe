package utils

import "testing"

func TestCalculateIndexByPage(t *testing.T) {
	type testCase struct {
		page          int
		pageSize      int
		total         int
		expectedStart int
		expectedEnd   int
	}

	testCases := []testCase{
		{page: 1, pageSize: 5, total: 20, expectedStart: 0, expectedEnd: 5},
		{page: 3, pageSize: 5, total: 20, expectedStart: 10, expectedEnd: 15},
		{page: 0, pageSize: 5, total: 20, expectedStart: 0, expectedEnd: 5},
		{page: 1, pageSize: 0, total: 20, expectedStart: 0, expectedEnd: 10},
		{page: 1, pageSize: -1, total: 20, expectedStart: 0, expectedEnd: 20},
		{page: 5, pageSize: 5, total: 20, expectedStart: 20, expectedEnd: 20},
	}

	for _, tc := range testCases {
		start, end := CalculateIndexByPage(tc.page, tc.pageSize, tc.total)
		if start != tc.expectedStart {
			t.Errorf("For page %d, pageSize %d, total %d, expected start index %d, but got %d",
				tc.page, tc.pageSize, tc.total, tc.expectedStart, start)
		}

		if end != tc.expectedEnd {
			t.Errorf("For page %d, pageSize %d, total %d, expected end index %d, but got %d",
				tc.page, tc.pageSize, tc.total, tc.expectedEnd, end)
		}
	}
}
