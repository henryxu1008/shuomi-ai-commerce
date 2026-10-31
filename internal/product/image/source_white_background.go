package image

import "context"

const SourceWhiteBackgroundOperation = "render_source_white_background"

type SourceWhiteBackgroundRequest struct {
	Source        Asset
	SourceBytes   []byte `json:"-"`
	Product       ProductContext
	Authorization *UsageQuote
}
type SourceWhiteBackground interface {
	EditSourceWhiteBackground(context.Context, SourceWhiteBackgroundRequest) (Candidate, error)
}
type sourceWhiteBackgroundCapability struct{ backend SourceWhiteBackground }

func NewSourceWhiteBackgroundCapability(backend SourceWhiteBackground) (SourceWhiteBackground, error) {
	if isNilCapability(backend) {
		return nil, ErrInputInvalid
	}
	return &sourceWhiteBackgroundCapability{backend: backend}, nil
}
func (c *sourceWhiteBackgroundCapability) EditSourceWhiteBackground(ctx context.Context, request SourceWhiteBackgroundRequest) (Candidate, error) {
	if err := contextError(ctx); err != nil {
		return Candidate{}, err
	}
	source, product, err := validatedInput(request.Source, request.Product)
	if err != nil || len(request.SourceBytes) == 0 || len(request.SourceBytes) > MaxInlineArtifactBytes {
		return Candidate{}, ErrInputInvalid
	}
	authorization, err := cloneUsageAuthorization(request.Authorization, SourceWhiteBackgroundOperation)
	if err != nil {
		return Candidate{}, err
	}
	cloned := SourceWhiteBackgroundRequest{Source: source, SourceBytes: append([]byte(nil), request.SourceBytes...), Product: product, Authorization: authorization}
	if err := contextError(ctx); err != nil {
		return Candidate{}, err
	}
	candidate, err := c.backend.EditSourceWhiteBackground(ctx, cloned)
	if contextErr := contextError(ctx); contextErr != nil {
		return Candidate{}, contextErr
	}
	if err != nil {
		return Candidate{}, capabilityError(err)
	}
	return validateCandidate(candidate, cloned.Source, RoleWhiteBackground, SourceWhiteBackgroundOperation, forbiddenArtifactURLs(cloned.Source))
}
