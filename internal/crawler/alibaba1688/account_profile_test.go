package alibaba1688

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccountProfileContainsNoPasswordField(t *testing.T) {
	_, exists := reflect.TypeOf(AccountProfile{}).FieldByName("Password")
	assert.False(t, exists)
}

func TestAccountProfileErrorCodesRemainStable(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		code string
	}{
		{"nil", nil, ""},
		{"unclassified", errors.New("unavailable"), ""},
		{"cancellation", context.Canceled, ""},
		{"unavailable", newAccountProfileError(AccountProfileUnavailable, "unavailable"), "source_account_unavailable"},
		{"wrapped disabled", fmt.Errorf("resolve: %w", newAccountProfileError(AccountProfileDisabled, "disabled")), "source_account_disabled"},
		{"source access unavailable", newSourceAccessError(AccountProfileUnavailable), "source_account_unavailable"},
		{"source access disabled", newSourceAccessError(AccountProfileDisabled), "source_account_disabled"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, AccountProfileErrorCode(tt.err))
		})
	}
}

func TestAccountProfileResolutionRejectsWrongScopeBeforeAccountBrowser(t *testing.T) {
	for _, profile := range []AccountProfile{
		{ID: 3001, TenantID: 102, ProfileDir: "foreign"},
		{ID: 3002, TenantID: 101, ProfileDir: "other-account"},
		{ID: 3001, TenantID: 101},
	} {
		resolver := &fakeAccountProfileResolver{profile: profile}
		processor := &fakeAlibaba1688TaskProcessor{globalErr: NewPublicAccessError(PublicAccessFailureChallenge, errors.New("challenge"))}
		service := newTestAlibaba1688Service(processor, resolver)

		err := (&Crawler1688Processor{service: service}).ProcessTask(context.Background(), crawler1688WorkerJob(t, 101, 3001))

		assert.Equal(t, AccountProfileUnavailable, AccountProfileErrorCode(err))
		assert.True(t, resolver.called)
		assert.Nil(t, processor.profile)
	}
}
