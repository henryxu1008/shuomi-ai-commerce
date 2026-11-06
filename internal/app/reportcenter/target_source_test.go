package reportcenterapp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	target "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	"task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/collection"
	rc "task-processor/internal/reportcenter"
	"task-processor/internal/storecenter"
)

type targetReadFunc func(context.Context, string) (target.TargetRecord, error)

func (f targetReadFunc) ReadRecord(ctx context.Context, id string) (target.TargetRecord, error) {
	return f(ctx, id)
}

type listingReadPolicy struct{}

func (listingReadPolicy) Authorize(_ string, _ []string, permission string) bool {
	return permission == preparation.PermissionRead
}

func nativeTargetFixture(scope rc.Scope) target.TargetRecord {
	sourceID, storeID := uuid.NewString(), uuid.NewString()
	r := target.TargetRecord{
		ID: uuid.NewString(), TargetID: target.TargetIdentity(collection.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, sourceID, storeID), Revision: 2,
		Source:           preparation.SourceItem{ID: sourceID, Source: collection.Source{ProductKey: "product-a", Version: 3, PublicationID: uuid.NewString(), Kind: "product"}},
		EffectiveVersion: 3, ProductHash: strings.Repeat("a", 64), InventoryHash: strings.Repeat("b", 64), RulesHash: strings.Repeat("c", 64),
		Merchant:          storecenter.ProductMerchantBinding{OrganizationID: scope.OrganizationID, StoreID: storeID, Site: "shein-us", ApplicationID: "private-application-handle"},
		Input:             target.TargetInput{SourceID: sourceID, StoreID: storeID, ExpectedRevision: 1, EffectiveVersion: 3, Draft: goods.OfficialDraftInput{Product: model.PublishProduct{Names: []model.LanguageContent{{Language: "en", Name: "Saved title"}}, Descriptions: []model.LanguageContent{{Language: "en", Name: "<b>plain historical text</b>"}}, SKCs: []model.ProductSKC{{SKUs: []model.ProductSKU{{SupplierSKU: "not-a-full-sku-export"}}}}}, Images: []goods.OfficialImageSlot{{AssetID: "private-asset-handle"}}}},
		Result:            goods.OfficialDraft{Issues: []goods.OfficialIssue{{Field: "images", Code: "missing", Message: "原保存时图片未完成"}}, SubmissionPayload: json.RawMessage(`{"private":"provider-payload-secret"}`)},
		ImageObservations: []goods.OfficialImageObservation{{SourceURL: "https://private.invalid/image", RemoteURL: "https://private.invalid/temporary"}},
		CreatedAt:         time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
	}
	return r
}

func TestNativeTargetReportCapturesExactHistoricalWhitelist(t *testing.T) {
	scope := rc.Scope{OrganizationID: "org-a", ActorID: "actor-a"}
	identity := authidentity.AuthenticatedIdentity{UserID: scope.ActorID, TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, EffectiveMemberID: "member-a", TokenExpiresAt: time.Now().Add(time.Hour).UTC()}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), identity)
	r := nativeTargetFixture(scope)
	calls := 0
	s := Sources{Policy: listingReadPolicy{}, TargetRecords: targetReadFunc(func(got context.Context, id string) (target.TargetRecord, error) {
		calls++
		current, _ := authidentity.AuthenticatedIdentityFromContext(got)
		require.Equal(t, identity, current)
		require.Equal(t, r.ID, id)
		return r, nil
	})}
	first, err := s.Read(ctx, scope, "SHEIN_TARGET_RECORD", r.ID)
	require.NoError(t, err)
	body, digest, err := first.Bytes()
	require.NoError(t, err)
	wantVersion := "sha256:" + collection.Digest(struct {
		Projection string `json:"projection"`
		BodyDigest string `json:"bodyDigest"`
	}{"shein-target-report-v1", collection.Digest(r)})
	require.Equal(t, wantVersion, first.Ref.Version)
	require.Equal(t, r.CreatedAt, *first.SourceAt)
	for _, value := range []string{r.ID, r.TargetID, r.ProductHash, "Saved title", "plain historical text", "原保存时图片未完成", "未重新", "NOT_RUN"} {
		require.Contains(t, string(body), value)
	}
	for _, secret := range []string{"provider-payload-secret", "private.invalid", "private-application-handle", "private-asset-handle", "not-a-full-sku-export"} {
		require.NotContains(t, string(body), secret)
	}
	second, err := s.Read(ctx, scope, "SHEIN_TARGET_RECORD", r.ID)
	require.NoError(t, err)
	_, nextDigest, err := second.Bytes()
	require.NoError(t, err)
	require.Equal(t, first.Ref, second.Ref)
	require.Equal(t, digest, nextDigest)
	require.Equal(t, 2, calls)
	// This exact ID belongs only to the native owner; no legacy fallback.
	s.Policy = allowPolicy(true)
	_, err = s.Read(ctx, scope, "SHEIN_RECORD", r.ID)
	require.ErrorIs(t, err, rc.ErrUnavailable)
	require.Equal(t, 2, calls)
}

func TestNativeTargetReportRejectsWrongOwnerShapeAndBounds(t *testing.T) {
	scope := rc.Scope{OrganizationID: "org-a", ActorID: "actor-a"}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: scope.ActorID, TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, EffectiveMemberID: "member-a", TokenExpiresAt: time.Now().Add(time.Hour)})
	for name, mutate := range map[string]func(*target.TargetRecord){
		"id":           func(r *target.TargetRecord) { r.ID = uuid.NewString() },
		"organization": func(r *target.TargetRecord) { r.Merchant.OrganizationID = "other-org" },
		"non-shein":    func(r *target.TargetRecord) { r.Merchant.Platform = storecenter.PlatformWalmart },
		"incomplete":   func(r *target.TargetRecord) { r.Source.ID = "" },
		"text bound":   func(r *target.TargetRecord) { r.Input.Draft.Product.Names[0].Name = strings.Repeat("x", 16001) },
		"all issues required": func(r *target.TargetRecord) {
			for i := 0; i < 256; i++ {
				r.Result.Issues = append(r.Result.Issues, goods.OfficialIssue{Field: "field", Code: "code", Message: "message"})
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := nativeTargetFixture(scope)
			id := r.ID
			mutate(&r)
			s := Sources{Policy: allowPolicy(true), TargetRecords: targetReadFunc(func(context.Context, string) (target.TargetRecord, error) { return r, nil })}
			_, err := s.Read(ctx, scope, "SHEIN_TARGET_RECORD", id)
			require.Error(t, err)
		})
	}
	for _, sourceErr := range []error{preparation.ErrForbidden, target.ErrNotFound, collection.ErrForbidden} {
		s := Sources{Policy: allowPolicy(true), TargetRecords: targetReadFunc(func(context.Context, string) (target.TargetRecord, error) { return target.TargetRecord{}, sourceErr })}
		_, err := s.Read(ctx, scope, "SHEIN_TARGET_RECORD", uuid.NewString())
		if sourceErr == target.ErrNotFound {
			require.ErrorIs(t, err, rc.ErrNotFound)
		} else {
			require.ErrorIs(t, err, rc.ErrForbidden)
		}
	}
}
