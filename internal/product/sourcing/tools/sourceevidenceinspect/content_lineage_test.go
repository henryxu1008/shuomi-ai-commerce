package sourceevidenceinspect

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/enrichment"
)

type effectiveContentFixture struct {
	*sourceStub
	observation EffectiveSourceObservation
}

func (s effectiveContentFixture) ReadEffectiveSource(context.Context, commercetool.Principal, catalog.PublishedSnapshot) (EffectiveSourceObservation, error) {
	return s.observation, nil
}

func TestSourceToolContentLineageCannotPretendToBeTitleOnly(t *testing.T) {
	ctx, snapshots, sources, definition, deps, _ := invocationFixture(t)
	base := sources.value.Receipt.CatalogVersion
	snapshots.value.Version, snapshots.value.PublicationID = base+1, "review:content"
	selection, err := enrichment.NewContentSelection("en-US", []string{"title", "description"})
	require.NoError(t, err)
	observation := EffectiveSourceObservation{Requested: CatalogReference{"product", strconv.FormatUint(base+1, 10), "review:content"}, Original: sources.value, Applied: []AppliedPublicationReference{{AppliedTitleReference: AppliedTitleReference{ProposalID: uuid.NewString(), BaseVersion: strconv.FormatUint(base, 10), BasePublicationID: "publication", Version: strconv.FormatUint(base+1, 10), PublicationID: "review:content"}, Policy: enrichment.ContentPolicyVersion, ContentSelection: selection}}}
	invoker, err := NewInvoker(snapshots, effectiveContentFixture{sources, observation}, definition, deps)
	require.NoError(t, err)
	result, err := invoker.Invoke(ctx, callMetadata(), Input{ProductKey: "product", CatalogVersion: strconv.FormatUint(base+1, 10)})
	require.NoError(t, err)
	var output Output
	require.NoError(t, json.Unmarshal(result.Output, &output))
	require.Empty(t, output.Applied, "content adoption is not a title-only Apply receipt")
	require.NotNil(t, output.PublicationLineage)
	require.Equal(t, "product-review-publication-lineage-v1", output.PublicationLineage.Schema)
	require.Equal(t, observation.Applied, output.PublicationLineage.References)
	require.Equal(t, "publication", output.PublicationID)
	require.Equal(t, "review:content", output.Requested.PublicationID)
	// A current title-only result keeps its original wire shape.
	observation.Applied[0].Policy = "title-review-v1"
	observation.Applied[0].ContentSelection = enrichment.ContentSelection{}
	invoker, err = NewInvoker(snapshots, effectiveContentFixture{sources, observation}, definition, deps)
	require.NoError(t, err)
	result, err = invoker.Invoke(ctx, callMetadata(), Input{ProductKey: "product", CatalogVersion: strconv.FormatUint(base+1, 10)})
	require.NoError(t, err)
	output = Output{}
	require.NoError(t, json.Unmarshal(result.Output, &output))
	require.Nil(t, output.PublicationLineage)
	require.Equal(t, []AppliedTitleReference{observation.Applied[0].AppliedTitleReference}, output.Applied)
}
