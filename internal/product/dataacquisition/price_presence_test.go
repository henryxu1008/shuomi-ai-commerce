package dataacquisition

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
)

func TestFrozenPriceRequiresExplicitCanonicalAmount(t *testing.T) {
	for _, amount := range []string{``, `,"unitPriceFen":null`, `,"unitPriceFen":"01"`, `,"unitPriceFen":0`, `,"unitPriceFen":"-1"`, `,"unitPriceFen":"0","unitPriceFen":"5"`} {
		t.Run(amount, func(t *testing.T) {
			var p PriceSnapshot
			err := json.Unmarshal([]byte(`{"offerId":"offer","pricingVersion":"v1","currency":"CNY"`+amount+`}`), &p)
			if err == nil {
				err = p.Validate()
			}
			require.Error(t, err, "absent, null or malformed amount is not an explicit zero")
		})
	}
	for _, amount := range []string{"0", "7"} {
		var p PriceSnapshot
		require.NoError(t, json.Unmarshal([]byte(`{"offerId":"offer","pricingVersion":"v1","currency":"CNY","unitPriceFen":"`+amount+`"}`), &p))
		require.NoError(t, p.Validate())
	}
}

type missingPriceCharges struct{ calls int }

func (c *missingPriceCharges) Lookup(context.Context, orgresource.ConsumerChargeIdentity) (orgresource.ConsumerChargeReceipt, error) {
	c.calls++
	return orgresource.ConsumerChargeReceipt{}, orgresource.ErrReservationNotFound
}
func (c *missingPriceCharges) Reserve(context.Context, orgresource.ConsumerChargeIdentity) (orgresource.ConsumerChargeReceipt, error) {
	c.calls++
	return orgresource.ConsumerChargeReceipt{}, ErrUnknown
}
func (c *missingPriceCharges) Reconcile(context.Context, orgresource.ConsumerChargeIdentity) (orgresource.ConsumerChargeReceipt, error) {
	c.calls++
	return orgresource.ConsumerChargeReceipt{}, ErrUnknown
}

func TestMissingPriceTerminalCannotReachResourceFastPath(t *testing.T) {
	scope := collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	job := Job{ID: uuid.NewString(), Scope: scope, State: "FAILED", Discovered: true, Deadline: time.Now().Add(-time.Hour)}
	item := Item{ID: uuid.NewString(), State: "FAILED", ReservationID: uuid.NewString()}
	repo := &resultRepository{job: job, item: item}
	charges := &missingPriceCharges{}
	svc := &Service{repo: repo, charges: charges, now: time.Now}
	// A stale or forged caller-supplied price cannot authorize the canonical row.
	forged := job
	forged.Price = PriceSnapshot{OfferID: "forged", Version: "v1", Currency: "CNY", UnitPriceFen: 0}
	require.Error(t, svc.ProcessItem(context.Background(), forged, item, "deadline"))
	require.Zero(t, charges.calls)
	require.Error(t, svc.Run(context.Background(), scope, job.ID))
	require.Zero(t, charges.calls)
}
