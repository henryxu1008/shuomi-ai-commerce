package httpapi

import (
	profileStore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/accountprofile"
	kernelmodule "github.com/henryxu1008/shuomi-ai-commerce/internal/kernel/module"
	"gorm.io/gorm"
)

func buildAccountProfileModule(db *gorm.DB) (kernelmodule.Module, error) {
	repository, err := profileStore.New(db)
	if err != nil {
		return nil, err
	}
	return accountProfileModule{repository: repository}, nil
}
