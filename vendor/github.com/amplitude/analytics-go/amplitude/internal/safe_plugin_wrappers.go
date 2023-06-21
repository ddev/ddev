package internal

import (
	"github.com/amplitude/analytics-go/amplitude/types"
)

type SafeBeforePluginWrapper struct {
	Plugin        types.BeforePlugin
	Logger        types.Logger
	isInitialized bool
}

func (w *SafeBeforePluginWrapper) Name() string {
	return w.Plugin.Name()
}

func (w *SafeBeforePluginWrapper) Type() types.PluginType {
	return w.Plugin.Type()
}

func (w *SafeBeforePluginWrapper) Setup(config types.Config) {
	defer func() {
		if r := recover(); r != nil {
			w.Logger.Errorf("Panic in plugin %s.Setup: %s", w.Plugin.Name(), r)
		}
	}()

	w.Plugin.Setup(config)
	w.isInitialized = true
}

func (w *SafeBeforePluginWrapper) Execute(event *types.Event) (result *types.Event) {
	if !w.isInitialized {
		return event
	}

	defer func() {
		if r := recover(); r != nil {
			w.Logger.Errorf("Panic in plugin %s.Execute: %s", w.Plugin.Name(), r)

			result = event
		}
	}()

	return w.Plugin.Execute(event)
}

type SafeEnrichmentPluginWrapper struct {
	Plugin        types.EnrichmentPlugin
	Logger        types.Logger
	isInitialized bool
}

func (w *SafeEnrichmentPluginWrapper) Name() string {
	return w.Plugin.Name()
}

func (w *SafeEnrichmentPluginWrapper) Type() types.PluginType {
	return w.Plugin.Type()
}

func (w *SafeEnrichmentPluginWrapper) Setup(config types.Config) {
	defer func() {
		if r := recover(); r != nil {
			w.Logger.Errorf("Panic in plugin %s.Setup: %s", w.Plugin.Name(), r)
		}
	}()

	w.Plugin.Setup(config)
	w.isInitialized = true
}

func (w *SafeEnrichmentPluginWrapper) Execute(event *types.Event) (result *types.Event) {
	if !w.isInitialized {
		return event
	}

	defer func() {
		if r := recover(); r != nil {
			w.Logger.Errorf("Panic in plugin %s.Execute: %s", w.Plugin.Name(), r)

			result = event
		}
	}()

	return w.Plugin.Execute(event)
}

type SafeDestinationPluginWrapper struct {
	Plugin        types.DestinationPlugin
	Logger        types.Logger
	isInitialized bool
}

func (w *SafeDestinationPluginWrapper) Name() string {
	return w.Plugin.Name()
}

func (w *SafeDestinationPluginWrapper) Type() types.PluginType {
	return w.Plugin.Type()
}

func (w *SafeDestinationPluginWrapper) Setup(config types.Config) {
	defer func() {
		if r := recover(); r != nil {
			w.Logger.Errorf("Panic in plugin %s.Setup: %s", w.Plugin.Name(), r)
		}
	}()

	w.Plugin.Setup(config)
	w.isInitialized = true
}

func (w *SafeDestinationPluginWrapper) Execute(event *types.Event) {
	if !w.isInitialized {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			w.Logger.Errorf("Panic in plugin %s.Execute: %s", w.Plugin.Name(), r)
		}
	}()

	w.Plugin.Execute(event)
}

type SafeExtendedDestinationPluginWrapper struct {
	Plugin        types.ExtendedDestinationPlugin
	Logger        types.Logger
	isInitialized bool
}

func (w *SafeExtendedDestinationPluginWrapper) Name() string {
	return w.Plugin.Name()
}

func (w *SafeExtendedDestinationPluginWrapper) Type() types.PluginType {
	return w.Plugin.Type()
}

func (w *SafeExtendedDestinationPluginWrapper) Setup(config types.Config) {
	defer func() {
		if r := recover(); r != nil {
			w.Logger.Errorf("Panic in plugin %s.Setup: %s", w.Plugin.Name(), r)
		}
	}()

	w.Plugin.Setup(config)
	w.isInitialized = true
}

func (w *SafeExtendedDestinationPluginWrapper) Execute(event *types.Event) {
	if !w.isInitialized {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			w.Logger.Errorf("Panic in plugin %s.Execute: %s", w.Plugin.Name(), r)
		}
	}()

	w.Plugin.Execute(event)
}

func (w *SafeExtendedDestinationPluginWrapper) Flush() {
	if !w.isInitialized {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			w.Logger.Errorf("Panic in plugin %s.Flush: %s", w.Plugin.Name(), r)
		}
	}()

	w.Plugin.Flush()
}

func (w *SafeExtendedDestinationPluginWrapper) Shutdown() {
	if !w.isInitialized {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			w.Logger.Errorf("Panic in plugin %s.Shutdown: %s", w.Plugin.Name(), r)
		}
	}()

	w.Plugin.Shutdown()
}
