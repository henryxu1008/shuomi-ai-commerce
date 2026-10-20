package platformbase

import (
	"testing"

	"task-processor/internal/core/config"
	appfetcher "task-processor/internal/crawler/fetcher"

	"github.com/stretchr/testify/require"
)

func TestResolvePlatformFetcherType(t *testing.T) {
	t.Parallel()

	cfg := config.NewDefaultConfig()
	cfg.Platforms.Shein.FetchMode = "local"
	cfg.Platforms.Temu.FetchMode = "remote-api"

	fetcherType, err := ResolvePlatformFetcherType(cfg, "shein")
	require.NoError(t, err)
	require.Equal(t, appfetcher.LocalFetcher, fetcherType)

	fetcherType, err = ResolvePlatformFetcherType(cfg, "temu")
	require.NoError(t, err)
	require.Equal(t, appfetcher.RemoteAPIFetcher, fetcherType)

	fetcherType, err = ResolvePlatformFetcherType(cfg, "")
	require.NoError(t, err)
	require.Equal(t, appfetcher.FetcherType(""), fetcherType)
}

func TestResolvePlatformFetcherTypeRejectsInvalidMode(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"invalid", "distributed"} {
		cfg := config.NewDefaultConfig()
		cfg.Platforms.Shein.FetchMode = mode
		_, err := ResolvePlatformFetcherType(cfg, "shein")
		require.Error(t, err, "mode %q must be rejected", mode)
	}
}
