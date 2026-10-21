package sheinlogin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"task-processor/internal/core/config"
)

// PrepareOwnedRoots is serve-only. The first installation requires an empty
// explicitly configured root. An existing directory is never adopted silently.
func PrepareOwnedRoots(c Config) error {
	for _, root := range []string{c.ProfileRoot, c.ArtifactRoot} {
		if e := noSymlinkComponents(root); e != nil {
			return e
		}
		if e := os.MkdirAll(root, 0700); e != nil {
			return ErrUnavailable
		}
		marker := filepath.Join(root, ".shein-login-owner")
		data, e := os.ReadFile(marker)
		if e == nil {
			if string(data) != "shein-login:v1:"+c.InstallationID || config.VerifyPrivateFiles(context.Background(), []string{marker}) != nil {
				return ErrInvalidInput
			}
			continue
		}
		if !os.IsNotExist(e) {
			return ErrUnavailable
		}
		entries, e := os.ReadDir(root)
		if e != nil || len(entries) != 0 {
			return ErrInvalidInput
		}
		f, e := os.OpenFile(marker, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return ErrInvalidInput
		}
		_, e = f.WriteString("shein-login:v1:" + c.InstallationID)
		closeErr := f.Close()
		if e != nil || closeErr != nil {
			return ErrUnavailable
		}
		if config.VerifyPrivateFiles(context.Background(), []string{marker}) != nil {
			return ErrInvalidInput
		}
	}
	return nil
}
func noSymlinkComponents(path string) error {
	if !filepath.IsAbs(path) {
		return ErrInvalidInput
	}
	current := filepath.Clean(path)
	for {
		i, e := os.Lstat(current)
		if e == nil && i.Mode()&os.ModeSymlink != 0 {
			return ErrInvalidInput
		}
		if e != nil && !os.IsNotExist(e) {
			return ErrUnavailable
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return nil
}
func prepareProfile(root string, a Account) (string, error) {
	if noSymlinkComponents(root) != nil {
		return "", ErrInvalidInput
	}
	marker, e := os.ReadFile(filepath.Join(root, ".shein-login-owner"))
	if e != nil || string(marker) != "shein-login:v1:"+a.Scope.InstallationID {
		return "", ErrInvalidInput
	}
	dir, e := resolveProfileDir(root, a.Scope)
	if e != nil || noSymlinkComponents(dir) != nil {
		return "", ErrInvalidInput
	}
	if e = os.MkdirAll(dir, 0700); e != nil {
		return "", ErrUnavailable
	}
	path := filepath.Join(dir, ".current-scope.json")
	if data, e := os.ReadFile(path); e == nil {
		var prior struct {
			OrganizationID, StoreID, BindingRevision, InstallationID string
			Generation                                               int64
		}
		if DecodeStrict(data, &prior, 4096) != nil || prior.OrganizationID != a.Scope.OrganizationID || prior.StoreID != a.Scope.StoreID || prior.BindingRevision != a.Scope.BindingRevision || prior.InstallationID != a.Scope.InstallationID || prior.Generation > a.Generation {
			return "", ErrInvalidInput
		}
	} else if !os.IsNotExist(e) {
		return "", ErrUnavailable
	}
	data, _ := json.Marshal(struct {
		OrganizationID, StoreID, BindingRevision, InstallationID string
		Generation                                               int64
	}{a.Scope.OrganizationID, a.Scope.StoreID, a.Scope.BindingRevision, a.Scope.InstallationID, a.Generation})
	if os.WriteFile(path, data, 0600) != nil {
		return "", ErrUnavailable
	}
	return dir, nil
}
func safeArtifactRoot(root string, a Account) (string, error) {
	marker, e := os.ReadFile(filepath.Join(root, ".shein-login-owner"))
	if e != nil || strings.TrimSpace(string(marker)) != "shein-login:v1:"+a.Scope.InstallationID || noSymlinkComponents(root) != nil {
		return "", ErrInvalidInput
	}
	return root, nil
}
