package manifest

import (
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/ddev/ddev/pkg/manifest/internal"
	"github.com/ddev/ddev/pkg/manifest/types"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
)

type messageTypes struct {
	messageType types.MessageType
	messages    []internal.Message
}

// Shows messages provided in the manifest.json to the user.
func (m *Manifest) ShowMessages() {
	defer util.TimeTrack(time.Now(), "ShowMessages()")()

	// Show infos and warning.
	version, err := semver.NewVersion(versionconstants.DdevVersion)
	if err != nil {
		util.Warning("Failed to parse the DDEV version `%s` into a semver.Version.", versionconstants.DdevVersion)
		return
	}

	for _, messages := range []messageTypes{
		{messageType: types.Warning, messages: m.manifest.Messages.Warnings},
		{messageType: types.Info, messages: m.manifest.Messages.Infos},
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

// ShowTips tips provided in the manifest.json to the user.
// TODO limit to once a day maybe see PR.
// TODO beautify output
func (m *Manifest) ShowTips() {
	defer util.TimeTrack(time.Now(), "ShowTips()")()

	// Show tips.
	if !m.tipsDisabled {
		tips := len(m.manifest.Messages.Tips.Messages)
		if tips > 0 {
			m.manifest.Messages.Tips.Last++
			if m.manifest.Messages.Tips.Last > tips {
				m.manifest.Messages.Tips.Last = 1
			}

			util.Success("\n%s", m.manifest.Messages.Tips.Messages[m.manifest.Messages.Tips.Last-1])

			m.write()
		}
	}
}
