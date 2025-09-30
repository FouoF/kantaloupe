package utils

import "github.com/dynamia-ai/kantaloupe/api/types"

func NewPage(page, pageSize int32, total int) *types.Pagination {
	if page < 1 {
		page = DefaultPage
	}
	if pageSize == 0 {
		pageSize = DefaultPageSize
	}
	if pageSize < 0 {
		pageSize = int32(total)
	}

	pages := CalculatePages(pageSize, total)
	pagination := types.Pagination{
		Page:     page,
		PageSize: pageSize,
		Total:    int32(total),
		Pages:    pages,
	}
	if pageSize == MaxPageSize {
		pagination.PageSize = -1
	}
	return &pagination
}

func PagedItems[T any](items []T, page, pageSize int32) []T {
	start, end := CalculateIndexByPage(int(page), int(pageSize), len(items))
	return items[start:end]
}

func CalculateIndexByPage(page, pageSize, total int) (int, int) {
	if page < 1 {
		page = DefaultPage
	}
	if pageSize == 0 {
		pageSize = DefaultPageSize
	}
	if pageSize < 0 {
		pageSize = total
	}

	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}

func CalculatePages(pageSize int32, total int) int32 {
	if pageSize <= 0 {
		return 0
	}

	pages := total / int(pageSize)
	if total%int(pageSize) != 0 {
		pages++
	}
	return int32(pages)
}
