package lexer_test

import (
	"os"
	"testing"

	"github.com/hackclub/hackatime-cli/pkg/lexer"

	"github.com/stretchr/testify/assert"
)

func TestForth_AnalyseText(t *testing.T) {
	data, err := os.ReadFile("testdata/forth_command.frt")
	assert.NoError(t, err)

	l := lexer.Forth{}.Lexer()

	assert.Equal(t, float32(0.3), l.AnalyseText(string(data)))
}
