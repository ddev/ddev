package manifest

import (
	"time"

	"github.com/Masterminds/semver/v3"
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
	manifest := GetManifest(24 * time.Hour)

	// Show infos and warning.
	version, err := semver.NewVersion(versionconstants.DdevVersion)
	if err != nil {
		util.Warning("Failed to parse the DDEV version `%s` into a semver.Version.", versionconstants.DdevVersion)
		return
	}

	for _, messagesOfType := range []messageTypes{
		{messageType: types.Warning, messages: manifest.Messages.Warnings},
		{messageType: types.Info, messages: manifest.Messages.Infos},
	} {
		for _, message := range messagesOfType.messages {
			constraint, err := semver.NewConstraint(message.Versions)
			if err != nil {
				continue
			}

			if !constraint.Check(version) && constraint.String() != "" {
				continue
			}

			switch messagesOfType.messageType {
			case types.Warning:
				util.Warning(message.Message)
			default:
				util.Success(message.Message)
			}
		}
	}

	// Show tips.
	tips := len(manifest.Messages.Tips.Messages)
	if tips > 0 {
		manifest.Messages.Tips.Last++
		if manifest.Messages.Tips.Last >= tips {
			manifest.Messages.Tips.Last = 0
		}

		util.Success(manifest.Messages.Tips.Messages[manifest.Messages.Tips.Last])

		UpdateManifest(manifest)
	}
}
