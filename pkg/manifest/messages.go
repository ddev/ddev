package manifest

import (
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/manifest/types"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
)

type messageTypes struct {
	messageType types.MessageType
	messages    []types.Message
}

// Shows messages provided in the manifest.json to the user.
func ShowMessages() {
	defer util.TimeTrack(time.Now(), "ShowMessages()")()

	updateInterval := globalconfig.DdevGlobalConfig.ManifestUpdateInterval
	if updateInterval <= 0 {
		updateInterval = 24
	}

	manifest := NewManifest(time.Duration(updateInterval) * time.Hour)

	// Show infos and warning.
	version, err := semver.NewVersion(versionconstants.DdevVersion)
	if err != nil {
		util.Warning("Failed to parse the DDEV version `%s` into a semver.Version.", versionconstants.DdevVersion)
		return
	}

	for _, messages := range []messageTypes{
		{messageType: types.Warning, messages: manifest.Manifest.Messages.Warnings},
		{messageType: types.Info, messages: manifest.Manifest.Messages.Infos},
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
				util.Warning(message.Message)
			default:
				util.Success(message.Message)
			}
		}
	}

	// Show tips.
	tips := len(manifest.Manifest.Messages.Tips.Messages)
	if tips > 0 {
		manifest.Manifest.Messages.Tips.Last++
		if manifest.Manifest.Messages.Tips.Last > tips {
			manifest.Manifest.Messages.Tips.Last = 1
		}

		util.Success(manifest.Manifest.Messages.Tips.Messages[manifest.Manifest.Messages.Tips.Last-1])

		manifest.Write()
	}
}
