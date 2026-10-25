package proposal

import "errors"

var (
	ErrInputInvalid          = errors.New("enrichment input invalid")
	ErrEvidenceInsufficient  = errors.New("enrichment evidence insufficient")
	ErrCapabilityUnsupported = errors.New("enrichment capability unsupported")
	ErrPolicyRejected        = errors.New("enrichment policy rejected")
	ErrOutputValidation      = errors.New("enrichment output validation failed")
)
