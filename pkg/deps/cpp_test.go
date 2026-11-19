package deps_test

import (
	"testing"

	"github.com/hackclub/hackatime-cli/pkg/deps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserCPP_Parse(t *testing.T) {
	parser := deps.ParserCPP{}

	dependencies, err := parser.Parse(t.Context(), "testdata/cpp.cpp")
	require.NoError(t, err)

	assert.Equal(t, []string{
		"openssl",
		"wakatime",
	}, dependencies)
}
