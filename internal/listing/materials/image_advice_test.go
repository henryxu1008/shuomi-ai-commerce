package materials

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/shein/goods"
	"task-processor/internal/product/collection"
)

func TestImageAdviceDistinguishesUnknownDimensionsFromRealMismatch(t *testing.T) {
	for _, scenario := range []string{"unknown", "known_conflict", "recorded_dimensions", "foreign_observation"} {
		t.Run(scenario, func(t *testing.T) {
			i := materialInspection(t)
			i.Base.Input.Draft.Images = []goods.OfficialImageSlot{{Group: "skc", AssetID: "approved", Sort: 1, Type: 1}}
			if scenario == "known_conflict" {
				i.Inventory.Assets[0].Width, i.Inventory.Assets[0].Height = 100, 100
			}
			if scenario == "recorded_dimensions" || scenario == "foreign_observation" {
				o := goods.OfficialImageObservation{AssetID: "approved", SourceURL: i.Inventory.Assets[0].URL, Width: 900, Height: 900, Type: 1, ContentHash: strings.Repeat("a", 64), Bytes: 1000, MediaType: "image/png"}
				if scenario == "foreign_observation" {
					o.SourceURL = "https://other.test/image.png"
				}
				i.Base.ImageObservations = []goods.OfficialImageObservation{o}
			}
			i.Base.InventoryHash = collection.Digest(i.Inventory)
			value, err := NewInspection(i.Base, i.Product, i.Rules, i.Inventory, time.Now().UTC(), time.Now().UTC())
			require.NoError(t, err)
			d, err := value.Disclose()
			require.NoError(t, err)
			want := "unknown"
			if scenario == "known_conflict" || scenario == "recorded_dimensions" {
				want = "known"
			}
			require.Equal(t, want, d.ImageAdvice.Facts[0].DimensionsStatus)
			for _, check := range d.ImageAdvice.Checks {
				if check.Code == "dimensions_conflict" {
					require.Equal(t, "known_conflict", scenario)
				}
			}
			if scenario == "known_conflict" {
				found := false
				for _, check := range d.ImageAdvice.Checks {
					found = found || check.Code == "dimensions_conflict"
				}
				require.True(t, found)
			}
		})
	}
}
