package amplitude

import (
	"github.com/amplitude/analytics-go/amplitude/constants"
	"github.com/amplitude/analytics-go/amplitude/internal"
	"github.com/amplitude/analytics-go/amplitude/loggers"
	"github.com/amplitude/analytics-go/amplitude/plugins/before"
	"github.com/amplitude/analytics-go/amplitude/plugins/destination"
	"github.com/amplitude/analytics-go/amplitude/storages"
)

type Client interface {
	Track(event Event)
	Identify(identify Identify, eventOptions EventOptions)
	GroupIdentify(groupType string, groupName string, identify Identify, eventOptions EventOptions)
	SetGroup(groupType string, groupName []string, eventOptions EventOptions)
	Revenue(revenue Revenue, eventOptions EventOptions)

	Flush()
	Shutdown()

	Add(plugin Plugin)
	Remove(pluginName string)

	Config() Config
}

func NewClient(config Config) Client {
	setConfigDefaultValues(&config)
	setSafeExecuteCallback(&config)
	config.Logger.Debugf("Client initialized")

	client := &client{
		config:   config,
		optOut:   internal.NewAtomicBool(config.OptOut),
		timeline: &timeline{logger: config.Logger},
	}

	client.Add(destination.NewAmplitudePlugin())
	client.Add(before.NewContextPlugin())

	return client
}

type client struct {
	config   Config
	timeline *timeline
	optOut   *internal.AtomicBool
}

func (c *client) Config() Config {
	return c.config
}

// Track processes and sends the given event object.
func (c *client) Track(event Event) {
	if !c.enabled() {
		return
	}

	if event.Plan == nil {
		event.Plan = c.config.Plan
	}

	if event.IngestionMetadata == nil {
		event.IngestionMetadata = c.config.IngestionMetadata
	}

	if event.EventOptions.UserID == "" && event.UserID != "" {
		event.EventOptions.UserID = event.UserID
	}

	if event.EventOptions.DeviceID == "" && event.DeviceID != "" {
		event.EventOptions.DeviceID = event.DeviceID
	}

	c.config.Logger.Debugf("Track event: \n\t%+v", event)
	c.timeline.Process(&event)
}

// Identify sends an identify event to update user Properties.
func (c *client) Identify(identify Identify, eventOptions EventOptions) {
	if !c.enabled() {
		return
	}

	validateErrors, validateWarnings := identify.Validate()

	for _, validateWarning := range validateWarnings {
		c.config.Logger.Warnf("Identify: %s", validateWarning)
	}

	if len(validateErrors) > 0 {
		for _, validateError := range validateErrors {
			c.config.Logger.Errorf("Identify: %s", validateError)
		}
	} else {
		identifyEvent := Event{
			EventType:      constants.IdentifyEventType,
			EventOptions:   eventOptions,
			UserProperties: identify.Properties,
		}

		c.Track(identifyEvent)
	}
}

// GroupIdentify sends a group identify event to update group Properties.
func (c *client) GroupIdentify(groupType string, groupName string, identify Identify, eventOptions EventOptions) {
	if !c.enabled() {
		return
	}

	validateErrors, validateWarnings := identify.Validate()

	for _, validateWarning := range validateWarnings {
		c.config.Logger.Warnf("Identify: %s", validateWarning)
	}

	if len(validateErrors) > 0 {
		for _, validateError := range validateErrors {
			c.config.Logger.Errorf("Invalid Identify: %s", validateError)
		}
	} else {
		groupIdentifyEvent := Event{
			EventType:       constants.GroupIdentifyEventType,
			EventOptions:    eventOptions,
			Groups:          map[string][]string{groupType: {groupName}},
			GroupProperties: identify.Properties,
		}

		c.Track(groupIdentifyEvent)
	}
}

// Revenue sends a revenue event with revenue info in eventProperties.
func (c *client) Revenue(revenue Revenue, eventOptions EventOptions) {
	if !c.enabled() {
		return
	}

	if validateErrors := revenue.Validate(); len(validateErrors) > 0 {
		for _, validateError := range validateErrors {
			c.config.Logger.Errorf("Invalid Revenue: %s", validateError)
		}
	} else {
		revenueEvent := Event{
			EventType:    constants.RevenueEventType,
			EventOptions: eventOptions,
			EventProperties: map[string]interface{}{
				constants.RevenueProductID:  revenue.ProductID,
				constants.RevenueQuantity:   revenue.Quantity,
				constants.RevenuePrice:      revenue.Price,
				constants.RevenueType:       revenue.RevenueType,
				constants.RevenueReceipt:    revenue.Receipt,
				constants.RevenueReceiptSig: revenue.ReceiptSig,
				constants.DefaultRevenue:    revenue.Revenue,
			},
		}
		c.Track(revenueEvent)
	}
}

// SetGroup sends an identify event to put a user in group(s)
// by setting group type and group name as user property for a user.
func (c *client) SetGroup(groupType string, groupName []string, eventOptions EventOptions) {
	if !c.enabled() {
		return
	}

	identify := Identify{}
	identify.Set(groupType, groupName)
	c.Identify(identify, eventOptions)
}

// Flush flushes all events waiting to be sent in the buffer.
func (c *client) Flush() {
	c.timeline.Flush()
}

// Add adds the plugin object to client instance.
// Events tracked by this client instance will be processed by instances' plugins.
func (c *client) Add(plugin Plugin) {
	safePluginWrapper := c.timeline.AddPlugin(plugin)
	if safePluginWrapper != nil {
		safePluginWrapper.Setup(c.config)
	}
}

// Remove removes the plugin object from client instance.
func (c *client) Remove(pluginName string) {
	c.timeline.RemovePlugin(pluginName)
}

// Shutdown shuts the client instance down from accepting new events.
func (c *client) Shutdown() {
	c.optOut.Set()

	c.config.Logger.Debugf("Client shutdown")
	c.timeline.Shutdown()
}

func (c *client) enabled() bool {
	return !c.optOut.IsSet()
}

func setConfigDefaultValues(config *Config) {
	if config.FlushInterval == 0 {
		config.FlushInterval = constants.DefaultConfig.FlushInterval
	}

	if config.FlushQueueSize == 0 {
		config.FlushQueueSize = constants.DefaultConfig.FlushQueueSize
	}

	if config.FlushSizeDivider == 0 {
		config.FlushSizeDivider = constants.DefaultConfig.FlushSizeDivider
	}

	if config.FlushMaxRetries == 0 {
		config.FlushMaxRetries = constants.DefaultConfig.FlushMaxRetries
	}

	if config.ConnectionTimeout == 0 {
		config.ConnectionTimeout = constants.DefaultConfig.ConnectionTimeout
	}

	if config.MaxStorageCapacity == 0 {
		config.MaxStorageCapacity = constants.DefaultConfig.MaxStorageCapacity
	}

	if config.RetryBaseInterval == 0 {
		config.RetryBaseInterval = constants.DefaultConfig.RetryBaseInterval
	}

	if config.RetryThrottledInterval == 0 {
		config.RetryThrottledInterval = constants.DefaultConfig.RetryThrottledInterval
	}

	if config.Logger == nil {
		config.Logger = loggers.NewDefaultLogger()
	}

	if config.StorageFactory == nil {
		config.StorageFactory = storages.NewInMemoryEventStorage
	}

	if config.ServerZone == "" {
		config.ServerZone = constants.DefaultConfig.ServerZone
	}

	if config.ServerURL == "" {
		if config.UseBatch {
			config.ServerURL = constants.ServerBatchURLs[config.ServerZone]
		} else {
			config.ServerURL = constants.ServerURLs[config.ServerZone]
		}
	}
}

func setSafeExecuteCallback(config *Config) {
	executeCallback := config.ExecuteCallback
	if executeCallback == nil {
		return
	}

	config.ExecuteCallback = func(result ExecuteResult) {
		defer func() {
			if r := recover(); r != nil {
				config.Logger.Errorf("Panic in callback: %s", r)
			}
		}()

		executeCallback(result)
	}
}
