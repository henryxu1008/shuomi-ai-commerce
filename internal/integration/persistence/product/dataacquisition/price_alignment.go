package dataacquisitionpersistence

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

// AlignPriceContractTx is only called by the explicit Product owner while the
// two native tables are locked. It never changes a business row or a grant.
func AlignPriceContractTx(ctx context.Context, tx *gorm.DB) error {
	current, err := PriceContractState(ctx, tx)
	if err != nil {
		return err
	}
	if current {
		return VerifyMissingPriceFacts(ctx, tx)
	}
	if err = VerifyPrePriceFacts(ctx, tx); err != nil {
		return err
	}
	var contract map[string]json.RawMessage
	if err = json.Unmarshal(nativePriceContract, &contract); err != nil {
		return err
	}
	var constraints [][]any
	if err = json.Unmarshal(contract["constraints"], &constraints); err != nil {
		return err
	}
	definition := ""
	for _, c := range constraints {
		if c[0] == "data_acquisition_jobs" && c[1] == "data_acquisition_price_snapshot" {
			definition, _ = c[7].(string)
		}
	}
	if definition == "" {
		return fmt.Errorf("native price CHECK definition unavailable")
	}
	if err = tx.WithContext(ctx).Exec(`ALTER TABLE public.data_acquisition_jobs ADD COLUMN price_snapshot jsonb; ALTER TABLE public.data_acquisition_jobs ADD CONSTRAINT data_acquisition_price_snapshot ` + definition).Error; err != nil {
		return err
	}
	current, err = PriceContractState(ctx, tx)
	if err != nil {
		return err
	}
	if !current {
		return fmt.Errorf("native price contract postcondition unavailable")
	}
	return VerifyMissingPriceFacts(ctx, tx)
}
