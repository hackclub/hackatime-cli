package ini

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMutexClockAfterUsesRequestedDuration(t *testing.T) {
	delay := 20 * time.Millisecond
	start := time.Now()

	<-(&mutexClock{}).After(delay)

	assert.GreaterOrEqual(t, time.Since(start), delay)
}

func TestConfigMutexName(t *testing.T) {
	name := configMutexName("/tmp/wakatime.cfg")

	assert.True(t, strings.HasPrefix(name, "hackatime-cli-config-"))
	assert.LessOrEqual(t, len(name), 40)
}
