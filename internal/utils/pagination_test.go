package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculatePagination_BasicScenario(t *testing.T) {
	result := CalculatePagination(100, 1, 10)

	assert.Equal(t, int64(100), result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.Limit)
	assert.Equal(t, 10, result.Pages) // 100 / 10 = 10 pages
}

func TestCalculatePagination_WithRemainder(t *testing.T) {
	result := CalculatePagination(105, 2, 10)

	assert.Equal(t, int64(105), result.Total)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 10, result.Limit)
	assert.Equal(t, 11, result.Pages) // ceil(105 / 10) = 11 pages
}

func TestCalculatePagination_ExactDivision(t *testing.T) {
	result := CalculatePagination(50, 3, 25)

	assert.Equal(t, int64(50), result.Total)
	assert.Equal(t, 3, result.Page)
	assert.Equal(t, 25, result.Limit)
	assert.Equal(t, 2, result.Pages) // 50 / 25 = 2 pages exactly
}

func TestCalculatePagination_ZeroTotal(t *testing.T) {
	result := CalculatePagination(0, 1, 10)

	assert.Equal(t, int64(0), result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.Limit)
	assert.Equal(t, 0, result.Pages) // 0 / 10 = 0 pages
}

func TestCalculatePagination_OneItem(t *testing.T) {
	result := CalculatePagination(1, 1, 10)

	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.Limit)
	assert.Equal(t, 1, result.Pages) // ceil(1 / 10) = 1 page
}

func TestCalculatePagination_LargeNumbers(t *testing.T) {
	result := CalculatePagination(1000000, 100, 100)

	assert.Equal(t, int64(1000000), result.Total)
	assert.Equal(t, 100, result.Page)
	assert.Equal(t, 100, result.Limit)
	assert.Equal(t, 10000, result.Pages) // 1000000 / 100 = 10000 pages
}

func TestCalculatePagination_SmallLimit(t *testing.T) {
	result := CalculatePagination(10, 5, 3)

	assert.Equal(t, int64(10), result.Total)
	assert.Equal(t, 5, result.Page)
	assert.Equal(t, 3, result.Limit)
	assert.Equal(t, 4, result.Pages) // ceil(10 / 3) = 4 pages
}

func TestCalculatePagination_LimitOne(t *testing.T) {
	result := CalculatePagination(5, 3, 1)

	assert.Equal(t, int64(5), result.Total)
	assert.Equal(t, 3, result.Page)
	assert.Equal(t, 1, result.Limit)
	assert.Equal(t, 5, result.Pages) // 5 / 1 = 5 pages
}

func TestCalculatePagination_EdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		total    int64
		page     int
		limit    int
		expected PaginationMeta
	}{
		{
			name:  "Large page number",
			total: 10,
			page:  999,
			limit: 10,
			expected: PaginationMeta{
				Total: 10,
				Page:  999,
				Limit: 10,
				Pages: 1,
			},
		},
		{
			name:  "Very large limit",
			total: 10,
			page:  1,
			limit: 1000,
			expected: PaginationMeta{
				Total: 10,
				Page:  1,
				Limit: 1000,
				Pages: 1,
			},
		},
		{
			name:  "Fractional division result",
			total: 7,
			page:  1,
			limit: 3,
			expected: PaginationMeta{
				Total: 7,
				Page:  1,
				Limit: 3,
				Pages: 3, // ceil(7/3) = 3
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := CalculatePagination(tc.total, tc.page, tc.limit)
			assert.Equal(t, tc.expected, result)
		})
	}
}
