package remoteconfig

import (
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/ddev/ddev/pkg/config/remoteconfig/internal"
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
)

type messageTypes struct {
	messageType types.MessageType
	messages    []internal.Message
}

// Shows messages provided by the remote config to the user.
func (c *remoteConfig) ShowMessages() {
	defer util.TimeTrack()()

	// Show infos and warning.
	version, err := semver.NewVersion(versionconstants.DdevVersion)
	if err != nil {
		util.Warning("Failed to parse the DDEV version `%s` into a semver.Version.", versionconstants.DdevVersion)
		return
	}

	for _, messages := range []messageTypes{
		{messageType: types.Warning, messages: c.remoteConfig.Messages.Warnings},
		{messageType: types.Info, messages: c.remoteConfig.Messages.Infos},
	} {
		for _, message := range messages.messages {
			if message.Versions != "" {
				constraint, err := semver.NewConstraint(message.Versions)

				if err != nil {
					continue
				}

				if !constraint.Check(version) && constraint.String() != "" {
					continue
				}
			}

			switch messages.messageType {
			case types.Warning:
				util.Warning("\n%s", message.Message)
			default:
				util.Success("\n%s", message.Message)
			}
		}
	}
}

// ShowTicker shows ticker messages provided by the remote config to the user.
// TODO beautify output
func (c *remoteConfig) ShowTicker() {
	defer util.TimeTrack()()

	if c.showTickerMessage() {
		messages := len(c.remoteConfig.Messages.Ticker.Messages)
		if messages > 0 {
			message := c.state.LastTickerMessage

			for {
				message++
				if message > messages {
					message = 1
				}

				// TODO add conditions

				if message == c.state.LastTickerMessage {
					break
				}

				util.Success("\n%s", c.remoteConfig.Messages.Ticker.Messages[message-1])
			}

			c.state.LastTickerMessage = message
			c.state.LastTickerMessageAt = time.Now()
			if err := c.state.Save(); err != nil {
				util.Debug("Error while saving state: %s", err)
			}
		}
	}
}

// isTickerDisabled returns true if tips should not be shown to the user which
// can be achieved by setting the related global config or also via the remote
// config.
func (c *remoteConfig) isTickerDisabled() bool {
	return c.tickerDisabled || c.remoteConfig.Messages.Ticker.Disabled
}

// getTickerInterval returns the ticker interval. The processing order is
// defined as follows, the first defined value is returned:
//   - global config
//   - remote config
//   - const tickerInterval
func (c *remoteConfig) getTickerInterval() time.Duration {
	if c.tickerInterval > 0 {
		return time.Duration(c.tickerInterval) * time.Hour
	}

	if c.remoteConfig.Messages.Ticker.Interval > 0 {
		return time.Duration(c.remoteConfig.Messages.Ticker.Interval) * time.Hour
	}

	return time.Duration(tickerInterval) * time.Hour
}

// showTickerMessage returns true if the ticker is not disabled and the ticker
// interval has been elapsed.
func (c *remoteConfig) showTickerMessage() bool {
	return !c.isTickerDisabled() && c.state.LastTickerMessageAt.Add(c.getTickerInterval()).Before(time.Now())
}
