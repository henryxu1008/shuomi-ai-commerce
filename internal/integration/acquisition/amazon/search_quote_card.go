package amazon

import (
	"strings"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/dataacquisition"
	"golang.org/x/net/html"
)

// CaptureQuoteCard preserves quote context as private text, including unfamiliar offer markup.
func (card USSearchCard) CaptureQuoteCard() *dataacquisition.AmazonQuoteCard {
	evidence := &dataacquisition.AmazonQuoteCard{Status: dataacquisition.AmazonQuoteCardUnavailable}
	if card.node == nil {
		return evidence
	}

	remaining := dataacquisition.MaxAmazonQuoteCardBytes
	node := projectQuoteNode(card.node, card.node, &remaining)
	if remaining < 0 {
		evidence.Status = dataacquisition.AmazonQuoteCardTooLarge
		return evidence
	}
	var markup strings.Builder
	if node == nil || html.Render(&markup, node) != nil {
		return evidence
	}
	if markup.Len() > dataacquisition.MaxAmazonQuoteCardBytes {
		evidence.Status = dataacquisition.AmazonQuoteCardTooLarge
		return evidence
	}
	evidence.Status = dataacquisition.AmazonQuoteCardCaptured
	evidence.Markup = markup.String()
	return evidence
}

func projectQuoteNode(node, root *html.Node, remaining *int) *html.Node {
	if node.Type != html.ElementNode && node.Type != html.TextNode {
		return nil
	}
	if node.Type == html.ElementNode {
		if node.Namespace != "" || (node != root && attr(node, "data-component-type") == "s-search-result") {
			return nil
		}
		switch node.Data {
		case "script", "style", "template", "noscript", "iframe", "object", "embed", "link", "meta", "base",
			"input", "textarea", "select", "video", "audio", "canvas",
			"title", "xmp", "plaintext", "noembed", "noframes":
			return nil
		}
	}
	// Account for a lower bound on rendered bytes during projection, then check
	// the exact size after HTML escaping. Oversize evidence has no partial text.
	*remaining -= len(node.Data)
	if *remaining < 0 {
		return nil
	}
	projected := &html.Node{Type: node.Type, Data: node.Data, DataAtom: node.DataAtom}
	for _, a := range node.Attr {
		if a.Namespace != "" {
			continue
		}
		switch a.Key {
		case "class", "id", "role", "hidden", "aria-hidden", "aria-label", "aria-labelledby", "aria-describedby",
			"title", "data-asin", "data-component-type", "data-cy", "data-a-size", "data-a-color", "data-a-strike":
		case "style":
			a.Val = quoteVisibilityStyle(a.Val)
			if a.Val == "" {
				continue
			}
		default:
			continue
		}
		*remaining -= len(a.Key) + len(a.Val)
		if *remaining < 0 {
			return nil
		}
		projected.Attr = append(projected.Attr, a)
	}
	for child := node.FirstChild; child != nil && *remaining >= 0; child = child.NextSibling {
		if projectedChild := projectQuoteNode(child, root, remaining); projectedChild != nil {
			projected.AppendChild(projectedChild)
		}
	}
	return projected
}

func quoteVisibilityStyle(style string) string {
	var declarations []string
	for declaration := range strings.SplitSeq(style, ";") {
		key, value, found := strings.Cut(declaration, ":")
		if !found {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "display", "visibility", "opacity", "content-visibility", "text-decoration", "text-decoration-line":
			// Keep simple values, excluding CSS functions and URLs.
			if strings.TrimSpace(value) != "" && strings.IndexFunc(value, func(r rune) bool {
				return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(" \t.-%!#", r))
			}) == -1 {
				declarations = append(declarations, strings.TrimSpace(declaration))
			}
		}
	}
	return strings.Join(declarations, "; ")
}
