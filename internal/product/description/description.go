// Package description owns the bounded, versioned representation of product
// descriptions. It has no storage, network or authorization side effects.
package description

import (
	"bytes"
	"errors"
	"io"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const HTML = "html-v1"
const MaxBytes = 32768

var ErrInvalid = errors.New("invalid product description")

var allowedTags = map[string]bool{"p": true, "br": true, "h2": true, "h3": true, "strong": true, "em": true, "s": true, "ul": true, "ol": true, "li": true, "blockquote": true, "a": true, "img": true}
var policy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	for tag := range allowedTags {
		p.AllowElements(tag)
	}
	p.AllowAttrs("href", "target", "rel").OnElements("a")
	p.AllowAttrs("src", "alt", "title").OnElements("img")
	p.AllowURLSchemes("https")
	return p
}()

func validURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && u.Fragment == "" && !strings.ContainsAny(value, "\x00\r\n\t ") && !strings.Contains(value, "\\") && len(value) <= 2048
}
func fragment(value string) ([]*html.Node, error) {
	return html.ParseFragment(strings.NewReader(value), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div})
}
func rendered(nodes []*html.Node) string {
	var b bytes.Buffer
	for _, node := range nodes {
		_ = html.Render(&b, node)
	}
	return b.String()
}

// Validate rejects unsafe/unsupported content instead of silently changing the
// command payload. The original bytes remain the immutable fact and hash input.
// A nil image pool checks shape only; a non-nil pool enforces exact references.
func Validate(value, format string, images []string) error {
	if format == "" {
		return nil
	}
	if format != HTML || len(value) > MaxBytes || !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
		return ErrInvalid
	}
	// Token checks also reject ignored document tags and comments that a fragment
	// parser could otherwise discard before the sanitizer comparison.
	z := html.NewTokenizer(strings.NewReader(value))
	count, depth := 0, 0
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			if z.Err() != io.EOF {
				return ErrInvalid
			}
			break
		}
		count++
		if count > 3072 {
			return ErrInvalid
		}
		if kind == html.CommentToken || kind == html.DoctypeToken {
			return ErrInvalid
		}
		if kind == html.StartTagToken || kind == html.SelfClosingTagToken || kind == html.EndTagToken {
			token := z.Token()
			if !allowedTags[token.Data] {
				return ErrInvalid
			}
			if kind == html.StartTagToken && token.Data != "img" && token.Data != "br" {
				depth++
				if depth > 32 {
					return ErrInvalid
				}
			}
			if kind == html.EndTagToken {
				depth--
				if depth < 0 {
					return ErrInvalid
				}
			}
		}
	}
	nodes, err := fragment(value)
	if err != nil {
		return ErrInvalid
	}
	cleaned, err := fragment(policy.Sanitize(value))
	if err != nil || rendered(nodes) != rendered(cleaned) {
		return ErrInvalid
	}
	imageSet := map[string]bool{}
	for _, image := range images {
		imageSet[image] = true
	}
	count = 0
	var walk func(*html.Node, int) bool
	walk = func(n *html.Node, depth int) bool {
		count++
		if count > 1024 || depth > 32 {
			return false
		}
		if n.Type == html.ElementNode {
			attrs := map[string]string{}
			for _, a := range n.Attr {
				if a.Namespace != "" || attrs[a.Key] != "" {
					return false
				}
				attrs[a.Key] = a.Val
			}
			if n.Data == "img" && (!validURL(attrs["src"]) || images != nil && !imageSet[attrs["src"]]) {
				return false
			}
			if n.Data == "a" {
				if href, ok := attrs["href"]; ok && !validURL(href) {
					return false
				}
				if target, ok := attrs["target"]; ok && target != "_blank" {
					return false
				}
				if attrs["target"] == "_blank" {
					rel := map[string]bool{}
					for _, v := range strings.Fields(attrs["rel"]) {
						rel[v] = true
					}
					if !rel["noopener"] || !rel["noreferrer"] || !rel["nofollow"] {
						return false
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if !walk(c, depth+1) {
				return false
			}
		}
		return true
	}
	for _, n := range nodes {
		if !walk(n, 1) {
			return ErrInvalid
		}
	}
	return nil
}

// ImageReferences reads resources from the already validated description.
// It has no object-storage or network effects.
func ImageReferences(value, format string) []string {
	if format != HTML {
		return nil
	}
	nodes, err := fragment(value)
	if err != nil {
		return nil
	}
	var result []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					result = append(result, attr.Val)
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	for _, node := range nodes {
		walk(node)
	}
	return result
}

// PlainText is a transient projection for existing text-only contracts. Images
// and link destinations are not copied or fetched. Plain input is unchanged.
func PlainText(value, format string) string {
	if format == "" {
		return value
	}
	if format != HTML {
		return ""
	}
	nodes, err := fragment(policy.Sanitize(value))
	if err != nil {
		return ""
	}
	var b strings.Builder
	newline := func() {
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
			b.WriteByte('\n')
		}
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			return
		}
		if n.Data == "img" {
			return
		}
		block := n.Data == "p" || n.Data == "h2" || n.Data == "h3" || n.Data == "li" || n.Data == "blockquote" || n.Data == "ul" || n.Data == "ol"
		if block {
			newline()
		}
		if n.Data == "br" {
			b.WriteByte('\n')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if block {
			newline()
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	return strings.TrimSpace(b.String())
}
