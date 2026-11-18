package amazon

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type USSearchCard struct {
	ASIN      string
	MainImage string
	PriceUSD  float64
	node      *html.Node
}

// ParseUSSearchCards keeps a search image and displayed price on the same
// organic result card selected by discovery.
func ParseUSSearchCards(raw string, limit int) ([]USSearchCard, error) {
	cards, err := discoveryCards(raw, limit)
	if err != nil {
		return nil, err
	}
	results := make([]USSearchCard, 0, len(cards))
	for _, card := range cards {
		image := walk(card.node, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "img" && hasClass(n, "s-image") &&
				inSearchCard(n, card.node)
		})
		result := USSearchCard{ASIN: card.ASIN, PriceUSD: usSearchPrice(card.node), node: card.node}
		if image != nil {
			result.MainImage = attr(image, "src")
		}
		results = append(results, result)
	}
	return results, nil
}

func inSearchCard(node, card *html.Node) bool {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if attr(parent, "data-component-type") == "s-search-result" {
			return parent == card
		}
	}
	return false
}

var exactUSDPrice = regexp.MustCompile(`^(?:\$|US\$|USD )(?:[0-9]{1,9}|[0-9]{1,3}(?:,[0-9]{3}){1,2})(?:\.[0-9]{1,2})?$`)

func usSearchPrice(card *html.Node) float64 {
	var price float64
	for area := range card.Descendants() {
		if attr(area, "data-cy") != "price-recipe" || !inSearchCard(area, card) {
			continue
		}
		for node := range area.Descendants() {
			parent := node.Parent
			// The full accessible amount on the primary offer is separate
			// from crossed-out prices and smaller per-unit amounts.
			if !hasClass(node, "a-offscreen") || !hasClass(parent, "a-price") ||
				crossedOutSearchPrice(parent) || attr(parent, "data-a-size") != "xl" ||
				attr(parent, "data-a-color") != "base" || !inSearchCard(node, card) {
				continue
			}
			if node.FirstChild == nil || node.FirstChild != node.LastChild || node.FirstChild.Type != html.TextNode {
				return 0
			}
			amount := exactSearchPrice(textOf(node))
			if price > 0 || amount == 0 || !clearSearchPriceContext(card, area, node, amount) {
				return 0
			}
			price = amount
		}
	}
	return price
}

func exactSearchPrice(quote string) float64 {
	if exactUSDPrice.MatchString(quote) {
		amount, currency := parsePrice(quote, "us")
		if amount > 0 && currency == "USD" {
			return amount
		}
	}
	return 0
}

func crossedOutSearchPrice(node *html.Node) bool {
	return node.Type == html.ElementNode && (node.Data == "del" || node.Data == "s" || node.Data == "strike" ||
		(hasClass(node, "a-price") && (hasClass(node, "a-text-price") || attr(node, "data-a-strike") == "true")))
}

func clearSearchPriceContext(card, area, quote *html.Node, price float64) bool {
	for node := quote; node != card.Parent; node = node.Parent {
		if qualifiedSearchPrice(node) || crossedOutSearchPrice(node) || attr(node, "aria-hidden") == "true" {
			return false
		}
	}
	var visual *html.Node
	for child := quote.Parent.FirstChild; child != nil; child = child.NextSibling {
		if child != quote && attr(child, "aria-hidden") == "true" {
			amount := strings.Join(strings.Fields(textOf(child)), "")
			if visual != nil || exactSearchPrice(amount) != price || walk(child, func(node *html.Node) bool {
				return qualifiedSearchPrice(node) || crossedOutSearchPrice(node) ||
					(node != child && attr(node, "aria-hidden") == "true")
			}) != nil {
				return false
			}
			visual = child
		}
	}
	return onlySearchPriceText(area, quote, visual)
}

func qualifiedSearchPrice(node *html.Node) bool {
	for _, attribute := range node.Attr {
		switch attribute.Key {
		case "hidden":
			return true
		case "aria-label", "aria-labelledby", "aria-describedby", "title":
			if strings.TrimSpace(attribute.Val) != "" {
				return true
			}
		}
	}
	style := strings.Join(strings.Fields(strings.ToLower(attr(node, "style"))), "")
	return hasClass(node, "aok-hidden") || hasClass(node, "a-hidden") ||
		strings.Contains(style, "display:none") || strings.Contains(style, "visibility:hidden") ||
		strings.Contains(style, "line-through")
}

// Unknown adjoining text might qualify the amount or make it a range.
// Accept only the exact quote, its matching visual rendering and struck-out references.
func onlySearchPriceText(node, quote, visual *html.Node) bool {
	if node == visual || crossedOutSearchPrice(node) {
		return true
	}
	if qualifiedSearchPrice(node) || attr(node, "aria-hidden") == "true" {
		return false
	}
	if node == quote {
		return true
	}
	if node.Type == html.TextNode {
		return strings.TrimSpace(node.Data) == ""
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if !onlySearchPriceText(child, quote, visual) {
			return false
		}
	}
	return true
}
