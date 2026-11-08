package storecenterapp

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

func TestSavedTemuOrdersReadOnlyOriginalRegionalFacts(t *testing.T) {
	for _, mode := range []storecenter.OfficialMerchantMode{storecenter.MerchantLocal, storecenter.MerchantSemiManaged} {
		t.Run(string(mode), func(t *testing.T) {
			_, original, p, s := newTemuRulesAccess(t, storecenter.PartitionGlobal, storecenter.MerchantLocal)
			p.application.SupportedModes = storecenter.ModesLocal | storecenter.ModesSemiManaged
			p.application.Version = "v1:temu:GLOBAL:manual_token:3:r2"
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
			r := &observationMaterialReader{material: m}
			registry, e := NewOfficialApplicationRegistry([]OfficialApplicationRegistration{{Provider: p, Protection: savedTemuNoSecrets{}}})
			require.NoError(t, e)
			access, e := NewOfficialObservationAccess(r, observationLiveAccess{}, registry)
			require.NoError(t, e)
			access.now = func() time.Time { return p.facts.VerifiedAt.Add(time.Second) }
			subject := storecenter.ObservationSubject{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID, Purpose: storecenter.ObservationPurposeOrders}
			b, e := access.ReadSavedTemuOrderBinding(context.Background(), subject, "store-a")
			require.NoError(t, e)
			require.True(t, b.ValidTemuOrderApplication())
			require.False(t, b.ValidTemuApplication())
			require.Equal(t, 0, p.identityReads)
			require.Equal(t, subject, r.subject)
			for _, change := range []string{"product", "sync", "identity", "scope", "expired"} {
				saved := r.material
				sub := subject
				switch change {
				case "product":
					sub.Purpose = storecenter.ObservationPurposeProducts
				case "sync":
					sub.Sync = true
				case "identity":
					r.material.Connection.MerchantIdentityHash = strings.Repeat("e", 64)
				case "scope":
					r.material.Attempt.VerificationJSON = strings.Replace(r.material.Attempt.VerificationJSON, "bg.order.list.v2.get", "bad scope", 1)
				case "expired":
					r.material.ServiceExpiresAt = access.now()
				}
				_, e = access.ReadSavedTemuOrderBinding(context.Background(), sub, "store-a")
				require.Error(t, e)
				r.material = saved
			}
			require.Equal(t, 0, p.identityReads)
		})
	}
}

type savedTemuNoSecrets struct {
	storecenter.OfficialCredentialProtection
}

func (savedTemuNoSecrets) Open(storecenter.OfficialConnectionAttempt, string, string) (storecenter.OfficialMerchantCredential, error) {
	panic("saved order read must not open credentials")
}
