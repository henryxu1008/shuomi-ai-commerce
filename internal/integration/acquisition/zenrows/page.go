package zenrows

import (
	"encoding/json"

	"task-processor/internal/product/dataacquisition"
)

func pageCards(body []byte, limit int) []dataacquisition.SHEINUSTrialCard {
	var envelope struct {
		Parsed struct {
			Category struct {
				Currency string
			} `json:"category_context"`
			Products []json.RawMessage
		}
	}
	if json.Unmarshal(body, &envelope) != nil {
		return nil
	}
	products := envelope.Parsed.Products
	var cards []dataacquisition.SHEINUSTrialCard
	for i, raw := range products[:min(len(products), limit)] {
		var card struct {
			GoodsID   string `json:"goods_id"`
			Title     string
			URL       string `json:"product_url"`
			Price     json.RawMessage
			PriceNote json.RawMessage `json:"price_note"`
		}
		if json.Unmarshal(raw, &card) != nil {
			continue
		}
		var note string
		amount := string(card.Price)
		if len(card.PriceNote) != 0 && json.Unmarshal(card.PriceNote, &note) != nil {
			amount = ""
		}
		cards = append(cards, dataacquisition.SHEINUSTrialCard{
			Position: i + 1, GoodsID: card.GoodsID, Title: card.Title, URL: card.URL,
			Amount: amount, Currency: envelope.Parsed.Category.Currency, PromotionNote: note,
		})
	}
	return cards
}
