package skillsandbox

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	osb "github.com/alibaba/OpenSandbox/sdks/sandbox/go"
	strict "sigs.k8s.io/json"
	"task-processor/internal/agent"
)

// Config comes only from private installation configuration. TrialEvidence is
// the deployment owner's explicit reference to completed D-IT1 checks, never a
// guest assertion. Absence leaves this profile closed; the POC does not supply it.
type Config struct {
	Domain         string                 `json:"domain"`
	CAFile         string                 `json:"caFile"`
	CredentialFile string                 `json:"credentialFile"`
	TrialEvidence  string                 `json:"trialEvidence"`
	Descriptor     agent.ScriptDescriptor `json:"descriptor"`
}
type credential struct {
	User      string `json:"user"`
	Password  string `json:"password"`
	Lifecycle string `json:"lifecycle"`
}
type Executor struct {
	descriptor   agent.ScriptDescriptor
	lifecycle    *osb.LifecycleClient
	client       *http.Client
	origin       string
	headers      map[string]string
	gate         chan struct{}
	unknownUntil time.Time // ambiguous creation consumes capacity until its TTL
}

func privateFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, agent.ErrUnavailable
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 16<<10+1))
	if err != nil || len(raw) > 16<<10 {
		return nil, agent.ErrUnavailable
	}
	return raw, nil
}
func (cfg Config) Validate() error {
	proof, err := hex.DecodeString(strings.TrimPrefix(cfg.TrialEvidence, "sha256:"))
	if !cfg.Descriptor.Valid() || len(proof) != 32 || err != nil || "sha256:"+hex.EncodeToString(proof) != cfg.TrialEvidence || cfg.CAFile == "" || cfg.CredentialFile == "" {
		return agent.ErrUnavailable
	}
	u, err := url.Parse("https://" + cfg.Domain)
	if err != nil || len(cfg.Domain) > 256 || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return agent.ErrInvalid
	}
	return nil
}
func New(cfg Config) (*Executor, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	u, _ := url.Parse("https://" + cfg.Domain)
	ca, err := privateFile(cfg.CAFile)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		return nil, agent.ErrUnavailable
	}
	raw, err := privateFile(cfg.CredentialFile)
	if err != nil {
		return nil, err
	}
	var secret credential
	violations, err := strict.UnmarshalStrict(raw, &secret)
	if err != nil || len(violations) != 0 || secret.User == "" || strings.ContainsAny(secret.User, ":\r\n") || len(secret.Password) < 16 || len(secret.Lifecycle) < 16 {
		return nil, agent.ErrUnavailable
	}
	transport := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 2 * time.Second}).DialContext, DisableCompression: true, TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 2 * time.Second, ResponseHeaderTimeout: 5 * time.Second, MaxConnsPerHost: 4}
	client := &http.Client{Transport: boundedTransport{base: transport, origin: u.String()}, CheckRedirect: func(*http.Request, []*http.Request) error { return errOrigin }}
	request, _ := http.NewRequest(http.MethodGet, u.String(), nil)
	request.SetBasicAuth(secret.User, secret.Password)
	headers := map[string]string{"Authorization": request.Header.Get("Authorization"), "OPEN-SANDBOX-API-KEY": secret.Lifecycle}
	lifecycle := osb.NewLifecycleClient(u.String()+"/v1", secret.Lifecycle, osb.WithHTTPClient(client), osb.WithHeaders(headers))
	return &Executor{descriptor: cfg.Descriptor, lifecycle: lifecycle, client: client, origin: u.String(), headers: headers, gate: make(chan struct{}, 1)}, nil
}
func (e *Executor) Descriptor() agent.ScriptDescriptor {
	if e == nil {
		return agent.ScriptDescriptor{}
	}
	return e.descriptor
}
