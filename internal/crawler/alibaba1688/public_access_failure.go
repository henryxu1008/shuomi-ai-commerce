package alibaba1688

type PublicAccessFailureKind string

const (
	PublicAccessFailureBrowser       PublicAccessFailureKind = "browser"
	PublicAccessFailureChallenge     PublicAccessFailureKind = "challenge"
	PublicAccessFailureMissingFields PublicAccessFailureKind = "missing_fields"
	PublicAccessFailureValidation    PublicAccessFailureKind = "validation"
	PublicAccessFailureInvalidURL    PublicAccessFailureKind = "invalid_url"
	PublicAccessFailureTransport     PublicAccessFailureKind = "transport"
)

type PublicAccessError struct {
	Kind PublicAccessFailureKind
	Err  error
}

func NewPublicAccessError(kind PublicAccessFailureKind, err error) error {
	return &PublicAccessError{Kind: kind, Err: err}
}

func (e *PublicAccessError) Error() string {
	if e == nil || e.Err == nil {
		return "public 1688 access failed"
	}
	return e.Err.Error()
}

func (e *PublicAccessError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
