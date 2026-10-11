package browser

import (
	"context"
	"encoding/json"
	sigjson "sigs.k8s.io/json"
	"task-processor/internal/product/sourcing"
)

var _ sourcing.PublicProductDiscoverer = (*Client)(nil)

// Discover shares the acquisition client's slots, throttle and challenge cooldown.
// It reads one anonymous visible page and never enters challenge solving.
func (c *Client) Discover(ctx context.Context, input sourcing.DiscoveryInput) (sourcing.DiscoveryEvidence, error) {
	canonical, e := sourcing.Canonical1688Discovery(sourcing.DiscoveryPage, input.URL)
	if e != nil || canonical.URL != input.URL || input.Mode != sourcing.DiscoveryPage && input.Mode != sourcing.DiscoveryKeyword {
		return sourcing.DiscoveryEvidence{}, sourcing.ErrInvalidAcquisition
	}
	if input.Mode == sourcing.DiscoveryKeyword {
		expected, e := sourcing.Canonical1688Discovery(input.Mode, input.Keyword)
		if e != nil || expected != input {
			return sourcing.DiscoveryEvidence{}, sourcing.ErrInvalidAcquisition
		}
	}
	if c == nil || c.opts.validate() != nil {
		return sourcing.DiscoveryEvidence{}, ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, c.opts.budget())
	defer cancel()
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	case <-ctx.Done():
		return sourcing.DiscoveryEvidence{}, ctx.Err()
	default:
		return sourcing.DiscoveryEvidence{}, ErrCapacity
	}
	if e := c.throttle.Wait(ctx); e != nil {
		return sourcing.DiscoveryEvidence{}, e
	}
	raw, e := c.collectPage(ctx, input.URL, false, &input, discoveryScript)
	if e != nil {
		return sourcing.DiscoveryEvidence{}, e
	}
	return decodeDiscovery(input, raw)
}
func decodeDiscovery(input sourcing.DiscoveryInput, raw any) (sourcing.DiscoveryEvidence, error) {
	result := sourcing.DiscoveryEvidence{SourceURL: input.URL, Products: []string{}}
	encoded, e := json.Marshal(raw)
	if e != nil || len(encoded) > 65536 {
		return result, ErrUnsupported
	}
	var body struct {
		Products  []string `json:"products"`
		Truncated bool     `json:"truncated"`
	}
	strict, e := sigjson.UnmarshalStrict(encoded, &body, sigjson.DisallowDuplicateFields, sigjson.DisallowUnknownFields)
	if e != nil || len(strict) > 0 || len(body.Products) == 0 || len(body.Products) > sourcing.MaxPlanProducts {
		return result, ErrUnsupported
	}
	seen := map[string]bool{}
	for _, url := range body.Products {
		canonical, e := sourcing.Canonical1688Source(url)
		if e != nil || canonical.URL != url || seen[url] {
			return result, ErrUnsupported
		}
		seen[url] = true
	}
	result.Products = body.Products
	result.Truncated = body.Truncated
	return result, nil
}

const discoveryScript = `() => {
 const products=[], seen=new Set(); let truncated=false;
 const links=document.querySelectorAll('a[href]');
 for (let i=0;i<links.length;i++) {
   if(i>=5000){truncated=true;break;}
   const a=links[i], style=getComputedStyle(a);
   if(!a.getClientRects().length || style.visibility==='hidden' || style.display==='none')continue;
   let u;try{u=new URL(a.href,location.href);}catch{continue;}
   if(u.protocol!=='https:' || u.hostname!=='detail.1688.com' || u.username || u.password || u.port && u.port!=='443' || !/^\/offer\/[1-9][0-9]{0,19}\.html$/.test(u.pathname))continue;
   const url='https://detail.1688.com'+u.pathname;
   if(seen.has(url))continue;seen.add(url);
   if(products.length>=200){truncated=true;break;}
   products.push(url);
 }
 return {products,truncated};
}`
