package types

import (
	"github.com/ddev/ddev/pkg/config/remoteconfig/internal"
)

type RemoteConfigStorage interface {
	Read() (internal.RemoteConfig, error)
	Write(internal.RemoteConfig) error
}
