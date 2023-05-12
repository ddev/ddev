package types

import (
	"time"

	"github.com/ddev/ddev/pkg/manifest/internal"
)

type ManifestStorage interface {
	LastUpdate() time.Time
	Push(manifest internal.Manifest) error
	Pull() (internal.Manifest, error)
}
