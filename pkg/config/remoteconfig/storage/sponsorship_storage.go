package storage

import (
	"encoding/gob"
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
)

// SponsorshipStorage defines interface for sponsorship data persistence
type SponsorshipStorage interface {
	Read() (*types.SponsorshipData, error)
	Write(*types.SponsorshipData) error
}

// NewSponsorshipFileStorage creates a new file-based sponsorship storage
func NewSponsorshipFileStorage(fileName string) SponsorshipStorage {
	return &sponsorshipFileStorage{
		fileName: fileName,
		loaded:   false,
	}
}

type sponsorshipFileStorage struct {
	fileName string
	loaded   bool
	data     sponsorshipFileStorageData
}

// sponsorshipFileStorageData is the structure used for the file
type sponsorshipFileStorageData struct {
	SponsorshipData types.SponsorshipData
}

func (s *sponsorshipFileStorage) Read() (*types.SponsorshipData, error) {
	err := s.loadData()
	if err != nil {
		return &types.SponsorshipData{}, err
	}

	return &s.data.SponsorshipData, nil
}

func (s *sponsorshipFileStorage) Write(sponsorshipData *types.SponsorshipData) error {
	s.data.SponsorshipData = *sponsorshipData

	// Ensure directory exists
	dir := filepath.Dir(s.fileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return s.saveData()
}

// loadData reads the sponsorship data from the data file
func (s *sponsorshipFileStorage) loadData() error {
	// The cache is read once per run time, early exit if already loaded
	if s.loaded {
		return nil
	}

	file, err := os.Open(s.fileName)
	// If the file does not exist, early exit
	if err != nil {
		s.loaded = true
		return nil
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

// saveData writes the sponsorship data to the file
func (s *sponsorshipFileStorage) saveData() error {
	file, err := os.Create(s.fileName)

	if err == nil {
		defer file.Close()

		encoder := gob.NewEncoder(file)
		err = encoder.Encode(&s.data)
	}

	return err
}
