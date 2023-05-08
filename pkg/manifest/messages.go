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

// Shows infos and warnings provided in the manifest.json to the user.
func ShowMessages() {
	messages := GetManifest(24 * time.Hour).Messages

	version, err := semver.NewVersion(versionconstants.DdevVersion)
	if err != nil {
		util.Warning("Failed to parse the DDEV version `%s` into a semver.Version.", versionconstants.DdevVersion)
		return
	}

	for _, messagesOfType := range []messageTypes{
		{messageType: types.Warning, messages: messages.Warnings},
		{messageType: types.Info, messages: messages.Infos},
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
}
