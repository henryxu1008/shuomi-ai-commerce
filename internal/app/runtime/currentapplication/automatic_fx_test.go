package currentapplication

import "testing"

func TestAutomaticFXConfigurationRequiresExplicitBoundedTemporal(t *testing.T) {
	var off *AutomaticFXConfig
	if err := off.validate(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		address, namespace string
		valid              bool
	}{
		{"127.0.0.1:7233", "default", true},
		{"example.com:7233", "default", false},
		{"127.0.0.1:7233", "", false},
	} {
		c := &AutomaticFXConfig{TemporalAddress: tc.address, TemporalNamespace: tc.namespace}
		if (c.validate() == nil) != tc.valid {
			t.Fatalf("config %#v", tc)
		}
	}
}
