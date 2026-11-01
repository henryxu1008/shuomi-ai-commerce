package httpapi

import (
	"errors"
	module "task-processor/internal/app/httpapi/fulfillment"
	"task-processor/internal/httproute"
)

func WithFulfillment(d module.Dependencies) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.fulfillments++; o.fulfillment = &d }
}
func validateFulfillmentDescriptor(d httproute.Descriptor) error {
	for _, expected := range (*module.Module)(nil).Routes() {
		if d.Path == expected.Path && d.Method == expected.Method && d.Module == expected.Module && d.Permission == expected.Permission && d.AuthPolicy == expected.AuthPolicy && d.OrganizationAccessPolicy == expected.OrganizationAccessPolicy && d.OrganizationTargetResolver == nil && d.RequestTimeout == expected.RequestTimeout && d.RejectUnreadRequestBody == expected.RejectUnreadRequestBody && d.Handler != nil {
			return nil
		}
	}
	return errors.New("fulfillment route loses current scoped authority")
}
