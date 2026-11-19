//go:build !linux

package system_test

import (
	"runtime"
	"testing"

	"github.com/hackclub/hackatime-cli/pkg/system"
	"github.com/stretchr/testify/assert"
)

func TestOSName(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("skipping test on non-darwin and non-windows system")
	}

	name := system.OSName(t.Context())

	assert.Equal(t, runtime.GOOS, name)
}
