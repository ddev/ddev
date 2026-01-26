package storage

import (
	"encoding/gob"
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
)

// AddonStorage defines interface for addon data persistence
type AddonStorage interface {
	Read() (*types.AddonData, error)
	Write(*types.AddonData) error
}

// NewAddonFileStorage creates a new file-based addon storage
func NewAddonFileStorage(fileName string) AddonStorage {
	return &addonFileStorage{
		fileName: fileName,
		loaded:   false,
	}
}

type addonFileStorage struct {
	fileName string
	loaded   bool
	data     addonFileStorageData
}

// addonFileStorageData is the structure used for the file
type addonFileStorageData struct {
	AddonData types.AddonData
}

func (s *addonFileStorage) Read() (*types.AddonData, error) {
	err := s.loadData()
	if err != nil {
		return &types.AddonData{}, err
	}

	return &s.data.AddonData, nil
}

func (s *addonFileStorage) Write(addonData *types.AddonData) error {
	s.data.AddonData = *addonData

	// Ensure directory exists
	dir := filepath.Dir(s.fileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return s.saveData()
}

// loadData reads the addon data from the data file
func (s *addonFileStorage) loadData() error {
	// The cache is read once per run time, early exit if already loaded
	if s.loaded {
		return nil
	}

	file, err := os.Open(s.fileName)
	// If the file does not exist, early exit with success
	if err != nil {
		if os.IsNotExist(err) {
			s.loaded = true
			return nil
		}
		// For other errors, return the error
		return err
	}

	defer file.Close()

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&s.data)

	// If the file was properly read, mark the cache as loaded
	if err == nil {
		s.loaded = true
	}

	return err
}

// saveData writes the addon data to the file
func (s *addonFileStorage) saveData() error {
	file, err := os.Create(s.fileName)

	if err == nil {
		defer file.Close()

		encoder := gob.NewEncoder(file)
		err = encoder.Encode(&s.data)
	}

	return err
}
