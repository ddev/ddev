package types

import (
	"time"

	"github.com/ddev/ddev/pkg/config/remoteconfig/internal"
)

type RemoteConfigStorage interface {
	LastUpdate() time.Time
	Push(manifest internal.RemoteConfig) error
	Pull() (internal.RemoteConfig, error)
}
