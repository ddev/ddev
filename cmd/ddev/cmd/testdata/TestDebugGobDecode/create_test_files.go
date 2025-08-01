// This file creates test gob files for the debug gob-decode tests
// Run with: go run create_test_files.go
package main

import (
	"encoding/gob"
	"os"
	"time"
)

// Types matching the debug-gob-decode.go structures
type Message struct {
	Message    string   `json:"message"`
	Title      string   `json:"title,omitempty"`
	Conditions []string `json:"conditions,omitempty"`
	Versions   string   `json:"versions,omitempty"`
}

type Notifications struct {
	Interval int       `json:"interval"`
	Infos    []Message `json:"infos"`
	Warnings []Message `json:"warnings"`
}

type Ticker struct {
	Interval int       `json:"interval"`
	Messages []Message `json:"messages"`
}

type Messages struct {
	Notifications Notifications `json:"notifications"`
	Ticker        Ticker        `json:"ticker"`
}

type Remote struct {
	Owner    string `json:"owner,omitempty"`
	Repo     string `json:"repo,omitempty"`
	Ref      string `json:"ref,omitempty"`
	Filepath string `json:"filepath,omitempty"`
}

type RemoteConfig struct {
	UpdateInterval int      `json:"update-interval,omitempty"`
	Remote         Remote   `json:"remote,omitempty"`
	Messages       Messages `json:"messages,omitempty"`
}

type fileStorageData struct {
	RemoteConfig RemoteConfig
}

type SponsorshipData struct {
	Sponsors     []Sponsor `json:"sponsors,omitempty"`
	TotalIncome  float64   `json:"total_income,omitempty"`
	SponsorCount int       `json:"sponsor_count,omitempty"`
}

type Sponsor struct {
	Name        string  `json:"name,omitempty"`
	Amount      float64 `json:"amount,omitempty"`
	Currency    string  `json:"currency,omitempty"`
	Type        string  `json:"type,omitempty"`
	Description string  `json:"description,omitempty"`
}

type sponsorshipFileStorageData struct {
	SponsorshipData SponsorshipData
}

type StorageEvent struct {
	EventType  string                 `json:"event_type,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	DeviceID   string                 `json:"device_id,omitempty"`
	Time       int64                  `json:"time,omitempty"`
	EventProps map[string]interface{} `json:"event_properties,omitempty"`
	UserProps  map[string]interface{} `json:"user_properties,omitempty"`
}

type eventCache struct {
	LastSubmittedAt time.Time       `json:"last_submitted_at"`
	Events          []*StorageEvent `json:"events"`
}

type customData struct {
	Name    string            `json:"name"`
	Count   int               `json:"count"`
	Enabled bool              `json:"enabled"`
	Tags    []string          `json:"tags"`
	Meta    map[string]string `json:"meta"`
}

func main() {
	// Create test remote config file
	remoteConfigData := fileStorageData{
		RemoteConfig: RemoteConfig{
			UpdateInterval: 24,
			Remote: Remote{
				Owner:    "test-owner",
				Repo:     "test-repo",
				Ref:      "test-ref",
				Filepath: "test-config.jsonc",
			},
			Messages: Messages{
				Notifications: Notifications{
					Interval: 12,
					Infos:    []Message{{Message: "Test info message"}},
					Warnings: []Message{{Message: "Test warning message"}},
				},
				Ticker: Ticker{
					Interval: 6,
					Messages: []Message{
						{Message: "Test ticker message 1"},
						{Message: "Test ticker message 2", Title: "Custom Title"},
					},
				},
			},
		},
	}

	file, _ := os.Create("test-remote-config.gob")
	gob.NewEncoder(file).Encode(remoteConfigData)
	file.Close()

	// Create test amplitude cache file
	amplitudeData := eventCache{
		LastSubmittedAt: time.Date(2024, 8, 1, 12, 0, 0, 0, time.UTC),
		Events: []*StorageEvent{
			{
				EventType: "test_event_1",
				UserID:    "user123",
				DeviceID:  "device456",
				Time:      1722544763,
				EventProps: map[string]interface{}{
					"test_prop": "test_value",
					"count":     42,
				},
				UserProps: map[string]interface{}{
					"user_type": "developer",
				},
			},
			{
				EventType: "test_event_2",
				DeviceID:  "device789",
				Time:      1722544800,
				EventProps: map[string]interface{}{
					"action": "debug_command",
				},
			},
		},
	}

	file, _ = os.Create("test-amplitude-cache.gob")
	gob.NewEncoder(file).Encode(amplitudeData)
	file.Close()

	// Create test sponsorship data file
	sponsorshipData := sponsorshipFileStorageData{
		SponsorshipData: SponsorshipData{
			Sponsors: []Sponsor{
				{
					Name:        "ACME Corporation",
					Amount:      1000.00,
					Currency:    "USD",
					Type:        "monthly",
					Description: "Supporting open source development",
				},
				{
					Name:        "Developer Jane",
					Amount:      50.00,
					Currency:    "USD",
					Type:        "one-time",
					Description: "Thanks for the great tool!",
				},
			},
			TotalIncome:  1050.00,
			SponsorCount: 2,
		},
	}

	file, _ = os.Create("test-sponsorship-data.gob")
	gob.NewEncoder(file).Encode(sponsorshipData)
	file.Close()

	// Create test generic gob file with primitive types that can be decoded as interface{}
	genericData := map[string]interface{}{
		"name":    "generic_test_data",
		"count":   123,
		"enabled": true,
		"tags":    []string{"test", "gob", "fallback"},
		"meta": map[string]string{
			"version": "1.0",
			"type":    "test",
		},
	}

	file, _ = os.Create("test-generic.gob")
	gob.NewEncoder(file).Encode(genericData)
	file.Close()
}