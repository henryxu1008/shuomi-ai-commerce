package storecenterapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	f "task-processor/internal/fulfillment"
	sf "task-processor/internal/marketplace/shein/fulfillment"
	"task-processor/internal/storecenter"
	"time"
)

type OfficialFulfillmentProvider interface {
	QueryFulfillmentSource(context.Context, storecenter.OfficialMerchantCredential, string, string) (sf.Source, error)
	QueryFulfillmentOptions(context.Context, storecenter.OfficialMerchantCredential, sf.Source, sf.Selection) (sf.ShippingOptions, error)
	BuildFulfillmentPlan(context.Context, storecenter.OfficialMerchantCredential, sf.Source, sf.Delivery, sf.Selection) (sf.Plan, error)
	ExecuteFulfillmentPlan(context.Context, storecenter.OfficialMerchantCredential, sf.Plan) (sf.MutationResult, error)
}

// FulfillmentAccess uses the current Store owner directly. Observation handles remain read-only.
type FulfillmentAccess struct {
	Reader        storecenter.FulfillmentReader
	Authorization storecenter.FulfillmentAuthorizer
	Applications  *OfficialApplicationRegistry
}

type FulfillmentMerchant struct {
	owner                      *FulfillmentAccess
	subject                    storecenter.FulfillmentSubject
	store, kind                string
	binding                    MerchantBinding
	connection, credentialHash string
}

func (a *FulfillmentAccess) Open(ctx context.Context, s storecenter.FulfillmentSubject, store, kind string) (*FulfillmentMerchant, error) {
	if a == nil || a.Reader == nil || a.Authorization == nil || a.Applications == nil || !storecenter.ValidFulfillmentPurpose(s.Purpose) || kind != "consumer_order" && kind != "platform_supply" {
		return nil, f.ErrUnavailable
	}
	material, err := a.Reader.ReadFulfillment(ctx, s, store, a.Authorization, time.Now())
	if err != nil {
		return nil, f.ErrForbidden
	}
	entry, err := a.Applications.resolve(material.Attempt.AppID, material.Attempt.AppVersion, material.Platform, material.Attempt.IdentityPartition)
	if err != nil {
		return nil, f.ErrUnavailable
	}
	if _, ok := entry.provider.(OfficialFulfillmentProvider); !ok {
		return nil, f.ErrUnavailable
	}
	if kind == "platform_supply" && entry.mode != storecenter.ApplicationFullyManaged || kind == "consumer_order" && entry.mode == storecenter.ApplicationFullyManaged {
		return nil, f.ErrConflict
	}
	credential, err := entry.protection.Open(material.Attempt, material.Attempt.KeyID, material.Attempt.Ciphertext)
	if err != nil || !validOfficialMaterial(s.OrganizationID, store, material, credential, entry, time.Now()) {
		return nil, f.ErrForbidden
	}
	binding := officialMerchantBinding(s.OrganizationID, store, material, credential, entry.mode)
	if kind == "platform_supply" {
		binding.Site = "unknown"
	}
	return &FulfillmentMerchant{a, s, store, kind, binding, material.Connection.AttemptID, privateCredentialHash(material)}, nil
}
func (h *FulfillmentMerchant) BindingDigest() string {
	if h == nil {
		return ""
	}
	raw, _ := json.Marshal(h.binding)
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}
func (h *FulfillmentMerchant) current(ctx context.Context) (officialApplicationEntry, storecenter.OfficialMerchantCredential, error) {
	var empty officialApplicationEntry
	if h == nil || h.owner == nil {
		return empty, storecenter.OfficialMerchantCredential{}, f.ErrForbidden
	}
	fresh, err := h.owner.Open(ctx, h.subject, h.store, h.kind)
	if err != nil || fresh.binding != h.binding || fresh.connection != h.connection || fresh.credentialHash != h.credentialHash {
		return empty, storecenter.OfficialMerchantCredential{}, f.ErrForbidden
	}
	material, err := h.owner.Reader.ReadFulfillment(ctx, h.subject, h.store, h.owner.Authorization, time.Now())
	if err != nil || material.Connection.AttemptID != h.connection || privateCredentialHash(material) != h.credentialHash {
		return empty, storecenter.OfficialMerchantCredential{}, f.ErrForbidden
	}
	entry, err := h.owner.Applications.resolve(material.Attempt.AppID, material.Attempt.AppVersion, material.Platform, material.Attempt.IdentityPartition)
	if err != nil {
		return empty, storecenter.OfficialMerchantCredential{}, f.ErrUnavailable
	}
	credential, err := entry.protection.Open(material.Attempt, material.Attempt.KeyID, material.Attempt.Ciphertext)
	if err != nil || !validOfficialMaterial(h.subject.OrganizationID, h.store, material, credential, entry, time.Now()) {
		return empty, storecenter.OfficialMerchantCredential{}, f.ErrForbidden
	}
	return entry, credential, nil
}
func (h *FulfillmentMerchant) Source(ctx context.Context, id string) (sf.Source, error) {
	entry, credential, err := h.current(ctx)
	if err != nil {
		return sf.Source{}, err
	}
	return entry.provider.(OfficialFulfillmentProvider).QueryFulfillmentSource(ctx, credential, h.kind, id)
}
func (h *FulfillmentMerchant) Check(ctx context.Context) error {
	_, _, err := h.current(ctx)
	return err
}

func (h *FulfillmentMerchant) Options(ctx context.Context, source sf.Source, selection sf.Selection) (sf.ShippingOptions, error) {
	entry, credential, err := h.current(ctx)
	if err != nil {
		return sf.ShippingOptions{}, err
	}
	return entry.provider.(OfficialFulfillmentProvider).QueryFulfillmentOptions(ctx, credential, source, selection)
}
func (h *FulfillmentMerchant) Build(ctx context.Context, source sf.Source, delivery sf.Delivery, selection sf.Selection) (sf.Plan, error) {
	entry, credential, err := h.current(ctx)
	if err != nil {
		return sf.Plan{}, err
	}
	return entry.provider.(OfficialFulfillmentProvider).BuildFulfillmentPlan(ctx, credential, source, delivery, selection)
}
func (h *FulfillmentMerchant) Send(ctx context.Context, plan sf.Plan) (sf.MutationResult, error) {
	empty := sf.MutationResult{}
	if h == nil || h.subject.Purpose != "fulfillment.execute" {
		return empty, f.ErrForbidden
	}
	if plan.Purpose == "accept_address" || plan.Purpose == "read_address" {
		addressSubject := h.subject
		addressSubject.Purpose = "fulfillment.address"
		material, err := h.owner.Reader.ReadFulfillment(ctx, addressSubject, h.store, h.owner.Authorization, time.Now())
		if err != nil || material.Connection.AttemptID != h.connection || privateCredentialHash(material) != h.credentialHash {
			return empty, f.ErrForbidden
		}
	}
	entry, credential, err := h.current(ctx)
	if err != nil {
		return empty, err
	}
	return entry.provider.(OfficialFulfillmentProvider).ExecuteFulfillmentPlan(ctx, credential, plan)
}
