package dataacquisition

import (
	"encoding/json"
	"net/url"
	"strings"

	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

type SHEINUSTrialSubmission struct {
	SHEINUSTrialUsage
	Cards []SHEINUSTrialCard
}

type SHEINUSTrialCard struct {
	Position                                             int
	GoodsID, Title, URL, Amount, Currency, PromotionNote string
}

type SHEINUSTrialPrepared struct {
	SHEINUSCapture
	Position int
}

func (a SHEINUSTrialApproval) PageQuery() SHEINUSQuery {
	return SHEINUSQuery{
		QueryRef: collection.StableID("shein-us-trial-query", a.ID), Market: "US", SourceRegion: "US",
		Query: SHEINUSQuerySummary{Mode: "category", CategoryRef: "1727"},
	}
}

// PrepareSHEINUSTrialCards derives one immutable observation per eligible
// product. Input positions are from the original array before validation.
func PrepareSHEINUSTrialCards(a SHEINUSTrialApproval, cards []SHEINUSTrialCard) []SHEINUSTrialPrepared {
	q := a.PageQuery()
	seen := make(map[string]bool)
	var prepared []SHEINUSTrialPrepared
	last := 0
	for _, card := range cards {
		if card.Position <= last || card.Position > sheinUSFirstPage().CardLimit {
			return nil
		}
		last = card.Position
		if seen[card.GoodsID] || !sheinGoodsID.MatchString(card.GoodsID) || !sheinText(card.Title, 2000) {
			continue
		}
		link, err := url.Parse(card.URL)
		if err != nil || link.User != nil || link.Fragment != "" {
			continue
		}
		link.RawQuery = ""
		link.ForceQuery = false
		o := SHEINUSObservation{
			Platform: "shein", Market: "US", NativeGoodsID: card.GoodsID, SourceURL: link.String(),
			JobID: a.ID, QueryRef: q.QueryRef, ObservationVersion: sheinUSObservationVersion,
			ParserVersion: "zenrows-category-page-v1", Title: &card.Title,
			Missing: []string{"capturedAt", "variantId"},
		}
		price := SHEINUSCardPrice{Amount: card.Amount, Currency: card.Currency, Basis: "listing_display"}
		if strings.TrimSpace(card.PromotionNote) != "" {
			price.PromotionNote = &card.PromotionNote
		}
		if price.valid() {
			o.Price = &price
			if price.PromotionNote == nil {
				o.Missing = append(o.Missing, "price.promotionNote")
			}
			o.Missing = append(o.Missing, "price.taxIncluded", "price.unit")
		} else {
			o.Missing = append(o.Missing, "price")
		}
		raw, err := json.Marshal(o)
		if err != nil {
			continue
		}
		c := SHEINUSCapture{Scope: a.Scope, JobID: a.ID, ItemID: collection.StableID("shein-us-trial-item", a.ID, card.GoodsID), NativeGoodsID: card.GoodsID, Query: q, Raw: raw}
		if _, err = c.Publication(sourcing.RawSnapshotChecksum(string(raw))); err != nil {
			continue
		}
		seen[card.GoodsID] = true
		prepared = append(prepared, SHEINUSTrialPrepared{SHEINUSCapture: c, Position: card.Position})
	}
	return prepared
}
