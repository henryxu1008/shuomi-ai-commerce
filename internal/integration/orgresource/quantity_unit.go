package orgresourceadapter

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"strconv"
	"task-processor/internal/ledger/orgresource"
)

func resourceQuantityUnit(kind string) string {
	if kind == string(orgresource.ResourceAIPoint) {
		return orgresource.AIPointQuantityUnitV1
	}
	return ""
}
func validResourceQuantityUnit(kind, unit string) error {
	if unit != resourceQuantityUnit(kind) {
		return orgresource.ErrAIPointUnitVersion
	}
	return nil
}

// VerifyQuantitySchema only reads schema metadata. Existing unitless or partial
// ledgers cannot be upgraded, relabelled or interpreted as micro-points.
func VerifyQuantitySchema(ctx context.Context, db *gorm.DB) error {
	if db == nil || ctx == nil {
		return orgresource.ErrAIPointUnitVersion
	}
	for _, model := range resourceSchemaModels() {
		statement := &gorm.Statement{DB: db}
		if err := statement.Parse(model); err != nil {
			return err
		}
		columns, err := db.WithContext(ctx).Migrator().ColumnTypes(model)
		if err != nil {
			return fmt.Errorf("%w: %s", orgresource.ErrAIPointUnitVersion, statement.Schema.Table)
		}
		if len(columns) != len(statement.Schema.DBNames) {
			return orgresource.ErrAIPointUnitVersion
		}
		names := map[string]bool{}
		for _, column := range columns {
			names[column.Name()] = true
		}
		for _, name := range statement.Schema.DBNames {
			if !names[name] {
				return orgresource.ErrAIPointUnitVersion
			}
		}
		for name := range statement.Schema.ParseCheckConstraints() {
			if !db.WithContext(ctx).Migrator().HasConstraint(model, name) {
				return orgresource.ErrAIPointUnitVersion
			}
		}
	}
	return nil
}

func (r *organizationResourceBucketRow) AfterFind(*gorm.DB) error {
	return validResourceQuantityUnit(r.ResourceType, r.QuantityUnit)
}
func (r *organizationResourceBucketRow) BeforeCreate(*gorm.DB) error {
	return validResourceQuantityUnit(r.ResourceType, r.QuantityUnit)
}

func (r *organizationResourceDebtRow) AfterFind(*gorm.DB) error {
	return validResourceQuantityUnit(r.ResourceType, r.QuantityUnit)
}
func (r *organizationResourceDebtRow) BeforeCreate(*gorm.DB) error {
	return validResourceQuantityUnit(r.ResourceType, r.QuantityUnit)
}

func (r *organizationResourceEventRow) AfterFind(*gorm.DB) error {
	return validResourceQuantityUnit(r.ResourceType, r.QuantityUnit)
}
func (r *organizationResourceEventRow) BeforeCreate(*gorm.DB) error {
	return validResourceQuantityUnit(r.ResourceType, r.QuantityUnit)
}

func (r *organizationResourceReservationRow) AfterFind(*gorm.DB) error {
	return validResourceQuantityUnit(r.ResourceType, r.QuantityUnit)
}
func (r *organizationResourceReservationRow) BeforeCreate(*gorm.DB) error {
	return validResourceQuantityUnit(r.ResourceType, r.QuantityUnit)
}

func (r *organizationResourceSourceClaimRow) AfterFind(*gorm.DB) error {
	return validResourceQuantityUnit(r.ResourceType, r.QuantityUnit)
}
func (r *organizationResourceSourceClaimRow) BeforeCreate(*gorm.DB) error {
	return validResourceQuantityUnit(r.ResourceType, r.QuantityUnit)
}

func (r *memberAIPointLimitRow) AfterFind(*gorm.DB) error {
	if r.QuantityUnit != orgresource.AIPointQuantityUnitV1 {
		return orgresource.ErrAIPointUnitVersion
	}
	return nil
}

func (r *memberAIPointLimitRow) BeforeCreate(*gorm.DB) error {
	if r.QuantityUnit != orgresource.AIPointQuantityUnitV1 {
		return orgresource.ErrAIPointUnitVersion
	}
	return nil
}

func (r *memberAIPointMonthRow) AfterFind(*gorm.DB) error {
	if r.QuantityUnit != orgresource.AIPointQuantityUnitV1 {
		return orgresource.ErrAIPointUnitVersion
	}
	return nil
}

func (r *memberAIPointMonthRow) BeforeCreate(*gorm.DB) error {
	if r.QuantityUnit != orgresource.AIPointQuantityUnitV1 {
		return orgresource.ErrAIPointUnitVersion
	}
	return nil
}

func formatResourceQuantity(kind string, value int64) string {
	if kind == string(orgresource.ResourceAIPoint) {
		return orgresource.FormatAIPointMicropoints(value)
	}
	return strconv.FormatInt(value, 10)
}
