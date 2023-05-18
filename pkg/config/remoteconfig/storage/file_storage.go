package storage

import (
	"encoding/gob"
	"os"
	"time"

	"github.com/ddev/ddev/pkg/config/remoteconfig/internal"
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
)

func NewFileStorage(fileName string) types.RemoteConfigStorage {
	return &fileStorage{
		fileName: fileName,
		loaded:   false,
	}
}

type fileStorage struct {
	fileName string
	loaded   bool
	data     fileStorageData
}

// fileStorageData is the structure used for the file.
type fileStorageData struct {
	LastUpdate   time.Time
	RemoteConfig internal.RemoteConfig
}

func (s *fileStorage) LastUpdate() time.Time {
	err := s.loadData()
	if err != nil {
		return time.Time{}
	}

	return s.data.LastUpdate
}

func (s *fileStorage) Pull() (messages internal.RemoteConfig, err error) {
	err = s.loadData()
	if err != nil {
		return
	}

	return s.data.RemoteConfig, nil
}

func (s *fileStorage) Push(manifest internal.RemoteConfig) (err error) {
	s.data.LastUpdate = time.Now()
	s.data.RemoteConfig = manifest

	err = s.saveData()

	return
}

// loadData reads the messages from the data file.
func (s *fileStorage) loadData() error {
	// The cache is read once per run time, early exit if already loaded.
	if s.loaded {
		return nil
	}

	file, err := os.Open(s.fileName)
	// If the file does not exists, early exit.
	if err != nil {
		s.loaded = true

		return nil
	}

	defer file.Close()

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&s.data)

	// If the file was properly read, mark the cache as loaded.
	if err == nil {
		s.loaded = true
	}

	return err
}

// saveData writes the messages to the message file.
func (s *fileStorage) saveData() error {
	file, err := os.Create(s.fileName)

	if err == nil {
		defer file.Close()

		encoder := gob.NewEncoder(file)
		err = encoder.Encode(&s.data)
	}

	return err
}
