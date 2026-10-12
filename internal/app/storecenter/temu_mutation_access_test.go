package storecenterapp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/storecenter"
)

func (p *temuRulesProvider) CreateLocalGoods(context.Context, storecenter.OfficialMerchantCredential, model.LocalGoodsV2) (temugoods.CreatedGoods, error) {
	p.metadataReads++
	id := int64(9007199254740993)
	return temugoods.CreatedGoods{Local: &model.LocalGoodsCreated{GoodsID: &id}}, nil
}
func (p *temuRulesProvider) UploadLocalImage(context.Context, storecenter.OfficialMerchantCredential, model.LocalImageUpload) (model.LocalImageUploaded, string, error) {
	p.metadataReads++
	return model.LocalImageUploaded{}, "", nil
}

func TestTemuMutationUsesOriginalPurposeAndOnlyOneDispatch(t *testing.T) {
	for _, purpose := range []string{storecenter.ProductPurposePublish, storecenter.ProductPurposeImage} {
		t.Run(purpose, func(t *testing.T) {
			a, _, p, subject := newTemuRulesAccess(t, storecenter.PartitionGlobal, storecenter.MerchantLocal)
			_, err := a.AuthorizeTemuMutation(context.Background(), subject, "store-a", nil)
			require.ErrorIs(t, err, ErrProductAccessChanged)
			subject.Purpose = purpose
			h, err := a.AuthorizeTemuMutation(context.Background(), subject, "store-a", nil)
			require.NoError(t, err)
			raw, err := json.Marshal(h)
			require.NoError(t, err)
			require.JSONEq(t, `{}`, string(raw))
			if purpose == storecenter.ProductPurposePublish {
				_, _, err = h.UploadLocalImage(context.Background(), model.LocalImageUpload{})
				require.ErrorIs(t, err, ErrProductAccessChanged)
				_, err = h.CreateLocalGoods(context.Background(), model.LocalGoodsV2{})
				require.NoError(t, err)
				_, err = h.CreateLocalGoods(context.Background(), model.LocalGoodsV2{})
			} else {
				_, err = h.CreateLocalGoods(context.Background(), model.LocalGoodsV2{})
				require.ErrorIs(t, err, ErrProductAccessChanged)
				_, _, err = h.UploadLocalImage(context.Background(), model.LocalImageUpload{})
				require.NoError(t, err)
				_, _, err = h.UploadLocalImage(context.Background(), model.LocalImageUpload{})
			}
			require.ErrorIs(t, err, ErrProductAccessChanged)
			require.Equal(t, 1, p.metadataReads)
		})
	}
}

func TestTemuMutationRechecksAuthorityBeforeProvider(t *testing.T) {
	for _, change := range []string{"grant", "connection", "credential", "service", "mode", "expired"} {
		t.Run(change, func(t *testing.T) {
			a, reader, p, subject := newTemuRulesAccess(t, storecenter.PartitionGlobal, storecenter.MerchantLocal)
			subject.Purpose = storecenter.ProductPurposePublish
			h, err := a.AuthorizeTemuMutation(context.Background(), subject, "store-a", nil)
			require.NoError(t, err)
			switch change {
			case "grant":
				reader.err = storecenter.ErrNotFound
			case "connection":
				reader.material.Connection.Version++
			case "credential":
				reader.material.Attempt.Ciphertext = "changed"
			case "service":
				reader.material.ServiceExpiresAt = a.now()
			case "mode":
				reader.material.Connection.Mode = storecenter.MerchantSemiManaged
			case "expired":
				a.now = func() time.Time { return h.authority.expiresAt }
			}
			_, err = h.CreateLocalGoods(context.Background(), model.LocalGoodsV2{})
			require.ErrorIs(t, err, ErrProductAccessChanged)
			require.Zero(t, p.metadataReads)
		})
	}
}
