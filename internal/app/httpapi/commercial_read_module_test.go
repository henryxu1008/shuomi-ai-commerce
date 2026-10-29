package httpapi

import (
	"context"
	"net/http"
	"sync/atomic"

	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCommercialReadModuleRequiresDurableConfiguration(t *testing.T) {
	module, err := buildUnifiedCommercialRead(context.Background(), nil, nil, nil)
	require.Error(t, err)
	require.Nil(t, module)
}
func TestCommercialRecoveryRunsImmediatelyAndStopsWithServer(t *testing.T) {
	server := &http.Server{}
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := make(chan struct{}, 2)
	var calls atomic.Int32
	startCommercialRecoveryLoop(parent, server, func(context.Context) error {
		calls.Add(1)
		called <- struct{}{}
		return nil
	}, time.Hour, "current resource purchase", nil)
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("initial recovery did not run")
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatalf("recovery calls after shutdown = %d", calls.Load())
	}
}

func TestCommercialRecoveryStopsWithRuntimeContext(t *testing.T) {
	server := &http.Server{}
	parent, cancel := context.WithCancel(context.Background())
	started := make(chan struct{}, 1)
	stopped := make(chan struct{}, 1)
	startCommercialRecoveryLoop(parent, server, func(ctx context.Context) error {
		started <- struct{}{}
		<-ctx.Done()
		stopped <- struct{}{}
		return ctx.Err()
	}, time.Hour, "current resource purchase", nil)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("initial recovery did not run")
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("recovery did not stop with runtime context")
	}
}
