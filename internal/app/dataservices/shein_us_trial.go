package dataservicesapp

import (
	"database/sql"

	jobstore "task-processor/internal/integration/persistence/product/dataacquisition"
	"task-processor/internal/product/dataacquisition"
)

// NewSHEINUSTrial assembles the isolated one-page trial. Serving assemblies do
// not register this constructor, a sender, or the trial schema.
func NewSHEINUSTrial(db *sql.DB, authority dataacquisition.SHEINUSTrialAuthority, sender dataacquisition.SHEINUSTrialSender, lookup dataacquisition.SHEINUSTrialLookup) (*dataacquisition.SHEINUSTrial, error) {
	store, err := jobstore.NewSHEINUSTrialStore(db)
	if err != nil {
		return nil, err
	}
	return dataacquisition.NewSHEINUSTrial(store, authority, sender, lookup)
}
