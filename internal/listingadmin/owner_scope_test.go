package listingadmin

import (
	"context"
	"testing"
)

func TestWithOwnerUserIDSuppliesInternalOwner(t *testing.T) {
	got := requestUserIDFromContext(WithOwnerUserID(context.Background(), "job-sub"))
	if got != "job-sub" {
		t.Fatalf("requestUserIDFromContext(WithOwnerUserID()) = %q; want job-sub", got)
	}
}
