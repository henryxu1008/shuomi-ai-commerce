package setupcontrol

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"task-processor/internal/httproute"
	"time"
)

type Module struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Configured bool   `json:"configured"`
	Installed  string `json:"installed"`
	Enabled    bool   `json:"enabled"`
	UIEnabled  *bool  `json:"uiEnabled"`
	Mutable    bool   `json:"mutable"`
	Note       string `json:"note"`
}
type Instance struct {
	ID           string   `json:"id"`
	Project      string   `json:"project"`
	Origin       string   `json:"origin"`
	BackendImage string   `json:"backendImage"`
	UIImage      string   `json:"uiImage"`
	Modules      []Module `json:"modules,omitempty"`
	Blocks       []string `json:"blocks"`
	Pending      *Plan    `json:"pending,omitempty"`
}
type Profile struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Note  string `json:"note"`
}
type Snapshot struct {
	History       []Plan     `json:"history"`
	HostFreeBytes *uint64    `json:"hostFreeBytes"`
	Pending       []Plan     `json:"pending"`
	DockerReady   bool       `json:"dockerReady"`
	Version       string     `json:"version"`
	Instances     []Instance `json:"instances"`
	Profiles      []Profile  `json:"profiles"`
	Blocks        []string   `json:"blocks"`
}
type Catalog interface {
	Snapshot(context.Context) (Snapshot, error)
	Inspect(context.Context, string) (Instance, error)
}

// HTTP is exclusively a loopback, host-launched installer. It never accepts a
// tenant token as installation authority and does not expose its private driver.
type HTTP struct {
	host, origin        string
	control             *Control
	catalog             Catalog
	assets              http.Handler
	mu                  sync.Mutex
	pair, session, csrf string
	expires             time.Time
	attempts            int
}

func RandomSecret() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func NewHTTP(host, code string, c *Control, catalog Catalog, assets http.Handler) *HTTP {
	return &HTTP{host: host, origin: "http://" + host, pair: code, control: c, catalog: catalog, assets: assets}
}
func equal(a, b string) bool {
	return len(a) > 0 && len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func output(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, e error) {
	status := 503
	code := ErrUnavailable.Error()
	switch {
	case errors.Is(e, ErrInvalid):
		status = 400
		code = e.Error()
	case errors.Is(e, ErrDrift), errors.Is(e, ErrBusy), errors.Is(e, ErrUnresolved), errors.Is(e, ErrExpired):
		status = 409
		code = e.Error()
	}
	output(w, status, map[string]string{"error": code})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
		problem(w, ErrInvalid)
		return false
	}
	b, e := io.ReadAll(io.LimitReader(r.Body, 16385))
	if e != nil || httproute.DecodeJSON(b, v, 16384, true) != nil {
		problem(w, ErrInvalid)
		return false
	}
	return true
}
func (s *HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	if r.Host != s.host || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != s.origin) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		http.Error(w, "Forbidden", 403)
		return
	}
	if r.URL.RawQuery != "" {
		problem(w, ErrInvalid)
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/api/") {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		s.assets.ServeHTTP(w, r)
		return
	}
	if r.Method != "GET" && r.Header.Get("Origin") != s.origin {
		http.Error(w, "Forbidden", 403)
		return
	}
	if r.URL.Path == "/api/pair" && r.Method == "POST" {
		var v struct {
			Code string `json:"code"`
		}
		if !decode(w, r, &v) {
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.attempts++
		if s.attempts > 5 || !equal(s.pair, v.Code) {
			http.Error(w, "Pairing refused", 403)
			return
		}
		s.pair = ""
		s.session = RandomSecret()
		s.csrf = RandomSecret()
		s.expires = time.Now().Add(time.Hour)
		http.SetCookie(w, &http.Cookie{Name: "setup_session", Value: s.session, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 3600})
		output(w, 200, map[string]string{"csrf": s.csrf})
		return
	}
	s.mu.Lock()
	cookie, e := r.Cookie("setup_session")
	authorized := e == nil && equal(s.session, cookie.Value) && time.Now().Before(s.expires)
	csrf := s.csrf
	s.mu.Unlock()
	if !authorized {
		http.Error(w, "Pairing required", 401)
		return
	}
	if r.Method != "GET" && !equal(csrf, r.Header.Get("X-Setup-CSRF")) {
		http.Error(w, "Forbidden", 403)
		return
	}
	if r.URL.Path == "/api/session" && r.Method == "GET" {
		output(w, 200, map[string]string{"csrf": csrf})
		return
	}
	if r.URL.Path == "/api/logout" && r.Method == "POST" {
		s.mu.Lock()
		s.session = ""
		s.csrf = ""
		s.mu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: "setup_session", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
		output(w, 200, struct{}{})
		return
	}
	if r.URL.Path == "/api/environment" && r.Method == "GET" {
		v, e := s.catalog.Snapshot(r.Context())
		if e == nil {
			v.Pending, e = s.control.Outstanding()
		}
		if e == nil {
			v.History, e = s.control.History()
		}
		if e != nil {
			problem(w, e)
		} else {
			output(w, 200, v)
		}
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 3 && parts[1] == "instances" && r.Method == "GET" {
		v, e := s.catalog.Inspect(r.Context(), parts[2])
		if e == nil {
			v.Pending, e = s.control.Pending(parts[2])
		}
		if e != nil {
			problem(w, e)
		} else {
			output(w, 200, v)
		}
		return
	}
	if r.URL.Path == "/api/plans" && r.Method == "POST" {
		var v Request
		if !decode(w, r, &v) {
			return
		}
		p, e := s.control.Plan(r.Context(), v)
		if e != nil {
			problem(w, e)
		} else {
			output(w, 200, p)
		}
		return
	}
	if len(parts) == 3 && parts[1] == "plans" && r.Method == "GET" {
		p, e := s.control.Read(parts[2])
		if e != nil {
			problem(w, e)
		} else {
			output(w, 200, p)
		}
		return
	}
	if len(parts) == 4 && parts[1] == "plans" && r.Method == "POST" && parts[3] == "recovery-preview" {
		var input struct {
			Action        string `json:"action"`
			NetworkSubnet string `json:"networkSubnet"`
		}
		if !decode(w, r, &input) {
			return
		}
		p, e := s.control.PrepareRecovery(r.Context(), parts[2], input.Action, input.NetworkSubnet)
		if e != nil {
			problem(w, e)
		} else {
			output(w, 200, p)
		}
		return
	}
	if len(parts) == 4 && parts[1] == "plans" && r.Method == "POST" && parts[3] == "recovery-confirm" {
		var input struct {
			PreviewID string `json:"previewId"`
		}
		if !decode(w, r, &input) {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
		defer cancel()
		p, e := s.control.ConfirmRecovery(ctx, parts[2], input.PreviewID)
		if e != nil {
			problem(w, e)
		} else {
			output(w, 200, p)
		}
		return
	}
	if len(parts) == 4 && parts[1] == "plans" && r.Method == "POST" && (parts[3] == "apply" || parts[3] == "reconcile") {
		var v struct{}
		if !decode(w, r, &v) {
			return
		}
		// Once the intent is durable, browser disconnection cannot cancel or replay
		// a native operation. A process crash leaves its original receipt unresolved.
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
		defer cancel()
		var p Plan
		var e error
		if parts[3] == "apply" {
			p, e = s.control.Apply(ctx, parts[2])
		} else {
			p, e = s.control.Reconcile(ctx, parts[2])
		}
		if e != nil {
			problem(w, e)
		} else {
			output(w, 200, p)
		}
		return
	}
	http.NotFound(w, r)
}
