package strutil

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	bracketsRe    = regexp.MustCompile(`(\[(.*?)\]|\((.*?)\))`)
	nonAlphaNumRe = regexp.MustCompile(`[^a-zA-Z0-9\s]+`)
)

// RemoveNonAlphaNum removes all special characters in the string
func RemoveNonAlphaNum(s string) string {
	return nonAlphaNumRe.ReplaceAllString(s, "")
}

// RemoveContentIntoBrackets removes content inside brackets, including brackets
func RemoveContentIntoBrackets(s string) string {
	return bracketsRe.ReplaceAllString(s, "")
}

// RemoveExtraSpaces removes unnecessary spaces in the string
// For example RemoveExtraSpaces("hello  world  ") return "hello world"
func RemoveExtraSpaces(s string) string {
	idx := 0

	return strings.Trim(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			idx++
			if idx > 1 {
				return -1
			}
		} else if idx > 0 {
			idx = 0
		}

		return r
	}, s), " \t")
}

// CamelCase returns a string that is a camel case
func CamelCase(s string) string {
	tokens := strings.Split(strings.ToLower(s), " ")
	for i := range tokens {
		tokens[i] = strings.Title(tokens[i])
	}

	return strings.Join(tokens, "")
}
