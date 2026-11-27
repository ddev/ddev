package webui

import (
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/ddev/ddev/pkg/config/remoteconfig"
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/config/state/storage/yaml"
	"github.com/ddev/ddev/pkg/globalconfig"
)

// remote is what DDEV downloads for `ddev start` to show: the remote config's
// tips and notices, and the sponsorship status.
type remote struct {
	config      types.RemoteConfig
	sponsorship types.SponsorshipManager
}

// newRemote reads them from the global config directory and, like
// `ddev start`, downloads newer copies when those are old.
var newRemote = func() remote {
	ddevMu.Lock()
	dir := globalconfig.GetGlobalDdevDir()
	rc := globalconfig.DdevGlobalConfig.RemoteConfig
	cfg := remoteconfig.Config{
		Local:          remoteconfig.Local{Path: dir},
		URL:            rc.RemoteConfigURL,
		UpdateInterval: rc.UpdateInterval,
		TickerInterval: globalconfig.DdevGlobalConfig.Messages.TickerInterval,
	}
	ddevMu.Unlock()
	state := yaml.NewState(filepath.Join(dir, ".state.yaml"))
	return remote{
		config:      remoteconfig.New(&cfg, state, globalconfig.IsInternetActive),
		sponsorship: remoteconfig.NewSponsorshipManager(dir, state, globalconfig.IsInternetActive, rc.UpdateInterval, rc.SponsorshipDataURL),
	}
}

var remoteCache struct {
	sync.Mutex
	r  remote
	at time.Time
}

// remoteMessages reads them again each hour, so that an app left open for
// days gets the notices published since it started.
func remoteMessages() remote {
	c := &remoteCache
	c.Lock()
	defer c.Unlock()
	if c.at.IsZero() || time.Since(c.at) > time.Hour {
		c.r, c.at = newRemote(), time.Now()
	}
	return c.r
}

type message struct {
	Title   string `json:"title,omitempty"`
	Message string `json:"message"`
}

// sponsorship is how much DDEV's sponsors give each month, in dollars, and
// the goal, with the message `ddev start` shows about them.
type sponsorship struct {
	Message string  `json:"message,omitempty"`
	Monthly float64 `json:"monthly"`
	Goal    float64 `json:"goal,omitempty"`
	Percent float64 `json:"percent,omitempty"`
}

// messages are the remote config's tips and its info and warning
// notifications, and the sponsorship status.
type messages struct {
	Tips        []message    `json:"tips"`
	Infos       []message    `json:"infos"`
	Warnings    []message    `json:"warnings"`
	Sponsorship *sponsorship `json:"sponsorship,omitempty"`
}

func toMessages(in []types.Message) []message {
	out := []message{}
	for _, m := range in {
		out = append(out, message{Title: m.Title, Message: m.Message})
	}
	return out
}

func (s *Server) handleMessages(w http.ResponseWriter, _ *http.Request) {
	r := remoteMessages()
	infos, warnings := r.config.NotificationMessages()
	out := messages{Tips: toMessages(r.config.TickerMessages()), Infos: toMessages(infos), Warnings: toMessages(warnings)}
	// As `ddev start`, say nothing before the data is there.
	if d, err := r.sponsorship.GetSponsorshipData(); err == nil && d != nil && d.TotalMonthlyAverageIncome > 0 {
		out.Sponsorship = &sponsorship{Message: d.AppreciationMessage, Monthly: d.TotalMonthlyAverageIncome, Goal: d.CurrentGoal.TargetAmount, Percent: d.CurrentGoal.ProgressPercentage}
	}
	writeJSON(w, http.StatusOK, out)
}
