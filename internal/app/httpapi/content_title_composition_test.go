package httpapi

import (
	"context"
	"errors"
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

func TestContentChatFullProductAgentCompositionAdmission(t *testing.T) {
	stop := errors.New("reached workbench construction")
	for _, tc := range []struct {
		name         string
		content      bool
		collections  bool
		sharedPool   bool
		wantAdmitted bool
	}{
		{"full runtime with content", true, true, false, true},
		{"missing content consumer", false, true, false, false},
		{"missing collections", true, false, false, false},
		{"shared run pool", true, true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run, chatDB := &gorm.DB{}, &gorm.DB{}
			if tc.sharedPool {
				chatDB = run
			}
			product := ProductAgentDependencies{Enabled: true, RunDB: run}
			if tc.content {
				product.ContentTextPolicies = map[string]governed.RoutePolicy{"org": {}}
			}
			chat := AIWorkbenchDependencies{DB: chatDB, ContentTasks: true}
			factories := currentApplicationFactories{
				buildWorkbench: func(*config.Config, *logrus.Logger) (workbenchContextBuildResult, error) {
					return workbenchContextBuildResult{}, stop
				},
				buildSourceAccount: func(*gorm.DB, *authz.ListingKitAuthorizer) (kernelmodule.Module, error) { return nil, nil },
				buildCommercial:    func(*gorm.DB, *gorm.DB, *authz.ListingKitAuthorizer) (kernelmodule.Module, error) { return nil, nil },
			}
			options := []CurrentApplicationOption{WithProductAgent(product), WithProductAcquisition(&gorm.DB{}), WithAIWorkbench(chat)}
			if tc.collections {
				options = append(options, WithProductCollections())
			}
			_, err := buildCurrentApplication(context.Background(), &gorm.DB{}, currentApplicationTestConfig(), logrus.New(), factories, options...)
			if tc.wantAdmitted {
				require.ErrorContains(t, err, "resource consumers require their resource owner pool")
			} else {
				require.ErrorContains(t, err, "explicit execution mode")
			}
		})
	}
}
