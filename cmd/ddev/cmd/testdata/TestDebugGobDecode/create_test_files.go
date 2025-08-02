// This file creates test gob files for the debug gob-decode tests
// Run with: go run create_test_files.go
package main

import (
	"encoding/gob"
	"os"
	"time"

	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
)

// Types matching the debug-gob-decode.go structures
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

type fileStorageData struct {
	RemoteConfig types.RemoteConfigData
}

type sponsorshipFileStorageData struct {
	SponsorshipData types.SponsorshipData
}

func main() {
	// Create test remote config file
	remoteConfigData := fileStorageData{
		RemoteConfig: types.RemoteConfigData{
			UpdateInterval: 24,
			Remote: types.Remote{
				Owner:    "test-owner",
				Repo:     "test-repo",
				Ref:      "test-ref",
				Filepath: "test-config.jsonc",
			},
			Messages: types.Messages{
				Notifications: types.Notifications{
					Interval: 12,
					Infos:    []types.Message{{Message: "Test info message"}},
					Warnings: []types.Message{{Message: "Test warning message"}},
				},
				Ticker: types.Ticker{
					Interval: 6,
					Messages: []types.Message{
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
		SponsorshipData: types.SponsorshipData{
			GitHubDDEVSponsorships: types.GitHubSponsorship{
				TotalMonthlySponsorship: 1000,
				TotalSponsors:           2,
				SponsorsPerTier: map[string]int{
					"Gold":   1,
					"Silver": 1,
				},
			},
			GitHubRfaySponsorships: types.GitHubSponsorship{
				TotalMonthlySponsorship: 0,
				TotalSponsors:           0,
				SponsorsPerTier:         map[string]int{},
			},
			MonthlyInvoicedSponsorships: types.InvoicedSponsorship{
				TotalMonthlySponsorship: 0,
				TotalSponsors:           0,
				MonthlySponsorsPerTier:  map[string]int{},
			},
			AnnualInvoicedSponsorships: types.AnnualSponsorship{
				TotalAnnualSponsorships:      0,
				TotalSponsors:                0,
				MonthlyEquivalentSponsorship: 0,
				AnnualSponsorsPerTier:        map[string]int{},
			},
			PaypalSponsorships:        0,
			TotalMonthlyAverageIncome: 1050.00,
			UpdatedDateTime:           time.Now(),
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
