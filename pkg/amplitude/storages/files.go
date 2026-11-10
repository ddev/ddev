package storages

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// LockFile holds an exclusive lock on path. It gives up after a second,
// because a command shouldn't wait on instrumentation.
func LockFile(path string) (unlock func(), ok bool) {
	fl := flock.New(path)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if locked, err := fl.TryLockContext(ctx, 20*time.Millisecond); err != nil || !locked {
		_ = fl.Close()
		return nil, false
	}

	return func() {
		_ = fl.Unlock()
		_ = fl.Close()
	}, true
}

// WriteFileAtomic replaces path with content through a temporary file, so
// an exit mid-write can't truncate it.
func WriteFileAtomic(path string, content []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}

	_, err = file.Write(content)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(file.Name(), path)
	}
	if err != nil {
		_ = os.Remove(file.Name())
	}

	return err
}
