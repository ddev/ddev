package internal

import (
	"net/http"

	"github.com/amplitude/analytics-go/amplitude/types"
)

type AmplitudeResponse struct {
	Status int   `json:"-"`
	Err    error `json:"-"`

	Code  int    `json:"code"`
	Error string `json:"error"`

	MissingField               string           `json:"missing_field"`
	EventsWithInvalidFields    map[string][]int `json:"events_with_invalid_fields"`
	EventsWithMissingFields    map[string][]int `json:"events_with_missing_fields"`
	EventsWithInvalidIDLengths map[string][]int `json:"events_with_invalid_id_lengths"`
	SilencedEvents             []int            `json:"silenced_events"`

	ThrottledEvents           []int          `json:"throttled_events"`
	ExceededDailyQuotaUsers   map[string]int `json:"exceeded_daily_quota_users"`
	ExceededDailyQuotaDevices map[string]int `json:"exceeded_daily_quota_devices"`
}

func (r AmplitudeResponse) normalizedStatus() int {
	switch {
	case 200 <= r.Status && r.Status < 300:
		return http.StatusOK
	case r.Status == http.StatusTooManyRequests:
		return r.Status
	case r.Status == http.StatusRequestEntityTooLarge:
		return r.Status
	case r.Status == http.StatusRequestTimeout:
		return r.Status
	case 400 <= r.Status && r.Status < 500:
		return http.StatusBadRequest
	case 500 <= r.Status:
		return http.StatusInternalServerError
	default:
		return -1
	}
}

func (r AmplitudeResponse) invalidOrSilencedEventIndexes() map[int]struct{} {
	result := make(map[int]struct{})

	for _, indexes := range r.EventsWithMissingFields {
		for _, index := range indexes {
			result[index] = struct{}{}
		}
	}

	for _, indexes := range r.EventsWithInvalidFields {
		for _, index := range indexes {
			result[index] = struct{}{}
		}
	}

	for _, indexes := range r.EventsWithInvalidIDLengths {
		for _, index := range indexes {
			result[index] = struct{}{}
		}
	}

	for _, index := range r.SilencedEvents {
		result[index] = struct{}{}
	}

	return result
}

func (r AmplitudeResponse) hasThrottledEventAtIndex(eventIndex int) bool {
	for _, index := range r.ThrottledEvents {
		if eventIndex == index {
			return true
		}
	}

	return false
}

func (r AmplitudeResponse) hasExceededDailyQuota(event *types.Event) bool {
	if _, ok := r.ExceededDailyQuotaUsers[event.UserID]; ok {
		return true
	}

	if _, ok := r.ExceededDailyQuotaDevices[event.DeviceID]; ok {
		return true
	}

	return false
}
