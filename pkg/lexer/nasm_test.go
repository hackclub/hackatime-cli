package lexer_test

import (
	"os"
	"testing"

	"github.com/hackclub/hackatime-cli/pkg/lexer"

	"github.com/stretchr/testify/assert"
)

func TestNASM_AnalyseText(t *testing.T) {
	data, err := os.ReadFile("testdata/nasm.asm")
	assert.NoError(t, err)

	l := lexer.NASM{}.Lexer()

	assert.Equal(t, float32(0), l.AnalyseText(string(data)))
}
