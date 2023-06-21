package amplitude

import (
	"sync"

	"github.com/amplitude/analytics-go/amplitude/internal"
)

type timeline struct {
	logger             Logger
	beforePlugins      []BeforePlugin
	enrichmentPlugins  []EnrichmentPlugin
	destinationPlugins []DestinationPlugin
	mu                 sync.RWMutex
}

func (t *timeline) Process(event *Event) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	event = t.applyBeforePlugins(event)
	if event == nil {
		return
	}

	event = t.applyEnrichmentPlugins(event)
	if event == nil {
		return
	}

	t.applyDestinationPlugins(event)
}

func (t *timeline) applyBeforePlugins(event *Event) *Event {
	result := event

	for _, plugin := range t.beforePlugins {
		result = t.executeBeforePlugin(plugin, result)
		if result == nil {
			return nil
		}
	}

	return result
}

func (t *timeline) applyEnrichmentPlugins(event *Event) *Event {
	result := event

	for _, plugin := range t.enrichmentPlugins {
		result = t.executeEnrichmentPlugin(plugin, result)
		if result == nil {
			return nil
		}
	}

	return result
}

func (t *timeline) applyDestinationPlugins(event *Event) {
	var wg sync.WaitGroup

	for _, plugin := range t.destinationPlugins {
		clone := event.Clone()

		wg.Add(1)

		go t.executeDestinationPlugin(plugin, &clone, &wg)
	}

	wg.Wait()
}

func (t *timeline) executeBeforePlugin(plugin BeforePlugin, event *Event) (result *Event) {
	return plugin.Execute(event)
}

func (t *timeline) executeEnrichmentPlugin(plugin EnrichmentPlugin, event *Event) (result *Event) {
	return plugin.Execute(event)
}

func (t *timeline) executeDestinationPlugin(plugin DestinationPlugin, event *Event, wg *sync.WaitGroup) {
	defer wg.Done()
	plugin.Execute(event)
}

func (t *timeline) AddPlugin(plugin Plugin) Plugin {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch plugin.Type() {
	case PluginTypeBefore:
		plugin, ok := plugin.(BeforePlugin)
		if !ok {
			t.logger.Errorf("Plugin %s doesn't implement Before interface", plugin.Name())
		}

		wrapper := &internal.SafeBeforePluginWrapper{Plugin: plugin, Logger: t.logger}
		t.beforePlugins = append(t.beforePlugins, wrapper)

		return wrapper
	case PluginTypeEnrichment:
		plugin, ok := plugin.(EnrichmentPlugin)
		if !ok {
			t.logger.Errorf("Plugin %s doesn't implement Enrichment interface", plugin.Name())
		}

		wrapper := &internal.SafeEnrichmentPluginWrapper{Plugin: plugin, Logger: t.logger}
		t.enrichmentPlugins = append(t.enrichmentPlugins, wrapper)

		return wrapper
	case PluginTypeDestination:
		plugin, ok := plugin.(DestinationPlugin)
		if !ok {
			t.logger.Errorf("Plugin %s doesn't implement Destination interface", plugin.Name())
		}

		var wrapper DestinationPlugin
		if extendedPlugin, ok := plugin.(ExtendedDestinationPlugin); ok {
			wrapper = &internal.SafeExtendedDestinationPluginWrapper{Plugin: extendedPlugin, Logger: t.logger}
		} else {
			wrapper = &internal.SafeDestinationPluginWrapper{Plugin: plugin, Logger: t.logger}
		}

		t.destinationPlugins = append(t.destinationPlugins, wrapper)

		return wrapper
	default:
		t.logger.Errorf("Plugin %s - unknown type %s", plugin.Name(), plugin.Type())

		return nil
	}
}

func (t *timeline) RemovePlugin(pluginName string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i := len(t.beforePlugins) - 1; i >= 0; i-- {
		if t.beforePlugins[i].Name() == pluginName {
			t.beforePlugins = append(t.beforePlugins[:i], t.beforePlugins[i+1:]...)
		}
	}

	for i := len(t.enrichmentPlugins) - 1; i >= 0; i-- {
		if t.enrichmentPlugins[i].Name() == pluginName {
			t.enrichmentPlugins = append(t.enrichmentPlugins[:i], t.enrichmentPlugins[i+1:]...)
		}
	}

	for i := len(t.destinationPlugins) - 1; i >= 0; i-- {
		if t.destinationPlugins[i].Name() == pluginName {
			t.destinationPlugins = append(t.destinationPlugins[:i], t.destinationPlugins[i+1:]...)
		}
	}
}

func (t *timeline) Flush() {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var wg sync.WaitGroup

	for _, plugin := range t.destinationPlugins {
		if plugin, ok := plugin.(ExtendedDestinationPlugin); ok {
			wg.Add(1)

			go func(plugin ExtendedDestinationPlugin) {
				defer wg.Done()
				plugin.Flush()
			}(plugin)
		}
	}

	wg.Wait()
}

func (t *timeline) Shutdown() {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var wg sync.WaitGroup

	for _, plugin := range t.destinationPlugins {
		if plugin, ok := plugin.(ExtendedDestinationPlugin); ok {
			wg.Add(1)

			go func(plugin ExtendedDestinationPlugin) {
				defer wg.Done()
				plugin.Shutdown()
			}(plugin)
		}
	}

	wg.Wait()
}
