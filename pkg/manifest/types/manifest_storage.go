package types

import "time"

type ManifestStorage interface {
	LastUpdate() time.Time
	Push(manifest *Manifest) error
	Pull() (Manifest, error)
}
