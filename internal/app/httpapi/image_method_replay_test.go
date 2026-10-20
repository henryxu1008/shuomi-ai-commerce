package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/imageagent"
	imagestore "task-processor/internal/imageagent/store"
	configstore "task-processor/internal/integration/persistence/agentconfig"
)

type replayImageMethod struct{}

func (replayImageMethod) AcquireImageMethod(context.Context, imageagent.ExecutionIdentity, agent.SkillVersionRef, string) (string, func(), error) {
	return "A clear product photograph.", func() {}, nil
}

func TestFullImageOriginalPreparedMethodDoesNotRequireNewSkillSelection(t *testing.T) {
	const sourceID = "d1abe8da-b381-4924-8d15-d79bdbfacf70"
	const requestID = "b912e7d4-df80-44a5-8510-3cdf91d5b8dd"
	catalog, err := imageagent.NormalizeAssetCatalog(imageagent.AssetCatalog{ProductContext: imageagent.ProductContextRef{ProductID: "product", Title: "Product", SourceSnapshotVersion: 1}, Assets: []imageagent.AuthorizedAsset{{ID: "source-1", Type: imageagent.AuthorizedAssetSource, URL: "https://source.example/image.png", Width: 1024, Height: 1024}}})
	require.NoError(t, err)
	contexts := imageSetPrepareACKContexts{preparation: imageagent.ImageSetPreparation{Source: imageagent.ImageSourceBinding{ContextKind: imageagent.ImageSourceAcquisition, ProductID: "product", OperationID: sourceID, OriginalPublicationID: "publication", OriginalVersion: 1, EffectiveVersion: 1, CatalogHash: catalog.Manifest.Hash}, Target: imageagent.ImageTarget{Platform: "product"}, Catalog: catalog, Observations: []imageagent.ImageSourceObservation{{AssetID: "source-1", SHA256: strings.Repeat("c", 64), Bytes: 10, Width: 1024, Height: 1024, MediaType: "image/png"}}}}
	repo := &imageSetPrepareACKRepository{Repository: imagestore.NewMemoryRepository()}
	service, err := imageagent.NewService(repo, imageSetHTTPWorkflow{}, closedImageSetCatalog{}, imageagent.WithOrganizationScope(), imageagent.WithImageMethodContext(replayImageMethod{}), imageagent.WithImageSetDependencies(imageagent.ImageSetDependencies{Configuration: imageSetPrepareACKConfiguration{}, Contexts: contexts, Quotes: imageSetPrepareACKQuote{}, HardLimits: agentconfig.ImageRunLimits{Images: 32, Points: 100, ElapsedSeconds: 3600}}))
	require.NoError(t, err)
	identity := authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member"}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), identity)
	ref := agent.SkillVersionRef{Kind: "skill-version-v1", ID: requestID, OwnerKind: "PLATFORM", OwnerID: "official", Version: 2, Digest: strings.Repeat("f", 64), PolicyVersion: "skill-method-v1"}
	input := imageagent.PrepareImageSetInput{Text: &imageagent.ImageTextSelection{Mode: "none"}, Method: &ref, ExpectedModeRevision: "0", ContextKind: imageagent.ImageSourceAcquisition, ContextID: sourceID, RequestID: requestID, Template: &agentconfig.TemplateRef{TemplateID: requestID, Revision: "1"}, Target: imageagent.ImageTargetSelection{Platform: "product"}, SharedOriginalIDs: []string{"source-1"}}
	_, err = service.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	g, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	configuration, err := configstore.New(g)
	require.NoError(t, err)
	module := fullImageModule{application: &fullImageApplication{service: service, configuration: configuration}, bind: func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }}
	router := gin.New()
	for _, route := range module.routes() {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	body, _ := json.Marshal(map[string]any{"text": input.Text, "expectedModeRevision": "0", "template": input.Template, "target": input.Target, "sharedOriginalIds": input.SharedOriginalIDs, "skill": map[string]any{"skillId": ref.ID, "version": ref.Version, "digest": ref.Digest}})
	mock.ExpectQuery("SELECT").WillReturnError(agentconfig.ErrUnavailable)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workbench/sourcing/1688/acquisitions/"+sourceID+"/images/prepare", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", requestID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req.WithContext(ctx))
	require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
	require.JSONEq(t, `{"code":"OUTCOME_UNKNOWN"}`, w.Body.String())
	require.Equal(t, 1, repo.initializations)
	require.NoError(t, mock.ExpectationsWereMet())
}
