package messages

import (
	"path/filepath"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/messages/storages"
	"github.com/ddev/ddev/pkg/messages/types"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
)

var (
	fileStorage   types.MessagesStorage
	githubStorage types.MessagesStorage
)

type messageTypes struct {
	messageType types.MessageType
	messages    []types.Message
}

// Shows infos and warnings provided in the manifest.json to the user.
func ShowMessages() {
	messages := getMessages(24 * time.Hour)

	for _, messagesOfType := range []messageTypes{
		{messageType: types.Warning, messages: messages.Warnings},
		{messageType: types.Info, messages: messages.Infos},
	} {
		for _, message := range messagesOfType.messages {
			constraint, err := semver.NewConstraint(message.Versions)
			if err != nil {
				continue
			}

			version, err := semver.NewVersion(versionconstants.DdevVersion)
			if err != nil {
				continue
			}

			if !constraint.Check(version) {
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

// getMessages updates the messages if needed and returns them.
func getMessages(updateInterval time.Duration) (messages types.Messages) {
	var err error

	if fileStorage == nil {
		messageFile := filepath.Join(globalconfig.GetGlobalDdevDir(), ".messages")
		fileStorage = storages.NewFileStorage(messageFile)
	}

	// Check if an update is needed.
	if fileStorage.LastUpdate().Add(updateInterval).Before(time.Now()) {
		if githubStorage == nil {
			githubStorage = storages.NewGithubStorage(`ddev`, `ddev`, `manifest.json`)
		}

		messages, err = githubStorage.Pull()

		if err == nil {
			// Push the downloaded messages to the local storage.
			err = fileStorage.Push(&messages)

			if err != nil {
				util.Error("Error while writing messages: %s", err)
			}
		} else {
			util.Error("Error while downloading messages: %s", err)
		}
	}

	// Pull the messages to return
	messages, err = fileStorage.Pull()

	if err != nil {
		util.Error("Error while loading messages: %s", err)
	}

	return
}
