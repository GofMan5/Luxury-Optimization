package optimizer

import (
	"strings"
	"unicode"
)

func displayText(value string) string {
	value = strings.Map(func(character rune) rune {
		if unicode.IsControl(character) || unicode.Is(unicode.Cf, character) {
			return -1
		}
		return character
	}, value)
	runes := []rune(value)
	if len(runes) > 512 {
		return string(runes[:511]) + "…"
	}
	return value
}
