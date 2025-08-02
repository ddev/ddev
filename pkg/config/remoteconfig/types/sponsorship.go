package types

import "time"

// SponsorshipData represents the structure of the sponsorship data JSON
type SponsorshipData struct {
	GitHubDDEVSponsorships      GitHubSponsorship         `json:"github_ddev_sponsorships"`
	GitHubRfaySponsorships      GitHubSponsorship         `json:"github_rfay_sponsorships"`
	MonthlyInvoicedSponsorships InvoicedSponsorship       `json:"monthly_invoiced_sponsorships"`
	AnnualInvoicedSponsorships  AnnualSponsorship         `json:"annual_invoiced_sponsorships"`
	PaypalSponsorships          int                       `json:"paypal_sponsorships"`
	TotalMonthlyAverageIncome   float64                   `json:"total_monthly_average_income"`
	Goal                        SponsorshipGoal           `json:"goal"`
	History                     []SponsorshipHistoryEntry `json:"history"`
	SponsorAppreciationMessage  string                    `json:"sponsor_appreciation_message"`
	UpdatedDateTime             time.Time                 `json:"updated_datetime"`
}

// GitHubSponsorship represents GitHub sponsorship data
type GitHubSponsorship struct {
	TotalMonthlySponsorship int            `json:"total_monthly_sponsorship"`
	TotalSponsors           int            `json:"total_sponsors"`
	SponsorsPerTier         map[string]int `json:"sponsors_per_tier"`
}

// InvoicedSponsorship represents monthly invoiced sponsorship data
type InvoicedSponsorship struct {
	TotalMonthlySponsorship int            `json:"total_monthly_sponsorship"`
	TotalSponsors           int            `json:"total_sponsors"`
	MonthlySponsorsPerTier  map[string]int `json:"monthly_sponsors_per_tier"`
}

// AnnualSponsorship represents annual sponsorship data
type AnnualSponsorship struct {
	TotalAnnualSponsorships      int            `json:"total_annual_sponsorships"`
	TotalSponsors                int            `json:"total_sponsors"`
	MonthlyEquivalentSponsorship int            `json:"monthly_equivalent_sponsorship"`
	AnnualSponsorsPerTier        map[string]int `json:"annual_sponsors_per_tier"`
}

// SponsorshipGoal represents the sponsorship goal and progress
type SponsorshipGoal struct {
	GoalAmount float64 `json:"goal_amount"`
	GoalTitle  string  `json:"goal_title"`
	GoalURL    string  `json:"goal_url"`
}

// SponsorshipHistoryEntry represents a single entry in the sponsorship history
type SponsorshipHistoryEntry struct {
	Date                      time.Time `json:"date"`
	TotalMonthlyAverageIncome float64   `json:"total_monthly_average_income"`
	TotalSponsors             int       `json:"total_sponsors"`
}

// SponsorshipManager defines the interface for managing sponsorship data
type SponsorshipManager interface {
	GetSponsorshipData() (*SponsorshipData, error)
	GetTotalMonthlyIncome() float64
	GetTotalSponsors() int
	IsDataStale() bool
}
