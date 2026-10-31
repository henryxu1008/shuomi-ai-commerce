package openai

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCredentialConfigIsOnlyTheCurrentProviderSelection(t *testing.T) {
	typ := reflect.TypeOf(ClientConfig{})
	fields := []string{"APIKey", "Model", "BaseURL", "APIStyle", "Timeout"}
	require.Equal(t, len(fields), typ.NumField())
	for i, name := range fields {
		require.Equal(t, name, typ.Field(i).Name)
	}
}
