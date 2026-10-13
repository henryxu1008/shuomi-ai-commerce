package storeobservationsapp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	o "task-processor/internal/storecenter/observations"
)

// Unimplemented embedded ports panic if preparation accidentally consumes facts
// or dispatches work. Capability authorization is the only admitted call.
type preparationPorts struct {
	o.Repository
	o.Directory
	o.Starter
	o.Access
	denied bool
}

func (p *preparationPorts) Authorize(context.Context, o.Scope, o.Kind, bool) error {
	if p.denied {
		return o.ErrForbidden
	}
	return nil
}

func TestPreparationCapabilitiesNeverEnableProviderExecution(t *testing.T) {
	p := &preparationPorts{}
	ready := true
	a := &Application{Service: &o.Service{Repository: p, Directory: p, Starter: p, Access: p}, Ready: func() bool { return ready }, PreparationOnly: true}
	s := o.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}
	for _, k := range []o.Kind{o.Products, o.Orders} {
		c, e := a.Capabilities(context.Background(), s, k)
		require.NoError(t, e)
		require.False(t, c.Available)
		require.False(t, c.CanSync)
		b, e := json.Marshal(c)
		require.NoError(t, e)
		var flags map[string]any
		require.NoError(t, json.Unmarshal(b, &flags))
		require.Equal(t, true, flags["runtimeReady"])
		require.Equal(t, false, flags["providerConfigured"])
	}
	ready = false
	c, e := a.Capabilities(context.Background(), s, o.Products)
	require.NoError(t, e)
	b, _ := json.Marshal(c)
	var flags map[string]any
	require.NoError(t, json.Unmarshal(b, &flags))
	require.Equal(t, false, flags["runtimeReady"])
	p.denied = true
	_, e = a.Capabilities(context.Background(), s, o.Products)
	require.ErrorIs(t, e, o.ErrForbidden)
	_, e = (Access{}).Open(context.Background(), s, "store-a", o.Products, true, nil)
	require.ErrorIs(t, e, o.ErrUnavailable)
	p.denied = false
	a.PreparationOnly = false
	ready = true
	c, e = a.Capabilities(context.Background(), s, o.Products)
	require.NoError(t, e)
	require.True(t, c.Available)
	require.True(t, c.CanSync)
	require.True(t, c.ProviderConfigured)
	ready = false
	require.False(t, a.RuntimeReady())
	require.False(t, a.Available())
}
