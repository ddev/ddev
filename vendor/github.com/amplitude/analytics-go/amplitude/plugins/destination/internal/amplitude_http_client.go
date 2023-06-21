package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/amplitude/analytics-go/amplitude/types"
)

type AmplitudeHTTPClient interface {
	Send(payload AmplitudePayload) AmplitudeResponse
}

func NewAmplitudeHTTPClient(
	serverURL string, options AmplitudePayloadOptions, logger types.Logger, connectionTimeout time.Duration,
) AmplitudeHTTPClient {
	var payloadOptions *AmplitudePayloadOptions
	if options != (AmplitudePayloadOptions{}) {
		payloadOptions = &options
	}

	return &amplitudeHTTPClient{
		serverURL:      serverURL,
		logger:         logger,
		payloadOptions: payloadOptions,
		httpClient: &http.Client{
			Timeout: connectionTimeout,
		},
	}
}

type AmplitudePayloadOptions struct {
	MinIDLength int `json:"min_id_length,omitempty"`
}

type AmplitudePayload struct {
	APIKey  string                   `json:"api_key"`
	Events  []*types.Event           `json:"events"`
	Options *AmplitudePayloadOptions `json:"options,omitempty"`
}

type amplitudeHTTPClient struct {
	serverURL      string
	logger         types.Logger
	payloadOptions *AmplitudePayloadOptions
	httpClient     *http.Client
}

func (c *amplitudeHTTPClient) Send(payload AmplitudePayload) AmplitudeResponse {
	if len(payload.Events) == 0 {
		return AmplitudeResponse{}
	}

	payload.Options = c.payloadOptions
	payloadBytes, err := json.Marshal(payload)

	if err != nil {
		c.logger.Errorf("payload encoding failed: \n\tError: %w\n\tpayload: %+v", err, payload)

		return AmplitudeResponse{
			Err: fmt.Errorf("can't encode payload: %w", err),
		}
	}

	c.logger.Debugf("payloadBytes:\n\t%s", string(payloadBytes))

	request, err := http.NewRequest(http.MethodPost, c.serverURL, bytes.NewReader(payloadBytes))
	if err != nil {
		c.logger.Errorf("Building new request failed: \n\t%w", err)

		return AmplitudeResponse{
			Err: fmt.Errorf("can't build new request: %w", err),
		}
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "*/*")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return AmplitudeResponse{
			Err: fmt.Errorf("HTTP request failed: %w", err),
		}
	}

	defer func() {
		err := response.Body.Close()
		if err != nil {
			c.logger.Warnf("HTTP response, close body: %s", err)
		}
	}()

	c.logger.Infof("HTTP response code: %s", response.Status)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return AmplitudeResponse{
			Status: response.StatusCode,
			Err:    fmt.Errorf("can't read HTTP response body: %w", err),
		}
	}

	c.logger.Infof("HTTP response body: %s", string(body))

	var amplitudeResponse AmplitudeResponse
	if json.Valid(body) {
		_ = json.Unmarshal(body, &amplitudeResponse)
	}

	amplitudeResponse.Status = response.StatusCode

	return amplitudeResponse
}
