package zenrows

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
	strict "sigs.k8s.io/json"
	"task-processor/internal/integration/acquisition/amazon"
	"task-processor/internal/product/dataacquisition"
)

// NewAmazonFirstPageFetch fixes the vendor endpoint and request shape. The key
// is resolved only for the current frozen approval.
func NewAmazonFirstPageFetch(readKey func(context.Context) (string, error)) dataacquisition.AmazonFirstPageSender {
	return amazonFirstPageFetch(oneRequestClient(dataacquisition.AmazonFirstPageFetchTimeout), fetchURL, readKey)
}

func amazonFirstPageFetch(client *http.Client, endpoint string, readKey func(context.Context) (string, error)) dataacquisition.AmazonFirstPageSender {
	return func(ctx context.Context, approval dataacquisition.AmazonFirstPageApproval) dataacquisition.AmazonFirstPageSubmission {
		out := dataacquisition.AmazonFirstPageSubmission{Receipt: dataacquisition.AmazonFirstPageReceipt{
			ApprovalID: approval.ID, State: dataacquisition.AmazonFirstPageUnknown,
		}}
		if approval.Validate() != nil || readKey == nil {
			return out
		}
		ctx, cancel := context.WithDeadline(ctx, approval.SendBefore())
		defer cancel()
		if ctx.Err() != nil {
			return out
		}
		key, err := readKey(ctx)
		digest := sha256.Sum256([]byte(key))
		if err != nil || ctx.Err() != nil || len(key) == 0 || len(key) > 512 ||
			strings.TrimSpace(key) != key || strings.ContainsAny(key, "\x00\r\n\t") ||
			hex.EncodeToString(digest[:]) != approval.KeyDigest {
			return out
		}
		target := approval.TargetURL()
		params := url.Values{
			"apikey": {key}, "url": {target}, "extract": {"auto"},
			"js_render": {"true"}, "premium_proxy": {"true"}, "proxy_country": {"us"},
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
		if err != nil {
			return out
		}
		response, err := client.Do(req)
		if err != nil {
			return out
		}
		defer response.Body.Close()
		id := singleHeader(response.Header, "X-Request-Id")
		if !dataacquisition.ValidSHEINUSTrialRequestID(id) || strings.Contains(id, key) {
			return out
		}
		out.Receipt.RequestID = id
		if credits, err := strconv.ParseUint(singleHeader(response.Header, "X-Request-Credits"), 10, 31); err == nil {
			value := int(credits)
			out.Receipt.Credits = &value
			if value > dataacquisition.AmazonFirstPageRequestCredits || value > approval.MaximumCredits {
				out.Receipt.State = dataacquisition.AmazonFirstPageOverLimit
				return out
			}
		}
		if response.StatusCode != http.StatusOK || !matchesAmazonFirstPage(singleHeader(response.Header, "Zr-Final-Url"), approval.Keyword) {
			out.Receipt.State = dataacquisition.AmazonFirstPageRejected
			return out
		}
		body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
		if err != nil || ctx.Err() != nil || len(body) > maxResponseBytes || bytes.Contains(body, []byte(key)) {
			return out
		}
		out.Evidence = amazonSearchEvidence(body, time.Now().UTC())
		for _, fact := range out.Evidence {
			if strings.Contains(fact.Title, key) || strings.Contains(fact.MainImage, key) {
				out.Evidence = nil
				return out
			}
		}
		if len(out.Evidence) < 2 {
			out.Receipt.State = dataacquisition.AmazonFirstPageInsufficient
			out.Evidence = nil
		}
		return out
	}
}

func matchesAmazonFirstPage(raw, keyword string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "www.amazon.com" || u.Path != "/s" || u.User != nil || u.Fragment != "" {
		return false
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(q["k"]) != 1 || q.Get("k") != keyword {
		return false
	}
	page, hasPage := q["page"]
	if !hasPage {
		return len(q) == 1
	}
	return len(q) == 2 && len(page) == 1 && page[0] == "1"
}

func amazonSearchEvidence(body []byte, captured time.Time) []dataacquisition.Evidence {
	var response struct {
		HTML   string `json:"html"`
		Parsed struct {
			Results []struct {
				ASIN      string `json:"asin"`
				Title     string `json:"title"`
				Sponsored bool   `json:"is_sponsored"`
			} `json:"results"`
		} `json:"parsed"`
	}
	problems, err := strict.UnmarshalStrict(body, &response, strict.DisallowDuplicateFields)
	if err != nil || len(problems) != 0 {
		return nil
	}
	ids, err := amazon.ParseDiscovery(response.HTML, 200)
	if err != nil || len(ids) == 0 {
		return nil
	}
	doc, err := html.Parse(strings.NewReader(response.HTML))
	if err != nil {
		return nil
	}
	images := make(map[string]string)
	for node := range doc.Descendants() {
		if node.Type != html.ElementNode || node.Data != "img" || !slices.Contains(strings.Fields(nodeAttribute(node, "class")), "s-image") {
			continue
		}
		for card := node.Parent; card != nil; card = card.Parent {
			if nodeAttribute(card, "data-component-type") == "s-search-result" {
				id := nodeAttribute(card, "data-asin")
				if _, seen := images[id]; !seen {
					images[id] = nodeAttribute(node, "src")
				}
				break
			}
		}
	}
	titles := make(map[string]string)
	for _, card := range response.Parsed.Results {
		if _, seen := titles[card.ASIN]; !seen && !card.Sponsored {
			titles[card.ASIN] = card.Title
		}
	}
	facts := make([]dataacquisition.Evidence, 0, min(len(ids), dataacquisition.AmazonFirstPageLimit))
	for _, id := range ids {
		fact := dataacquisition.Evidence{
			Site: "us", ASIN: id, Title: titles[id], MainImage: images[id], Availability: "unknown",
			ParserVersion: dataacquisition.AmazonZenRowsSearchVersion, CapturedAt: captured.Format(time.RFC3339Nano),
			Missing: []string{"availability", "price", "currency", "description", "brand", "attributes", "variants", "rating", "reviewCount"},
		}
		if fact.Validate() == nil {
			facts = append(facts, fact)
			if len(facts) == dataacquisition.AmazonFirstPageLimit {
				break
			}
		}
	}
	return facts
}

func nodeAttribute(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}
