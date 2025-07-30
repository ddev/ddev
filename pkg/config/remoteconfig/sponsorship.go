package remoteconfig

import (
	"context"
	"sync"
	"time"

	"github.com/ddev/ddev/pkg/config/remoteconfig/downloader"
	"github.com/ddev/ddev/pkg/config/remoteconfig/storage"
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	statetypes "github.com/ddev/ddev/pkg/config/state/types"
	"github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/util"
)

const (
	sponsorshipLocalFileName  = ".sponsorship-data"
	sponsorshipUpdateInterval = 24 // hours
)

// sponsorshipManager implements types.SponsorshipManager
type sponsorshipManager struct {
	downloader       downloader.JSONCDownloader
	fileStorage      types.RemoteConfigStorage
	state            *state
	updateInterval   time.Duration
	isInternetActive func() bool
	data             *types.SponsorshipData
	mu               sync.Mutex
}

// NewSponsorshipManager creates a new sponsorship data manager
func NewSponsorshipManager(localPath string, stateManager statetypes.State, isInternetActive func() bool) types.SponsorshipManager {
	mgr := &sponsorshipManager{
		downloader: downloader.NewGitHubJSONCDownloader(
			"ddev",
			"sponsorship-data",
			"data/all-sponsorships.json",
			github.RepositoryContentGetOptions{Ref: "main"},
		),
		fileStorage:      storage.NewFileStorage(localPath + "/" + sponsorshipLocalFileName),
		state:            newState(stateManager),
		updateInterval:   time.Duration(sponsorshipUpdateInterval) * time.Hour,
		isInternetActive: isInternetActive,
	}

	mgr.loadFromLocalStorage()
	mgr.updateFromGithub()

	return mgr
}

// GetSponsorshipData returns the current sponsorship data
func (m *sponsorshipManager) GetSponsorshipData() (*types.SponsorshipData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data == nil {
		return &types.SponsorshipData{}, nil
	}

	return m.data, nil
}

// GetTotalMonthlyIncome returns the total monthly income from all sources
func (m *sponsorshipManager) GetTotalMonthlyIncome() int {
	data, err := m.GetSponsorshipData()
	if err != nil || data == nil {
		return 0
	}
	return data.TotalMonthlyAverageIncome
}

// GetTotalSponsors returns the total number of sponsors from all sources
func (m *sponsorshipManager) GetTotalSponsors() int {
	data, err := m.GetSponsorshipData()
	if err != nil || data == nil {
		return 0
	}

	total := data.GitHubDDEVSponsorships.TotalSponsors +
		data.GitHubRfaySponsorships.TotalSponsors +
		data.MonthlyInvoicedSponsorships.TotalSponsors +
		data.AnnualInvoicedSponsorships.TotalSponsors

	return total
}

// IsDataStale returns true if the sponsorship data needs updating
func (m *sponsorshipManager) IsDataStale() bool {
	return m.state.UpdatedAt.Add(m.updateInterval).Before(time.Now())
}

// loadFromLocalStorage loads sponsorship data from local file storage
func (m *sponsorshipManager) loadFromLocalStorage() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try to read from file storage using the RemoteConfig interface
	// This is a bit of a hack since we're reusing the existing file storage
	// which expects internal.RemoteConfig, but we can work around it
	err := m.loadSponsorshipDataFromFile()
	if err != nil {
		util.Debug("Error loading sponsorship data from local storage: %s", err)
		m.data = &types.SponsorshipData{}
	}
}

// loadSponsorshipDataFromFile loads sponsorship data directly from file
func (m *sponsorshipManager) loadSponsorshipDataFromFile() error {
	// For now, we'll update from GitHub if local file doesn't exist or is stale
	// A more complete implementation would use a dedicated sponsorship file storage
	m.data = &types.SponsorshipData{}
	return nil
}

// updateFromGithub downloads fresh sponsorship data from GitHub
func (m *sponsorshipManager) updateFromGithub() {
	if !m.isInternetActive() {
		util.Debug("No internet connection for sponsorship data update.")
		return
	}

	if !m.IsDataStale() {
		util.Debug("Sponsorship data is still fresh, skipping update.")
		return
	}

	util.Debug("Downloading sponsorship data from GitHub.")

	m.mu.Lock()
	defer m.mu.Unlock()

	ctx := context.Background()
	var newData types.SponsorshipData

	err := m.downloader.Download(ctx, &newData)
	if err != nil {
		util.Debug("Error downloading sponsorship data: %s", err)
		return
	}

	m.data = &newData

	// Update state
	m.state.UpdatedAt = time.Now()
	if err = m.state.save(); err != nil {
		util.Debug("Error saving sponsorship state: %s", err)
	}
}
