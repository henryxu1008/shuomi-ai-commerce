package currentapplication

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConfigDocumentParsingDoesNotApplyHostFilesystemSemantics(t *testing.T) {
	cfg, e := DecodeConfigDocument([]byte(`{"schemaVersion":1,"toolMarket":{"packageDirectory":"/private/tool-release"}}`))
	require.NoError(t, e)
	require.Equal(t, "/private/tool-release", cfg.ToolMarket.PackageDirectory)
	for _, raw := range []string{`{"schemaVersion":1,"schemaVersion":2}`, `{"schemaVersion":1,"unknown":true}`} {
		_, e = DecodeConfigDocument([]byte(raw))
		require.Error(t, e)
	}
}
