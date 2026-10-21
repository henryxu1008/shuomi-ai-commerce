package sheinlogin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCurrentBindingsRejectLegacyAndAmbiguousConfiguration(t *testing.T) {
	s, _, _, _, _ := currentService(t)
	cfg := s.config
	for _, mutate := range []func(*Config){func(c *Config) { c.Redis.Host = "" }, func(c *Config) { c.InstallationID = "1" }, func(c *Config) { c.Bindings = append(c.Bindings, c.Bindings[0]) }, func(c *Config) { c.Bindings[0].StoreID = "869" }, func(c *Config) { c.Bindings[0].Revision = "../legacy" }, func(c *Config) { c.ProfileRoot = "relative" }, func(c *Config) { c.StoreDatabase.User = "store_center_runtime" }, func(c *Config) { c.Bindings[0].LoginURL = "https://wrong.example" }} {
		changed := cfg
		changed.Bindings = append([]Binding(nil), cfg.Bindings...)
		mutate(&changed)
		if changed.Validate() == nil {
			t.Fatal("invalid explicit binding accepted")
		}
	}
	raw, _ := json.Marshal(cfg)
	var decoded Config
	if DecodeStrict(append(raw, []byte(` {}`)...), &decoded, 1<<20) == nil {
		t.Fatal("trailing object accepted")
	}
	if DecodeStrict([]byte(strings.TrimSuffix(string(raw), "}")+`,"legacy_store_id":869}`), &decoded, 1<<20) == nil {
		t.Fatal("legacy input accepted")
	}
	binding := cfg.Bindings[0]
	scope := cfg.Scope(binding)
	scope.Generation = 1
	account, e := binding.LoadAccount(context.Background(), scope)
	if e != nil || account.OrganizationID != "123" || account.StoreID != binding.StoreID {
		t.Fatal(account.OrganizationID, e)
	}
	data, _ := json.Marshal(account)
	if strings.Contains(string(data), "private-") {
		t.Fatal("account serialized credential")
	}
	// Changed metadata produces a new exact fingerprint; secret contents do not
	// become a fingerprint, queue field or second persisted identity.
	original := cfg.Scope(binding)
	binding.Headless = !binding.Headless
	if cfg.Scope(binding).ConfigFingerprint == original.ConfigFingerprint {
		t.Fatal("browser metadata omitted")
	}
	cfg.Bindings[0].CredentialFile = filepath.Join(t.TempDir(), "missing")
	if cfg.Validate() != nil {
		t.Fatal("producer validation read credential")
	}
	var object map[string]any
	if DecodeStrict(make([]byte, 65537), &object, 65536) == nil {
		t.Fatal("unbounded command accepted")
	}
	if _, e = ReadPrivate(context.Background(), "relative", 65536); e == nil {
		t.Fatal("relative credential file accepted")
	}
}
func TestOwnedProfileRootsRejectExistingContentsAndForeignInstallation(t *testing.T) {
	s, _, _, _, _ := currentService(t)
	root := s.config.ProfileRoot
	if e := os.MkdirAll(root, 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "old-profile"), []byte("synthetic"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := PrepareOwnedRoots(s.config); e == nil {
		t.Fatal("existing root adopted")
	}
	marker := filepath.Join(root, ".shein-login-owner")
	if _, e := os.Stat(marker); !os.IsNotExist(e) {
		t.Fatal("unowned root changed")
	}
}
