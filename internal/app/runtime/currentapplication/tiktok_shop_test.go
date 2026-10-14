package currentapplication

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTikTokConfigIsOptionalAndRequiresSeparatePrivateFiles(t *testing.T) {
	var absent *TikTokShopConfig
	require.NoError(t, absent.validate())
	c := TikTokShopConfig{AppKey: "app", ServiceID: "service", Revision: "r1", CallbackURL: "https://example.org/workbench/stores/tiktokshop/callback", AppSecretFile: filepath.Join(t.TempDir(), "secret"), CredentialKeyFile: filepath.Join(t.TempDir(), "key"), CredentialKeyID: "key-1"}
	require.NoError(t, c.validate())
	for _, mutate := range []func(*TikTokShopConfig){func(v *TikTokShopConfig) { v.ServiceID = "" }, func(v *TikTokShopConfig) { v.CallbackURL = "http://example.org/callback" }, func(v *TikTokShopConfig) { v.CredentialKeyFile = v.AppSecretFile }, func(v *TikTokShopConfig) { v.AppSecretFile = "relative" }, func(v *TikTokShopConfig) { v.Revision = "" }} {
		invalid := c
		mutate(&invalid)
		require.Error(t, invalid.validate())
	}
}
