package orgresource

import (
	"math"
	"strconv"
	"strings"
)

const AIPointQuantityUnitV1 = "ai-micropoint-v1"
const MicropointsPerAIPoint int64 = 1_000_000

// AIPointQuantity makes conversion at a trusted commercial/display boundary
// explicit. It neither authorizes a grant nor changes a stored balance's unit.
type AIPointQuantity struct {
	Micropoints int64
	UnitVersion string
}

func WholeAIPointQuantity(points int64) (AIPointQuantity, error) {
	if points < 0 || points > math.MaxInt64/MicropointsPerAIPoint {
		return AIPointQuantity{}, ErrInvalidInput
	}
	return AIPointQuantity{Micropoints: points * MicropointsPerAIPoint, UnitVersion: AIPointQuantityUnitV1}, nil
}

// ParseAIPointQuantity parses a display amount, without floating point or
// rounding. Limits and balances may be zero; a grant still requires positivity.
func ParseAIPointQuantity(text string) (AIPointQuantity, error) {
	if len(text) == 0 || len(text) > 24 {
		return AIPointQuantity{}, ErrInvalidInput
	}
	parts := strings.Split(text, ".")
	if len(parts) > 2 || parts[0] == "" || len(parts[0]) > 1 && parts[0][0] == '0' {
		return AIPointQuantity{}, ErrInvalidInput
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if fraction == "" || len(fraction) > 6 {
			return AIPointQuantity{}, ErrInvalidInput
		}
	}
	for _, r := range parts[0] + fraction {
		if r < '0' || r > '9' {
			return AIPointQuantity{}, ErrInvalidInput
		}
	}
	n, err := strconv.ParseInt(parts[0]+fraction+strings.Repeat("0", 6-len(fraction)), 10, 64)
	if err != nil {
		return AIPointQuantity{}, ErrInvalidInput
	}
	return AIPointQuantity{Micropoints: n, UnitVersion: AIPointQuantityUnitV1}, nil
}

func (quantity AIPointQuantity) PointsString() (string, error) {
	if quantity.UnitVersion != AIPointQuantityUnitV1 || quantity.Micropoints < 0 {
		return "", ErrInvalidInput
	}
	whole, fraction := quantity.Micropoints/MicropointsPerAIPoint, quantity.Micropoints%MicropointsPerAIPoint
	text := strconv.FormatInt(whole, 10)
	if fraction != 0 {
		digits := strconv.FormatInt(fraction+MicropointsPerAIPoint, 10)[1:]
		text += "." + strings.TrimRight(digits, "0")
	}
	return text, nil
}
