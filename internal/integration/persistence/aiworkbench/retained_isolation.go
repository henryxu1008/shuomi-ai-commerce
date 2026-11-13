package aiworkbenchpersistence

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"gorm.io/gorm"
)

func verifyRetainedIsolation(containerID, hostname string, interfaces []net.Interface) error {
	if len(containerID) != 64 || hostname != containerID[:12] {
		return errors.New("isolated container identity mismatch")
	}
	if _, err := hex.DecodeString(containerID); err != nil {
		return errors.New("invalid isolated container identity")
	}
	if len(interfaces) != 1 || interfaces[0].Name != "lo" || interfaces[0].Flags&net.FlagLoopback == 0 {
		return errors.New("isolated process has a network interface")
	}
	return nil
}

func verifyIsolatedSession(conn *gorm.DB, target RetainedTarget) error {
	var pid int
	if err := conn.Raw(`SELECT pg_backend_pid()`).Scan(&pid).Error; err != nil || pid <= 1 {
		return errors.New("database backend identity unavailable")
	}
	for _, ns := range []string{"mnt", "net", "pid"} {
		self, err := os.Readlink("/proc/self/ns/" + ns)
		if err != nil {
			return errors.New("process namespace unavailable")
		}
		backend, err := os.Readlink(fmt.Sprintf("/proc/%d/ns/%s", pid, ns))
		if err != nil || self != backend {
			return errors.New("database backend is outside isolated process namespaces")
		}
	}
	return retainedIdentity(conn, target)
}

// CheckRetainedIsolatedSession is read-only. It proves only the selected
// database/session and process namespaces, not backup or owner admission.
func CheckRetainedIsolatedSession(ctx context.Context, db *gorm.DB, target RetainedTarget, containerID string) error {
	if runtime.GOOS != "linux" || ctx == nil || db == nil || db.Dialector.Name() != "postgres" || target.Address != "" || target.Port != 0 || target.SessionUser != "chat1260_restore_admin" {
		return errors.New("not an isolated local connection")
	}
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	if err := verifyRetainedIsolation(containerID, hostname, interfaces); err != nil {
		return err
	}
	return db.WithContext(ctx).Connection(func(conn *gorm.DB) error { return verifyIsolatedSession(conn, target) })
}

// ApplyRetainedIsolated is an isolated-copy primitive, not an authorization
// decision: its caller must first verify separately protected source/restore
// records, owner attestations and actual Docker identity. It cannot operate on
// the original instance; it requires an isolated Linux container and Unix DB
// socket, and repeats the exact PostgreSQL identity check inside the transaction.
func ApplyRetainedIsolated(ctx context.Context, db *gorm.DB, target RetainedTarget, containerID string) (RetainedState, error) {
	if runtime.GOOS != "linux" || ctx == nil || db == nil || db.Dialector.Name() != "postgres" || target.Address != "" || target.Port != 0 || target.SessionUser != "chat1260_restore_admin" || strings.TrimSpace(target.MaintenanceWindow) == "" || strings.TrimSpace(target.BackupEvidence) == "" || !time.Now().Before(target.NotAfter) {
		return "", errors.New("not an isolated local maintenance connection")
	}
	hostname, err := os.Hostname()
	if err != nil {
		return "", errors.New("isolated hostname unavailable")
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", errors.New("isolated network unavailable")
	}
	if err := verifyRetainedIsolation(containerID, hostname, interfaces); err != nil {
		return "", err
	}
	windowCtx, cancel := context.WithDeadline(ctx, target.NotAfter)
	defer cancel()
	var result RetainedState
	err = db.WithContext(windowCtx).Connection(func(conn *gorm.DB) error {
		if err := verifyIsolatedSession(conn, target); err != nil {
			return err
		}
		var err error
		result, err = prepareRetainedOnConnection(conn, target)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("isolated preparation not confirmed; inspect the same target before retry: %w", err)
	}
	return result, nil
}
