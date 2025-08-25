package utils

import (
	"math"
	"strconv"
)

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Pages int   `json:"pages"`
}

// CalculatePagination calculates pagination metadata
func CalculatePagination(total int64, page, limit int) PaginationMeta {
	pages := int(math.Ceil(float64(total) / float64(limit)))

	return PaginationMeta{
		Total: total,
		Page:  page,
		Limit: limit,
		Pages: pages,
	}
}

// IntToString converts an integer to string
func IntToString(i int) string {
	return strconv.Itoa(i)
}
