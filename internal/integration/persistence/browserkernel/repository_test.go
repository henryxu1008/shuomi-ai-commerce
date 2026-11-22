package kernelpersistence

import (
	"context"
	k "github.com/henryxu1008/shuomi-ai-commerce/internal/browserkernel"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestInvalidOwnerPoolFailsClosed(t *testing.T) {
	for _, db := range []*gorm.DB{nil, {}, {Config: &gorm.Config{}}} {
		_, e := New(db)
		require.ErrorIs(t, e, k.ErrUnavailable)
		require.ErrorIs(t, VerifySchema(context.Background(), db), k.ErrUnavailable)
		require.ErrorIs(t, InstallSchema(db), k.ErrUnavailable)
		require.ErrorIs(t, GrantRuntime(db, "browser_kernel_runtime"), k.ErrUnavailable)
	}
}
