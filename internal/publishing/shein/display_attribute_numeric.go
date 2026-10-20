package shein

import (
	"regexp"
)

var numericTokenPattern = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)
