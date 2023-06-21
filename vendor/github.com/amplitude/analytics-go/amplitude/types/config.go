package types

import (
	"time"
)

type Config struct {
	APIKey                 string
	FlushInterval          time.Duration
	FlushQueueSize         int
	FlushSizeDivider       int
	FlushMaxRetries        int
	Logger                 Logger
	MinIDLength            int
	ExecuteCallback        func(result ExecuteResult)
	ServerZone             ServerZone
	UseBatch               bool
	StorageFactory         func() EventStorage
	OptOut                 bool
	Plan                   *Plan
	IngestionMetadata      *IngestionMetadata
	ServerURL              string
	ConnectionTimeout      time.Duration
	MaxStorageCapacity     int
	RetryBaseInterval      time.Duration
	RetryThrottledInterval time.Duration
}

func NewConfig(apiKey string) Config {
	return Config{
		APIKey: apiKey,
	}
}

func (c Config) IsValid() bool {
	if c.APIKey == "" || c.FlushQueueSize <= 0 || c.FlushInterval <= 0 || c.MinIDLength < 0 {
		return false
	}

	return true
}
