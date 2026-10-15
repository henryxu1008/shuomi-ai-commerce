package httpapi

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	governed "task-processor/internal/integration/aicapability/einomodel"
	kernelmodule "task-processor/internal/kernel/module"
)

func TestContentTitleOnlyGlobalCompositionRejectsFullChatPlannerAndSharedPool(t *testing.T) {
	for _, kind := range []string{"full", "planner", "shared-pool"} {
		t.Run(kind, func(t *testing.T) {
			run, chatDB := &gorm.DB{}, &gorm.DB{}
			chat := AIWorkbenchDependencies{DB: chatDB, ConversationOnly: true}
			switch kind {
			case "full":
				chat.ConversationOnly = false
			case "planner":
				chat.PlanningTextPolicies = map[string]governed.RoutePolicy{"org": {}}
			case "shared-pool":
				chat.DB = run
			}
			built := false
			factories := currentApplicationFactories{
				buildWorkbench: func(*config.Config, *logrus.Logger) (workbenchContextBuildResult, error) {
					built = true
					return workbenchContextBuildResult{}, nil
				},
				buildSourceAccount: func(*gorm.DB, *authz.ListingKitAuthorizer) (kernelmodule.Module, error) {
					built = true
					return nil, nil
				},
				buildCommercial: func(*gorm.DB, *gorm.DB, *authz.ListingKitAuthorizer) (kernelmodule.Module, error) {
					built = true
					return nil, nil
				},
			}
			server, err := buildCurrentApplication(context.Background(), &gorm.DB{}, currentApplicationTestConfig(), logrus.New(), factories, WithProductAgent(ProductAgentDependencies{ContentTitleOnly: true, RunDB: run}), WithProductAcquisition(&gorm.DB{}), WithAIWorkbench(chat))
			require.Nil(t, server)
			require.ErrorContains(t, err, "explicit execution mode")
			require.False(t, built)
		})
	}
}
