package storecenterapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/amazon"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/credentialprotection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	amazonmarket "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/amazonus"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

type AmazonProductConnectionReader interface {
	ReadAmazonProductConnection(context.Context, string, string, string, string, storecenter.AmazonAuthorizationApplication, time.Time) (storecenter.AmazonProductConnection, error)
}

type AmazonProductProvider interface {
	RefreshAccess(context.Context, string) (amazon.Tokens, error)
	GetProductTypeDefinition(context.Context, string, string, string) (amazon.ProductTypeDefinition, error)
	Preview(context.Context, string, string, string, amazon.ListingInput) (amazon.PreviewResult, error)
	CheckSKUUnused(context.Context, string, string, string) (bool, error)
	CheckGTINUnused(context.Context, string, string) (bool, error)
	Submit(context.Context, string, string, string, amazon.ListingInput) (amazon.SubmissionReceipt, error)
	GetListing(context.Context, string, string, string) (amazon.ListingObservation, error)
}

type AmazonProductAccess struct {
	reader    AmazonProductConnectionReader
	provider  AmazonProductProvider
	protector *credentialprotection.AESGCM
	app       storecenter.AmazonAuthorizationApplication
}

func NewAmazonProductAccess(reader AmazonProductConnectionReader, provider AmazonProductProvider, protector *credentialprotection.AESGCM, app storecenter.AmazonAuthorizationApplication) (*AmazonProductAccess, error) {
	if reader == nil || provider == nil || protector == nil || app.Revision == "" || app.RedirectURI == "" {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	if _, e := amazon.ConsentURL(app.ID, "configuration-check", app.Beta); e != nil {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	return &AmazonProductAccess{reader: reader, provider: provider, protector: protector, app: app}, nil
}

// AmazonProductSession holds a short-lived token only inside the Store app.
// No field is exported to a client, record or background workflow.
type AmazonProductSession struct {
	sellerID, accessToken string
	binding               storecenter.ProductMerchantBinding
	provider              AmazonProductProvider
}

func (a *AmazonProductAccess) Open(ctx context.Context, subject storecenter.ProductExecutionSubject, storeID string, expected *storecenter.ProductMerchantBinding) (*AmazonProductSession, error) {
	connection, err := a.reader.ReadAmazonProductConnection(ctx, subject.OrganizationID, subject.ActorID, subject.MemberID, storeID, a.app, time.Now().UTC())
	if err != nil || expected != nil && connection.Binding != *expected {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	plain, err := a.protector.Open(connection.KeyID, connection.Ciphertext, connection.CredentialAAD())
	if err != nil {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	defer clear(plain)
	tokens, err := a.provider.RefreshAccess(ctx, string(plain))
	if err != nil || tokens.AccessToken == "" {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	if !a.Current(ctx, subject, connection.Binding) {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	return &AmazonProductSession{sellerID: connection.SellerID, accessToken: tokens.AccessToken, binding: connection.Binding, provider: a.provider}, nil
}

func (a *AmazonProductAccess) Current(ctx context.Context, subject storecenter.ProductExecutionSubject, binding storecenter.ProductMerchantBinding) bool {
	current, err := a.Binding(ctx, subject, binding.StoreID)
	return err == nil && current == binding
}

func (a *AmazonProductAccess) Binding(ctx context.Context, subject storecenter.ProductExecutionSubject, storeID string) (storecenter.ProductMerchantBinding, error) {
	current, err := a.reader.ReadAmazonProductConnection(ctx, subject.OrganizationID, subject.ActorID, subject.MemberID, storeID, a.app, time.Now().UTC())
	return current.Binding, err
}

func (s *AmazonProductSession) Binding() storecenter.ProductMerchantBinding { return s.binding }

// OccupancyPartition is only for the isolated Listing occupancy table. It
// must never be included in a Target, receipt projection or provider request.
func (s *AmazonProductSession) OccupancyPartition() string {
	digest := sha256.Sum256([]byte("amazon-us-seller-occupancy-v1\x00" + s.sellerID))
	return hex.EncodeToString(digest[:])
}

func (s *AmazonProductSession) GetDefinition(ctx context.Context, productType string) (amazon.ProductTypeDefinition, error) {
	return s.provider.GetProductTypeDefinition(ctx, s.accessToken, s.sellerID, productType)
}

func (s *AmazonProductSession) Preview(ctx context.Context, sku string, input amazon.ListingInput) (amazon.PreviewResult, error) {
	return s.provider.Preview(ctx, s.accessToken, s.sellerID, sku, input)
}

func (s *AmazonProductSession) CheckSKUUnused(ctx context.Context, sku string) (bool, error) {
	return s.provider.CheckSKUUnused(ctx, s.accessToken, s.sellerID, sku)
}

func (s *AmazonProductSession) CheckGTINUnused(ctx context.Context, gtin string) (bool, error) {
	return s.provider.CheckGTINUnused(ctx, s.accessToken, gtin)
}

func (a *AmazonProductAccess) Submit(ctx context.Context, scope collection.Scope, session *AmazonProductSession, draft amazonmarket.Draft,
	permit *submission.AmazonWritePermit, authorize func(context.Context) error,
) (amazon.SubmissionReceipt, error) {
	subject := storecenter.ProductExecutionSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID}
	if session == nil || authorize == nil || !a.Current(ctx, subject, session.binding) {
		return amazon.SubmissionReceipt{}, storecenter.ErrOfficialConnectionUnavailable
	}
	if err := authorize(ctx); err != nil {
		return amazon.SubmissionReceipt{}, err
	}
	if ctx.Err() != nil ||
		permit.Consume(scope, session.binding, draft, time.Now()) != nil {
		return amazon.SubmissionReceipt{}, storecenter.ErrOfficialConnectionUnavailable
	}
	return a.provider.Submit(ctx, session.accessToken, session.sellerID, draft.SKU,
		amazon.ListingInput{ProductType: draft.ProductType, Attributes: draft.Attributes})
}

func (s *AmazonProductSession) GetListing(ctx context.Context, sku string) (amazon.ListingObservation, error) {
	return s.provider.GetListing(ctx, s.accessToken, s.sellerID, sku)
}
