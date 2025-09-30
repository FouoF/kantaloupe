package utils

import (
	"math"
)

// Define events for execute space objects.
const (
	// DefaultPageSize indicates that default query page size.
	DefaultPageSize = 10
	// DefaultPage indicates that default query page index.
	DefaultPage = 1
	// MaxPageSize indicates that max query page size without paging.
	MaxPageSize = math.MaxInt32
	// MaxPerPageSize indicates that max query page size with paging.
	MaxPerPageSize = 100
)

type OptionFunc func(*ListOptions)

func WithLabelSelector(labels string) func(*ListOptions) {
	return func(opts *ListOptions) {
		opts.LabelSelector = labels
	}
}

func WithPage(page, pageSize int32) func(*ListOptions) {
	return func(opts *ListOptions) {
		opts.Page = page
		opts.PageSize = pageSize
	}
}

func WithSort(sortBy, sortDir string) func(*ListOptions) {
	return func(opts *ListOptions) {
		opts.SortBy = sortBy
		opts.SortDir = sortDir
	}
}

func WithFuzzyName(fuzzy string) func(*ListOptions) {
	return func(opts *ListOptions) {
		opts.FuzzyName = fuzzy
	}
}

func NewListOption(funcs ...OptionFunc) *ListOptions {
	options := &ListOptions{
		ClusterOptions: &ListClusterOptions{},
	}
	for _, f := range funcs {
		f(options)
	}

	if options.Page < 1 {
		options.Page = DefaultPage
	}
	if options.PageSize == 0 {
		options.PageSize = DefaultPageSize
	} else if options.PageSize < 0 {
		options.PageSize = MaxPageSize
	}
	return options
}

type ListOptions struct {
	Page           int32
	PageSize       int32
	Pages          int32
	Total          int
	FuzzyName      string // fuzzy search by name
	SortBy         string // sort
	SortDir        string // result order, default asc
	LabelSelector  string
	ClusterOptions *ListClusterOptions
}

type ListClusterOptions struct {
	Type     string
	State    string
	Provider string
}

func WithClusterType(t string) func(*ListOptions) {
	return func(opts *ListOptions) {
		opts.ClusterOptions.Type = t
	}
}

func WithClusterState(state string) func(*ListOptions) {
	return func(opts *ListOptions) {
		opts.ClusterOptions.State = state
	}
}

func WithClusterProvider(provider string) func(*ListOptions) {
	return func(opts *ListOptions) {
		opts.ClusterOptions.Provider = provider
	}
}
