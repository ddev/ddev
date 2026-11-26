package webui

import (
	"net/http"
	"path/filepath"
	"sync"

	"github.com/ddev/ddev/pkg/config/remoteconfig"
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/config/state/storage/yaml"
	"github.com/ddev/ddev/pkg/globalconfig"
)

// remoteMessages reads DDEV's remote config, which holds the tips and notices
// `ddev start` shows. Like `ddev start`, it downloads a newer one when the
// copy in the global config directory is old.
var remoteMessages = sync.OnceValue(func() types.RemoteConfig {
	ddevMu.Lock()
	dir := globalconfig.GetGlobalDdevDir()
	cfg := remoteconfig.Config{
		Local:          remoteconfig.Local{Path: dir},
		URL:            globalconfig.DdevGlobalConfig.RemoteConfig.RemoteConfigURL,
		UpdateInterval: globalconfig.DdevGlobalConfig.RemoteConfig.UpdateInterval,
		TickerInterval: globalconfig.DdevGlobalConfig.Messages.TickerInterval,
	}
	ddevMu.Unlock()
	return remoteconfig.New(&cfg, yaml.NewState(filepath.Join(dir, ".state.yaml")), globalconfig.IsInternetActive)
})

type message struct {
	Title   string `json:"title,omitempty"`
	Message string `json:"message"`
}

// messages are the remote config's tips and its info and warning
// notifications.
type messages struct {
	Tips     []message `json:"tips"`
	Infos    []message `json:"infos"`
	Warnings []message `json:"warnings"`
}

func toMessages(in []types.Message) []message {
	out := []message{}
	for _, m := range in {
		out = append(out, message{Title: m.Title, Message: m.Message})
	}
	return out
}

func (s *Server) handleMessages(w http.ResponseWriter, _ *http.Request) {
	rc := remoteMessages()
	infos, warnings := rc.NotificationMessages()
	writeJSON(w, http.StatusOK, messages{Tips: toMessages(rc.TickerMessages()), Infos: toMessages(infos), Warnings: toMessages(warnings)})
}
