package goods

import (
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/asset"
	"testing"
)

func TestSupplierImagesResolveOnlyApprovedSlotAndOneDefiniteOfficialURL(t *testing.T) {
	in := DraftInput{Mode: "fully_managed", Selection: model.RuleSelection{CatID: 34}, Supplier: &model.SupplierGoods{SKCs: []model.SupplierSKC{{SKUs: []model.SupplierSKU{{ExternalCode: "a"}}}}}, Images: []ImageSlot{{Field: "material", AssetID: "a", Usage: 5}, {Field: "gallery", AssetID: "a", Usage: 2}, {Field: "thumb", AssetID: "a", Usage: 3}}}
	inventory := asset.ApprovedAssetInventory{Assets: []asset.ApprovedAsset{{ID: "a", URL: "https://source.example.org/a.png"}}}
	observed := []ImageObservation{}
	for _, usage := range []int64{5, 2, 3} {
		observed = append(observed, ImageObservation{AssetID: "a", SourceURL: inventory.Assets[0].URL, CatID: 34, Usage: usage, ContentHash: strings.Repeat("a", 64), Width: 800, Height: 800, Bytes: 1234, MediaType: "image/png", RemoteURLs: []string{"https://official.example.org/a.png"}, ResponseHash: strings.Repeat("b", 64)})
	}
	out, issues := resolveImages(in, inventory, observed, nil)
	require.Empty(t, issues)
	require.Equal(t, "https://official.example.org/a.png", out.Supplier.MaterialImage)
	require.Equal(t, []string{"https://official.example.org/a.png"}, out.Supplier.CarouselImages)
	require.Equal(t, "https://official.example.org/a.png", out.Supplier.SKCs[0].SKUs[0].ThumbURL)
	require.Empty(t, in.Supplier.MaterialImage)
	observed[0].RemoteURLs = append(observed[0].RemoteURLs, "https://official.example.org/ambiguous.png")
	_, issues = resolveImages(in, inventory, observed, nil)
	require.NotEmpty(t, issues)
}
