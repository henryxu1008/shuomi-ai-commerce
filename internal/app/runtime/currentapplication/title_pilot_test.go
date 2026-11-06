package currentapplication

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestTitlePilotOperatorRequiresDedicatedComposition(t *testing.T) {
	db := func(name string) DatabaseConfig {
		return DatabaseConfig{Host: "127.0.0.1", Port: 15100, Database: name}
	}
	config := func() *Config {
		run, product, commercial := db("run"), db("product"), db("commercial")
		return &Config{TitlePilot: &TitlePilotConfig{Port: 15101, PostmasterStartedAt: time.Now(),
			DatabaseOIDs: map[string]uint32{"identity": 1, "product": 2, "review": 2, "asset": 3, "run": 4, "commercial": 5},
			BatchID:      uuid.New(), OrganizationID: "org", ActorID: "operator", MemberID: "member",
			SamplesDigest: strings.Repeat("a", 64), ProfileDigest: strings.Repeat("b", 64),
			CredentialRevision: "r1", PriceVersion: "p1", FXVersion: "fx1"},
			Listen:                ListenConfig{Host: "127.0.0.1", Port: 15102},
			Identity:              IdentityConfig{IssuerURL: "http://127.0.0.1:15103", AuthorizationAPIURL: "http://127.0.0.1:15103"},
			SourceAccountDatabase: db("identity"), ProductAcquisitionDatabase: &product, CommercialOwnerDatabase: &commercial,
			ProductAgent: &ProductAgentConfig{Enabled: true, AllowedOrganizationIDs: []string{"org"}, Database: run,
				ReviewDatabase: product, AssetDatabase: db("asset")},
			ChannelCredentials: &ChannelCredentialsConfig{Reader: run}, ModelRoutePricing: &ModelRoutePricingConfig{Writer: &run}}
	}
	require.NoError(t, (&Config{}).validateTitlePilot(), "ordinary composition needs no pilot configuration")
	require.NoError(t, config().validateTitlePilot(), "a complete private preflight needs no paid start authorization")
	for name, mutate := range map[string]func(*Config){
		"content-only":            func(c *Config) { c.ProductAgent.ContentOnly = true },
		"wrong credential target": func(c *Config) { c.ChannelCredentials.Reader.Database = "other" },
		"review not Product":      func(c *Config) { c.TitlePilot.DatabaseOIDs["review"] = 42 },
		"nonisolated identity":    func(c *Config) { c.Identity.IssuerURL = "https://identity.example.com" },
		"second organization": func(c *Config) {
			c.ProductAgent.AllowedOrganizationIDs = append(c.ProductAgent.AllowedOrganizationIDs, "other")
		},
		"missing execution approval": func(c *Config) {
			c.TitlePilot.NotBefore, c.TitlePilot.NotAfter = time.Now(), time.Now().Add(time.Minute)
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := config()
			mutate(c)
			require.ErrorIs(t, c.validateTitlePilot(), errTitlePilotIsolation)
		})
	}
}

func TestTitlePilotOperatorChecksLiveDatabaseIdentity(t *testing.T) {
	dsn := os.Getenv("ISSUE132_TEST_DSN")
	if dsn == "" {
		t.Skip("requires a task-owned ISSUE132_TEST_DSN")
	}
	require.Contains(t, dsn, "dbname=issue132_rehearsal")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	var pin struct {
		OID       uint32 `gorm:"column:oid"`
		StartedAt time.Time
	}
	require.NoError(t, db.Raw(`SELECT oid::bigint AS oid, pg_postmaster_start_time() AS started_at
		FROM pg_catalog.pg_database WHERE datname = current_database()`).Scan(&pin).Error)
	require.Positive(t, pin.OID)
	require.False(t, pin.StartedAt.IsZero())
	p := TitlePilotConfig{PostmasterStartedAt: pin.StartedAt, DatabaseOIDs: map[string]uint32{"run": pin.OID}}
	database := DatabaseConfig{Database: "issue132_rehearsal", User: "issue398_owner"}
	ctx := context.Background()
	require.NoError(t, p.verifyDatabase(ctx, db, database, "run"))
	require.ErrorIs(t, p.verifyDatabase(ctx, db, database, "review"), errTitlePilotIsolation)
	p.DatabaseOIDs["run"]++
	require.ErrorIs(t, p.verifyDatabase(ctx, db, database, "run"), errTitlePilotIsolation)
	p.DatabaseOIDs["run"] = pin.OID
	p.PostmasterStartedAt = pin.StartedAt.Add(time.Microsecond)
	require.ErrorIs(t, p.verifyDatabase(ctx, db, database, "run"), errTitlePilotIsolation)
	p.PostmasterStartedAt = pin.StartedAt
	database.User = "different_runtime_role"
	require.ErrorIs(t, p.verifyDatabase(ctx, db, database, "run"), errTitlePilotIsolation)
}

func TestTitlePilotOperatorSharesProcessLifetime(t *testing.T) {
	mainListener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	defer mainListener.Close()
	operatorListener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	defer operatorListener.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered, finished := make(chan struct{}), make(chan struct{})
	operator := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
		close(finished)
		w.WriteHeader(http.StatusRequestTimeout)
	})}
	main := &http.Server{Handler: http.NewServeMux()}
	done := make(chan error, 1)
	go func() {
		done <- serveWithTitlePilot(ctx, main, operator, mainListener, operatorListener, 4*time.Second)
	}()
	client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{}}
	t.Cleanup(client.CloseIdleConnections)
	responses := make(chan int, 1)
	go func() {
		resp, err := client.Get("http://" + operatorListener.Addr().String() + "/title-pilot/start")
		if err != nil {
			responses <- 0
			return
		}
		defer resp.Body.Close()
		responses <- resp.StatusCode
	}()
	select {
	case <-entered:
	case <-time.After(4 * time.Second):
		t.Fatal("operator request never entered")
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(4 * time.Second):
		t.Fatal("operator did not cancel its active request")
	}
	require.NoError(t, <-done)
	require.Equal(t, http.StatusRequestTimeout, <-responses)
	for _, listener := range []net.Listener{mainListener, operatorListener} {
		conn, err := net.DialTimeout("tcp4", listener.Addr().String(), time.Second)
		if conn != nil {
			_ = conn.Close()
		}
		require.Error(t, err)
	}
}
