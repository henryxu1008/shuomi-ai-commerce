package currentapplication

import (
	"testing"
)

func TestMembershipConfigurationRequiresSeparatePoolAndExactIdentityOrigin(t *testing.T) {
	identity := IdentityConfig{IssuerURL: "http://127.0.0.1:8800", AuthorizationAPIURL: "http://127.0.0.1:8800", ProjectID: "project"}
	valid := MembershipConfig{ProviderOrigin: "http://127.0.0.1:8800", ReadToken: "read-only", WriteToken: "write-only", Database: DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "organization_membership_runtime", Password: "password", Database: "membership", MaxConnections: 2}}
	if err := valid.validate(identity); err != nil {
		t.Fatal(err)
	}
	identity.IssuerURL = "https://localhost:18443"
	identity.AuthorizationAPIURL = "https://localhost:18443"
	valid.ProviderOrigin = "https://localhost:18443"
	if err := valid.validate(identity); err != nil {
		t.Fatal(err)
	}
	identity.IssuerURL = "http://127.0.0.1:8800"
	identity.AuthorizationAPIURL = "http://127.0.0.1:8800"
	valid.ProviderOrigin = "http://127.0.0.1:8800"
	for _, mutate := range []func(*MembershipConfig){
		func(c *MembershipConfig) { c.ProviderOrigin = "http://127.0.0.1:8801" },
		func(c *MembershipConfig) { c.ProviderOrigin = "http://example.com:8800" },
		func(c *MembershipConfig) { c.ProviderOrigin += "/api" },
		func(c *MembershipConfig) { c.ReadToken = "" },
		func(c *MembershipConfig) { c.WriteToken = c.ReadToken },
		func(c *MembershipConfig) { c.Database.User = "source_account_runtime" },
		func(c *MembershipConfig) { c.Database.Host = "remote" },
	} {
		candidate := valid
		mutate(&candidate)
		if candidate.validate(identity) == nil {
			t.Fatalf("accepted invalid config (details intentionally omitted)")
		}
	}
	identity.ProjectID = "invalid/project"
	if valid.validate(identity) == nil {
		t.Fatal("invalid trusted project accepted")
	}
}

func TestLoadConfigPublicIdentityWithMembership(t *testing.T) {
	cfg, err := LoadConfig(writeManifest(t, validManifest()))
	if err != nil {
		t.Fatal(err)
	}
	const origin = "https://auth-trial.shuomiai.com"
	cfg.Identity.IssuerURL, cfg.Identity.AuthorizationAPIURL = origin, origin
	cfg.Membership = &MembershipConfig{ProviderOrigin: origin, ReadToken: "read-only", WriteToken: "write-only", Database: DatabaseConfig{Host: "127.0.0.1", Port: 15432, User: "organization_membership_runtime", Password: "password", Database: "membership", MaxConnections: 2}}
	if err := cfg.validate(); err != nil {
		t.Fatalf("public identity with native membership rejected: %v", err)
	}
	for _, candidate := range []string{"https://other.example.com", "http://auth-trial.shuomiai.com", origin + "/", origin + "/api", origin + ":443", "https://localhost:18443"} {
		cfg.Membership.ProviderOrigin = candidate
		if cfg.validate() == nil {
			t.Fatalf("unadmitted membership origin accepted: %s", candidate)
		}
	}
}
