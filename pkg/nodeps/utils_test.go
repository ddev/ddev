package nodeps

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

type netResolverStub struct {
	sleepTime time.Duration
	err       error
}

func (t netResolverStub) LookupHost(ctx context.Context, _ string) ([]string, error) {
	select {
	case <-time.After(t.sleepTime):
	case <-ctx.Done():
		return nil, errors.New("context timed out")
	}
	return nil, t.err
}

func resetVariables() {
	// resetVariables back to defaults
	isInternetActiveNetResolver = net.DefaultResolver
	isInternetActiveAlreadyChecked = false
	isInternetActiveResult = false
}

func TestIsInternetActiveErrorOccurred(t *testing.T) {
	resetVariables()

	isInternetActiveNetResolver = netResolverStub{
		sleepTime: 0,
		err:       errors.New("test error"),
	}

	assert.False(t, IsInternetActive())
}

func TestIsInternetActiveTimeout(t *testing.T) {
	resetVariables()

	isInternetActiveNetResolver = netResolverStub{
		sleepTime: 1000 * time.Millisecond,
	}

	assert.False(t, IsInternetActive())
}

func TestIsInternetActiveAlreadyChecked(t *testing.T) {
	resetVariables()

	isInternetActiveAlreadyChecked = true
	isInternetActiveResult = true

	assert.True(t, IsInternetActive())
}

func TestIsInternetActive(t *testing.T) {
	resetVariables()

	isInternetActiveNetResolver = netResolverStub{
		sleepTime: 0,
	}

	// should return true
	assert.True(t, IsInternetActive())
	// should have set the isInternetActiveAlreadyChecked to true
	assert.True(t, isInternetActiveAlreadyChecked)
	// result should still be true
	assert.True(t, isInternetActiveResult)
	// and calling it again, should also still be true
	assert.True(t, IsInternetActive())
}

func TestRandomString(t *testing.T) {
	randomString := RandomString(10)

	// is RandomString as long as required
	assert.Equal(t, 10, len(randomString))
}
