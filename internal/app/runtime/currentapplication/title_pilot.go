package currentapplication

import (
	"context"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"task-processor/internal/agent"
)

// TitlePilotConfig is an opt-in in the existing private application manifest.
// Pin the live dedicated PostgreSQL instance and each database before enabling
// this operator listener. The optional execution window covers the whole tuple.
type TitlePilotConfig struct {
	Port                int               `json:"port"`
	PostmasterStartedAt time.Time         `json:"postmasterStartedAt"`
	DatabaseOIDs        map[string]uint32 `json:"databaseOids"`
	BatchID             uuid.UUID         `json:"batchId"`
	OrganizationID      string            `json:"organizationId"`
	ActorID             string            `json:"actorId"`
	MemberID            string            `json:"memberId"`
	SamplesDigest       string            `json:"samplesDigest"`
	ProfileDigest       string            `json:"profileDigest"`
	CredentialRevision  string            `json:"credentialRevision"`
	PriceVersion        string            `json:"priceVersion"`
	FXVersion           string            `json:"fxVersion"`
	ExecutionApprovalID string            `json:"executionApprovalId,omitempty"`
	NotBefore           time.Time         `json:"notBefore,omitempty"`
	NotAfter            time.Time         `json:"notAfter,omitempty"`
}

var errTitlePilotIsolation = errors.New("title pilot requires the complete dedicated local application and exact database identities")

func (p TitlePilotConfig) Address() string {
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(p.Port))
}

func (cfg *Config) titlePilotDatabases() map[string]DatabaseConfig {
	return map[string]DatabaseConfig{
		"identity": cfg.SourceAccountDatabase, "product": *cfg.ProductAcquisitionDatabase, "review": cfg.ProductAgent.ReviewDatabase,
		"asset": cfg.ProductAgent.AssetDatabase, "run": cfg.ProductAgent.Database, "commercial": *cfg.CommercialOwnerDatabase,
	}
}

func (cfg *Config) validateTitlePilot() error {
	p := cfg.TitlePilot
	if p == nil {
		return nil
	}
	a := cfg.ProductAgent
	hasWindow := p.ExecutionApprovalID != "" || !p.NotBefore.IsZero() || !p.NotAfter.IsZero()
	if a == nil || !a.Enabled || a.ContentOnly || a.ContentTitleOnly || cfg.ProductAcquisitionDatabase == nil || cfg.CommercialOwnerDatabase == nil ||
		cfg.ChannelCredentials == nil || cfg.ModelRoutePricing == nil || cfg.ModelRoutePricing.Writer == nil ||
		len(a.AllowedOrganizationIDs) != 1 || a.AllowedOrganizationIDs[0] != p.OrganizationID ||
		p.BatchID == uuid.Nil || !(agent.ExecutionIdentity{OrganizationID: p.OrganizationID, ActorID: p.ActorID, MemberID: p.MemberID}).Valid() ||
		p.Port < 1 || p.Port > 65535 || p.Port == cfg.Listen.Port || p.PostmasterStartedAt.IsZero() ||
		hasWindow && (!agent.ValidID(p.ExecutionApprovalID) || p.NotBefore.IsZero() || !p.NotAfter.After(p.NotBefore)) {
		return errTitlePilotIsolation
	}
	if _, err := validateLoopbackURL("issuerURL", cfg.Identity.IssuerURL); err != nil {
		return errTitlePilotIsolation
	}
	if _, err := validateLoopbackURL("authorizationAPIURL", cfg.Identity.AuthorizationAPIURL); err != nil {
		return errTitlePilotIsolation
	}
	for _, digest := range []string{p.SamplesDigest, p.ProfileDigest} {
		if raw, err := hex.DecodeString(digest); err != nil || len(raw) != 32 {
			return errTitlePilotIsolation
		}
	}
	for _, version := range []string{p.CredentialRevision, p.PriceVersion, p.FXVersion} {
		if !boundedValue(version, 128) {
			return errTitlePilotIsolation
		}
	}
	dbs := cfg.titlePilotDatabases()
	if len(p.DatabaseOIDs) != len(dbs) || !sameDatabaseTarget(cfg.ChannelCredentials.Reader, a.Database) ||
		!sameDatabaseTarget(*cfg.ModelRoutePricing.Writer, a.Database) || p.DatabaseOIDs["review"] != p.DatabaseOIDs["product"] ||
		!sameDatabaseTarget(dbs["product"], dbs["review"]) {
		return errTitlePilotIsolation
	}
	for owner, db := range dbs {
		if db.Host != "127.0.0.1" || db.Port != a.Database.Port || db.Port == p.Port || p.DatabaseOIDs[owner] == 0 ||
			owner != "run" && (db.Database == a.Database.Database || p.DatabaseOIDs[owner] == p.DatabaseOIDs["run"]) {
			return errTitlePilotIsolation
		}
	}
	return nil
}

func (cfg *Config) verifyTitlePilotDatabases(ctx context.Context, pools map[string]*gorm.DB) error {
	for owner, want := range cfg.titlePilotDatabases() {
		if err := cfg.TitlePilot.verifyDatabase(ctx, pools[owner], want, owner); err != nil {
			return err
		}
	}
	return nil
}

func (p TitlePilotConfig) verifyDatabase(ctx context.Context, db *gorm.DB, want DatabaseConfig, owner string) error {
	expectedOID := p.DatabaseOIDs[owner]
	if db == nil || expectedOID == 0 || p.PostmasterStartedAt.IsZero() {
		return errTitlePilotIsolation
	}
	var fact struct {
		Name      string
		Role      string
		OID       uint32 `gorm:"column:oid"`
		StartedAt time.Time
	}
	q := db.WithContext(ctx).Raw(`SELECT current_database() AS name, session_user AS role, oid::bigint AS oid,
		pg_postmaster_start_time() AS started_at FROM pg_catalog.pg_database WHERE datname = current_database()`).Scan(&fact)
	if q.Error != nil || q.RowsAffected != 1 || fact.Name != want.Database || fact.Role != want.User ||
		fact.OID != expectedOID || !fact.StartedAt.Equal(p.PostmasterStartedAt) {
		return errTitlePilotIsolation
	}
	return nil
}

// Serve both listeners under the same process lifetime. Stop accepting and
// cancel the operator request before waiting for existing outcome settlement.
func serveWithTitlePilot(ctx context.Context, main, operator *http.Server, listener, local net.Listener, timeout time.Duration) error {
	requests, cancelRequests := context.WithCancel(ctx)
	defer cancelRequests()
	operator.BaseContext = func(net.Listener) context.Context { return requests }
	results := make(chan error, 2)
	go func() { results <- main.Serve(listener) }()
	go func() { results <- operator.Serve(local) }()
	remaining := 2
	var result error
	select {
	case result = <-results:
		remaining--
	case <-ctx.Done():
	}
	cancelRequests()
	shutdown, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	shutdowns := make(chan error, 2)
	for _, server := range []*http.Server{main, operator} {
		go func() {
			err := server.Shutdown(shutdown)
			if err != nil {
				err = errors.Join(err, server.Close())
			}
			shutdowns <- err
		}()
	}
	if errors.Is(result, http.ErrServerClosed) {
		result = nil
	}
	result = errors.Join(result, <-shutdowns, <-shutdowns)
	for range remaining {
		if err := <-results; err != nil && !errors.Is(err, http.ErrServerClosed) {
			result = errors.Join(result, err)
		}
	}
	return result
}
