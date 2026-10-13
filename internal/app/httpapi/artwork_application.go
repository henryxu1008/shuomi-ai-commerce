package httpapi

import (
	"gorm.io/gorm"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/imageagent"
	imagestore "task-processor/internal/imageagent/store"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
)

func buildArtworkOriginGuard(productDB, imageDB, configurationDB *gorm.DB) (*productsourcing.ArtworkOriginGuard, error) {
	snapshots, err := catalogstore.NewBoundedSnapshotReader(productDB, 2<<20)
	if err != nil {
		return nil, err
	}
	configuration, err := configstore.New(configurationDB)
	if err != nil {
		return nil, err
	}
	images := imagestore.NewOrganizationRepository(imageDB)
	facts, ok := images.(imageagent.GenerationFactRepository)
	if !ok {
		return nil, imageagent.ErrCommandBlocked
	}
	return &productsourcing.ArtworkOriginGuard{Snapshots: snapshots, Images: images, Facts: facts, Configuration: configuration}, nil
}
