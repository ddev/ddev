package types

import (
	"time"
)

type EventOptions struct {
	UserID             string             `json:"user_id,omitempty"`
	DeviceID           string             `json:"device_id,omitempty"`
	Time               int64              `json:"time,omitempty"`
	InsertID           string             `json:"insert_id,omitempty"`
	Library            string             `json:"library,omitempty"`
	LocationLat        float64            `json:"location_lat,omitempty"`
	LocationLng        float64            `json:"location_lng,omitempty"`
	AppVersion         string             `json:"app_version,omitempty"`
	VersionName        string             `json:"version_name,omitempty"`
	Platform           string             `json:"platform,omitempty"`
	OSName             string             `json:"os_name,omitempty"`
	OSVersion          string             `json:"os_version,omitempty"`
	DeviceBrand        string             `json:"device_brand,omitempty"`
	DeviceManufacturer string             `json:"device_manufacturer,omitempty"`
	DeviceModel        string             `json:"device_model,omitempty"`
	Carrier            string             `json:"carrier,omitempty"`
	Country            string             `json:"country,omitempty"`
	Region             string             `json:"region,omitempty"`
	City               string             `json:"city,omitempty"`
	DMA                string             `json:"dma,omitempty"`
	IDFA               string             `json:"idfa,omitempty"`
	IDFV               string             `json:"idfv,omitempty"`
	ADID               string             `json:"adid,omitempty"`
	AndroidID          string             `json:"android_id,omitempty"`
	Language           string             `json:"language,omitempty"`
	IP                 string             `json:"ip,omitempty"`
	Price              float64            `json:"price,omitempty"`
	Quantity           int                `json:"quantity,omitempty"`
	Revenue            float64            `json:"revenue,omitempty"`
	ProductID          string             `json:"productId,omitempty"`
	RevenueType        string             `json:"revenueType,omitempty"`
	EventID            int                `json:"event_id,omitempty"`
	SessionID          int                `json:"session_id,omitempty"`
	PartnerID          string             `json:"partner_id,omitempty"`
	Plan               *Plan              `json:"plan,omitempty"`
	IngestionMetadata  *IngestionMetadata `json:"ingestion_metadata,omitempty"`
}

func (eo *EventOptions) SetTime(time time.Time) {
	eo.Time = time.UnixMilli()
}

func (eo *EventOptions) Clone() *EventOptions {
	if eo == nil {
		return nil
	}

	clone := *eo

	if eo.Plan != nil {
		plan := *eo.Plan
		clone.Plan = &plan
	}

	if eo.IngestionMetadata != nil {
		ingestionMetadata := *eo.IngestionMetadata
		clone.IngestionMetadata = &ingestionMetadata
	}

	return &clone
}
