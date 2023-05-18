package remoteconfig

import (
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

// ShowTicker tips provided by the remote config to the user.
// TODO limit to once a day maybe see PR.
// TODO beautify output
func (c *remoteConfig) ShowTicker() {
	defer util.TimeTrack()()

	// Show ticker if not disabled.
	if !c.isTickerDisabled() {
		tips := len(c.remoteConfig.Messages.Ticker.Messages)
		if tips > 0 {
			tip := c.remoteConfig.Messages.Ticker.Last

			for {
				tip++
				if tip > tips {
					tip = 1
				}

				// TODO add conditions

				if tip == c.remoteConfig.Messages.Ticker.Last {
					break
				}

				util.Success("\n%s", c.remoteConfig.Messages.Ticker.Messages[tip-1])
			}

			c.remoteConfig.Messages.Ticker.Last = tip
			c.write()
		}
	}
}

// isTickerDisabled returns true if tips should not be shown to the user which
// can be achieved by setting the related global config or also via the remote
// config.
func (c *remoteConfig) isTickerDisabled() bool {
	return c.tickerDisabled || c.remoteConfig.Messages.Ticker.Disabled
}
