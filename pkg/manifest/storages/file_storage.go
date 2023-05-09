package storages

import (
	"encoding/gob"
	"os"
	"time"

	"github.com/ddev/ddev/pkg/manifest/types"
)

func NewFileStorage(fileName string) types.ManifestStorage {
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
	LastUpdate time.Time
	Manifest   types.Manifest
}

func (s *fileStorage) LastUpdate() time.Time {
	err := s.loadData()
	if err != nil {
		return time.Time{}
	}

	return s.data.LastUpdate
}

func (s *fileStorage) Pull() (messages types.Manifest, err error) {
	err = s.loadData()
	if err != nil {
		return
	}

	return s.data.Manifest, nil
}

func (s *fileStorage) Push(manifest types.Manifest) (err error) {
	s.data.LastUpdate = time.Now()
	s.data.Manifest = manifest

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

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&s.data)
	file.Close()

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
		encoder := gob.NewEncoder(file)
		err = encoder.Encode(&s.data)
	}
	file.Close()

	return err
}
