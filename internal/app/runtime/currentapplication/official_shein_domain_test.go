package currentapplication

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
)

func TestSheinApplicationModePinsProductionAPIOrigin(t *testing.T) {
	for _, mode := range []storecenter.OfficialApplicationType{storecenter.ApplicationSelfOperated, storecenter.ApplicationSemiManaged, storecenter.ApplicationFullyManaged} {
		t.Run(string(mode), func(t *testing.T) {
			root := t.TempDir()
			config := OfficialStoreConnectionConfig{Type: mode, AppID: "offline-app", Version: "v1", CallbackURL: "https://example.invalid/callback", AppSecretFile: filepath.Join(root, "app-secret"), CredentialKeyFile: filepath.Join(root, "merchant-key"), CredentialKeyID: "offline-key"}
			want, wrong := "https://openapi.sheincorp.com", "https://openapi.sheincorp.cn"
			if mode == storecenter.ApplicationFullyManaged {
				want, wrong = wrong, want
			}
			config.APIOrigin = want
			require.NoError(t, config.validate())
			config.APIOrigin = wrong
			require.Error(t, config.validate(), "wrong application/host pairing must fail before secret files or HTTP")
		})
	}
}
