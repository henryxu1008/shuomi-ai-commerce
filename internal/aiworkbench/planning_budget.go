package aiworkbench

import (
	"strconv"
	"task-processor/internal/aicapability"
)

// PlanningBudget is a user's accepted bound. The server always computes the
// actual bound from its current route; this object is never a price source.
type PlanningBudget struct {
	ProfileDigest       string `json:"profileDigest"`
	Sequence            uint64 `json:"sequence"`
	MaximumCostMicros   int64  `json:"maximumCostMicros"`
	MaximumPointsMicros string `json:"maximumPointsMicros"`
	Currency            string `json:"currency"`
}

func NewPlanningBudget(profile aicapability.ModelProfile, sequence uint64) (PlanningBudget, error) {
	if sequence == 0 || profile.Validate() != nil {
		return PlanningBudget{}, ErrUnavailable
	}
	digest, err := profile.Digest()
	if err != nil {
		return PlanningBudget{}, ErrUnavailable
	}
	cost, err := profile.MaximumCost()
	if err != nil {
		return PlanningBudget{}, ErrUnavailable
	}
	points, err := profile.OfficialTariff.ReserveTextMicropoints(profile.MaximumPromptTokens, profile.MaximumCompletionTokens)
	if err != nil {
		return PlanningBudget{}, ErrUnavailable
	}
	return PlanningBudget{ProfileDigest: digest, Sequence: sequence, MaximumCostMicros: cost, MaximumPointsMicros: strconv.FormatInt(points, 10), Currency: profile.Currency}, nil
}

func (accepted PlanningBudget) Check(current PlanningBudget) error {
	acceptedPoints, ae := strconv.ParseInt(accepted.MaximumPointsMicros, 10, 64)
	currentPoints, ce := strconv.ParseInt(current.MaximumPointsMicros, 10, 64)
	if ae != nil || ce != nil || strconv.FormatInt(acceptedPoints, 10) != accepted.MaximumPointsMicros || strconv.FormatInt(currentPoints, 10) != current.MaximumPointsMicros || accepted.ProfileDigest == "" || accepted.ProfileDigest != current.ProfileDigest || accepted.Sequence == 0 || accepted.Sequence != current.Sequence || accepted.Currency == "" || accepted.Currency != current.Currency || accepted.MaximumCostMicros < 0 || acceptedPoints < 0 || current.MaximumCostMicros < 0 || currentPoints < 0 || current.MaximumCostMicros > accepted.MaximumCostMicros || currentPoints > acceptedPoints {
		return ErrProposalStale
	}
	return nil
}
