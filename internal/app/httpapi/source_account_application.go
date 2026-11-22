package httpapi

import (
	"context"

	"gorm.io/gorm"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	sourceaccountstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/sourceaccountregistry"
	kernelmodule "github.com/henryxu1008/shuomi-ai-commerce/internal/kernel/module"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/sourceaccountregistry"
	sourceaccounthttpapi "github.com/henryxu1008/shuomi-ai-commerce/internal/sourceaccountregistry/httpapi"
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
