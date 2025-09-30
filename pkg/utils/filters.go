package utils

import (
	"github.com/dlclark/regexp2"
)

const (
	Regex = regexp2.IgnorePatternWhitespace | regexp2.IgnoreCase
)

type NameGetter interface {
	GetName() string
}

func FilterByFuzzyName[T NameGetter](items []T, keyword string) []T {
	res := []T{}
	regex, err := regexp2.Compile(keyword, Regex)
	if err != nil {
		return res
	}

	for idx := range items {
		item := items[idx]
		match, _ := regex.MatchString(item.GetName())
		if match {
			res = append(res, item)
		}
	}
	return res
}

func MatchByFuzzyName[T NameGetter](item T, keyword string) bool {
	regex, err := regexp2.Compile(keyword, Regex)
	if err != nil {
		return false
	}

	match, _ := regex.MatchString(item.GetName())
	return match
}
