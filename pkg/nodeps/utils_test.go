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

// LookupHost is a custom version of net.LookupHost that just wastes some time and then returns
func (t netResolverStub) LookupHost(ctx context.Context, _ string) ([]string, error) {
	select {
	case <-time.After(t.sleepTime):
	case <-ctx.Done():
		return nil, errors.New("context timed out")
	}
	return nil, t.err
}

// resetVariables resets the global variables IsInternetActive() uses back to their defaults
func resetVariables() {
	isInternetActiveNetResolver = net.DefaultResolver
	isInternetActiveAlreadyChecked = false
	isInternetActiveResult = false
}

// TestIsInternetActiveErrorOccurred tests if IsInternetActive() returns false when LookupHost returns an error
func TestIsInternetActiveErrorOccurred(t *testing.T) {
	resetVariables()

	isInternetActiveNetResolver = netResolverStub{
		sleepTime: 0,
		err:       errors.New("test error"),
	}

	assert.False(t, IsInternetActive())
}

// TestIsInternetActiveTimeout tests if IsInternetActive() returns false when it times out
func TestIsInternetActiveTimeout(t *testing.T) {
	resetVariables()

	isInternetActiveNetResolver = netResolverStub{
		sleepTime: 1000 * time.Millisecond,
	}

	assert.False(t, IsInternetActive())
}

// TestIsInternetActiveAlreadyChecked tests if IsInternetActive() returns true when it has already
// been called and returned true on an earlier execution.
func TestIsInternetActiveAlreadyChecked(t *testing.T) {
	resetVariables()

	isInternetActiveAlreadyChecked = true
	isInternetActiveResult = true

	assert.True(t, IsInternetActive())
}

// TestIsInternetActive tests if IsInternetActive() returns true, when the LookupHost call goes well
// and if it properly sets the globals so it won't execute the LookupHost again.
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

// TestRandomString tests if RandomString returns the correct character length
func TestRandomString(t *testing.T) {
	randomString := RandomString(10)

	// is RandomString as long as required
	assert.Equal(t, 10, len(randomString))
}
