package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	recovery "task-processor/internal/app/modelinvocationrecovery"
	store "task-processor/internal/integration/persistence/aiworkbench"
)

func runFormalChild(capPath string, mode string) error {
	raw, err := consumeIsolatedCapability(capPath)
	if err != nil {
		return err
	}
	defer clear(raw)
	if err := dropToPostgres(); err != nil {
		return err
	}
	var req formalChildRequest
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return errors.New("invalid original maintenance capability")
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return errors.New("invalid original maintenance capability")
	}
	if mode == "state" || mode == "capture" {
		if err := validateFormalAnchor(req.Anchor, req.Target); err != nil {
			return err
		}
		if mode == "capture" && !formalCaptureWindowValid(req.Admission, req.Target, time.Now()) {
			return errors.New("frozen maintenance window required")
		}
		dsn := fmt.Sprintf("host=/var/run/postgresql port=%d user=business_cluster_admin dbname=product_agent sslmode=disable", req.Anchor.SocketPort)
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if err != nil {
			return errors.New("original inspection connection unavailable")
		}
		sqlDB, err := db.DB()
		if err != nil {
			return errors.New("original inspection connection unavailable")
		}
		defer sqlDB.Close()
		deadline := time.Now().Add(45 * time.Second)
		if mode == "capture" {
			if req.Admission.FreezeExpiresAt.Before(deadline) {
				deadline = req.Admission.FreezeExpiresAt
			}
			if req.Admission.ExpiresAt.Before(deadline) {
				deadline = req.Admission.ExpiresAt
			}
			if req.Target.NotAfter.Before(deadline) {
				deadline = req.Target.NotAfter
			}
		}
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()
		if mode == "capture" {
			facts, acl, err := store.CaptureRetainedOriginalInContainer(ctx, db, req.Target)
			if err != nil {
				return errors.New("original baseline unavailable")
			}
			if ctx.Err() != nil || !formalCaptureWindowValid(req.Admission, req.Target, time.Now()) {
				return errors.New("frozen maintenance window expired during capture")
			}
			fmt.Printf("ORIGINAL_WORKBENCH_BASELINE %s %s\n", facts, acl)
			return nil
		}
		state, err := store.InspectRetainedOriginalInContainer(ctx, db, req.Target)
		if err != nil {
			return errors.New("original target state unknown")
		}
		fmt.Println("ORIGINAL_WORKBENCH_STATE_" + string(state))
		return nil
	}
	if err := validateFormalAdmission(req.Anchor, req.Admission, req.Target, req.Admission.AnchorSHA256, time.Now()); err != nil {
		return err
	}
	if req.Anchor.ReservationBindingSHA256 != originalReservationIdentitySHA256 || req.Anchor.InvocationBindingSHA256 != originalInvocationBindingSHA256 {
		return errors.New("original invocation anchor mismatch")
	}
	recoveryRaw, err := io.ReadAll(io.LimitReader(os.Stdin, (16<<10)+1))
	if err != nil {
		return errors.New("protected recovery input unavailable")
	}
	defer clear(recoveryRaw)
	cfg, err := decodeFormalRecoveryInput(recoveryRaw, req.Anchor)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithDeadline(context.Background(), req.Target.NotAfter)
	defer cancel()
	// Existing owner entry uses read-only database connections for inspect; it
	// never constructs a provider client or calls resolve in this path.
	if err := inspectFormalReservation(ctx, req.Anchor, req.Admission, cfg, originalReservationIdentitySHA256, recovery.Run); err != nil {
		return err
	}
	if mode == "probe" {
		fmt.Println("FORMAL_INSPECT_VERIFIED")
		return nil
	}
	dsn := fmt.Sprintf("host=/var/run/postgresql port=%d user=business_cluster_admin dbname=product_agent sslmode=disable", req.Anchor.SocketPort)
	owner, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return errors.New("original owner connection unavailable")
	}
	ownerRaw, err := owner.DB()
	if err != nil {
		return errors.New("original owner connection unavailable")
	}
	defer ownerRaw.Close()
	if _, err := store.ApplyRetainedOriginalInContainer(ctx, owner, req.Target); err != nil {
		return errors.New("original preparation not confirmed; inspect the same target before retry")
	}
	runtimeDSN := fmt.Sprintf("host=/var/run/postgresql port=%d user=ai_workbench_runtime dbname=product_agent sslmode=disable", req.Anchor.SocketPort)
	runtimeDB, err := gorm.Open(postgres.Open(runtimeDSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return errors.New("committed or unknown; runtime connection unavailable")
	}
	runtimeRaw, err := runtimeDB.DB()
	if err != nil {
		return errors.New("committed or unknown; runtime connection unavailable")
	}
	defer runtimeRaw.Close()
	if err := store.CheckRetainedRuntimeInContainer(ctx, runtimeDB, req.Target); err != nil {
		return errors.New("committed or unknown; runtime verification failed")
	}
	fmt.Println("ORIGINAL_PREPARATION_VERIFIED")
	return nil
}
