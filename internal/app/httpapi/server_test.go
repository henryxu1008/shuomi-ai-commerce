package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"task-processor/internal/authidentity"
	zitadelruntime "task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	"task-processor/internal/workbenchcontext"
	workbenchcontexthttpapi "task-processor/internal/workbenchcontext/httpapi"
)

type mountedOrganizationResolverStub struct {
	events *[]string
	roles  []string
	bearer *string
}

type mountedVerifierStub struct {
	identity authidentity.AuthenticatedIdentity
	err      error
	events   *[]string
	token    *string
}

func (stub mountedVerifierStub) Verify(_ context.Context, token string) (authidentity.AuthenticatedIdentity, error) {
	if stub.events != nil {
		*stub.events = append(*stub.events, "authentication")
	}
	if stub.token != nil {
		*stub.token = token
	}
	return stub.identity, stub.err
}

type appHTTPRoundTripFunc func(*http.Request) (*http.Response, error)

type workbenchBodyBoundaryReader struct {
	payload          []byte
	offset           int
	readsPastPayload int
}

func (reader *workbenchBodyBoundaryReader) Read(buffer []byte) (int, error) {
	if reader.offset >= len(reader.payload) {
		reader.readsPastPayload++
		return 0, io.EOF
	}
	read := copy(buffer, reader.payload[reader.offset:])
	reader.offset += read
	return read, nil
}

func (roundTrip appHTTPRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func mountedVerifiedIdentity() authidentity.AuthenticatedIdentity {
	return authidentity.AuthenticatedIdentity{UserID: "user-1", TokenExpiresAt: time.Now().Add(time.Minute)}
}

type mountedOrganizationResolverError struct{ err error }

type mountedWorkbenchAuditStub struct {
	records []workbenchcontext.AuditEvent
	err     error
}

func (stub *mountedWorkbenchAuditStub) Record(_ context.Context, event workbenchcontext.AuditEvent) error {
	stub.records = append(stub.records, event)
	return stub.err
}

type mountedCapturingOrganizationResolver struct {
	calls  int
	target string
}

func (resolver *mountedCapturingOrganizationResolver) Resolve(_ context.Context, _ httproute.OrganizationAccessPolicy, input workbenchcontext.ResolveInput) (authidentity.AuthenticatedIdentity, error) {
	resolver.calls++
	resolver.target = input.RequestedOrganizationID
	identity := input.Identity
	identity.EffectiveOrganizationID = input.RequestedOrganizationID
	return identity, nil
}

func (stub mountedOrganizationResolverError) Resolve(context.Context, httproute.OrganizationAccessPolicy, workbenchcontext.ResolveInput) (authidentity.AuthenticatedIdentity, error) {
	return authidentity.AuthenticatedIdentity{}, stub.err
}

func TestWorkbenchLiveSwitchRejectsInvalidTargetBeforeGrantLookup(t *testing.T) {
	testCases := []struct {
		name   string
		body   string
		header string
	}{
		{name: "unknown field", body: `{"organizationId":"org-a","unexpected":true}`},
		{name: "blank", body: `{"organizationId":" "}`},
		{name: "trailing", body: `{"organizationId":"org-a"}{}`},
		{name: "header mismatch", body: `{"organizationId":"org-a"}`, header: "org-b"},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &mountedCapturingOrganizationResolver{}
			handlerCalls := 0
			router := gin.New()
			mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
				Method:                     http.MethodPut,
				Path:                       "/api/v1/workbench/context/effective-organization",
				AuthPolicy:                 httproute.AuthPolicyVerifiedIdentity,
				OrganizationAccessPolicy:   httproute.OrganizationAccessPolicyLiveSwitch,
				OrganizationTargetResolver: workbenchcontexthttpapi.ResolveSwitchOrganizationTarget,
				Handler: func(c *gin.Context) {
					handlerCalls++
					c.Status(http.StatusNoContent)
				},
			}}, routeAuthDependencies{
				workbenchVerifier:    mountedVerifierStub{identity: mountedVerifiedIdentity()},
				organizationResolver: resolver,
			})
			request := httptest.NewRequest(http.MethodPut, "/api/v1/workbench/context/effective-organization", strings.NewReader(tt.body))
			request.Header.Set("Authorization", "Bearer current-request-token")
			request.Header.Set("X-Requested-Organization-ID", tt.header)
			request.Header.Set("X-Request-ID", "req-123")
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
			require.JSONEq(t, `{"code":"INVALID_REQUEST","message":"Request is invalid","requestId":"req-123","fieldErrors":[]}`, response.Body.String())
			require.Zero(t, resolver.calls)
			require.Zero(t, handlerCalls)
		})
	}
}

func TestWorkbenchLiveSwitchRejects4097ByteBodyWithoutReadingPastLimitBeforeGrantLookup(t *testing.T) {
	reader := &workbenchBodyBoundaryReader{payload: []byte(strings.Repeat("x", 4097))}
	resolver := &mountedCapturingOrganizationResolver{}
	handlerCalls := 0
	router := gin.New()
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
		Method:                     http.MethodPut,
		Path:                       "/api/v1/workbench/context/effective-organization",
		AuthPolicy:                 httproute.AuthPolicyVerifiedIdentity,
		OrganizationAccessPolicy:   httproute.OrganizationAccessPolicyLiveSwitch,
		OrganizationTargetResolver: workbenchcontexthttpapi.ResolveSwitchOrganizationTarget,
		Handler: func(c *gin.Context) {
			handlerCalls++
			c.Status(http.StatusNoContent)
		},
	}}, routeAuthDependencies{
		workbenchVerifier:    mountedVerifierStub{identity: mountedVerifiedIdentity()},
		organizationResolver: resolver,
		auditRecorder:        &mountedWorkbenchAuditStub{},
	})
	request := httptest.NewRequest(http.MethodPut, "/api/v1/workbench/context/effective-organization", reader)
	request.Header.Set("Authorization", "Bearer current-request-token")
	request.Header.Set("X-Request-ID", "req-oversize")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	require.JSONEq(t, `{"code":"INVALID_REQUEST","message":"Request is invalid","requestId":"req-oversize","fieldErrors":[]}`, response.Body.String())
	require.Zero(t, reader.readsPastPayload)
	require.Zero(t, resolver.calls)
	require.Zero(t, handlerCalls)
}

func TestWorkbenchLiveSwitchUsesBodyTargetAndRestoresBodyForHandler(t *testing.T) {
	const body = `{"organizationId":"org-a"}`
	resolver := &mountedCapturingOrganizationResolver{}
	handlerBody := ""
	router := gin.New()
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
		Method:                     http.MethodPut,
		Path:                       "/api/v1/workbench/context/effective-organization",
		AuthPolicy:                 httproute.AuthPolicyVerifiedIdentity,
		OrganizationAccessPolicy:   httproute.OrganizationAccessPolicyLiveSwitch,
		OrganizationTargetResolver: workbenchcontexthttpapi.ResolveSwitchOrganizationTarget,
		Handler: func(c *gin.Context) {
			data, err := io.ReadAll(c.Request.Body)
			require.NoError(t, err)
			handlerBody = string(data)
			c.Status(http.StatusNoContent)
		},
	}}, routeAuthDependencies{
		workbenchVerifier:    mountedVerifierStub{identity: mountedVerifiedIdentity()},
		organizationResolver: resolver,
		auditRecorder:        &mountedWorkbenchAuditStub{},
	})
	request := httptest.NewRequest(http.MethodPut, "/api/v1/workbench/context/effective-organization", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer current-request-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	require.Equal(t, 1, resolver.calls)
	require.Equal(t, "org-a", resolver.target)
	require.Equal(t, body, handlerBody)
}

func TestWorkbenchLiveSwitchRequiresAuditBeforeReportingSuccess(t *testing.T) {
	now := time.Date(2026, time.August, 31, 8, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name         string
		auditError   error
		wantStatus   int
		wantHandlers int
	}{
		{name: "recorded", wantStatus: http.StatusNoContent, wantHandlers: 1},
		{name: "audit unavailable", auditError: errors.New("audit sink unavailable"), wantStatus: http.StatusServiceUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			audit := &mountedWorkbenchAuditStub{err: tt.auditError}
			handlerCalls := 0
			router := gin.New()
			route := httproute.Descriptor{
				Method: http.MethodPut, Path: "/api/v1/workbench/context/effective-organization",
				AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveSwitch,
				OrganizationTargetResolver: workbenchcontexthttpapi.ResolveSwitchOrganizationTarget,
				Handler:                    func(c *gin.Context) { handlerCalls++; c.Status(http.StatusNoContent) },
			}
			mountRoutesWithAuthDependencies(router, []httproute.Descriptor{route}, routeAuthDependencies{
				workbenchVerifier: mountedVerifierStub{identity: authidentity.AuthenticatedIdentity{
					UserID: "user-1", HomeOrganizationID: "org-home", TokenExpiresAt: now.Add(time.Minute),
				}},
				organizationResolver: &mountedCapturingOrganizationResolver{},
				auditRecorder:        audit,
				auditNow:             func() time.Time { return now },
			})
			request := httptest.NewRequest(http.MethodPut, route.Path, strings.NewReader(`{"organizationId":"org-target"}`))
			request.Header.Set("Authorization", "Bearer current-request-token")
			request.Header.Set("X-Request-ID", "req-switch")
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			require.Equal(t, tt.wantStatus, response.Code, response.Body.String())
			require.Equal(t, tt.wantHandlers, handlerCalls)
			require.Equal(t, []workbenchcontext.AuditEvent{{
				Subject: "user-1", HomeOrganizationID: "org-home", EffectiveOrganizationID: "org-target",
				Resource: route.Path, Action: workbenchcontext.AuditActionOrganizationSwitch,
				Result: workbenchcontext.AuditResultSuccess, Timestamp: now, RequestID: "req-switch",
			}}, audit.records)
		})
	}
}

func TestWorkbenchLiveSwitchFailureIsAuditedWithoutChangingTheStableDenial(t *testing.T) {
	now := time.Date(2026, time.August, 31, 8, 15, 0, 0, time.UTC)
	audit := &mountedWorkbenchAuditStub{err: errors.New("audit unavailable")}
	router := gin.New()
	route := httproute.Descriptor{
		Method: http.MethodPut, Path: "/api/v1/workbench/context/effective-organization",
		AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveSwitch,
		OrganizationTargetResolver: workbenchcontexthttpapi.ResolveSwitchOrganizationTarget,
		Handler:                    func(*gin.Context) { t.Fatal("denied switch handler ran") },
	}
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{route}, routeAuthDependencies{
		workbenchVerifier:    mountedVerifierStub{identity: authidentity.AuthenticatedIdentity{UserID: "user-1", HomeOrganizationID: "org-home", TokenExpiresAt: now.Add(time.Minute)}},
		organizationResolver: mountedOrganizationResolverError{err: workbenchcontext.ErrOrganizationAccessDenied},
		auditRecorder:        audit, auditNow: func() time.Time { return now },
	})
	request := httptest.NewRequest(route.Method, route.Path, strings.NewReader(`{"organizationId":"org-target"}`))
	request.Header.Set("Authorization", "Bearer current-request-token")
	request.Header.Set("X-Request-ID", "req-switch-denied")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusForbidden, response.Code)
	require.Contains(t, response.Body.String(), "ORGANIZATION_ACCESS_DENIED")
	require.Equal(t, []workbenchcontext.AuditEvent{{
		Subject: "user-1", HomeOrganizationID: "org-home", EffectiveOrganizationID: "org-target",
		Resource: route.Path, Action: workbenchcontext.AuditActionOrganizationSwitch,
		Result: workbenchcontext.AuditResultOrganizationAccessDenied, Timestamp: now, RequestID: "req-switch-denied",
	}}, audit.records)
}

func TestWorkbenchPostAuthenticationOrganizationAndRoleDenialsAreAudited(t *testing.T) {
	now := time.Date(2026, time.August, 31, 8, 30, 0, 0, time.UTC)
	t.Run("organization denial", func(t *testing.T) {
		audit := &mountedWorkbenchAuditStub{err: errors.New("audit unavailable")}
		router := gin.New()
		route := httproute.Descriptor{Method: http.MethodGet, Path: "/api/v1/workbench/resource", AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyCachedRead, Handler: func(*gin.Context) { t.Fatal("denied handler ran") }}
		mountRoutesWithAuthDependencies(router, []httproute.Descriptor{route}, routeAuthDependencies{
			workbenchVerifier:    mountedVerifierStub{identity: authidentity.AuthenticatedIdentity{UserID: "user-1", HomeOrganizationID: "org-home", TokenExpiresAt: now.Add(time.Minute)}},
			organizationResolver: mountedOrganizationResolverError{err: workbenchcontext.ErrOrganizationAccessDenied},
			auditRecorder:        audit, auditNow: func() time.Time { return now },
		})
		request := httptest.NewRequest(route.Method, route.Path, nil)
		request.Header.Set("Authorization", "Bearer current-request-token")
		request.Header.Set("X-Requested-Organization-ID", "org-target")
		request.Header.Set("X-Request-ID", "req-denied")
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		require.Equal(t, http.StatusForbidden, response.Code)
		require.Contains(t, response.Body.String(), "ORGANIZATION_ACCESS_DENIED")
		require.Equal(t, 1, len(audit.records))
		require.Equal(t, "org-target", audit.records[0].EffectiveOrganizationID)
		require.Equal(t, workbenchcontext.AuditResultOrganizationAccessDenied, audit.records[0].Result)
	})

	t.Run("role denial", func(t *testing.T) {
		audit := &mountedWorkbenchAuditStub{err: errors.New("audit unavailable")}
		router := gin.New()
		route := httproute.Descriptor{Method: http.MethodPost, Path: "/api/v1/workbench/admin-action", Module: "listing-kit-admin", Permission: authz.PermissionListingKitAdminWrite, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, Handler: func(*gin.Context) { t.Fatal("denied handler ran") }}
		mountRoutesWithAuthDependencies(router, []httproute.Descriptor{route}, routeAuthDependencies{
			workbenchVerifier:    mountedVerifierStub{identity: authidentity.AuthenticatedIdentity{UserID: "user-1", HomeOrganizationID: "org-home", TokenExpiresAt: now.Add(time.Minute)}},
			organizationResolver: mountedOrganizationResolverStub{events: &[]string{}, roles: []string{"listingkit_viewer"}},
			auditRecorder:        audit, auditNow: func() time.Time { return now },
		})
		request := httptest.NewRequest(route.Method, route.Path, nil)
		request.Header.Set("Authorization", "Bearer current-request-token")
		request.Header.Set("X-Request-ID", "req-role")
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		require.Equal(t, http.StatusForbidden, response.Code)
		require.Contains(t, response.Body.String(), "PERMISSION_DENIED")
		require.Equal(t, 1, len(audit.records))
		require.Equal(t, "org-a", audit.records[0].EffectiveOrganizationID)
		require.Equal(t, authz.PermissionListingKitAdminWrite, audit.records[0].Action)
		require.Equal(t, workbenchcontext.AuditResultPermissionDenied, audit.records[0].Result)
	})
}

func (stub mountedOrganizationResolverStub) Resolve(_ context.Context, _ httproute.OrganizationAccessPolicy, input workbenchcontext.ResolveInput) (authidentity.AuthenticatedIdentity, error) {
	*stub.events = append(*stub.events, "organization")
	if stub.bearer != nil {
		*stub.bearer = input.BearerToken
	}
	identity := input.Identity
	identity.TenantID = "org-a"
	identity.EffectiveOrganizationID = "org-a"
	identity.Roles = append([]string(nil), stub.roles...)
	return identity, nil
}

func TestWorkbenchAuthHandlerOrderIsAuthenticationOrganizationRoleThenHandler(t *testing.T) {
	events := []string{}
	resolvedBearer := ""
	router := gin.New()
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
		Method:                   http.MethodGet,
		Path:                     "/api/v1/workbench/order",
		AuthPolicy:               httproute.AuthPolicyVerifiedIdentity,
		OrganizationAccessPolicy: httproute.OrganizationAccessPolicyCachedRead,
		Handler: func(c *gin.Context) {
			events = append(events, "handler")
			c.Status(http.StatusNoContent)
		},
	}}, routeAuthDependencies{
		workbenchVerifier:    mountedVerifierStub{identity: mountedVerifiedIdentity(), events: &events},
		organizationResolver: mountedOrganizationResolverStub{events: &events, roles: []string{"listingkit_admin"}, bearer: &resolvedBearer},
		roleMiddleware: func(httproute.Descriptor) gin.HandlerFunc {
			return func(c *gin.Context) {
				events = append(events, "role")
				identity, ok := authidentity.AuthenticatedIdentityFromContext(c.Request.Context())
				if !ok || identity.EffectiveOrganizationID != "org-a" || len(identity.Roles) != 1 || identity.Roles[0] != "listingkit_admin" {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
				c.Next()
			}
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workbench/order", nil)
	request.Header.Set("Authorization", "Bearer pre-auth-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	require.Equal(t, []string{"authentication", "organization", "role", "handler"}, events)
	require.Equal(t, "pre-auth-token", resolvedBearer)
}

func TestProductionAuthShapeUsesIndependentServerAndWorkbenchAuthentication(t *testing.T) {
	zitadel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"introspection_endpoint": "http://" + r.Host + "/oauth/v2/introspect",
			})
		case "/oauth/v2/introspect":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"active": true, "sub": "user-1", "exp": time.Now().Add(time.Minute).Unix(),
				"urn:zitadel:iam:user:resourceowner:id": "org-home",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer zitadel.Close()
	events := []string{}
	authorization, err := buildRouteAuthorization(&config.Config{ListingKit: config.ListingKitConfig{Zitadel: config.ListingKitZitadelConfig{
		IssuerURL: zitadel.URL, ClientID: "listingkit-client", ProjectID: "project-1",
		AuthorizationRequired: true, AllowedTenantIDs: []string{"org-other"},
	}}})
	require.NoError(t, err)
	authorization = authorization.withWorkbench(routeAuthDependencies{
		workbenchVerifier: zitadelruntime.NewVerifier(zitadelruntime.Config{
			IssuerURL: zitadel.URL, ClientID: "listingkit-client", ProjectID: "project-1", HTTPClient: zitadel.Client(),
		}),
		organizationResolver: mountedOrganizationResolverStub{events: &events, roles: []string{"listingkit_viewer"}},
	})
	router := gin.New()
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{
		{
			Method: http.MethodGet, Path: "/api/v1/workbench/context", AuthPolicy: httproute.AuthPolicyVerifiedIdentity,
			OrganizationAccessPolicy: httproute.OrganizationAccessPolicyCachedRead,
			Handler: func(c *gin.Context) {
				events = append(events, "workbench-handler")
				c.Status(http.StatusNoContent)
			},
		},
		{
			Method: http.MethodGet, Path: "/api/v1/listing-kits/tasks", Module: "listing-kit",
			AuthPolicy: httproute.AuthPolicyVerifiedIdentity,
			Handler:    func(c *gin.Context) { t.Fatal("listing kit handler ran after allowlist denial") },
		},
	}, authorization)

	workbenchRequest := httptest.NewRequest(http.MethodGet, "/api/v1/workbench/context", nil)
	workbenchRequest.Header.Set("Authorization", "Bearer user-token")
	workbenchResponse := httptest.NewRecorder()
	router.ServeHTTP(workbenchResponse, workbenchRequest)
	require.Equal(t, http.StatusNoContent, workbenchResponse.Code, workbenchResponse.Body.String())
	require.Equal(t, []string{"organization", "workbench-handler"}, events)

	legacyRequest := httptest.NewRequest(http.MethodGet, "/api/v1/listing-kits/tasks", nil)
	legacyRequest.Header.Set("Authorization", "Bearer user-token")
	legacyResponse := httptest.NewRecorder()
	router.ServeHTTP(legacyResponse, legacyRequest)
	require.Equal(t, http.StatusForbidden, legacyResponse.Code, legacyResponse.Body.String())
	require.Contains(t, legacyResponse.Body.String(), "zitadel_access_denied")
}

func TestLegacyBlankOrganizationPolicySkipsResolver(t *testing.T) {
	assertOrganizationPolicySkipsResolver(t, "")
}

func TestExplicitNoneOrganizationPolicySkipsResolver(t *testing.T) {
	assertOrganizationPolicySkipsResolver(t, httproute.OrganizationAccessPolicyNone)
}

func TestWorkbenchOrganizationPolicyFailsClosedWithoutAuthenticationMiddleware(t *testing.T) {
	router := gin.New()
	route := httproute.Descriptor{
		Method: http.MethodGet, Path: "/workbench", AuthPolicy: httproute.AuthPolicyVerifiedIdentity,
		OrganizationAccessPolicy: httproute.OrganizationAccessPolicyCachedRead,
		Handler: func(c *gin.Context) {
			t.Fatal("handler ran without authentication middleware")
		},
	}
	router.Handle(route.Method, route.Path, append(routeAuthHandlers(route, routeAuthorization{}), route.Handler)...)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/workbench", nil))

	require.Equal(t, http.StatusUnauthorized, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), "AUTHENTICATION_REQUIRED")
}

func TestWorkbenchOrganizationResolverNilFailsClosed(t *testing.T) {
	router := gin.New()
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
		Method: http.MethodGet, Path: "/workbench", AuthPolicy: httproute.AuthPolicyVerifiedIdentity,
		OrganizationAccessPolicy: httproute.OrganizationAccessPolicyCachedRead,
		Handler:                  func(c *gin.Context) { t.Fatal("handler ran without organization resolver") },
	}}, routeAuthDependencies{workbenchVerifier: mountedVerifierStub{identity: mountedVerifiedIdentity()}})
	request := httptest.NewRequest(http.MethodGet, "/workbench", nil)
	request.Header.Set("Authorization", "Bearer current-request-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusServiceUnavailable, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), "DEPENDENCY_UNAVAILABLE")
}

func TestWorkbenchCurrentIdentityPreservesBearerTokenForUserScopedHandlers(t *testing.T) {
	router := gin.New()
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
		Method:     http.MethodGet,
		Path:       "/account/identity/profile",
		AuthPolicy: httproute.AuthPolicyCurrentIdentity,
		Handler: func(c *gin.Context) {
			require.Equal(t, "current-request-token", zitadelruntime.BearerTokenFromContext(c.Request.Context()))
			c.Status(http.StatusNoContent)
		},
	}}, routeAuthDependencies{
		workbenchVerifier: mountedVerifierStub{identity: mountedVerifiedIdentity()},
	})

	request := httptest.NewRequest(http.MethodGet, "/account/identity/profile", nil)
	request.Header.Set("Authorization", "Bearer current-request-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
}

func assertOrganizationPolicySkipsResolver(t *testing.T, policy httproute.OrganizationAccessPolicy) {
	t.Helper()
	events := []string{}
	router := gin.New()
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
		Method: http.MethodGet, Path: "/policy", AuthPolicy: httproute.AuthPolicyPublic,
		OrganizationAccessPolicy: policy,
		Handler: func(c *gin.Context) {
			events = append(events, "handler")
			c.Status(http.StatusNoContent)
		},
	}}, routeAuthDependencies{
		organizationResolver: mountedOrganizationResolverStub{events: &events},
	})
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/policy", nil))

	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	require.Equal(t, []string{"handler"}, events)
}

type mountedGrantLoader struct {
	grants []authidentity.OrganizationGrant
}

func (loader mountedGrantLoader) Load(context.Context, workbenchcontext.GrantSource, workbenchcontext.GrantRequest) (workbenchcontext.GrantResult, error) {
	return workbenchcontext.GrantResult{Grants: loader.grants}, nil
}

func (mountedGrantLoader) Invalidate(string, string) {}

func TestMountedWorkbenchContextUsesCurrentOpaqueOrganizationAndScopedRoles(t *testing.T) {
	for _, organizationID := range []string{"org-canonical", "246", "92233720368547758080001"} {
		t.Run(organizationID, func(t *testing.T) {
			now := time.Now().UTC()
			resolver := workbenchcontext.NewResolver(mountedGrantLoader{grants: []authidentity.OrganizationGrant{{
				OrganizationID: organizationID, ProjectID: "project-1", Roles: []string{"listingkit_viewer"},
			}}}, "project-1", "v1", nil)
			router := gin.New()
			mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
				Method: http.MethodGet, Path: "/api/v1/workbench/context", AuthPolicy: httproute.AuthPolicyVerifiedIdentity,
				OrganizationAccessPolicy: httproute.OrganizationAccessPolicyContextRead,
				Handler: func(c *gin.Context) {
					identity, ok := authidentity.AuthenticatedIdentityFromContext(c.Request.Context())
					require.True(t, ok)
					require.Equal(t, organizationID, identity.TenantID)
					require.Equal(t, organizationID, identity.EffectiveOrganizationID)
					require.Equal(t, []string{"listingkit_viewer"}, identity.Roles)
					c.JSON(http.StatusOK, gin.H{
						"effectiveOrganizationId": identity.EffectiveOrganizationID,
						"roles":                   identity.Roles,
					})
				},
			}}, routeAuthDependencies{
				workbenchVerifier: mountedVerifierStub{identity: authidentity.AuthenticatedIdentity{
					TenantID: "246", UserID: "user-1", HomeOrganizationID: organizationID,
					Roles: []string{"legacy_admin"}, TokenExpiresAt: now.Add(time.Minute),
				}},
				organizationResolver: resolver,
			})
			request := httptest.NewRequest(http.MethodGet, "/api/v1/workbench/context", nil)
			request.Header.Set("Authorization", "Bearer current-request-token")
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			require.JSONEq(t, fmt.Sprintf(`{"effectiveOrganizationId":%q,"roles":["listingkit_viewer"]}`, organizationID), response.Body.String())
		})
	}
}

func TestWorkbenchScopedRoleDoesNotCarryAdminFromOrganizationAToViewerOrganizationB(t *testing.T) {
	now := time.Now().UTC()
	resolver := workbenchcontext.NewResolver(mountedGrantLoader{grants: []authidentity.OrganizationGrant{
		{OrganizationID: "org-a", ProjectID: "project-1", Roles: []string{"listingkit_admin"}},
		{OrganizationID: "org-b", ProjectID: "project-1", Roles: []string{"listingkit_viewer"}},
	}}, "project-1", "v1", nil)
	handlerCalls := 0
	router := gin.New()
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
		Method:                   http.MethodPost,
		Path:                     "/api/v1/workbench/admin-action",
		Module:                   "listing-kit-admin",
		Permission:               authz.PermissionListingKitAdminWrite,
		AuthPolicy:               httproute.AuthPolicyVerifiedIdentity,
		OrganizationAccessPolicy: httproute.OrganizationAccessPolicyCachedRead,
		Handler: func(c *gin.Context) {
			handlerCalls++
			c.Status(http.StatusNoContent)
		},
	}}, routeAuthDependencies{
		workbenchVerifier: mountedVerifierStub{identity: authidentity.AuthenticatedIdentity{
			TenantID: "org-a", UserID: "user-1", HomeOrganizationID: "org-a",
			Roles: []string{"listingkit_admin"}, TokenExpiresAt: now.Add(time.Minute),
		}},
		organizationResolver: resolver,
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/workbench/admin-action", nil)
	request.Header.Set("Authorization", "Bearer current-request-token")
	request.Header.Set("X-Requested-Organization-ID", "org-b")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
	require.Equal(t, 0, handlerCalls)
	var deniedBody struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &deniedBody))
	require.Equal(t, "PERMISSION_DENIED", deniedBody.Code)

	request = httptest.NewRequest(http.MethodPost, "/api/v1/workbench/admin-action", nil)
	request.Header.Set("Authorization", "Bearer current-request-token")
	request.Header.Set("X-Requested-Organization-ID", "org-a")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	require.Equal(t, 1, handlerCalls)
}

func TestWorkbenchOrganizationContextReadClearsLegacyBusinessScopeWhenSelectionIsRequired(t *testing.T) {
	now := time.Now().UTC()
	resolver := workbenchcontext.NewResolver(mountedGrantLoader{grants: []authidentity.OrganizationGrant{
		{OrganizationID: "org-a", ProjectID: "project-1", Roles: []string{"listingkit_admin"}},
		{OrganizationID: "org-b", ProjectID: "project-1", Roles: []string{"listingkit_viewer"}},
	}}, "project-1", "v1", nil)
	router := gin.New()
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
		Method: http.MethodGet, Path: "/api/v1/workbench/context", AuthPolicy: httproute.AuthPolicyVerifiedIdentity,
		OrganizationAccessPolicy: httproute.OrganizationAccessPolicyContextRead,
		Handler: func(c *gin.Context) {
			identity, ok := authidentity.AuthenticatedIdentityFromContext(c.Request.Context())
			require.True(t, ok)
			require.Equal(t, "user-1", identity.UserID)
			require.Equal(t, "org-home", identity.HomeOrganizationID)
			require.Empty(t, identity.TenantID)
			require.Empty(t, identity.EffectiveOrganizationID)
			require.Empty(t, identity.Roles)
			require.Empty(t, c.GetHeader("X-Tenant-ID"))
			require.Empty(t, c.GetHeader("tenant-id"))
			require.Empty(t, c.GetHeader("X-User-Roles"))
			c.Status(http.StatusNoContent)
		},
	}}, routeAuthDependencies{
		workbenchVerifier: mountedVerifierStub{identity: authidentity.AuthenticatedIdentity{
			TenantID: "org-home", UserID: "user-1", HomeOrganizationID: "org-home",
			Roles: []string{"legacy_admin"}, TokenExpiresAt: now.Add(time.Minute),
		}},
		organizationResolver: resolver,
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workbench/context", nil)
	request.Header.Set("Authorization", "Bearer current-request-token")
	request.Header.Set("X-Tenant-ID", "org-home")
	request.Header.Set("tenant-id", "org-home")
	request.Header.Set("X-User-Roles", "legacy_admin")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
}

func TestCurrentIdentityWithVerifiedRolesPreservesGlobalAdminRole(t *testing.T) {
	router := gin.New()
	handled := false
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
		Method: http.MethodPost, Path: "/api/v1/account/referrals/withdrawals/id/review",
		AuthPolicy: httproute.AuthPolicyCurrentIdentityWithVerifiedRoles,
		Permission: authz.PermissionListingKitAdminWrite,
		Handler: func(c *gin.Context) {
			identity, ok := authidentity.AuthenticatedIdentityFromContext(c.Request.Context())
			require.True(t, ok)
			require.Equal(t, []string{"platform_admin"}, identity.Roles)
			require.Empty(t, identity.OrganizationGrants)
			require.Empty(t, identity.EffectiveOrganizationID)
			handled = true
			c.Status(http.StatusNoContent)
		},
	}}, routeAuthDependencies{
		workbenchVerifier: mountedVerifierStub{identity: authidentity.AuthenticatedIdentity{UserID: "admin-1", Roles: []string{"platform_admin"}, TokenExpiresAt: time.Now().Add(time.Minute)}},
		authorizer:        appHTTPTestRouteAuthorization.authorizer,
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/account/referrals/withdrawals/id/review", nil)
	request.Header.Set("Authorization", "Bearer current-request-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	require.True(t, handled)
}

func TestWorkbenchOrganizationAuthenticationFailureUsesStableProtocol(t *testing.T) {
	router := gin.New()
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
		Method:                   http.MethodGet,
		Path:                     "/api/v1/workbench/context",
		AuthPolicy:               httproute.AuthPolicyVerifiedIdentity,
		OrganizationAccessPolicy: httproute.OrganizationAccessPolicyContextRead,
		Handler: func(c *gin.Context) {
			t.Fatal("handler ran after authentication failure")
		},
	}}, routeAuthDependencies{
		workbenchVerifier:    mountedVerifierStub{err: errors.New("provider said secret-token and org-other")},
		organizationResolver: mountedOrganizationResolverStub{events: &[]string{}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workbench/context", nil)
	request.Header.Set("Authorization", "Bearer secret-token")
	request.Header.Set("X-Request-ID", "req-123")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
	var body struct {
		Code        string `json:"code"`
		Message     string `json:"message"`
		RequestID   string `json:"requestId"`
		FieldErrors []any  `json:"fieldErrors"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Equal(t, "AUTHENTICATION_REQUIRED", body.Code)
	require.Equal(t, "req-123", body.RequestID)
	require.NotNil(t, body.FieldErrors)
	require.NotContains(t, response.Body.String(), "secret-token")
	require.NotContains(t, response.Body.String(), "org-other")
}

func TestWorkbenchVerifierDependencyFailureUsesStableUnavailableProtocol(t *testing.T) {
	verifier := zitadelruntime.NewVerifier(zitadelruntime.Config{
		IssuerURL: "https://issuer.example", ClientID: "listingkit-client",
		HTTPClient: &http.Client{Transport: appHTTPRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial failed with secret-token and org-other")
		})},
	})
	router := gin.New()
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
		Method: http.MethodGet, Path: "/api/v1/workbench/context", AuthPolicy: httproute.AuthPolicyVerifiedIdentity,
		OrganizationAccessPolicy: httproute.OrganizationAccessPolicyContextRead,
		Handler:                  func(c *gin.Context) { t.Fatal("handler ran after verifier dependency failure") },
	}}, routeAuthDependencies{
		workbenchVerifier:    verifier,
		organizationResolver: mountedOrganizationResolverStub{events: &[]string{}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workbench/context", nil)
	request.Header.Set("Authorization", "Bearer secret-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusServiceUnavailable, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), "DEPENDENCY_UNAVAILABLE")
	require.NotContains(t, response.Body.String(), "secret-token")
	require.NotContains(t, response.Body.String(), "org-other")
}

func TestWorkbenchVerifierPathPreservesGinRecoveryOnHandlerPanic(t *testing.T) {
	router := gin.New()
	router.Use(gin.Recovery())
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
		Method: http.MethodGet, Path: "/workbench", AuthPolicy: httproute.AuthPolicyVerifiedIdentity,
		OrganizationAccessPolicy: httproute.OrganizationAccessPolicyCachedRead,
		Handler:                  func(*gin.Context) { panic("handler panic") },
	}}, routeAuthDependencies{
		workbenchVerifier:    mountedVerifierStub{identity: mountedVerifiedIdentity()},
		organizationResolver: mountedOrganizationResolverStub{events: &[]string{}, roles: []string{"listingkit_viewer"}},
	})
	request := httptest.NewRequest(http.MethodGet, "/workbench", nil)
	request.Header.Set("Authorization", "Bearer current-request-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusInternalServerError, response.Code)
}

func TestWorkbenchOrganizationScopedSuccessStreamsWithoutResponseBuffering(t *testing.T) {
	events := []string{}
	response := httptest.NewRecorder()
	flushedBeforeReturn := false
	visibleStatusBeforeReturn := 0
	visibleBodyBeforeReturn := ""
	visibleHeaderBeforeReturn := ""
	router := gin.New()
	mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
		Method: http.MethodGet, Path: "/workbench/stream", Module: "listing-kit-admin",
		Permission: authz.PermissionListingKitAdminRead, AuthPolicy: httproute.AuthPolicyVerifiedIdentity,
		OrganizationAccessPolicy: httproute.OrganizationAccessPolicyCachedRead,
		Handler: func(c *gin.Context) {
			c.Header("X-Workbench-Stream", "visible")
			c.Header("Content-Length", "5")
			c.Status(http.StatusPartialContent)
			_, _ = c.Writer.WriteString("hello")
			c.Writer.Flush()
			flushedBeforeReturn = response.Flushed
			visibleStatusBeforeReturn = response.Code
			visibleBodyBeforeReturn = response.Body.String()
			visibleHeaderBeforeReturn = response.Header().Get("X-Workbench-Stream")
		},
	}}, routeAuthDependencies{
		workbenchVerifier:    mountedVerifierStub{identity: mountedVerifiedIdentity()},
		organizationResolver: mountedOrganizationResolverStub{events: &events, roles: []string{"listingkit_admin"}},
	})
	request := httptest.NewRequest(http.MethodGet, "/workbench/stream", nil)
	request.Header.Set("Authorization", "Bearer current-request-token")

	router.ServeHTTP(response, request)

	require.True(t, flushedBeforeReturn)
	require.Equal(t, http.StatusPartialContent, visibleStatusBeforeReturn)
	require.Equal(t, "hello", visibleBodyBeforeReturn)
	require.Equal(t, "visible", visibleHeaderBeforeReturn)
	require.Equal(t, http.StatusPartialContent, response.Code)
	require.Equal(t, "hello", response.Body.String())
	require.Equal(t, "visible", response.Header().Get("X-Workbench-Stream"))
	require.Equal(t, "5", response.Header().Get("Content-Length"))
}

func TestWorkbenchOrganizationErrorsUseStableNonEnumeratingProtocol(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "selection", err: workbenchcontext.ErrOrganizationSelectionRequired, wantStatus: http.StatusConflict, wantCode: "ORGANIZATION_SELECTION_REQUIRED"},
		{name: "denied", err: workbenchcontext.ErrOrganizationAccessDenied, wantStatus: http.StatusForbidden, wantCode: "ORGANIZATION_ACCESS_DENIED"},
		{name: "revoked", err: workbenchcontext.ErrOrganizationAccessRevoked, wantStatus: http.StatusForbidden, wantCode: "ORGANIZATION_ACCESS_REVOKED"},
		{name: "suspended", err: workbenchcontext.ErrOrganizationSuspended, wantStatus: http.StatusForbidden, wantCode: "ORGANIZATION_SUSPENDED"},
		{name: "dependency", err: errors.New("provider exposed org-other and secret-token"), wantStatus: http.StatusServiceUnavailable, wantCode: "DEPENDENCY_UNAVAILABLE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			mountRoutesWithAuthDependencies(router, []httproute.Descriptor{{
				Method: http.MethodGet, Path: "/api/v1/workbench/context", AuthPolicy: httproute.AuthPolicyVerifiedIdentity,
				OrganizationAccessPolicy: httproute.OrganizationAccessPolicyCachedRead,
				Handler:                  func(c *gin.Context) { t.Fatal("handler ran after organization resolution error") },
			}}, routeAuthDependencies{
				workbenchVerifier:    mountedVerifierStub{identity: mountedVerifiedIdentity()},
				organizationResolver: mountedOrganizationResolverError{err: tt.err},
			})
			request := httptest.NewRequest(http.MethodGet, "/api/v1/workbench/context", nil)
			request.Header.Set("Authorization", "Bearer secret-token")
			request.Header.Set("X-Requested-Organization-ID", "org-other")
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			require.Equal(t, tt.wantStatus, response.Code, response.Body.String())
			var body struct {
				Code string `json:"code"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			require.Equal(t, tt.wantCode, body.Code)
			require.NotContains(t, response.Body.String(), "org-other")
			require.NotContains(t, response.Body.String(), "secret-token")
		})
	}
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestBuildHTTPServerFromRoutesAtBindsLoopback(t *testing.T) {
	server := buildHTTPServerFromRoutesAt("127.0.0.1", 18085, nil, appHTTPTestRouteAuthorization)
	if server.Addr != "127.0.0.1:18085" {
		t.Fatalf("server address = %q, want loopback-only listener", server.Addr)
	}
	if server.ReadTimeout != 0 {
		t.Fatalf("server read timeout = %s, want route-scoped body deadlines", server.ReadTimeout)
	}
	if got := buildHTTPServerFromRoutes(18085, nil, appHTTPTestRouteAuthorization).Addr; got != ":18085" {
		t.Fatalf("default server address = %q, want backward-compatible wildcard", got)
	}
}

func TestHTTPServersKeepRouteAuthorizationIsolated(t *testing.T) {
	issuerA := newRouteAuthorizationIssuer(t, "token-a", "role-a")
	issuerB := newRouteAuthorizationIssuer(t, "token-b", "role-b")

	authA, err := buildRouteAuthorization(&config.Config{ListingKit: config.ListingKitConfig{
		PlatformAdminRoles: []string{"role-a"},
		Zitadel: config.ListingKitZitadelConfig{
			IssuerURL: issuerA.URL,
			ClientID:  "client-a",
		},
	}})
	require.NoError(t, err)
	authB, err := buildRouteAuthorization(&config.Config{ListingKit: config.ListingKitConfig{
		PlatformAdminRoles: []string{"role-b"},
		Zitadel: config.ListingKitZitadelConfig{
			IssuerURL: issuerB.URL,
			ClientID:  "client-b",
		},
	}})
	require.NoError(t, err)

	route := httproute.Descriptor{
		Method: http.MethodPost,
		Path:   "/api/v1/listing-kits/platform/test",
		Module: "listing-kit-platform-admin",
		Handler: func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		},
	}
	serverA := buildHTTPServerFromRoutes(0, []httproute.Descriptor{route}, authA)
	serverB := buildHTTPServerFromRoutes(0, []httproute.Descriptor{route}, authB)

	assertStatus := func(server *http.Server, token string, want int) {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, route.Path, nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)
		require.Equal(t, want, response.Code, response.Body.String())
	}
	assertStatus(serverA, "token-a", http.StatusNoContent)
	assertStatus(serverB, "token-b", http.StatusNoContent)
	assertStatus(serverA, "token-b", http.StatusUnauthorized)
	assertStatus(serverB, "token-a", http.StatusUnauthorized)
}

func newRouteAuthorizationIssuer(t *testing.T, acceptedToken, role string) *httptest.Server {
	t.Helper()
	var issuer *httptest.Server
	issuer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"introspection_endpoint": issuer.URL + "/oauth/v2/introspect",
			})
		case "/oauth/v2/introspect":
			active := r.FormValue("token") == acceptedToken
			_ = json.NewEncoder(w).Encode(map[string]any{
				"active":                                active,
				"sub":                                   "user-" + role,
				"user_id":                               "user-" + role,
				"urn:zitadel:iam:user:resourceowner:id": "tenant-" + role,
				"urn:zitadel:iam:org:project:roles":     map[string]any{role: map[string]any{}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(issuer.Close)
	return issuer
}

func TestCurrentSMSWebhookDoesNotRelaxProtectedImageRoutes(t *testing.T) {
	webhookCalled := false
	listingKitCalled := false
	server := buildHTTPServerFromRoutes(0, []httproute.Descriptor{
		{
			Method: http.MethodPost,
			Path:   zitadelSMSPath,
			Module: "zitadel-sms", AuthPolicy: httproute.AuthPolicyPublic,
			Handler: func(c *gin.Context) {
				webhookCalled = true
				c.Status(http.StatusNoContent)
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/api/v1/image-agent/runs",
			Module: "image-agent", AuthPolicy: httproute.AuthPolicyVerifiedIdentity,
			Handler: func(c *gin.Context) {
				listingKitCalled = true
				c.Status(http.StatusOK)
			},
		},
	}, appHTTPTestRouteAuthorization)

	webhookResponse := httptest.NewRecorder()
	server.Handler.ServeHTTP(webhookResponse, httptest.NewRequest(http.MethodPost, zitadelSMSPath, nil))
	require.Equal(t, http.StatusNoContent, webhookResponse.Code)
	require.True(t, webhookCalled)

	listingKitResponse := httptest.NewRecorder()
	server.Handler.ServeHTTP(listingKitResponse, httptest.NewRequest(http.MethodGet, "/api/v1/image-agent/runs", nil))
	require.Equal(t, http.StatusUnauthorized, listingKitResponse.Code)
	require.False(t, listingKitCalled)
}
