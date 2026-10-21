package sheinloginworker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"task-processor/internal/app/configadapter"
	"task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	memberstore "task-processor/internal/integration/persistence/organization/membership"
	"task-processor/internal/platform/database"
	"task-processor/internal/sheinlogin"
	"task-processor/internal/sheinlogin/bootstrap"
	"task-processor/internal/storecenter"
)

type runtime struct {
	config  sheinlogin.Config
	service *sheinlogin.Service
	auth    *sheinlogin.Authorization
	closers []func() error
	ready   func(context.Context) bool
}

func (r *runtime) Close() {
	for i := len(r.closers) - 1; i >= 0; i-- {
		_ = r.closers[i]()
	}
}
func build(ctx context.Context, path string) (*runtime, error) {
	cfg, e := sheinlogin.LoadConfig(ctx, path)
	if e != nil {
		return nil, e
	}
	r := &runtime{config: cfg}
	success := false
	defer func() {
		if !success {
			r.Close()
		}
	}()
	db, e := database.OpenExistingWritableContext(ctx, configadapter.Database(&cfg.StoreDatabase))
	if e != nil {
		return nil, sheinlogin.ErrUnavailable
	}
	r.closers = append(r.closers, func() error { return database.Close(db) })
	sqlDB, e := db.DB()
	if e != nil {
		return nil, sheinlogin.ErrUnavailable
	}
	pings := []func(context.Context) error{sqlDB.PingContext}
	if storecenter.VerifySheinLoginRuntime(ctx, db) != nil {
		return nil, sheinlogin.ErrUnavailable
	}
	permissions, e := authz.NewListingKitAuthorizer(nil, nil)
	if e != nil {
		return nil, sheinlogin.ErrUnavailable
	}
	if cfg.Auth.RolePolicyDatabase != nil {
		if cfg.Auth.RolePolicyDatabase.User != memberstore.RolePolicyReaderRole {
			return nil, sheinlogin.ErrInvalidInput
		}
		policyDB, e := database.OpenExistingReadOnlyContext(ctx, configadapter.Database(cfg.Auth.RolePolicyDatabase))
		if e != nil {
			return nil, sheinlogin.ErrUnavailable
		}
		r.closers = append(r.closers, func() error { return database.Close(policyDB) })
		policySQL, e := policyDB.DB()
		if e != nil {
			return nil, sheinlogin.ErrUnavailable
		}
		pings = append(pings, policySQL.PingContext)
		reader, e := memberstore.NewRolePolicyReader(ctx, policyDB, cfg.Auth.ProjectID)
		if e != nil {
			return nil, sheinlogin.ErrUnavailable
		}
		permissions.SetRolePolicyReader(reader)
	}
	secret, e := sheinlogin.ReadPrivate(ctx, cfg.Auth.ClientSecretFile, 65536)
	if e != nil {
		return nil, e
	}
	httpClient := &http.Client{Timeout: 5 * time.Second}
	r.auth = &sheinlogin.Authorization{Verifier: zitadel.NewVerifier(zitadel.Config{IssuerURL: cfg.Auth.IssuerURL, ClientID: cfg.Auth.ClientID, ClientSecret: strings.TrimSpace(string(secret)), ProjectID: cfg.Auth.ProjectID, HTTPClient: httpClient}), Grants: zitadel.NewAuthorizationClient(cfg.Auth.AuthorizationAPIURL, httpClient), ServiceToken: func(ctx context.Context) (string, error) {
		data, e := sheinlogin.ReadPrivate(ctx, cfg.Auth.ServiceTokenFile, 65536)
		return strings.TrimSpace(string(data)), e
	}, ProjectID: cfg.Auth.ProjectID, IssuerURL: cfg.Auth.IssuerURL, ClientID: cfg.Auth.ClientID, Permissions: permissions}
	owner, e := storecenter.NewSheinLoginRepository(db, r.auth)
	if e != nil {
		return nil, e
	}
	result, e := bootstrap.BuildService(bootstrap.BuildInput{Config: cfg, Owner: owner})
	if e != nil {
		return nil, e
	}
	r.service = result.Service
	r.closers = append(r.closers, result.Close)
	r.ready = func(ctx context.Context) bool {
		return workerDependenciesReady(ctx, r.service.RedisReady, func(ctx context.Context) error {
			for _, ping := range pings {
				if err := ping(ctx); err != nil {
					return err
				}
			}
			return nil
		}, func(ctx context.Context) bool { return r.auth.Ready(ctx, cfg.Bindings[0].OrganizationID) })
	}
	success = true
	return r, nil
}
func Run(logger *logrus.Logger, o Options) error {
	if o.Action == "" {
		o.Action = "serve"
	}
	switch o.Action {
	case "serve", "login", "status", "verify", "cancel", "activate", "quiesce":
	default:
		return sheinlogin.ErrInvalidInput
	}
	if o.Input == nil {
		o.Input = os.Stdin
	}
	if o.Output == nil {
		o.Output = os.Stdout
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signalCh := o.ShutdownSignal
	if signalCh == nil {
		signalCh = make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(signalCh)
	}
	go func() {
		select {
		case <-signalCh:
			cancel()
		case <-ctx.Done():
		}
	}()
	startup, done := context.WithTimeout(ctx, 15*time.Second)
	r, e := build(startup, o.ConfigPath)
	done()
	if e != nil {
		return e
	}
	defer r.Close()
	if o.Action != "serve" {
		commandCtx, done := context.WithTimeout(ctx, 30*time.Second)
		defer done()
		return execute(commandCtx, r.service, r.auth, o)
	}
	if o.OrganizationID != "" || o.StoreID != "" || o.TokenFile != "" || o.AttemptID != "" || o.RequestKey != "" || o.ForceLogin {
		return sheinlogin.ErrInvalidInput
	}
	if e = sheinlogin.PrepareOwnedRoots(r.config); e != nil {
		return e
	}
	if o.Port > 0 {
		server := &http.Server{Addr: fmt.Sprintf(":%d", o.Port), Handler: healthHandler(r.ready), ReadHeaderTimeout: 5 * time.Second}
		go func() {
			if e := server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
				logger.Error("SHEIN worker health server unavailable")
				cancel()
			}
		}()
		defer func() {
			shutdown, done := context.WithTimeout(context.Background(), o.shutdownTimeout())
			defer done()
			_ = server.Shutdown(shutdown)
		}()
	}
	return r.service.RunWorker(ctx, "")
}

func workerDependenciesReady(ctx context.Context, redis func(context.Context) bool, store func(context.Context) error, authorization func(context.Context) bool) bool {
	return ctx.Err() == nil && redis != nil && store != nil && authorization != nil && redis(ctx) && store(ctx) == nil && authorization(ctx) && ctx.Err() == nil
}

func healthHandler(ready func(context.Context) bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"alive": true})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		check, done := context.WithTimeout(req.Context(), time.Second)
		defer done()
		available := ready != nil && ready(check)
		w.Header().Set("Content-Type", "application/json")
		if !available {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"ready": available})
	})
	return mux
}

type commandService interface {
	Execute(context.Context, sheinlogin.Command) (sheinlogin.Result, error)
}
type commandAuth interface {
	Authenticate(context.Context, string, string) (context.Context, string, string, error)
}

func execute(ctx context.Context, service commandService, auth commandAuth, o Options) error {
	token, e := sheinlogin.ReadPrivate(ctx, o.TokenFile, 65536)
	if e != nil {
		return e
	}
	authorized, actor, member, e := auth.Authenticate(ctx, strings.TrimSpace(string(token)), o.OrganizationID)
	if e != nil {
		return e
	}
	code := ""
	if o.Action == "verify" {
		data, e := io.ReadAll(io.LimitReader(o.Input, 65537))
		if e != nil || len(data) > 65536 {
			return sheinlogin.ErrInvalidInput
		}
		code = strings.TrimSpace(string(data))
	}
	result, e := service.Execute(authorized, sheinlogin.Command{Action: o.Action, OrganizationID: o.OrganizationID, StoreID: o.StoreID, ActorID: actor, MemberID: member, RequestKey: o.RequestKey, AttemptID: o.AttemptID, ForceLogin: o.ForceLogin, Code: code, ExpectedGeneration: o.ExpectedGeneration})
	if e != nil {
		return e
	}
	return json.NewEncoder(o.Output).Encode(result)
}
