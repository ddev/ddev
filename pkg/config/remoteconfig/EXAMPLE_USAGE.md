# Remote Config Refactoring - Usage Examples

## Existing Remote Config (Unchanged)

The existing remote config functionality continues to work exactly as before:

```go
import "github.com/ddev/ddev/pkg/config/remoteconfig"

// Initialize global remote config (existing usage)
remoteConfig := remoteconfig.InitGlobal(
    remoteconfig.Config{
        Local: remoteconfig.Local{Path: "/path/to/local"},
        Remote: remoteconfig.Remote{
            Owner: "ddev",
            Repo: "remote-config", 
            Filepath: "remote-config.jsonc",
        },
    },
    stateManager,
    isInternetActive,
)

// Use remote config (existing usage)
remoteconfig.GetGlobal().ShowNotifications()
remoteconfig.GetGlobal().ShowTicker()
```

## New Sponsorship Data Functionality

```go
import "github.com/ddev/ddev/pkg/config/remoteconfig"

// Initialize sponsorship manager
sponsorshipManager := remoteconfig.InitGlobalSponsorship(
    "/path/to/local", 
    stateManager, 
    isInternetActive,
)

// Get sponsorship data
data, err := sponsorshipManager.GetSponsorshipData()
if err != nil {
    log.Printf("Error getting sponsorship data: %v", err)
}

// Get total monthly income from all sources
totalIncome := sponsorshipManager.GetTotalMonthlyIncome()
fmt.Printf("Total monthly income: $%d\n", totalIncome)

// Get total number of sponsors
totalSponsors := sponsorshipManager.GetTotalSponsors()
fmt.Printf("Total sponsors: %d\n", totalSponsors)

// Check if data needs updating
if sponsorshipManager.IsDataStale() {
    fmt.Println("Sponsorship data is stale and will be updated")
}

// Access specific sponsorship details
fmt.Printf("GitHub DDEV sponsorships: $%d/month from %d sponsors\n", 
    data.GitHubDDEVSponsorships.TotalMonthlySponsorship,
    data.GitHubDDEVSponsorships.TotalSponsors)
```

## Generic JSONC Downloader (For Other Use Cases)

```go
import (
    "context"
    "github.com/ddev/ddev/pkg/config/remoteconfig/downloader"
    "github.com/ddev/ddev/pkg/github"
)

// Create a generic downloader for any JSONC file
genericDownloader := downloader.NewGitHubJSONCDownloader(
    "owner",
    "repo", 
    "path/to/file.json",
    github.RepositoryContentGetOptions{Ref: "main"},
)

// Define your custom struct
type MyCustomData struct {
    Field1 string `json:"field1"`
    Field2 int    `json:"field2"`
}

// Download and unmarshal
var customData MyCustomData
ctx := context.Background()
err := genericDownloader.Download(ctx, &customData)
if err != nil {
    log.Printf("Error downloading custom data: %v", err)
}
```

## Architecture Overview

- **Generic JSONC Downloader**: `pkg/config/remoteconfig/downloader/jsonc_downloader.go`
- **Sponsorship Types**: `pkg/config/remoteconfig/types/sponsorship.go`  
- **Sponsorship Manager**: `pkg/config/remoteconfig/sponsorship.go`
- **Factory Functions**: `pkg/config/remoteconfig/global.go`
- **Backward Compatible Storage**: `pkg/config/remoteconfig/storage/github_storage.go` (refactored to use generic downloader)

The refactoring maintains 100% backward compatibility while enabling flexible downloading of arbitrary JSONC files from GitHub repositories.