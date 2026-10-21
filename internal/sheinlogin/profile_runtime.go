package sheinlogin

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
	"task-processor/internal/storecenter"
)

func resolveProfileDir(root string, s storecenter.SheinLoginScope) (string, error) {
	if !filepath.IsAbs(root) || s.Validate() != nil || s.Generation < 1 {
		return "", ErrInvalidInput
	}
	sum := sha256.Sum256([]byte(s.OrganizationID))
	return filepath.Join(root, "v1", hex.EncodeToString(sum[:]), s.StoreID, s.BindingRevision), nil
}
func isProfileInUseError(e error) bool {
	if e == nil {
		return false
	}
	v := strings.ToLower(e.Error())
	return strings.Contains(v, "profile appears to be in use") || strings.Contains(v, "processsingleton") || strings.Contains(v, "profile directory") || strings.Contains(v, "singletonlock")
}
