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

	"gorm.io/driver/postgres"
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

func verifyPostgresNamespaces(conn *gorm.DB) error {
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
	return nil
}

func verifyIsolatedSession(conn *gorm.DB, target RetainedTarget) error {
	if err := verifyPostgresNamespaces(conn); err != nil {
		return err
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

// ApplyRetainedOriginalInContainer may only be invoked by the explicitly
// admitted host controller through the pinned original DB container. It is
// not a serving entry point. The local Unix session and backend namespaces are
// rechecked on the same physical connection before taking the transaction lock.
func ApplyRetainedOriginalInContainer(ctx context.Context, db *gorm.DB, target RetainedTarget) (RetainedState, error) {
	if runtime.GOOS != "linux" || ctx == nil || db == nil || db.Dialector.Name() != "postgres" || target.Database != "product_agent" || target.Address != "" || target.Port != 0 || target.SessionUser != "business_cluster_admin" || strings.TrimSpace(target.MaintenanceWindow) == "" || strings.TrimSpace(target.BackupEvidence) == "" || !time.Now().Before(target.NotAfter) {
		return "", errors.New("original local maintenance connection invalid")
	}
	windowCtx, cancel := context.WithDeadline(ctx, target.NotAfter)
	defer cancel()
	var result RetainedState
	err := db.WithContext(windowCtx).Connection(func(conn *gorm.DB) error {
		if err := verifyIsolatedSession(conn, target); err != nil {
			return err
		}
		var err error
		result, err = prepareRetainedOnConnection(conn, target)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("original preparation not confirmed; inspect the same target before retry: %w", err)
	}
	return result, nil
}

// CaptureRetainedOriginalInContainer reads the old baseline on the same
// container-local session used to verify its actual PostgreSQL backend.
func CaptureRetainedOriginalInContainer(ctx context.Context, db *gorm.DB, target RetainedTarget) (string, string, error) {
	if runtime.GOOS != "linux" || ctx == nil || db == nil || db.Dialector.Name() != "postgres" || target.Database != "product_agent" || target.Address != "" || target.Port != 0 || target.SessionUser != "business_cluster_admin" {
		return "", "", errors.New("original local inspection connection invalid")
	}
	var facts, acl string
	err := db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
		if err := verifyIsolatedSession(conn, target); err != nil {
			return err
		}
		var err error
		facts, acl, err = captureRetainedOnConnection(conn, target)
		return err
	})
	return facts, acl, err
}

// InspectRetainedOriginalInContainer classifies the same target without any
// write, including after a maintenance window or acknowledgement has expired.
func InspectRetainedOriginalInContainer(ctx context.Context, db *gorm.DB, target RetainedTarget) (RetainedState, error) {
	if runtime.GOOS != "linux" || ctx == nil || db == nil || db.Dialector.Name() != "postgres" || target.Database != "product_agent" || target.Address != "" || target.Port != 0 || target.SessionUser != "business_cluster_admin" {
		return "", errors.New("original local inspection connection invalid")
	}
	var state RetainedState
	err := db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
		if err := verifyIsolatedSession(conn, target); err != nil {
			return err
		}
		return conn.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("SET TRANSACTION READ ONLY, ISOLATION LEVEL REPEATABLE READ").Error; err != nil {
				return err
			}
			var err error
			state, err = retainedState(tx, target)
			return err
		})
	})
	return state, err
}

// CheckRetainedRuntimeInContainer checks the serving role and original DB on
// the same local socket, then runs the existing runtime schema verification.
func CheckRetainedRuntimeInContainer(ctx context.Context, db *gorm.DB, target RetainedTarget) error {
	if runtime.GOOS != "linux" || ctx == nil || db == nil || db.Dialector.Name() != "postgres" || target.Address != "" || target.Port != 0 {
		return errors.New("local runtime connection invalid")
	}
	return db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
		if err := verifyPostgresNamespaces(conn); err != nil {
			return err
		}
		var got struct {
			Database, Role, Address string
			SystemID                string `gorm:"column:system_id"`
			OID                     string `gorm:"column:oid"`
			Port                    int
		}
		if err := conn.Raw(`SELECT current_database() AS database, session_user AS role, (SELECT oid::text FROM pg_database WHERE datname=current_database()) AS oid, (SELECT system_identifier::text FROM pg_control_system()) AS system_id, coalesce(inet_server_addr()::text,'') AS address, coalesce(inet_server_port(),0) AS port`).Scan(&got).Error; err != nil || got.Database != target.Database || got.Role != "ai_workbench_runtime" || got.OID != target.DatabaseOID || got.SystemID != target.SystemIdentifier || got.Address != "" || got.Port != 0 {
			return errors.New("runtime socket target mismatch")
		}
		// A fresh GORM root avoids reusing the identity query's statement;
		// its pool is the same pinned physical connection, not the parent pool.
		clean, err := gorm.Open(postgres.New(postgres.Config{Conn: conn.Statement.ConnPool}), &gorm.Config{DisableAutomaticPing: true})
		if err != nil {
			return errors.New("runtime verification connection unavailable")
		}
		if err := validateRetainedDatabasePrivileges(clean); err != nil {
			return err
		}
		return VerifySchema(ctx, clean)
	})
}
