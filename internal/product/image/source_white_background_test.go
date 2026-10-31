package image

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type sourceWhiteBackgroundFunc func(context.Context, SourceWhiteBackgroundRequest) (Candidate, error)

func (f sourceWhiteBackgroundFunc) EditSourceWhiteBackground(ctx context.Context, r SourceWhiteBackgroundRequest) (Candidate, error) {
	return f(ctx, r)
}
func sourceWhiteRequest() SourceWhiteBackgroundRequest {
	return SourceWhiteBackgroundRequest{Source: validSourceAsset(), SourceBytes: []byte{1, 2, 3}, Product: validProductContext()}
}
func TestSourceWhiteBackgroundValidationAndOwnership(t *testing.T) {
	original := sourceWhiteRequest()
	q := validUsageAuthorization(SourceWhiteBackgroundOperation)
	original.Authorization = &q
	backend := sourceWhiteBackgroundFunc(func(_ context.Context, r SourceWhiteBackgroundRequest) (Candidate, error) {
		require.Equal(t, []byte{1, 2, 3}, r.SourceBytes)
		r.SourceBytes[0] = 9
		r.Product.Attributes["material"] = "changed"
		r.Authorization.Model = "changed"
		return Candidate{Asset: validInlineGeneratedAsset(RoleWhiteBackground, SourceWhiteBackgroundOperation, []byte{4}), Metadata: GenerationMetadata{Capability: SourceWhiteBackgroundOperation, ModelFamily: "model", InvocationID: "request", PromptReference: "prompt", PromptVersion: "v1"}}, nil
	})
	c, err := NewSourceWhiteBackgroundCapability(backend)
	require.NoError(t, err)
	got, err := c.EditSourceWhiteBackground(context.Background(), original)
	require.NoError(t, err)
	require.Equal(t, []byte{4}, got.Asset.Bytes)
	require.Equal(t, []byte{1, 2, 3}, original.SourceBytes)
	require.Equal(t, "steel", original.Product.Attributes["material"])
	require.Equal(t, q.Model, original.Authorization.Model)
}
func TestSourceWhiteBackgroundRejectsInvalidSourceWithoutDispatch(t *testing.T) {
	for _, mutate := range []func(*SourceWhiteBackgroundRequest){
		func(r *SourceWhiteBackgroundRequest) { r.SourceBytes = nil }, func(r *SourceWhiteBackgroundRequest) { r.SourceBytes = make([]byte, MaxInlineArtifactBytes+1) },
		func(r *SourceWhiteBackgroundRequest) { r.Source.Bytes = []byte{1} }, func(r *SourceWhiteBackgroundRequest) { r.Source.URL = "https://source.example:80:90/a.png" },
		func(r *SourceWhiteBackgroundRequest) { r.Source.URL = "https://source.example/a.png?first=1;second=2" }, func(r *SourceWhiteBackgroundRequest) { r.Product.Attributes = map[string]string{"bad": ""} },
	} {
		calls := 0
		c, err := NewSourceWhiteBackgroundCapability(sourceWhiteBackgroundFunc(func(context.Context, SourceWhiteBackgroundRequest) (Candidate, error) {
			calls++
			return Candidate{}, nil
		}))
		require.NoError(t, err)
		r := sourceWhiteRequest()
		mutate(&r)
		_, err = c.EditSourceWhiteBackground(context.Background(), r)
		require.ErrorIs(t, err, ErrInputInvalid)
		require.Zero(t, calls)
	}
}
func TestSourceWhiteBackgroundRejectsCanonicalAliasesAndInvalidMaterials(t *testing.T) {
	for _, url := range []string{"https://source.example:0443/a.png", "https://source.example:443/a.png", "https://[2001:db8::1]:00443/a.png", "https://cdn.example/a.png?first=%zz"} {
		r := sourceWhiteRequest()
		if url == "https://[2001:db8::1]:00443/a.png" {
			r.Source.URL = "https://[2001:0DB8:0:0:0:0:0:1]:443/a.png"
		}
		c, err := NewSourceWhiteBackgroundCapability(sourceWhiteBackgroundFunc(func(context.Context, SourceWhiteBackgroundRequest) (Candidate, error) {
			return Candidate{Asset: validGeneratedAsset(RoleWhiteBackground, SourceWhiteBackgroundOperation, url)}, nil
		}))
		require.NoError(t, err)
		_, err = c.EditSourceWhiteBackground(context.Background(), r)
		require.ErrorIs(t, err, ErrOutputValidation)
	}
	for _, mutate := range []func(*Asset){func(a *Asset) { a.Bytes = nil }, func(a *Asset) { a.URL = "https://generated.example/a.png" }, func(a *Asset) { a.Bytes = make([]byte, MaxInlineArtifactBytes+1) }, func(a *Asset) { a.Operations = []string{"placeholder"} }, func(a *Asset) { a.SourceAssetID = "other" }, func(a *Asset) { a.Width = 0 }} {
		c, _ := NewSourceWhiteBackgroundCapability(sourceWhiteBackgroundFunc(func(context.Context, SourceWhiteBackgroundRequest) (Candidate, error) {
			a := validInlineGeneratedAsset(RoleWhiteBackground, SourceWhiteBackgroundOperation, []byte{1})
			mutate(&a)
			return Candidate{Asset: a}, nil
		}))
		_, err := c.EditSourceWhiteBackground(context.Background(), sourceWhiteRequest())
		require.ErrorIs(t, err, ErrOutputValidation)
	}
}
func TestSourceWhiteBackgroundNilStableErrorsAndCancellation(t *testing.T) {
	var typedNil sourceWhiteBackgroundFunc
	_, err := NewSourceWhiteBackgroundCapability(typedNil)
	require.ErrorIs(t, err, ErrInputInvalid)
	_, err = NewSourceWhiteBackgroundCapability(nil)
	require.ErrorIs(t, err, ErrInputInvalid)
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	c, _ := NewSourceWhiteBackgroundCapability(sourceWhiteBackgroundFunc(func(context.Context, SourceWhiteBackgroundRequest) (Candidate, error) {
		calls++
		cancel()
		return Candidate{}, errors.New("private provider error")
	}))
	_, err = c.EditSourceWhiteBackground(ctx, sourceWhiteRequest())
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, calls)
	_, err = c.EditSourceWhiteBackground(ctx, sourceWhiteRequest())
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, calls)
	c, _ = NewSourceWhiteBackgroundCapability(sourceWhiteBackgroundFunc(func(context.Context, SourceWhiteBackgroundRequest) (Candidate, error) {
		return Candidate{}, errors.New("private provider error")
	}))
	_, err = c.EditSourceWhiteBackground(context.Background(), sourceWhiteRequest())
	require.ErrorIs(t, err, ErrExternalCapabilityUnavailable)
	require.NotContains(t, err.Error(), "private")
}
