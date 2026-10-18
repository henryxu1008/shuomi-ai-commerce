package publishing

import (
	"regexp"
)

var (
	leftParenthesisSpacingPattern = regexp.MustCompile(`(\S)\(`)
)
