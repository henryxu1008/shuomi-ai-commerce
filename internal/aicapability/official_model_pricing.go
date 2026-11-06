package aicapability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
)

// AIPointMicrounitV1 represents one million internal units per retail AI point.
// Installing or interpreting an existing balance in this unit is not permitted.
const AIPointMicrounitV1 = "ai-micropoint-v1"

// ReferenceTextTotalsV1 estimates from explicit prompt/completion totals using
// the platform's frozen reference rates. It makes no provider bill or cache claim.
const ReferenceTextTotalsV1 = "reference-text-totals-v1"

var ErrOfficialModelPrice = errors.New("invalid official model price or usage evidence")

func (p OfficialModelTariff) Valid() bool { return p.Validate() == nil }

// ReserveTextMicropoints prices an admitted text-token bound, never observed
// usage or a customer-supplied amount. Settlement uses the complete partitions.
func (p OfficialModelTariff) ReserveTextMicropoints(prompt, completion int64) (int64, error) {
	return p.ReservationMicropoints(BillableModelUsage{MappingVersion: p.UsageVersion, Complete: true, InputText: prompt, OutputText: completion})
}

// OfficialModelRates are currency micros per one million tokens. Cached inputs
// are subsets of their corresponding total inputs, never additional tokens.
type OfficialModelRates struct {
	// Currency micros per output image, for an explicitly admitted image-count protocol.
	OutputImagePerItem int64 `json:"outputImagePerItem,omitempty"`
	InputText          int64 `json:"inputText"`
	CachedInputText    int64 `json:"cachedInputText"`
	InputImage         int64 `json:"inputImage"`
	CachedInputImage   int64 `json:"cachedInputImage"`
	OutputText         int64 `json:"outputText"`
	OutputImage        int64 `json:"outputImage"`
}

// OfficialModelTariff is a frozen execution price, not a current configuration
// source. The configuration owner validates reference provenance, applicability,
// effective dates and credentials before constructing it. No caller-selected
// rates or FX can authorize dispatch through these arithmetic helpers.
type OfficialModelTariff struct {
	// WireModel is set only by the trusted qualifier when a channel uses a
	// different exact identifier. Model keeps its manufacturer price identity.
	WireModel                string             `json:"wireModel,omitempty"`
	UnitVersion              string             `json:"unitVersion"`
	PriceVersion             string             `json:"priceVersion"`
	FXVersion                string             `json:"fxVersion"`
	Currency                 string             `json:"currency"`
	CNYMicrosPerCurrencyUnit int64              `json:"cnyMicrosPerCurrencyUnit"`
	Model                    string             `json:"model"`
	Channel                  string             `json:"channel"`
	UsageVersion             string             `json:"usageVersion"`
	Rates                    OfficialModelRates `json:"rates"`
}

func (p OfficialModelTariff) MatchesWireModel(model string) bool {
	if p.WireModel != "" {
		return validTextFact(p.WireModel, 128) && p.WireModel == model
	}
	return p.Model == model
}

func (p OfficialModelTariff) IsReferenceEstimate() bool {
	return p.UsageVersion == ReferenceTextTotalsV1
}

// Empty means this tariff uses a different, separately admitted strategy.
func (p OfficialModelTariff) ReferencePriceVersion() string {
	if p.IsReferenceEstimate() || p.IsReferenceImageEstimate() {
		return p.PriceVersion
	}
	return ""
}

// BillableModelUsage is built by the trusted usage mapper only after checking
// the fields required by MappingVersion. For ReferenceTextTotalsV1, Complete
// means explicit basic totals, not all supplier billing subdivisions. It must
// never be inferred from Go zero values or establish a terminal outcome alone.
type BillableModelUsage struct {
	ReferenceImageTotals ReferenceImageTotals `json:"referenceImageTotals,omitzero"`
	OutputImageItems     int64                `json:",omitempty"`
	MappingVersion       string
	Complete             bool
	InputText            int64
	CachedInputText      int64
	InputImage           int64
	CachedInputImage     int64
	OutputText           int64
	OutputImage          int64
}

func (u BillableModelUsage) Validate() error {
	if u.MappingVersion == ReferenceImagesTotalsV1 {
		return u.validateReferenceImages()
	}
	if u.ReferenceImageTotals != (ReferenceImageTotals{}) {
		return ErrOfficialModelPrice
	}
	if u.MappingVersion == ReferenceTextTotalsV1 && (u.CachedInputText != 0 || u.InputImage != 0 || u.CachedInputImage != 0 || u.OutputImage != 0 || u.OutputImageItems != 0) {
		return ErrOfficialModelPrice
	}
	if !u.Complete || !validTextFact(u.MappingVersion, 128) || u.CachedInputText > u.InputText || u.CachedInputImage > u.InputImage {
		return ErrOfficialModelPrice
	}
	for _, quantity := range []int64{u.InputText, u.CachedInputText, u.InputImage, u.CachedInputImage, u.OutputText, u.OutputImage, u.OutputImageItems} {
		if quantity < 0 || quantity > 1<<30 {
			return ErrOfficialModelPrice
		}
	}
	return nil
}

func (p OfficialModelTariff) Validate() error {
	if p.WireModel != "" && !validTextFact(p.WireModel, 128) {
		return ErrOfficialModelPrice
	}
	if p.IsReferenceImageEstimate() && (p.Rates.InputText <= 0 || p.Rates.InputImage <= 0 || p.Rates.OutputImage <= 0 || p.Rates.CachedInputText != 0 || p.Rates.CachedInputImage != 0 || p.Rates.OutputText != 0 || p.Rates.OutputImagePerItem != 0) {
		return ErrOfficialModelPrice
	}
	if p.IsReferenceEstimate() && (p.Rates.InputText <= 0 || p.Rates.OutputText <= 0 || p.Rates.CachedInputText != 0 || p.Rates.InputImage != 0 || p.Rates.CachedInputImage != 0 || p.Rates.OutputImage != 0 || p.Rates.OutputImagePerItem != 0) {
		return ErrOfficialModelPrice
	}
	if p.UnitVersion != AIPointMicrounitV1 || p.CNYMicrosPerCurrencyUnit <= 0 || p.CNYMicrosPerCurrencyUnit > 1_000_000_000_000 || len(p.Currency) != 3 {
		return ErrOfficialModelPrice
	}
	for _, r := range p.Currency {
		if r < 'A' || r > 'Z' {
			return ErrOfficialModelPrice
		}
	}
	if p.Currency == "CNY" && p.CNYMicrosPerCurrencyUnit != 1_000_000 {
		return ErrOfficialModelPrice
	}
	for _, v := range []string{p.PriceVersion, p.FXVersion, p.Model, p.Channel, p.UsageVersion} {
		if !validTextFact(v, 128) {
			return ErrOfficialModelPrice
		}
	}
	rates := []int64{p.Rates.InputText, p.Rates.CachedInputText, p.Rates.InputImage, p.Rates.CachedInputImage, p.Rates.OutputText, p.Rates.OutputImage}
	if p.UsageVersion == "qwen-image-count-v1" {
		for _, rate := range rates {
			if rate != 0 {
				return ErrOfficialModelPrice
			}
		}
		if p.Rates.OutputImagePerItem <= 0 {
			return ErrOfficialModelPrice
		}
	} else if p.Rates.OutputImagePerItem != 0 {
		return ErrOfficialModelPrice
	}
	rates = append(rates, p.Rates.OutputImagePerItem)
	positive := false
	for _, rate := range rates {
		if rate < 0 || rate > 1_000_000_000_000 {
			return ErrOfficialModelPrice
		}
		positive = positive || rate > 0
	}
	if !positive || p.Rates.CachedInputText > p.Rates.InputText || p.Rates.CachedInputImage > p.Rates.InputImage {
		return ErrOfficialModelPrice
	}
	return nil
}

func (p OfficialModelTariff) Digest() (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return "", ErrOfficialModelPrice
	}
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:]), nil
}

func (p OfficialModelTariff) SettlementMicropoints(usage BillableModelUsage) (int64, error) {
	return p.micropoints(usage, false)
}

// ReservationMicropoints assumes no cache discount, and ceilings only once.
// Its input must itself have an admitted conservative token bound. In particular,
// feeding a Flare output estimate here does not establish a dispatch budget.
func (p OfficialModelTariff) ReservationMicropoints(bound BillableModelUsage) (int64, error) {
	return p.micropoints(bound, true)
}

func (p OfficialModelTariff) micropoints(u BillableModelUsage, ceiling bool) (int64, error) {
	if p.Validate() != nil || u.Validate() != nil || u.MappingVersion != p.UsageVersion {
		return 0, ErrOfficialModelPrice
	}
	quantities := []int64{u.InputText - u.CachedInputText, u.CachedInputText, u.InputImage - u.CachedInputImage, u.CachedInputImage, u.OutputText, u.OutputImage, u.OutputImageItems}
	pricedRates := p.Rates
	rates := []int64{pricedRates.InputText, pricedRates.CachedInputText, pricedRates.InputImage, pricedRates.CachedInputImage, pricedRates.OutputText, pricedRates.OutputImage, pricedRates.OutputImagePerItem}
	if p.IsReferenceImageEstimate() {
		quantities[0], quantities[2] = u.ReferenceImageTotals.Input, 0
		rates[0] = max(pricedRates.InputText, pricedRates.InputImage)
		if !ceiling && u.ReferenceImageTotals.InputTextKnown && u.ReferenceImageTotals.InputImageKnown {
			quantities[0], quantities[2] = u.InputText, u.InputImage
			rates[0] = pricedRates.InputText
		}
	}
	for i, quantity := range quantities {
		if quantity < 0 || quantity > 1<<30 || quantity > 0 && rates[i] == 0 {
			return 0, ErrOfficialModelPrice
		}
	}
	if ceiling && !p.IsReferenceImageEstimate() {
		quantities[0], quantities[1] = u.InputText, 0
		quantities[2], quantities[3] = u.InputImage, 0
	}
	sum := new(big.Int)
	for i, quantity := range quantities {
		term := new(big.Int).Mul(big.NewInt(quantity), big.NewInt(rates[i]))
		if i == 6 {
			term.Mul(term, big.NewInt(1_000_000))
		} // per-image micros, not micros per million tokens
		sum.Add(sum, term)
	}
	// sum / 10^12 is the exact currency amount. FX has scale 10^6,
	// and one micropoint is CNY 10^-8, so the final denominator is 10^10.
	sum.Mul(sum, big.NewInt(p.CNYMicrosPerCurrencyUnit))
	denominator := big.NewInt(10_000_000_000)
	if ceiling {
		sum.Add(sum, new(big.Int).Sub(denominator, big.NewInt(1)))
	}
	sum.Quo(sum, denominator)
	if !sum.IsInt64() || ceiling && sum.Sign() <= 0 {
		return 0, ErrOfficialModelPrice
	}
	return sum.Int64(), nil
}

// OfficialModelBilling binds a conservative admitted usage bound to the same
// frozen official tariff. It is neither a second tariff nor a charge owner.
type OfficialModelBilling struct {
	ReferenceEstimate    ReferenceImageEstimate `json:"referenceEstimate,omitzero"`
	OfficialTariff       OfficialModelTariff
	UsageBound           BillableModelUsage
	BoundEvidenceVersion string
}

func (b OfficialModelBilling) ReservationMicropoints(model string) (int64, error) {
	if b.OfficialTariff.IsReferenceImageEstimate() {
		if model != FlareImageModel || b.OfficialTariff.Channel != "openai-direct" || b.OfficialTariff.Model != model || b.ReferenceEstimate.Validate() != nil || b.BoundEvidenceVersion != "" || b.UsageBound != (BillableModelUsage{}) {
			return 0, ErrOfficialModelPrice
		}
		e := b.ReferenceEstimate
		return b.OfficialTariff.ReservationMicropoints(BillableModelUsage{MappingVersion: ReferenceImagesTotalsV1, Complete: true, OutputImage: e.OutputTokens,
			ReferenceImageTotals: ReferenceImageTotals{Known: true, Input: e.InputTokens, Output: e.OutputTokens, Total: e.InputTokens + e.OutputTokens}})
	}
	if b.ReferenceEstimate != (ReferenceImageEstimate{}) {
		return 0, ErrOfficialModelPrice
	}
	if b.OfficialTariff.Model != model || !validTextFact(b.BoundEvidenceVersion, 128) {
		return 0, ErrOfficialModelPrice
	}
	return b.OfficialTariff.ReservationMicropoints(b.UsageBound)
}
func (b OfficialModelBilling) ActualMicropoints(model string, u BillableModelUsage) (int64, error) {
	maximum, err := b.ReservationMicropoints(model)
	if err != nil || u.Validate() != nil {
		return 0, ErrOfficialModelPrice
	}
	if !b.OfficialTariff.IsReferenceImageEstimate() && (u.InputText > b.UsageBound.InputText || u.InputImage > b.UsageBound.InputImage || u.OutputText > b.UsageBound.OutputText || u.OutputImage > b.UsageBound.OutputImage || u.OutputImageItems > b.UsageBound.OutputImageItems) {
		return 0, ErrOfficialModelPrice
	}
	actual, err := b.OfficialTariff.SettlementMicropoints(u)
	if err != nil || actual > maximum {
		return 0, ErrOfficialModelPrice
	}
	return actual, nil
}
