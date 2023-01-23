package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/amplitude/analytics-go/amplitude/types"
)

type AmplitudeResponseProcessor interface {
	Process(events []*types.StorageEvent, response AmplitudeResponse) AmplitudeProcessorResult
}

type AmplitudeProcessorResult struct {
	Code              int
	Message           string
	EventsForCallback []*types.StorageEvent
	EventsForRetry    []*types.StorageEvent
}

func NewAmplitudeResponseProcessor(options AmplitudeResponseProcessorOptions) AmplitudeResponseProcessor {
	return &amplitudeResponseProcessor{
		Options: options,
	}
}

type AmplitudeResponseProcessorOptions struct {
	MaxRetries             int
	RetryBaseInterval      time.Duration
	RetryThrottledInterval time.Duration
	Now                    func() time.Time
	Logger                 types.Logger
}

type amplitudeResponseProcessor struct {
	Options AmplitudeResponseProcessorOptions
}

func (p *amplitudeResponseProcessor) Process(events []*types.StorageEvent, response AmplitudeResponse) AmplitudeProcessorResult {
	responseStatus := response.normalizedStatus()

	var urlErr *url.Error
	isURLErr := errors.As(response.Err, &urlErr)

	isSuccess := response.Err == nil && responseStatus == http.StatusOK

	var result AmplitudeProcessorResult

	switch {
	case isSuccess:
		result = p.processSuccess(events, response)
	case (isURLErr && urlErr.Timeout()) || responseStatus == http.StatusRequestTimeout || responseStatus == http.StatusInternalServerError:
		result = p.processTimeout(events, response)
	case responseStatus == http.StatusRequestEntityTooLarge:
		result = p.processTooLargeRequest(events, response)
	case responseStatus == http.StatusBadRequest:
		result = p.processBadRequest(events, response)
	case responseStatus == http.StatusTooManyRequests:
		result = p.processTooManyRequests(events, response)
	default:
		result = p.processUnknownError(events, response)
	}

	if !isSuccess && len(result.EventsForCallback) > 0 {
		eventsJSON, _ := json.Marshal(result.EventsForCallback)
		p.Options.Logger.Errorf("%s: code=%d, events=%s", result.Message, result.Code, eventsJSON)
	}

	return result
}

func (p *amplitudeResponseProcessor) processSuccess(events []*types.StorageEvent, response AmplitudeResponse) AmplitudeProcessorResult {
	return AmplitudeProcessorResult{
		Code:              response.Code,
		Message:           "Event sent successfully.",
		EventsForCallback: events,
	}
}

func (p *amplitudeResponseProcessor) processTimeout(events []*types.StorageEvent, response AmplitudeResponse) AmplitudeProcessorResult {
	eventsForCallback := make([]*types.StorageEvent, 0, len(events))
	eventsForRetry := make([]*types.StorageEvent, 0, len(events))
	now := p.Options.Now()

	for _, event := range events {
		if event.RetryCount >= p.Options.MaxRetries {
			eventsForCallback = append(eventsForCallback, event)
		} else {
			event.RetryCount++
			event.RetryAt = now.Add(p.retryInterval(event.RetryCount))
			eventsForRetry = append(eventsForRetry, event)
		}
	}

	result := AmplitudeProcessorResult{
		Code:              response.Code,
		Message:           fmt.Sprintf("Event reached max retry times %d", p.Options.MaxRetries),
		EventsForCallback: eventsForCallback,
		EventsForRetry:    eventsForRetry,
	}

	return result
}

func (p *amplitudeResponseProcessor) processTooLargeRequest(events []*types.StorageEvent, response AmplitudeResponse) AmplitudeProcessorResult {
	if len(events) == 1 {
		result := AmplitudeProcessorResult{
			Code:              response.Code,
			Message:           response.Error,
			EventsForCallback: events,
		}

		return result
	}

	p.Options.Logger.Warnf("RequestEntityTooLarge: chunk size is reduced")

	return AmplitudeProcessorResult{
		Code:           response.Code,
		Message:        response.Error,
		EventsForRetry: events,
	}
}

func (p *amplitudeResponseProcessor) processBadRequest(events []*types.StorageEvent, response AmplitudeResponse) AmplitudeProcessorResult {
	switch {
	case strings.HasPrefix(response.Error, "Invalid API key:"):
		result := AmplitudeProcessorResult{
			Code:              response.Code,
			Message:           "Invalid API key",
			EventsForCallback: events,
		}

		return result
	case response.MissingField != "":
		result := AmplitudeProcessorResult{
			Code:              response.Code,
			Message:           fmt.Sprintf("Request missing required field %s", response.MissingField),
			EventsForCallback: events,
		}

		return result
	}

	invalidIndexes := response.invalidOrSilencedEventIndexes()
	eventsForCallback := make([]*types.StorageEvent, 0, len(events))
	eventsForRetry := make([]*types.StorageEvent, 0, len(events))

	for i, event := range events {
		if _, ok := invalidIndexes[i]; ok {
			eventsForCallback = append(eventsForCallback, event)
		} else {
			eventsForRetry = append(eventsForRetry, event)
		}
	}

	result := AmplitudeProcessorResult{
		Code:              response.Code,
		Message:           response.Error,
		EventsForCallback: eventsForCallback,
		EventsForRetry:    eventsForRetry,
	}

	return result
}

func (p *amplitudeResponseProcessor) processTooManyRequests(events []*types.StorageEvent, response AmplitudeResponse) AmplitudeProcessorResult {
	eventsForCallback := make([]*types.StorageEvent, 0, len(events))
	eventsForRetry := make([]*types.StorageEvent, 0, len(events))
	eventsForRetryDelay := make([]*types.StorageEvent, 0, len(events))
	now := p.Options.Now()

	for i, event := range events {
		if response.hasThrottledEventAtIndex(i) {
			if response.hasExceededDailyQuota(event.Event) {
				eventsForCallback = append(eventsForCallback, event)
			} else {
				event.RetryAt = now.Add(p.Options.RetryThrottledInterval)
				eventsForRetryDelay = append(eventsForRetryDelay, event)
			}
		} else {
			eventsForRetry = append(eventsForRetry, event)
		}
	}

	result := AmplitudeProcessorResult{
		Code:              response.Code,
		Message:           "Exceeded daily quota",
		EventsForCallback: eventsForCallback,
		EventsForRetry:    append(eventsForRetry, eventsForRetryDelay...),
	}

	return result
}

func (p *amplitudeResponseProcessor) processUnknownError(events []*types.StorageEvent, response AmplitudeResponse) AmplitudeProcessorResult {
	errMessage := response.Error
	if response.Err != nil {
		errMessage = response.Err.Error()
	}

	if errMessage == "" {
		errMessage = "Unknown error"
	}

	result := AmplitudeProcessorResult{
		Code:              response.Code,
		Message:           errMessage,
		EventsForCallback: events,
	}

	return result
}

func (p *amplitudeResponseProcessor) retryInterval(retries int) time.Duration {
	return p.Options.RetryBaseInterval * (1 << ((retries - 1) / 2))
}
