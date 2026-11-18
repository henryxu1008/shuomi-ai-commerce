package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// startReferralMaturityLoop starts the due-claim sweep and returns a stop
// function that cancels and waits for the loop before the caller closes its pool.
func startReferralMaturityLoop(server *http.Server, mature func(context.Context, time.Time) error, interval time.Duration, logger *logrus.Logger) func() {
	if server == nil || mature == nil || interval <= 0 {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	server.RegisterOnShutdown(cancel)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer cancel()
		run := func() {
			if err := mature(ctx, time.Now().UTC()); err != nil && logger != nil {
				logger.WithError(err).Warn("referral earnings maturity sweep failed")
			}
		}
		run()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				run()
			case <-ctx.Done():
				return
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}
