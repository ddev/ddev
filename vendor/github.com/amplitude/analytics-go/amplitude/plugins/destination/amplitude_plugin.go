package destination

import (
	"net/http"
	"sync"
	"time"

	"github.com/amplitude/analytics-go/amplitude/plugins/destination/internal"

	"github.com/amplitude/analytics-go/amplitude/types"
)

func NewAmplitudePlugin() types.ExtendedDestinationPlugin {
	return &amplitudePlugin{}
}

type amplitudePlugin struct {
	config            types.Config
	storage           types.EventStorage
	client            internal.AmplitudeHTTPClient
	responseProcessor internal.AmplitudeResponseProcessor
	messageChannel    chan amplitudeMessage
	messageChannelMu  sync.RWMutex

	chunkSize   int
	sizeDivider int
}

type amplitudeMessage struct {
	event *types.Event
	wg    *sync.WaitGroup
}

func (p *amplitudePlugin) Name() string {
	return "amplitude"
}

func (p *amplitudePlugin) Type() types.PluginType {
	return types.PluginTypeDestination
}

func (p *amplitudePlugin) Setup(config types.Config) {
	p.config = config

	p.sizeDivider = config.FlushSizeDivider
	if p.sizeDivider < 1 {
		p.sizeDivider = 1
	}

	p.chunkSize = config.FlushQueueSize / p.sizeDivider
	if p.chunkSize < 1 {
		p.chunkSize = 1
	}

	p.storage = config.StorageFactory()
	p.messageChannel = make(chan amplitudeMessage, config.MaxStorageCapacity)

	if p.client == nil {
		p.client = internal.NewAmplitudeHTTPClient(
			config.ServerURL,
			internal.AmplitudePayloadOptions{MinIDLength: config.MinIDLength},
			config.Logger,
			config.ConnectionTimeout,
		)
	}

	if p.responseProcessor == nil {
		p.responseProcessor = internal.NewAmplitudeResponseProcessor(internal.AmplitudeResponseProcessorOptions{
			MaxRetries:             config.FlushMaxRetries,
			RetryBaseInterval:      config.RetryBaseInterval,
			RetryThrottledInterval: config.RetryThrottledInterval,
			Now:                    time.Now,
			Logger:                 config.Logger,
		})
	}

	go p.start(p.messageChannel)
}

func (p *amplitudePlugin) start(messageChannel <-chan amplitudeMessage) {
	defer func() {
		if r := recover(); r != nil {
			p.config.Logger.Errorf("Panic in AmplitudePlugin: %s", r)
		}
	}()

	defer func() {
		p.messageChannelMu.Lock()
		defer p.messageChannelMu.Unlock()

		p.messageChannel = nil
	}()

	autoFlushTicker := time.NewTicker(p.config.FlushInterval)
	defer autoFlushTicker.Stop()

	for {
		select {
		case <-autoFlushTicker.C:
			p.sendEventsFromStorage(nil)
		case message, ok := <-messageChannel:
			if !ok {
				return
			}

			if message.wg != nil {
				p.sendEventsFromStorage(message.wg)
				autoFlushTicker.Reset(p.config.FlushInterval)
			} else {
				p.storage.PushNew(&types.StorageEvent{Event: message.event})

				if p.storage.Count(time.Now()) >= p.chunkSize {
					p.sendEventsFromStorage(nil)
					autoFlushTicker.Reset(p.config.FlushInterval)
				}
			}
		}
	}
}

// Execute processes the event with plugins added to the destination plugin.
// Then pushed the event to storage waiting to be sent.
func (p *amplitudePlugin) Execute(event *types.Event) {
	if !IsValidAmplitudeEvent(event) {
		p.config.Logger.Errorf("Invalid event, EventType and either UserID or DeviceID cannot be empty: \n\t%+v", event)
	}

	p.messageChannelMu.RLock()
	defer p.messageChannelMu.RUnlock()

	select {
	case p.messageChannel <- amplitudeMessage{
		event: event,
		wg:    nil,
	}:
	default:
	}
}

func (p *amplitudePlugin) Flush() {
	p.messageChannelMu.RLock()
	defer p.messageChannelMu.RUnlock()

	if p.messageChannel == nil {
		return
	}

	p.flush(p.messageChannel)
}

func (p *amplitudePlugin) flush(messageChannel chan<- amplitudeMessage) {
	var flushWaitGroup sync.WaitGroup

	flushWaitGroup.Add(1)

	select {
	case messageChannel <- amplitudeMessage{
		event: nil,
		wg:    &flushWaitGroup,
	}:
	default:
		flushWaitGroup.Done()
	}

	flushWaitGroup.Wait()
}

func (p *amplitudePlugin) sendEventsFromStorage(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}

	for {
		storageEvents := p.storage.Pull(p.chunkSize, time.Now())
		if len(storageEvents) == 0 {
			break
		}

		events := make([]*types.Event, len(storageEvents))
		for i, storageEvent := range storageEvents {
			events[i] = storageEvent.Event
		}

		response := p.client.Send(internal.AmplitudePayload{
			APIKey: p.config.APIKey,
			Events: events,
		})

		result := p.responseProcessor.Process(storageEvents, response)

		if result.Code == http.StatusRequestEntityTooLarge && len(result.EventsForRetry) > 0 {
			p.reduceChunkSize()
		}

		if len(result.EventsForRetry) > 0 {
			p.storage.ReturnBack(result.EventsForRetry...)
		}

		executeCallback := p.config.ExecuteCallback
		if executeCallback != nil && len(result.EventsForCallback) > 0 {
			go func() {
				for _, event := range result.EventsForCallback {
					executeCallback(types.ExecuteResult{
						PluginName: p.Name(),
						Event:      event.Event,
						Code:       result.Code,
						Message:    result.Message,
					})
				}
			}()
		}
	}
}

func (p *amplitudePlugin) Shutdown() {
	p.messageChannelMu.Lock()

	if p.messageChannel == nil {
		p.messageChannelMu.Unlock()

		return
	}

	messageChannel := p.messageChannel
	p.messageChannel = nil
	p.messageChannelMu.Unlock()

	p.flush(messageChannel)
	close(messageChannel)
}

func (p *amplitudePlugin) reduceChunkSize() {
	p.sizeDivider++

	p.chunkSize = p.config.FlushQueueSize / p.sizeDivider
	if p.chunkSize < 1 {
		p.chunkSize = 1
	}
}

func (p *amplitudePlugin) SetHTTPClient(client internal.AmplitudeHTTPClient) {
	p.client = client
}

func (p *amplitudePlugin) SetResponseProcessor(responseProcessor internal.AmplitudeResponseProcessor) {
	p.responseProcessor = responseProcessor
}

func IsValidAmplitudeEvent(event *types.Event) bool {
	userID := event.EventOptions.UserID
	if userID == "" {
		userID = event.UserID
	}

	deviceID := event.EventOptions.DeviceID
	if deviceID == "" {
		deviceID = event.DeviceID
	}

	return event.EventType != "" && (userID != "" || deviceID != "")
}
