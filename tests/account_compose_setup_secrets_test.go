package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupConsumedInstallersDoNotPassPasswordsInArguments(t *testing.T) {
	for _, name := range []string{"native-modules-init.sh", "chat-init.sh", "ecoservices-init.sh"} {
		raw, err := os.ReadFile(filepath.Join("..", "deployments", "docker", "account-compose", name))
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.Contains(line, "psql ") && (strings.Contains(line, "-v password=") || strings.Contains(line, "-v ownerpassword=") || strings.Contains(line, "-v runtimepassword=")) {
				t.Errorf("%s exposes generated password in psql argv", name)
			}
		}
	}
}
