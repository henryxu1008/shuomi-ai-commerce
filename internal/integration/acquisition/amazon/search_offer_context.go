package amazon

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

const searchPriceLinkID = "price-link"

// searchPriceLinkLabel recognizes the description attached to the displayed
// offer in Amazon's search-card link layout. The ID is reused across cards;
// require the description in this offer's row and reject duplicates within it.
func searchPriceLinkLabel(card, area, main *html.Node) *html.Node {
	link := main.Parent
	if link == nil || link.Data != "a" || attr(link, "aria-describedby") != searchPriceLinkID {
		return nil
	}
	row := link.Parent
	if row == nil || row.Parent == nil || row.Parent.Parent != area || attr(card, "id") == searchPriceLinkID {
		return nil
	}
	nodes := searchContent(row)
	if len(nodes) != 2 || nodes[1] != link {
		return nil
	}
	label := nodes[0]
	if label.Data != "span" || attr(label, "id") != searchPriceLinkID || !hasClass(label, "aok-offscreen") ||
		plainSearchText(label) != "Price, product page" || qualifiedSearchPrice(label, nil) {
		return nil
	}
	for node := range card.Descendants() {
		if node != label && attr(node, "id") == searchPriceLinkID && inSearchCard(node, card) {
			return nil
		}
	}
	return label
}

// searchOfferAddenda identifies separately marked facts alongside the main
// offer. Unknown text within the price recipe still fails the context check.
func searchOfferAddenda(area, main *html.Node, ignored map[*html.Node]bool) bool {
	link := main.Parent
	nodes := searchContent(link)
	if len(nodes) == 0 || nodes[0] != main {
		return false
	}
	siblings := nodes[1:]
	if len(siblings) != 0 && isSearchCountPrice(siblings[0]) {
		ignored[siblings[0]] = true
		siblings = siblings[1:]
	}
	if len(siblings) != 0 {
		if len(siblings) != 2 || !isSearchListPrice(siblings[0], siblings[1]) {
			return false
		}
		ignored[siblings[0]], ignored[siblings[1]] = true, true
	}
	children := searchContent(area)
	if len(children) == 2 && children[0] == link.Parent.Parent && isSearchPromotionRow(children[1]) {
		ignored[children[1]] = true
	}
	return true
}

func isSearchCountPrice(node *html.Node) bool {
	if node.Data != "span" || !hasClass(node, "a-size-base") || !hasClass(node, "a-color-base") ||
		qualifiedSearchPrice(node, nil) || attr(node, "aria-hidden") == "true" {
		return false
	}
	children := searchContent(node)
	return len(children) == 3 &&
		children[0].Type == html.TextNode && strings.TrimSpace(children[0].Data) == "(" &&
		children[2].Type == html.TextNode && strings.TrimSpace(children[2].Data) == "/count)" &&
		attr(children[1], "data-a-strike") == "" && attr(children[1], "data-a-color") == "base" &&
		searchSecondaryAmount(children[1]) != ""
}

func isSearchListPrice(label, visual *html.Node) bool {
	if label.Data != "span" || !hasClass(label, "a-offscreen") || qualifiedSearchPrice(label, nil) ||
		visual.Data != "div" || !hasClass(visual, "aok-inline-block") || qualifiedSearchPrice(visual, nil) {
		return false
	}
	amount, listed := strings.CutPrefix(plainSearchText(label), "List: ")
	children := searchContent(visual)
	return listed && amount != "" && attr(visual, "aria-hidden") == "List: "+amount &&
		len(children) == 2 && children[0].Data == "span" && plainSearchText(children[0]) == "List:" &&
		!qualifiedSearchPrice(children[0], nil) && attr(children[1], "data-a-strike") == "true" &&
		attr(children[1], "data-a-color") == "secondary" && searchSecondaryAmount(children[1]) == amount
}

func searchSecondaryAmount(node *html.Node) string {
	if node.Data != "span" || !hasClass(node, "a-price") || !hasClass(node, "a-text-price") ||
		attr(node, "data-a-size") != "b" || qualifiedSearchPrice(node, nil) || attr(node, "aria-hidden") == "true" {
		return ""
	}
	children := searchContent(node)
	if len(children) != 2 || children[0].Data != "span" || !hasClass(children[0], "a-offscreen") ||
		children[1].Data != "span" || attr(children[1], "aria-hidden") != "true" ||
		qualifiedSearchPrice(children[0], nil) || qualifiedSearchPrice(children[1], nil) {
		return ""
	}
	amount := plainSearchText(children[0])
	if exactSearchPrice(amount) == 0 || plainSearchText(children[1]) != amount {
		return ""
	}
	return amount
}

func isSearchPromotionRow(node *html.Node) bool {
	if node.Data != "div" || !hasClass(node, "a-row") || !hasClass(node, "a-size-base") ||
		!hasClass(node, "a-color-secondary") || qualifiedSearchPrice(node, nil) {
		return false
	}
	children := searchContent(node)
	if len(children) != 2 || children[0].Data != "span" || children[1].Data != "span" ||
		!hasClass(children[0], "s-promotion-highlight-color") || !hasClass(children[1], "a-color-base") ||
		qualifiedSearchPrice(children[0], nil) || qualifiedSearchPrice(children[1], nil) {
		return false
	}
	// The styling alone is also used for conditions on an offer. Only the
	// observed percentage-and-quantity template describes a separate discount.
	percent := searchDisplayNumber(plainSearchText(children[0]), "Save ", "%")
	quantity := searchDisplayNumber(plainSearchText(children[1]), "on ", " select item(s)")
	return percent > 0 && percent <= 100 && quantity >= 2
}

func searchDisplayNumber(text, prefix, suffix string) uint64 {
	value, starts := strings.CutPrefix(text, prefix)
	value, ends := strings.CutSuffix(value, suffix)
	if !starts || !ends {
		return 0
	}
	number, err := strconv.ParseUint(value, 10, 31)
	if err != nil || strconv.FormatUint(number, 10) != value {
		return 0
	}
	return number
}

func searchContent(parent *html.Node) []*html.Node {
	var children []*html.Node
	for child := parent.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode || child.Type == html.TextNode && strings.TrimSpace(child.Data) != "" {
			children = append(children, child)
		}
	}
	return children
}

func plainSearchText(node *html.Node) string {
	text := node.FirstChild
	if text == nil || text.Type != html.TextNode || text.NextSibling != nil {
		return ""
	}
	return strings.TrimSpace(text.Data)
}
