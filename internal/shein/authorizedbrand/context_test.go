package authorizedbrand

import (
	"context"
	"testing"
)

func TestWithResolvedFromContext_PreservesEnabledValue(t *testing.T) {
	resolved := &Resolved{
		Enabled: true,
		Code:    "2fd1n",
		Name:    "Logitech罗技",
		NameEn:  "Logitech",
	}

	ctx := WithResolved(context.Background(), resolved)
	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("FromContext() ok = false, want true")
	}
	if got == nil {
		t.Fatal("FromContext() returned nil")
	}
	if *got != *resolved {
		t.Fatalf("FromContext() = %+v, want %+v", got, resolved)
	}
}

func TestWithResolvedFromContext_DropsDisabledValue(t *testing.T) {
	ctx := WithResolved(context.Background(), &Resolved{Code: "2fd1n"})
	got, ok := FromContext(ctx)
	if ok || got != nil {
		t.Fatalf("FromContext() = (%+v, %v), want (nil, false)", got, ok)
	}
}
