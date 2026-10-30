package kernelpersistence

import (
	"context"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	k "task-processor/internal/browserkernel"
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
