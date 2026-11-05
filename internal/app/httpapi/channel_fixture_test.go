package httpapi

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	credential "task-processor/internal/aicapability/channelcredential"
	channelstore "task-processor/internal/integration/persistence/channelcredential"
)

// Synthetic HTTP fixtures keep model/protocol in their separate RoutePolicy.
// Only connection identity and encrypted Key are seeded in the real new owner.
type httpModelFixture struct {
	TenantID, UserID, ClientName, APIKey, BaseURL, Model, APIStyle string
	TimeoutSecond                                                  int
	Enabled                                                        bool
	reference                                                      string
}
type httpChannelFixture struct {
	db    *gorm.DB
	owner *channelstore.Store
}

func installHTTPChannelFixture(db *gorm.DB) error {
	if e := db.Exec("DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='product_agent_owner') THEN CREATE ROLE product_agent_owner NOLOGIN; END IF; END $$").Error; e != nil {
		return e
	}
	// Only fresh, task-owned fixture databases reach this helper. Exercise the
	// canonical installer so its metadata view has the admitted owner identity.
	if e := db.Exec("GRANT USAGE, CREATE ON SCHEMA public TO product_agent_owner").Error; e != nil {
		return e
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec("SET LOCAL ROLE product_agent_owner").Error; e != nil {
			return e
		}
		return channelstore.EnsureSchema(context.Background(), tx)
	})
}
func newHTTPChannelFixture(db *gorm.DB) *httpChannelFixture {
	p, _ := channelstore.NewProtection("synthetic-http-v1", []byte(strings.Repeat("h", 32)))
	s, _ := channelstore.New(db, p)
	return &httpChannelFixture{db, s}
}
func (f *httpChannelFixture) seed(ctx context.Context, in httpModelFixture) error {
	if in.UserID != "" {
		return credential.ErrInvalid
	}
	var existing channelstore.ConnectionRow
	e := f.db.Where("organization_id=? AND connection_id=?", in.TenantID, in.ClientName).Take(&existing).Error
	revision := int64(0)
	if e == nil {
		revision = existing.Revision
	}
	_, e = f.owner.Publish(ctx, credential.Command{OrganizationID: in.TenantID, ActorID: "synthetic-operator", Key: uuid.NewString(), ConnectionID: in.ClientName, ExpectedRevision: revision, Provider: "synthetic", Channel: "fixture-channel", Account: "synthetic", Region: "global", BaseURL: in.BaseURL, Secret: func() string {
		if !in.Enabled {
			return ""
		}
		return in.APIKey
	}(), Enabled: in.Enabled})
	return e
}
func (f *httpChannelFixture) record(ctx context.Context, org, member, id string) (*httpModelFixture, error) {
	if member != "" {
		return nil, credential.ErrInvalid
	}
	m, e := f.owner.Metadata(ctx, org, id)
	if e != nil {
		return nil, e
	}
	s, e := f.owner.Resolve(ctx, org, id, m.Reference)
	if e != nil {
		return nil, e
	}
	return &httpModelFixture{TenantID: org, ClientName: id, BaseURL: m.BaseURL, APIKey: s.APIKey, Enabled: m.Enabled, reference: m.Reference}, nil
}
func (f *httpChannelFixture) Resolve(ctx context.Context, org, id, ref string) (credential.Secret, error) {
	return f.owner.Resolve(ctx, org, id, ref)
}
func httpCredentialReference(r httpModelFixture) string { return r.reference }
