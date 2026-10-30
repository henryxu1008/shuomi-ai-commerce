package httpapi

import (
	"context"

	"gorm.io/gorm"

	"task-processor/internal/authz"
	sourceaccountstore "task-processor/internal/integration/persistence/sourceaccountregistry"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/sourceaccountregistry"
	sourceaccounthttpapi "task-processor/internal/sourceaccountregistry/httpapi"
)

func buildSourceAccountModule(ctx context.Context, db *gorm.DB, authorizer *authz.ListingKitAuthorizer) (kernelmodule.Module, error) {
	if db == nil || authorizer == nil {
		return nil, sourceaccountregistry.ErrUnavailable
	}
	repository, err := sourceaccountstore.NewRepository(ctx, db)
	if err != nil {
		return nil, err
	}
	service, err := sourceaccountregistry.NewService(repository, authorizer)
	if err != nil {
		return nil, err
	}
	handler, err := sourceaccounthttpapi.NewHandler(service)
	if err != nil {
		return nil, err
	}
	return sourceaccounthttpapi.NewModule(handler), nil
}
