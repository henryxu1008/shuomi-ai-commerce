package imageagent

import (
	"context"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
)

// GenerationProviderTask is worker-only evidence for the original accepted
// request. The connection and credential version are frozen in Intent.
type GenerationProviderTask struct {
	ID            string
	AcceptedAt    time.Time
	FailureStatus string `json:",omitempty"`
	FailureDigest string `json:",omitempty"`
}

type GenerationProviderTaskRepository interface {
	RecordGenerationProviderTask(context.Context, GenerationIntent, string, time.Time) (GenerationFact, error)
	RecordGenerationProviderFailure(context.Context, GenerationIntent, string, string, string) (GenerationFact, error)
	RecordInvalidGenerationProviderOutput(context.Context, GenerationIntent, string) (GenerationFact, error)
}

func (f GenerationFact) HasSuccessProof() bool {
	return f.State == GenerationSucceeded && (f.Intent.UsesProviderUsage() || f.Settlement.State == "committed") && f.Validate() == nil
}

func (i GenerationIntent) UsesProviderUsage() bool {
	return i.BillingMode == aicapability.ModelUsageOnly
}

func (i GenerationIntent) validProviderUsage() bool {
	return i.Provider == "grsai" && i.Model == aicapability.GRSaiProductImageModel &&
		i.Protocol == aicapability.GRSaiProductImageProtocol && i.Quality == "auto" &&
		i.InputProtocol == ImageSetSchema && i.PromptVersion == ImageSetSchema &&
		generationDigest(i.InputDigest) && canonicalImageValue(i.ConnectionID) &&
		i.Billing == (ImageBilling{}) && i.PriceVersion == "" && i.Points == 0 &&
		i.LimitVersion == 0 && i.MonthStart.IsZero()
}

func (f GenerationFact) RecordProviderTask(id string, acceptedAt time.Time) (GenerationFact, error) {
	if err := f.Validate(); err != nil {
		return f, err
	}
	if !f.Intent.UsesProviderUsage() || !aicapability.ValidGRSaiProductImageTaskID(id) || acceptedAt.IsZero() {
		return f, ErrValidation
	}
	if f.ProviderTask.ID != "" {
		if f.ProviderTask.ID != id {
			return f, ErrRevisionConflict
		}
		return f, nil
	}
	if f.State != GenerationDispatchStarted && f.State != GenerationUnknown {
		return f, ErrRevisionConflict
	}
	f.ProviderTask = GenerationProviderTask{ID: id, AcceptedAt: acceptedAt.UTC()}
	return f, nil
}

func (f GenerationFact) RecordProviderFailure(id, status, digest string) (GenerationFact, error) {
	if err := f.Validate(); err != nil {
		return f, err
	}
	if !f.Intent.UsesProviderUsage() || id == "" || f.ProviderTask.ID != id ||
		!validProviderFailure(status, digest) {
		return f, ErrValidation
	}
	if f.State == GenerationSupplierFailed && f.ProviderTask.FailureStatus == status && f.ProviderTask.FailureDigest == digest {
		return f, nil
	}
	if f.State != GenerationDispatchStarted && f.State != GenerationUnknown {
		return f, ErrRevisionConflict
	}
	f.State = GenerationSupplierFailed
	f.ProviderTask.FailureStatus = status
	f.ProviderTask.FailureDigest = digest
	return f, nil
}

func (f GenerationFact) recordProviderSuccess(success GenerationSuccess) (GenerationFact, error) {
	if !f.validProviderSuccess(success) {
		return f, ErrValidation
	}
	if f.State == GenerationSucceeded && f.Success == success {
		return f, nil
	}
	if f.State != GenerationDispatchStarted && f.State != GenerationUnknown {
		return f, ErrRevisionConflict
	}
	f.State = GenerationSucceeded
	f.Success = success
	return f, nil
}

func (f GenerationFact) validProviderSuccess(success GenerationSuccess) bool {
	return f.ProviderTask.ID != "" && success.ResponseID == f.ProviderTask.ID && success.ProviderImageCount == 1 &&
		success.BillableUsage == (aicapability.BillableModelUsage{}) && !success.UsageKnown &&
		success.InputTokens == 0 && success.OutputTokens == 0 && success.TotalTokens == 0 && validGenerationSuccess(success)
}

func validProviderFailure(status, digest string) bool {
	return (status == "failed" || status == "violation") && generationDigest(digest)
}

func (f GenerationFact) validateProviderUsage() error {
	if f.Reservation != (GenerationReservationReceipt{}) || f.Settlement != (GenerationSettlementReceipt{}) ||
		f.ObservedOverage != (GenerationSuccess{}) || f.ProviderOutputInvalid && f.State != GenerationSucceeded {
		return ErrValidation
	}
	task := f.ProviderTask
	if task != (GenerationProviderTask{}) &&
		(!aicapability.ValidGRSaiProductImageTaskID(task.ID) || task.AcceptedAt.IsZero() || task.AcceptedAt.Location() != time.UTC) {
		return ErrValidation
	}
	if f.State != GenerationSupplierFailed && (task.FailureStatus != "" || task.FailureDigest != "") {
		return ErrValidation
	}
	switch f.State {
	case GenerationPrepared, GenerationNoEffect:
		if task == (GenerationProviderTask{}) && f.Success == (GenerationSuccess{}) {
			return nil
		}
	case GenerationDispatchStarted, GenerationUnknown:
		if f.Success == (GenerationSuccess{}) {
			return nil
		}
	case GenerationSucceeded:
		if f.validProviderSuccess(f.Success) {
			return nil
		}
	case GenerationSupplierFailed:
		if task.ID != "" && validProviderFailure(task.FailureStatus, task.FailureDigest) && f.Success == (GenerationSuccess{}) {
			return nil
		}
	}
	return ErrValidation
}

func (f GenerationFact) providerUsageProofDigest() string {
	if f.State != GenerationSucceeded && f.State != GenerationNoEffect && f.State != GenerationSupplierFailed {
		return ""
	}
	return generationHash(struct {
		Fingerprint string
		State       GenerationState
		Task        GenerationProviderTask
		Success     GenerationSuccess
	}{f.Fingerprint, f.State, f.ProviderTask, f.Success})
}
