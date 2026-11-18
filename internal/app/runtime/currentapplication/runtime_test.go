package currentapplication

import (
	"context"
	"errors"
	"net"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	coreconfig "task-processor/internal/core/config"
)

func TestRunReferralPoolFailureAndDisabledLifecycle(t *testing.T) {
	for _, stage := range []string{"disabled", "open", "construct", "listen", "cancel"} {
		t.Run(stage, func(t *testing.T) {
			cfg := referralRuntimeConfig(t)
			cfg.Referrals.Enabled = stage != "disabled"
			source, referrals := &gorm.DB{}, &gorm.DB{}
			var closed []*gorm.DB
			opened := 0
			maturityStopped := false
			stop := errors.New("fixture stop")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			dependencies := Dependencies{
				IdentityPreflight: func(context.Context, IdentityConfig) error { return nil },
				OpenSourceAccount: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },

				OpenReferrals: func(context.Context, DatabaseConfig) (*gorm.DB, error) {
					opened++
					if stage == "open" {
						return nil, stop
					}
					if stage == "cancel" {
						cancel()
					}
					return referrals, nil
				},
				NewApplication: func(context.Context, *gorm.DB, *coreconfig.Config, *logrus.Logger) (*http.Server, error) {
					return &http.Server{}, nil
				},
				NewReferralsApplication: func(_ context.Context, s, r *gorm.DB, maturityStop *func(), core *coreconfig.Config, _ *logrus.Logger) (*http.Server, error) {
					if s != source || r != referrals || core.Referrals.Prepared == nil || maturityStop == nil {
						t.Fatal("referral composition lost independent dependencies")
					}
					if stage == "construct" {
						return nil, stop
					}
					*maturityStop = func() { maturityStopped = true }
					return &http.Server{}, nil
				},
				Listen: func(string, string) (net.Listener, error) { return nil, stop },
				CloseDatabase: func(db *gorm.DB) error {
					if db == referrals && stage == "listen" && !maturityStopped {
						t.Error("referral pool closed before maturity stopped")
					}
					closed = append(closed, db)
					return nil
				},
			}
			if err := Run(ctx, cfg, logrus.New(), dependencies); err == nil {
				t.Fatal("expected bounded stop")
			}
			want := []*gorm.DB{source}
			if stage != "disabled" && stage != "open" {
				want = append([]*gorm.DB{referrals}, want...)
			}
			if !reflect.DeepEqual(closed, want) {
				t.Fatal("incorrect pool release order")
			}
			if stage == "disabled" && opened != 0 {
				t.Fatal("disabled referrals opened a pool")
			}
		})
	}
}

func TestRunClosesSourcePoolWhenResourceOwnerOpenFails(t *testing.T) {
	source := &gorm.DB{}
	want := errors.New("resource owner unavailable")
	cfg := runtimeTestConfig()
	owner := cfg.SourceAccountDatabase
	owner.User = "commercial_owner_runtime"
	cfg.CommercialOwnerDatabase = &owner
	closed := []*gorm.DB{}
	dependencies := runtimeDependencies{
		IdentityPreflight: func(context.Context, IdentityConfig) error { return nil },
		OpenSourceAccount: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },

		OpenCommercialOwner: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return nil, want },
		NewApplicationWithFeatures: func(context.Context, *gorm.DB, ApplicationFeatures, *coreconfig.Config, *logrus.Logger) (*http.Server, error) {
			return nil, nil
		},
		CloseDatabase: func(db *gorm.DB) error {
			closed = append(closed, db)
			return nil
		},
	}

	err := run(context.Background(), cfg, logrus.New(), dependencies)
	if !errors.Is(err, want) {
		t.Fatalf("run() error = %v, want wrapped %v", err, want)
	}
	if len(closed) != 1 || closed[0] != source {
		t.Fatalf("closed databases = %#v", closed)
	}
}

func TestRunRejectsUnavailableIdentityBeforeOpeningDatabase(t *testing.T) {
	want := errors.New("provider unavailable")
	opened := false
	err := run(context.Background(), runtimeTestConfig(), logrus.New(), runtimeDependencies{
		IdentityPreflight: func(context.Context, IdentityConfig) error { return want },
		OpenSourceAccount: func(context.Context, DatabaseConfig) (*gorm.DB, error) { opened = true; return &gorm.DB{}, nil },

		CloseDatabase: func(*gorm.DB) error { return nil },
	})
	if !errors.Is(err, want) || opened {
		t.Fatalf("run() = %v, database opened=%t", err, opened)
	}
}

func TestRunClosesCurrentPoolWhenListenFails(t *testing.T) {
	source := &gorm.DB{}
	want := errors.New("address already in use")
	closed := []*gorm.DB{}
	dependencies := runtimeDependencies{
		IdentityPreflight: func(context.Context, IdentityConfig) error { return nil },
		OpenSourceAccount: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },

		NewApplication: func(_ context.Context, gotSource *gorm.DB, _ *coreconfig.Config, _ *logrus.Logger) (*http.Server, error) {
			if gotSource != source {
				t.Fatal("application received wrong database pools")
			}
			return &http.Server{}, nil
		},
		Listen: func(network, address string) (net.Listener, error) {
			if network != "tcp" || address != "127.0.0.1:18443" {
				t.Fatalf("listen = %s %s", network, address)
			}
			return nil, want
		},
		CloseDatabase: func(db *gorm.DB) error {
			closed = append(closed, db)
			return nil
		},
	}

	err := run(context.Background(), runtimeTestConfig(), logrus.New(), dependencies)
	if !errors.Is(err, want) {
		t.Fatalf("run() error = %v, want wrapped %v", err, want)
	}
	if len(closed) != 1 || closed[0] != source {
		t.Fatalf("close order = %#v", closed)
	}
}

func TestRunStopsCommercialBillingRecoveryBeforeClosingOwnerPools(t *testing.T) {
	cfg := runtimeTestConfig()
	commercialConfig, moneyConfig := cfg.SourceAccountDatabase, cfg.SourceAccountDatabase
	commercialConfig.User = "commercial_owner_runtime"
	moneyConfig.User = "money_owner_runtime"
	cfg.CommercialOwnerDatabase, cfg.MoneyOwnerDatabase = &commercialConfig, &moneyConfig

	source, money := &gorm.DB{}, &gorm.DB{}
	commercial := emptyStorageCommercialOwner(t)
	listenErr := errors.New("address already in use")
	var closed []*gorm.DB
	stopped := false
	err := Run(t.Context(), cfg, logrus.New(), Dependencies{
		IdentityPreflight:   func(context.Context, IdentityConfig) error { return nil },
		OpenSourceAccount:   func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },
		OpenCommercialOwner: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return commercial, nil },
		OpenMoneyOwner:      func(context.Context, DatabaseConfig) (*gorm.DB, error) { return money, nil },
		NewApplicationWithFeatures: func(_ context.Context, _ *gorm.DB, features ApplicationFeatures, _ *coreconfig.Config, _ *logrus.Logger) (*http.Server, error) {
			if features.CommercialBillingRecoveryStop == nil {
				t.Fatal("commercial billing recovery did not receive its stop callback")
			}
			*features.CommercialBillingRecoveryStop = func() { stopped = true }
			return &http.Server{}, nil
		},
		Listen: func(string, string) (net.Listener, error) { return nil, listenErr },
		CloseDatabase: func(db *gorm.DB) error {
			if (db == commercial || db == money) && !stopped {
				t.Error("commercial billing recovery was still running when its database pool closed")
			}
			closed = append(closed, db)
			return nil
		},
	})
	if !errors.Is(err, listenErr) {
		t.Fatalf("Run() error = %v, want wrapped %v", err, listenErr)
	}
	if !stopped || !reflect.DeepEqual(closed, []*gorm.DB{money, commercial, source}) {
		t.Fatalf("commercial billing recovery stopped = %t, closed pools = %v", stopped, closed)
	}
}

func TestRunStopsResourceConsumerRecoveryWithoutMoneyPool(t *testing.T) {
	cfg := runtimeTestConfig()
	commercialConfig, productConfig := cfg.SourceAccountDatabase, cfg.SourceAccountDatabase
	commercialConfig.User = "commercial_owner_runtime"
	productConfig.User, productConfig.Database = "source_acquisition_runtime", "product_acquisition"
	cfg.CommercialOwnerDatabase, cfg.ProductAcquisitionDatabase = &commercialConfig, &productConfig

	source, product := &gorm.DB{}, &gorm.DB{}
	commercial := emptyStorageCommercialOwner(t)
	listenErr := errors.New("address already in use")
	var closed []*gorm.DB
	stopped := false
	err := Run(t.Context(), cfg, logrus.New(), Dependencies{
		IdentityPreflight:      func(context.Context, IdentityConfig) error { return nil },
		OpenSourceAccount:      func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },
		OpenCommercialOwner:    func(context.Context, DatabaseConfig) (*gorm.DB, error) { return commercial, nil },
		OpenProductAcquisition: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return product, nil },
		NewApplicationWithFeatures: func(_ context.Context, _ *gorm.DB, features ApplicationFeatures, _ *coreconfig.Config, _ *logrus.Logger) (*http.Server, error) {
			if features.MoneyOwnerDB != nil || features.ResourceConsumerRecoveryStop == nil {
				t.Fatal("resource consumer recovery requires a stop callback with or without a money owner pool")
			}
			*features.ResourceConsumerRecoveryStop = func() { stopped = true }
			return &http.Server{}, nil
		},
		Listen: func(string, string) (net.Listener, error) { return nil, listenErr },
		CloseDatabase: func(db *gorm.DB) error {
			if (db == commercial || db == product) && !stopped {
				t.Error("resource consumer recovery was still running when its database pool closed")
			}
			closed = append(closed, db)
			return nil
		},
	})
	if !errors.Is(err, listenErr) {
		t.Fatalf("Run() error = %v, want wrapped %v", err, listenErr)
	}
	if !stopped || !reflect.DeepEqual(closed, []*gorm.DB{product, commercial, source}) {
		t.Fatalf("resource consumer recovery stopped = %t, closed pools = %v", stopped, closed)
	}
}

func TestRunShutsDownApplicationAndPreservesPoolsUntilServeStops(t *testing.T) {
	source := &gorm.DB{}
	ctx, cancel := context.WithCancel(context.Background())
	var mu sync.Mutex
	closed := []*gorm.DB{}
	listening := make(chan struct{})
	dependencies := runtimeDependencies{
		IdentityPreflight: func(context.Context, IdentityConfig) error { return nil },
		OpenSourceAccount: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },

		NewApplication: func(context.Context, *gorm.DB, *coreconfig.Config, *logrus.Logger) (*http.Server, error) {
			return &http.Server{Handler: http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(http.StatusNoContent) })}, nil
		},
		Listen: func(string, string) (net.Listener, error) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err == nil {
				close(listening)
			}
			return listener, err
		},
		CloseDatabase: func(db *gorm.DB) error {
			mu.Lock()
			defer mu.Unlock()
			closed = append(closed, db)
			return nil
		},
	}

	done := make(chan error, 1)
	go func() { done <- run(ctx, runtimeTestConfig(), logrus.New(), dependencies) }()
	<-listening
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run() error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(closed) != 1 || closed[0] != source {
		t.Fatalf("close order = %#v", closed)
	}
}

func TestRunDrainsActiveRequestWhenListenerExits(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	mock.MatchExpectationsInOrder(false)
	mock.ExpectPing()
	mock.ExpectClose()
	source, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	requestStarted := make(chan struct{})
	requestFinished := make(chan struct{})
	releaseRequest := make(chan struct{})
	resumeRequest := sync.OnceFunc(func() { close(releaseRequest) })
	defer resumeRequest()
	shutdownStarted := make(chan struct{})
	poolClosed := make(chan struct{})
	var closedBeforeRequestFinished bool
	pingResult := make(chan error, 1)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		defer close(requestFinished)
		<-releaseRequest
		err := sqlDB.PingContext(r.Context())
		pingResult <- err
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})}
	server.RegisterOnShutdown(func() { close(shutdownStarted) })

	runtimeDone := make(chan error, 1)
	go func() {
		runtimeDone <- Run(ctx, runtimeTestConfig(), logrus.New(), Dependencies{
			IdentityPreflight: func(context.Context, IdentityConfig) error { return nil },
			OpenSourceAccount: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },
			NewApplication: func(context.Context, *gorm.DB, *coreconfig.Config, *logrus.Logger) (*http.Server, error) {
				return server, nil
			},
			Listen: func(string, string) (net.Listener, error) { return listener, nil },
			CloseDatabase: func(db *gorm.DB) error {
				pool, err := db.DB()
				if err != nil {
					return err
				}
				select {
				case <-requestFinished:
				default:
					closedBeforeRequestFinished = true
				}
				err = pool.Close()
				close(poolClosed)
				return err
			},
			ShutdownTimeout: time.Second,
		})
	}()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+listener.Addr().String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	type requestResult struct {
		status int
		err    error
	}
	requestDone := make(chan requestResult, 1)
	go func() {
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			requestDone <- requestResult{err: err}
			return
		}
		defer response.Body.Close()
		requestDone <- requestResult{status: response.StatusCode}
	}()

	select {
	case <-requestStarted:
	case err := <-runtimeDone:
		t.Fatalf("runtime exited before serving a request: %v", err)
	case <-ctx.Done():
		t.Fatal("request did not enter the handler")
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-shutdownStarted:
	case <-poolClosed:
		t.Error("pool closed before HTTP shutdown started with a request in flight")
	case <-ctx.Done():
		t.Fatal("runtime did not begin shutting down after the listener exited")
	}
	resumeRequest()

	if response := <-requestDone; response.err != nil || response.status != http.StatusNoContent {
		t.Errorf("in-flight HTTP request returned status %d, error %v", response.status, response.err)
	}
	select {
	case err := <-pingResult:
		if err != nil {
			t.Errorf("in-flight request could not use the database: %v", err)
		}
	default:
		t.Error("in-flight request did not reach the database")
	}
	var acceptErr *net.OpError
	if err := <-runtimeDone; !errors.Is(err, net.ErrClosed) || !errors.As(err, &acceptErr) || acceptErr.Op != "accept" {
		t.Errorf("Run() error = %v, want original listener error", err)
	}
	if closedBeforeRequestFinished {
		t.Error("pool began closing before the handler returned")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestRunForcesCloseAfterBoundedShutdownExpires(t *testing.T) {
	source := &gorm.DB{}
	ctx, cancel := context.WithCancel(context.Background())
	listening := make(chan string, 1)
	requestStarted := make(chan struct{})
	closed := 0
	dependencies := runtimeDependencies{
		IdentityPreflight: func(context.Context, IdentityConfig) error { return nil },
		OpenSourceAccount: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },

		NewApplication: func(context.Context, *gorm.DB, *coreconfig.Config, *logrus.Logger) (*http.Server, error) {
			return &http.Server{Handler: http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
				close(requestStarted)
				<-request.Context().Done()
			})}, nil
		},
		Listen: func(string, string) (net.Listener, error) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err == nil {
				listening <- listener.Addr().String()
			}
			return listener, err
		},
		CloseDatabase:   func(*gorm.DB) error { closed++; return nil },
		ShutdownTimeout: 20 * time.Millisecond,
	}
	done := make(chan error, 1)
	go func() { done <- run(ctx, runtimeTestConfig(), logrus.New(), dependencies) }()
	go func() { _, _ = http.Get("http://" + <-listening) }()
	<-requestStarted
	cancel()
	err := <-done
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("run() error = %v, want shutdown deadline exceeded", err)
	}
	if closed != 1 {
		t.Fatalf("closed databases = %d, want 2", closed)
	}
}

func TestRunPassesOneBoundedStartupContextToBothOpeners(t *testing.T) {
	want := errors.New("context-aware source opener")
	err := run(context.Background(), runtimeTestConfig(), logrus.New(), runtimeDependencies{
		IdentityPreflight: func(ctx context.Context, _ IdentityConfig) error {
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("identity preflight did not receive a deadline")
			}
			return nil
		},
		OpenSourceAccount: func(ctx context.Context, _ DatabaseConfig) (*gorm.DB, error) {
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > 15*time.Second {
				t.Fatalf("source opener context deadline = %v, ok=%t", deadline, ok)
			}
			return nil, want
		},

		CloseDatabase: func(*gorm.DB) error { return nil },
	})
	if !errors.Is(err, want) {
		t.Fatalf("run() error = %v, want wrapped %v", err, want)
	}
}

func runtimeTestConfig() *Config {
	return &Config{
		SchemaVersion: manifestSchemaVersion,
		Listen:        ListenConfig{Host: "127.0.0.1", Port: 18443},
		Identity: IdentityConfig{
			IssuerURL: "http://localhost:18080", AuthorizationAPIURL: "http://localhost:18080",
			ClientID: "client", ClientSecret: "secret", ProjectID: "project",
		},
		SourceAccountDatabase: DatabaseConfig{Host: "127.0.0.1", Port: 15432, User: "source_account_runtime", Password: "secret", Database: "task_processor", MaxConnections: 2},
	}
}
