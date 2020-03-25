package nodeps

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"net"
	"os"
	"testing"
	"time"
)

type netResolverStub struct {
	sleepTime time.Duration
	err       error
}

func (t netResolverStub) LookupHost(_ context.Context, _ string) ([]string, error) {
	time.Sleep(t.sleepTime)
	return nil, t.err
}

func TestMain(m *testing.M) {
	// reset back to defaults
	isInternetActiveNetResolver = net.DefaultResolver
	isInternetActiveAlreadyChecked = false
	isInternetActiveResult = false

	// run tests
	code := m.Run()
	os.Exit(code)
}

func TestIsInternetActiveErrorOccurred(t *testing.T) {
	isInternetActiveNetResolver = netResolverStub{
		sleepTime: 0,
		err:       errors.New("test error"),
	}

	assert.False(t, IsInternetActive())
}

func TestIsInternetActiveTimeout(t *testing.T) {
	isInternetActiveNetResolver = netResolverStub{
		sleepTime: 501 * time.Millisecond,
		err:       nil,
	}

	assert.False(t, IsInternetActive())
}

func TestIsInternetActiveAlreadyChecked(t *testing.T) {
	isInternetActiveAlreadyChecked = true
	isInternetActiveResult = true

	assert.True(t, IsInternetActive())
}

func TestIsInternetActive(t *testing.T) {
	isInternetActiveNetResolver = netResolverStub{
		sleepTime: 0,
		err:       nil,
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
