package walmart

import (
	"math/big"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const MaxOrderLines = 100

var orderDecimal = regexp.MustCompile(`^[0-9]{1,20}(\.[0-9]{1,8})?$`)
var orderCurrency = regexp.MustCompile(`^[A-Z]{3}$`)

// Order is an allowlisted consumer purchase order, not a procurement order or
// a SHEIN observation. Raw responses and customer contact data never enter it.
type Order struct {
	ID              string      `json:"id"`
	Site            string      `json:"site"`
	CustomerOrderID string      `json:"customerOrderId,omitempty"`
	Date            string      `json:"date"`
	Type            string      `json:"type,omitempty"`
	Lines           []OrderLine `json:"lines"`
}
type OrderLine struct {
	Number    string        `json:"number"`
	Quantity  string        `json:"quantity,omitempty"`
	Charges   []OrderCharge `json:"charges"`
	Ambiguous bool          `json:"ambiguous,omitempty"`
}
type OrderCharge struct {
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
}
type OrderPage struct {
	Total      int
	Items      []Order
	NextCursor string
}

func orderText(s string, n int) bool {
	return len(s) <= n && strings.TrimSpace(s) == s && !strings.ContainsFunc(s, unicode.IsControl)
}
func (o Order) Valid() bool {
	if o.ID == "" || !orderText(o.ID, 128) || o.Site != "walmart-us" || !orderText(o.CustomerOrderID, 128) || !orderText(o.Type, 64) || len(o.Lines) > MaxOrderLines {
		return false
	}
	if o.Date != "" {
		at, e := time.Parse(time.RFC3339Nano, o.Date)
		if e != nil || at.Year() < 1 || at.Year() > 9999 {
			return false
		}
	}
	seen := map[string]bool{}
	for _, line := range o.Lines {
		if line.Number == "" || !orderText(line.Number, 128) || seen[line.Number] || !orderText(line.Quantity, 32) || len(line.Charges) > 20 {
			return false
		}
		seen[line.Number] = true
		for _, charge := range line.Charges {
			if charge.Amount != "" && !orderDecimal.MatchString(charge.Amount) || charge.Currency != "" && !orderCurrency.MatchString(charge.Currency) {
				return false
			}
		}
	}
	return true
}

// ProductAmount sums the source's line PRODUCT fees once. Quantity, status
// groups, taxes, shipping and virtual-pack counts do not multiply these fees.
func (o Order) ProductAmount() (amount, currency string, missingAmount, missingCurrency bool) {
	total := new(big.Int)
	if len(o.Lines) == 0 {
		return "", "", true, true
	}
	for _, line := range o.Lines {
		if line.Ambiguous || len(line.Charges) != 1 {
			missingAmount = true
		}
		if len(line.Charges) == 0 {
			missingCurrency = true
		}
		for _, charge := range line.Charges {
			if charge.Amount == "" || !orderDecimal.MatchString(charge.Amount) {
				missingAmount = true
			} else {
				parts := strings.SplitN(charge.Amount, ".", 2)
				fraction := ""
				if len(parts) == 2 {
					fraction = parts[1]
				}
				scaled, ok := new(big.Int).SetString(parts[0]+fraction+strings.Repeat("0", 8-len(fraction)), 10)
				if !ok {
					missingAmount = true
				} else {
					total.Add(total, scaled)
				}
			}
			if !orderCurrency.MatchString(charge.Currency) {
				missingCurrency = true
			} else if currency == "" {
				currency = charge.Currency
			} else if currency != charge.Currency {
				missingCurrency = true
			}
		}
	}
	text := total.String()
	if len(text) < 9 {
		text = strings.Repeat("0", 9-len(text)) + text
	}
	whole, fraction := text[:len(text)-8], strings.TrimRight(text[len(text)-8:], "0")
	if len(fraction) < 2 {
		fraction += strings.Repeat("0", 2-len(fraction))
	}
	amount = whole + "." + fraction
	if missingAmount {
		amount = ""
	}
	if missingCurrency {
		currency = ""
	}
	return
}
