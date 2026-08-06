package lib_test

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"
)

func yamlToJSON(t *testing.T, source []byte) []byte {
	t.Helper()

	value, err := yaml.YAMLToJSON(source)
	require.NoError(t, err)

	return value
}
