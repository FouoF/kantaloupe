package utils

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

func IsValidDNS1123Name(str string) bool {
	return len(str) <= 246 && len(validation.IsDNS1123Subdomain(str)) == 0
}

func IsValidLabelName(str string) bool {
	// Label names can have prefixes
	splitIndex := strings.Index(str, "/")
	if splitIndex > 0 {
		return len(validation.IsDNS1123Subdomain(str[0:splitIndex])) == 0 && len(validation.IsValidLabelValue(str[splitIndex+1:])) == 0
	}
	return len(validation.IsValidLabelValue(str)) == 0
}

func IsValidLabelNames(labels map[string]string) bool {
	for name := range labels {
		if !IsValidLabelName(name) {
			return false
		}
	}
	return true
}

func IsValidAnnotationNames(annotations map[string]string) bool {
	return IsValidLabelNames(annotations)
}

func IsValidIntegerNum(numStr string) bool {
	if len(numStr) == 0 {
		return false
	}
	if _, err := strconv.Atoi(numStr); err != nil {
		return false
	}
	return true
}
