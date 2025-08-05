package remoteconfig

import (
	"context"
	"sync"
	"time"

	"github.com/ddev/ddev/pkg/config/remoteconfig/downloader"
	"github.com/ddev/ddev/pkg/config/remoteconfig/storage"
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	statetypes "github.com/ddev/ddev/pkg/config/state/types"
	"github.com/ddev/ddev/pkg/util"
)

const (
	sponsorshipLocalFileName  = ".sponsorship-data"
	sponsorshipUpdateInterval = 24 // hours
)

// sponsorshipManager implements types.SponsorshipManager
type sponsorshipManager struct {
	downloader       downloader.JSONCDownloader
	fileStorage      storage.SponsorshipStorage
	state            *state
	updateInterval   time.Duration
	isInternetActive func() bool
	data             *types.SponsorshipData
	mu               sync.Mutex
}


// NewSponsorshipManager creates a new sponsorship data manager using a direct URL
func NewSponsorshipManager(localPath string, stateManager statetypes.State, isInternetActive func() bool, updateInterval int, url string) types.SponsorshipManager {
	mgr := &sponsorshipManager{
		downloader:       downloader.NewURLJSONCDownloader(url),
		fileStorage:      storage.NewSponsorshipFileStorage(localPath + "/" + sponsorshipLocalFileName),
		state:            newState(stateManager),
		updateInterval:   time.Duration(updateInterval) * time.Hour,
		isInternetActive: isInternetActive,
	}

	mgr.loadFromLocalStorage()
	mgr.updateFromRemote()

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

// GetTotalMonthlyIncome returns the total monthly average income as float64
func (m *sponsorshipManager) GetTotalMonthlyIncome() float64 {
	data, err := m.GetSponsorshipData()
	if err != nil {
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

	data, err := m.fileStorage.Read()
	if err != nil {
		util.Debug("Error loading sponsorship data from local storage: %s", err)
		// Initialize with empty data as fallback
		m.data = &types.SponsorshipData{}
	} else {
		m.data = data
	}
}

// updateFromRemote downloads fresh sponsorship data from the remote source
func (m *sponsorshipManager) updateFromRemote() {
	if !m.isInternetActive() {
		util.Debug("No internet connection for sponsorship data update.")
		return
	}

	if !m.IsDataStale() {
		util.Debug("Sponsorship data is still fresh, skipping update.")
		return
	}

	util.Debug("Downloading sponsorship data from remote source.")

	m.mu.Lock()
	defer m.mu.Unlock()

	ctx := context.Background()
	var newData types.SponsorshipData

	err := m.downloader.Download(ctx, &newData)
	if err != nil {
		util.Debug("Error downloading sponsorship data from remote source: %s", err)
		// Don't update data if download fails, keep existing data
		return
	}

	m.data = &newData

	// Save to local storage
	if err := m.fileStorage.Write(&newData); err != nil {
		util.Debug("Error saving sponsorship data to local storage: %s", err)
	}

	// Update state
	m.state.UpdatedAt = time.Now()
	if err = m.state.save(); err != nil {
		util.Debug("Error saving sponsorship state: %s", err)
	}
}
