package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestSetupConsumedInstallersDoNotPassPasswordsInArguments(t *testing.T) {
	for _, name := range []string{"business-db-init.sh", "native-modules-init.sh", "chat-init.sh", "ecoservices-init.sh", "knowledge-storage-init.sh", "qualification-storage-init.sh", "ecoservices-storage-init.sh", "capacity-storage-init.sh"} {
		raw, err := os.ReadFile(filepath.Join("..", "deployments", "docker", "account-compose", name))
		if err != nil {
			t.Fatal(err)
		}
		logical := strings.ReplaceAll(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\\\n", "")
		for _, line := range strings.Split(logical, "\n") {
			if strings.Contains(line, "psql ") && regexp.MustCompile(`-v\s+"?(?:ownerpassword|runtimepassword|password)=`).MatchString(line) {
				t.Errorf("%s exposes generated password in psql argv", name)
			}
			commandParts := strings.Split(line, "|")
			command := commandParts[len(commandParts)-1]
			if (strings.Contains(command, "mc alias set ") || strings.Contains(command, "mc admin user add ")) && strings.Contains(command, "$(cat") {
				t.Errorf("%s exposes generated storage credential in mc argv", name)
			}
		}
	}
}
