package before

import (
	"time"

	"github.com/google/uuid"

	"github.com/amplitude/analytics-go/amplitude/constants"
	"github.com/amplitude/analytics-go/amplitude/types"
)

// ContextPlugin is the default Before plugin that add library info to event.
// It also sets event default timestamp and insertID if not set elsewhere.
type ContextPlugin struct {
	contextString string
}

func NewContextPlugin() types.BeforePlugin {
	return &ContextPlugin{
		contextString: constants.SdkLibrary + "/" + constants.SdkVersion,
	}
}

func (p *ContextPlugin) Name() string {
	return "context"
}

func (p *ContextPlugin) Type() types.PluginType {
	return types.PluginTypeBefore
}

func (p *ContextPlugin) Setup(types.Config) {
}

// Execute sets default timestamp and insertID if not set elsewhere
// It also adds SDK name and version to event library.
func (p *ContextPlugin) Execute(event *types.Event) *types.Event {
	if event.Time == 0 {
		event.Time = time.Now().UnixMilli()
	}

	if event.InsertID == "" {
		event.InsertID = uuid.NewString()
	}

	event.Library = p.contextString

	return event
}
