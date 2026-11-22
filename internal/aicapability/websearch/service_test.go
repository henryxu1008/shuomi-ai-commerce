package websearch

import (
	"context"
	"strings"
	"testing"
	"time"

	credential "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/channelcredential"
	"github.com/stretchr/testify/require"
)

type clockSkewRepo struct {
	Repository
	claimCalls int
}

func (*clockSkewRepo) Selection(context.Context, string) (Selection, error) {
	return Selection{Revision: 1, Source: Owned}, nil
}
func (*clockSkewRepo) Find(context.Context, Scope, string, string) (Receipt, error) {
	return Receipt{}, ErrNotFound
}
func (*clockSkewRepo) Hold(context.Context, Attempt) (Receipt, bool, time.Time, error) {
	at := time.Now().Add(time.Hour)
	time.Sleep(5 * time.Millisecond)
	return Receipt{StartedAt: at, Status: Held}, true, at.Add(time.Millisecond), nil
}
func (r *clockSkewRepo) Claim(context.Context, Attempt) (bool, error) {
	r.claimCalls++
	return true, nil
}
func (*clockSkewRepo) Finish(_ context.Context, _ Attempt, state string, usage *int) (Receipt, error) {
	return Receipt{Status: state, Usage: usage}, nil
}

type skewConnection struct{}

func (skewConnection) Metadata(_ context.Context, org, id string) (credential.Metadata, error) {
	return credential.Metadata{OrganizationID: org, ConnectionID: id, Provider: "brave", Channel: Connection, BaseURL: BaseURL, KeyPresent: true, Enabled: true, Reference: "reference"}, nil
}

func TestDispatchExpiryUsesDatabaseDurationAndLocalMonotonicClock(t *testing.T) {
	repo := &clockSkewRepo{}
	s := Service{Repo: repo, Metadata: skewConnection{}, FingerprintKey: []byte(strings.Repeat("s", 32))}
	ctx := context.Background()
	_, option, err := s.Current(ctx, "A")
	require.NoError(t, err)
	response, err := s.Submit(ctx, Scope{Organization: "A"}, "key", "public rule", option.Preview, func(ctx context.Context) error { return ctx.Err() })
	require.NoError(t, err)
	require.Equal(t, NotSent, response.Receipt.Status)
	require.Equal(t, 0, repo.claimCalls, "expired database window cannot authorize the application to send")
}
