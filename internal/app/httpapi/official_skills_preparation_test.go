package httpapi

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type failingOfficialPreparer struct{ called bool }

func (p *failingOfficialPreparer) PrepareOfficial(context.Context) error {
	p.called = true
	return errors.New("sensitive details")
}

type blockedOfficialPreparer struct{}

func (blockedOfficialPreparer) PrepareOfficial(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestOfficialPreparationHonorsStartupCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	logger := logrus.New()
	logger.SetOutput(&bytes.Buffer{})
	started := time.Now()
	prepareOfficialSkills(ctx, blockedOfficialPreparer{}, logger)
	require.Less(t, time.Since(started), time.Second)
}

func TestOfficialPreparationFailureDoesNotInterruptStartup(t *testing.T) {
	var logs bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logs)
	p := &failingOfficialPreparer{}
	prepareOfficialSkills(context.Background(), p, logger)
	require.True(t, p.called)
	require.Contains(t, logs.String(), "official skill preparation failed")
	require.NotContains(t, logs.String(), "sensitive details")
}
