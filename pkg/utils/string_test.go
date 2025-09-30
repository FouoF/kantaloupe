package utils

import "testing"

func TestRandomString(t *testing.T) {
	stringLength := 15
	randomString := RandomString(stringLength)
	if len(randomString) != stringLength {
		t.Errorf("Expected length %d, got %d", stringLength, len(randomString))
	}
	for _, char := range randomString {
		if !isInCharset(char) {
			t.Errorf("Unexpected character %c found in random string", char)
		}
	}
}

func isInCharset(char rune) bool {
	for _, c := range charset {
		if c == char {
			return true
		}
	}
	return false
}
