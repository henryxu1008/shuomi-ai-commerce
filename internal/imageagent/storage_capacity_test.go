package imageagent

import (
	"context"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
)

type terminalCapacityProbe struct {
	storagecapacity.Writer
	closes, aborts int
	proof          storagecapacity.CloseProof
	failure        error
}

func (p *terminalCapacityProbe) FindWrite(context.Context, string, string, string) (storagecapacity.Reservation, error) {
	return storagecapacity.Reservation{ID: "original", State: "DISPATCHED"}, nil
}
func (p *terminalCapacityProbe) CloseWrite(_ context.Context, _ string, _ string, proof storagecapacity.CloseProof) error {
	p.closes++
	p.proof = proof
	return p.failure
}
func (p *terminalCapacityProbe) AbortUnstarted(context.Context, string, string) error {
	p.aborts++
	return nil
}
func TestStorageClosureRequiresExactInvalidCommittedProof(t *testing.T) {
	for _, mode := range []string{"invalid_committed", "invalid_unsettled", "valid_output", "unknown", "corrupt_settlement", "close_retry"} {
		t.Run(mode, func(t *testing.T) {
			f, err := NewGenerationFact(generationTestIntent())
			require.NoError(t, err)
			f, err = f.BindReservation(generationTestReceipt(f))
			require.NoError(t, err)
			f, _, err = f.BeginDispatch()
			require.NoError(t, err)
			if mode == "unknown" {
				f, err = f.MarkUnknown()
				require.NoError(t, err)
			} else {
				success := GenerationSuccess{ResponseID: "result", ResultDigest: strings.Repeat("c", 64), ResultUnavailable: "invalid_result"}
				if mode == "valid_output" {
					success.ResultUnavailable = ""
					success.ResultURL = "https://images.example.test/output.png"
				}
				f, err = f.RecordSuccess(success)
				require.NoError(t, err)
				if mode != "invalid_unsettled" {
					f, err = f.BindSettlement(GenerationSettlementReceipt{IntentID: f.IntentID, Fingerprint: f.Fingerprint, OperationID: "image-finalize:" + f.IntentID, ReservationID: f.Reservation.ReservationID, Points: f.Intent.Points, ProofDigest: f.TerminalProofDigest(), State: "committed"})
					require.NoError(t, err)
				}
				if mode == "corrupt_settlement" {
					f.Settlement.ProofDigest = strings.Repeat("a", 64)
				}
			}
			probe := &terminalCapacityProbe{}
			if mode == "close_retry" {
				probe.failure = storagecapacity.ErrUnknown
			}
			err = closeGenerationCapacity(context.Background(), probe, f)
			switch mode {
			case "invalid_committed", "close_retry":
				require.Equal(t, 1, probe.closes)
				require.Equal(t, "invalid-generation:"+f.TerminalProofDigest(), probe.proof.TerminalReceipt)
				if mode == "close_retry" {
					require.ErrorIs(t, err, storagecapacity.ErrUnknown)
				} else {
					require.NoError(t, err)
				}
			case "corrupt_settlement":
				require.ErrorIs(t, err, ErrCorruptPersistedEffect)
				require.Zero(t, probe.closes)
			default:
				require.NoError(t, err)
				require.Zero(t, probe.closes)
			}
			require.Zero(t, probe.aborts, "paid invalid result must not use the no-generation abort path")
		})
	}
}
