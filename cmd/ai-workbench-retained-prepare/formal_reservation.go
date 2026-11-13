package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	sigjson "sigs.k8s.io/json"

	recovery "task-processor/internal/app/modelinvocationrecovery"
	resourceadapter "task-processor/internal/integration/orgresource"
	"task-processor/internal/ledger/orgresource"
)

func decodeFormalRecoveryInput(raw []byte, anchor formalAnchor) (recovery.Config, error) {
	if len(raw) == 0 || len(raw) > 16<<10 {
		return recovery.Config{}, errors.New("invalid protected recovery input")
	}
	h := sha256.Sum256(raw)
	if hex.EncodeToString(h[:]) != anchor.RecoveryConfigSHA256 {
		return recovery.Config{}, errors.New("recovery input digest mismatch")
	}
	var cfg recovery.Config
	strict, err := sigjson.UnmarshalStrict(raw, &cfg, sigjson.DisallowDuplicateFields, sigjson.DisallowUnknownFields)
	if err != nil || len(strict) != 0 {
		return recovery.Config{}, errors.New("invalid protected recovery input")
	}
	if err := validateFormalRecoveryEndpoints(anchor, cfg); err != nil {
		return recovery.Config{}, err
	}
	return cfg, nil
}

func validateFormalRecoveryEndpoints(anchor formalAnchor, cfg recovery.Config) error {
	local := func(host string) bool { return host == "127.0.0.1" || host == "localhost" }
	if cfg.ResolveEnabled || !local(cfg.InvocationDatabase.Host) || !local(cfg.ResourceDatabase.Host) || cfg.InvocationDatabase.Port != anchor.SocketPort || cfg.ResourceDatabase.Port != anchor.SocketPort || cfg.InvocationDatabase.Database != "product_agent" || cfg.ResourceDatabase.Database != "commercial" || cfg.InvocationDatabase.User != "product_agent_runtime" || cfg.ResourceDatabase.User != resourceadapter.RuntimeRole {
		return errors.New("recovery inspect endpoints do not match the original local cluster")
	}
	return nil
}

// This binds the specifically named original call and reserve, independently
// of database credentials. It is never evidence that the call was unbilled.
const originalReservationIdentitySHA256 = "bd4f4346c06cea3500f82c1c41ece41aa8654acc5dca1d18b8c3b75c3006ad72"
const originalInvocationBindingSHA256 = "8dd876a5dd40522f6fadc8de8f0214e794ae9280b701fa338eeac219bbc9ddf9"

type recoveryInspector func(context.Context, recovery.Config, string) (recovery.Output, error)

// The only action passed to the existing owner is inspect. This has no
// provider client and cannot call resolve or modify a reservation.
func inspectFormalReservation(ctx context.Context, anchor formalAnchor, a formalAdmission, cfg recovery.Config, pin string, inspect recoveryInspector) error {
	if ctx == nil || inspect == nil || cfg.ResolveEnabled {
		return errors.New("read-only recovery inspection required")
	}
	out, err := inspect(ctx, cfg, "inspect")
	if err != nil {
		return errors.New(formalRecoveryFailure(err))
	}
	return validateReservationInspection(anchor, a, cfg, out, pin)
}

func formalRecoveryFailure(err error) string {
	if errors.Is(err, recovery.ErrRecoveryUnavailable) {
		switch err.Error() {
		case recovery.ErrRecoveryUnavailable.Error() + ": invocation runtime qualification":
			return "invocation runtime qualification unavailable"
		case recovery.ErrRecoveryUnavailable.Error() + ": resource runtime qualification":
			return "resource runtime qualification unavailable"
		}
	}
	return "original invocation inspection unavailable"
}

func reservationIdentityDigest(cfg recovery.Config) string {
	b, _ := json.Marshal([]string{cfg.OrganizationID, cfg.MemberID, cfg.InvocationID, cfg.ReservationID, cfg.ExpectedDispatchFingerprint})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func validateReservationInspection(anchor formalAnchor, a formalAdmission, cfg recovery.Config, out recovery.Output, pin string) error {
	r := out.Reservation
	if !validDigest(pin) || cfg.ResolveEnabled || cfg.ExpectedDispatchFingerprint == "" || reservationIdentityDigest(cfg) != pin || anchor.ReservationBindingSHA256 != pin || a.ReservationBindingSHA256 != pin || !validDigest(anchor.InvocationBindingSHA256) || out.BindingDigest != anchor.InvocationBindingSHA256 || out.InvocationID != cfg.InvocationID || r.OrganizationID != cfg.OrganizationID || r.MemberID != cfg.MemberID || r.InvocationID != cfg.InvocationID || r.ReservationID != cfg.ReservationID || r.DispatchFingerprint != cfg.ExpectedDispatchFingerprint || r.QuantityUnit != orgresource.AIPointQuantityUnitV1 || r.Quantity != 15445282 || r.State != "reserved" || r.SettledAt != nil || r.SettlementOperationID != nil || out.Settled || out.SettledMicropoints != nil {
		return errors.New("original invocation or reservation inspection mismatch")
	}
	return nil
}
