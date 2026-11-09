package storecenterapp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/integration/temu"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
)

func (p *temuRulesProvider) QueryObservedTemuOrders(ctx context.Context, _ storecenter.OfficialMerchantCredential, _ o.TemuOrderQuery) (o.TemuOrderPage, error) {
	p.metadataReads++
	p.deadline, _ = ctx.Deadline()
	if p.afterMetadata != nil {
		p.afterMetadata()
	}
	return o.TemuOrderPage{Total: 0, Items: []o.TemuOrder{}}, nil
}

func newTemuOrderAccess(t *testing.T, mode storecenter.OfficialMerchantMode) (*OfficialObservationAccess, *observationMaterialReader, *temuRulesProvider, storecenter.ObservationSubject) {
	t.Helper()
	_, original, p, s := newTemuRulesAccess(t, storecenter.PartitionUS, storecenter.MerchantLocal)
	p.application.SupportedModes = storecenter.ModesLocal | storecenter.ModesSemiManaged
	p.application.Version = "v1:temu:US:manual_token:3:r2"
	p.facts.Mode = mode
	p.facts.Scopes = []string{"bg.order.list.v2.get", "bg.order.amount.query"}
	p.facts.ContractRevision = "temu-regional-order-identity-v2"
	if mode == storecenter.MerchantSemiManaged {
		p.facts.Namespace = storecenter.MerchantRegionalSemiNamespace
		p.facts.SemiUniqueID = "semi-1"
	}
	m := original.material
	m.Attempt.AppVersion = p.application.Version
	m.Connection.AppRevision = p.application.Version
	m.Connection.Mode = mode
	raw, e := json.Marshal(p.facts)
	require.NoError(t, e)
	m.Attempt.VerificationJSON = string(raw)
	m.Connection.MerchantIdentityHash, e = p.facts.MerchantIdentityHash()
	require.NoError(t, e)
	protection, e := temu.NewCredentialProtection("order-key", make([]byte, 32))
	require.NoError(t, e)
	m.Attempt.KeyID, m.Attempt.Ciphertext, e = protection.Seal(m.Attempt, storecenter.OfficialMerchantCredential{AppID: p.application.AppID, Temu: &storecenter.TemuMerchantCredential{AccessToken: "synthetic-order-token"}})
	require.NoError(t, e)
	registry, e := NewOfficialApplicationRegistry([]OfficialApplicationRegistration{{Provider: p, Protection: protection}})
	require.NoError(t, e)
	r := &observationMaterialReader{material: m}
	a, e := NewOfficialObservationAccess(r, observationLiveAccess{}, registry)
	require.NoError(t, e)
	a.now = func() time.Time { return p.facts.VerifiedAt.Add(time.Second) }
	return a, r, p, storecenter.ObservationSubject{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID, Purpose: storecenter.ObservationPurposeOrders, Sync: true}
}

func TestTemuOrdersUseOwnBindingAndDiscardLateAuthorityChanges(t *testing.T) {
	for _, mode := range []storecenter.OfficialMerchantMode{storecenter.MerchantLocal, storecenter.MerchantSemiManaged} {
		t.Run(string(mode), func(t *testing.T) {
			for _, change := range []string{"none", "revoked", "connection", "cipher", "service", "identity", "scope", "cancel"} {
				t.Run(change, func(t *testing.T) {
					a, r, p, s := newTemuOrderAccess(t, mode)
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					h, e := a.Authorize(ctx, s, "store-a", nil)
					require.NoError(t, e)
					require.True(t, h.Binding().ValidTemuOrderApplication())
					require.False(t, h.Binding().ValidTemuApplication())
					scope := o.Scope{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID}
					binding := h.Binding()
					p.afterMetadata = func() {
						switch change {
						case "revoked":
							r.err = storecenter.ErrNotFound
						case "connection":
							r.material.Connection.Version++
						case "cipher":
							r.material.Attempt.Ciphertext = "changed"
						case "service":
							r.material.ServiceExpiresAt = a.now()
						case "identity":
							p.facts.MerchantID = "another-mall"
						case "scope":
							p.facts.Scopes = []string{"product-only"}
						case "cancel":
							cancel()
						}
					}
					reader, ok := any(a).(o.TemuOrderReader)
					require.True(t, ok, "formal TemuOrders port must be wired")
					page, e := reader.ReadTemuOrders(ctx, scope, binding, o.TemuOrderQuery{Start: a.now().Add(-time.Hour).Truncate(time.Second), End: a.now().Truncate(time.Second), Page: 1})
					if change == "none" {
						require.NoError(t, e)
						require.NotNil(t, page.Items)
						require.WithinDuration(t, a.now().Add(30*time.Second), p.deadline, time.Second)
					} else {
						require.Error(t, e)
						require.Empty(t, page.Items)
					}
				})
			}
		})
	}
}

func TestTemuOrdersRejectProductsAndFreshIdentityOrScopeMismatch(t *testing.T) {
	for _, change := range []string{"product_purpose", "missing_list", "identity", "scope", "expired"} {
		t.Run(change, func(t *testing.T) {
			a, r, p, s := newTemuOrderAccess(t, storecenter.MerchantSemiManaged)
			switch change {
			case "product_purpose":
				s.Purpose = storecenter.ObservationPurposeProducts
			case "missing_list":
				p.facts.Scopes = []string{"bg.order.amount.query"}
				raw, _ := json.Marshal(p.facts)
				r.material.Attempt.VerificationJSON = string(raw)
			case "identity":
				p.facts.MerchantID = "other"
			case "scope":
				p.facts.Scopes = []string{"bg.order.list.v2.get"}
			case "expired":
				r.material.ServiceExpiresAt = a.now()
			}
			_, e := a.Authorize(context.Background(), s, "store-a", nil)
			require.Error(t, e)
		})
	}
}
