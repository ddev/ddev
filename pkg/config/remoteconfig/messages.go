package remoteconfig

import (
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/styles"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

type messageTypes struct {
	messageType types.MessageType
	messages    []types.Message
}

type conditionDefinition struct {
	name          string
	description   string
	conditionFunc func() bool
}

var conditionDefinitions = map[string]conditionDefinition{}

func init() {
	AddCondition("Disabled", "Permanently disables the message", func() bool { return false })
	AddCondition("Colima", "Running on Colima", dockerutil.IsColima)
	AddCondition("Lima", "Running on Lima", dockerutil.IsLima)
	AddCondition("DockerDesktop", "Running on Docker Desktop", dockerutil.IsDockerDesktop)
	AddCondition("WSL2", "Running on WSL2", nodeps.IsWSL2)
}

func AddCondition(name, description string, conditionFunc func() bool) {
	conditionDefinitions[strings.ToLower(name)] = conditionDefinition{
		name:          name,
		description:   description,
		conditionFunc: conditionFunc,
	}
}

func ListConditions() (conditions map[string]string) {
	conditions = make(map[string]string)

	for _, condition := range conditionDefinitions {
		conditions[condition.name] = condition.description
	}

	return
}

// ShowNotifications shows notifications provided by the remote config to the user.
func (c *remoteConfig) ShowNotifications() {
	// defer util.TimeTrack()()

	if !c.showNotifications() {
		return
	}

	for _, messages := range []messageTypes{
		{messageType: types.Info, messages: c.remoteConfig.Messages.Notifications.Infos},
		{messageType: types.Warning, messages: c.remoteConfig.Messages.Notifications.Warnings},
	} {
		t := table.NewWriter()

		var title string
		var i int

		switch messages.messageType {
		case types.Warning:
			applyTableStyle(warning, t)
			title = "Important Warning"
		default:
			applyTableStyle(information, t)
			title = "Important Message"
		}

		for _, message := range messages.messages {
			if !c.checkConditions(message.Conditions) || !c.checkVersions(message.Versions) {
				continue
			}

			t.AppendRow(table.Row{message.Message})
			i++
		}

		if i == 0 {
			continue
		}

		if i > 1 {
			title += "s"
		}

		t.AppendHeader(table.Row{title})

		output.UserOut.Print("\n", t.Render(), "\n")
	}

	c.state.LastNotificationAt = time.Now()
	if err := c.state.save(); err != nil {
		util.Debug("Error while saving state: %v", err)
	}
}

// ShowTicker shows ticker messages provided by the remote config to the user.
func (c *remoteConfig) ShowTicker() {
	// defer util.TimeTrack()()

	tickerData := c.getTicker()
	if !c.showTickerMessage() || len(tickerData.Messages) == 0 {
		return
	}

	messageOffset := c.state.LastTickerMessage
	messageCount := len(tickerData.Messages)

	if messageOffset == 0 {
		// As long as no message was shown, start with a random message. This
		// is important for short living instances e.g. Gitpod to not always
		// show the first message. A number from 0 to number of messages minus
		// 1 is generated.
		messageOffset = rand.Intn(messageCount)
	}

	for i := range tickerData.Messages {
		messageOffset++
		if messageOffset > messageCount {
			messageOffset = 1
		}

		message := &tickerData.Messages[i+messageOffset-1]

		if c.checkConditions(message.Conditions) && c.checkVersions(message.Versions) {
			t := table.NewWriter()
			applyTableStyle(ticker, t)

			var title string

			if message.Title != "" {
				title = message.Title
			} else {
				title = "Tip of the day"
			}

			t.AppendHeader(table.Row{title})
			t.AppendRow(table.Row{message.Message})

			output.UserOut.Print("\n", t.Render(), "\n")

			c.state.LastTickerMessage = messageOffset
			c.state.LastTickerAt = time.Now()
			if err := c.state.save(); err != nil {
				util.Debug("Error while saving state: %v", err)
			}

			break
		}
	}
}

// isNotificationsDisabled returns true if notifications should not be shown to
// the user which can be achieved by setting the related remote config.
func (c *remoteConfig) isNotificationsDisabled() bool {
	return c.getNotificationsInterval() < 0
}

// getNotificationsInterval returns the notifications interval. The processing
// order is defined as follows, the first defined value is returned:
//   - remote config
//   - const notificationsInterval
func (c *remoteConfig) getNotificationsInterval() time.Duration {
	if c.remoteConfig.Messages.Notifications.Interval != 0 {
		return time.Duration(c.remoteConfig.Messages.Notifications.Interval) * time.Hour
	}

	return time.Duration(notificationsInterval) * time.Hour
}

// showNotifications returns true if notifications are not disabled and the
// notifications interval has been elapsed.
func (c *remoteConfig) showNotifications() bool {
	return !output.JSONOutput &&
		!c.isNotificationsDisabled() &&
		c.state.LastNotificationAt.Add(c.getNotificationsInterval()).Before(time.Now())
}

// isTickerDisabled returns true if tips should not be shown to the user which
// can be achieved by setting the related global config or also via the remote
// config.
func (c *remoteConfig) isTickerDisabled() bool {
	return c.getTickerInterval() < 0
}

// getTickerInterval returns the ticker interval. The processing order is
// defined as follows, the first defined value is returned:
//   - global config
//   - remote config
//   - const tickerInterval
func (c *remoteConfig) getTickerInterval() time.Duration {
	if c.tickerInterval != 0 {
		return time.Duration(c.tickerInterval) * time.Hour
	}

	tickerData := c.getTicker()
	if tickerData.Interval != 0 {
		return time.Duration(tickerData.Interval) * time.Hour
	}

	return time.Duration(tickerInterval) * time.Hour
}

// showTickerMessage returns true if the ticker is not disabled and the ticker
// interval has been elapsed.
func (c *remoteConfig) showTickerMessage() bool {
	return !output.JSONOutput &&
		os.Getenv("CI") != "true" &&
		!c.isTickerDisabled() &&
		c.state.LastTickerAt.Add(c.getTickerInterval()).Before(time.Now())
}

// showSponsorshipMessage returns true if sponsorship message should be shown
// Uses same logic as ticker - once per day
func (c *remoteConfig) showSponsorshipMessage() bool {
	// Use the same interval as ticker for consistency (once per day)
	sponsorshipInterval := c.getTickerInterval()
	return !output.JSONOutput &&
		os.Getenv("CI") != "true" &&
		!c.isTickerDisabled() &&
		c.state.LastSponsorshipAt.Add(sponsorshipInterval).Before(time.Now())
}

func (c *remoteConfig) checkConditions(conditions []string) bool {
	for _, rawCondition := range conditions {
		condition, negated := strings.CutPrefix(strings.TrimSpace(rawCondition), "!")
		condition = strings.ToLower(strings.TrimSpace(condition))

		conditionDef, found := conditionDefinitions[condition]

		if found {
			conditionResult := conditionDef.conditionFunc()

			if (!negated && !conditionResult) || (negated && conditionResult) {
				return false
			}
		}
	}

	return true
}

func (c *remoteConfig) checkVersions(versions string) bool {
	versions = strings.TrimSpace(versions)
	if versions != "" {
		match, err := util.SemverValidate(versions, versionconstants.DdevVersion)
		if err != nil {
			util.Debug("Failed to validate DDEV version `%s` against constraint `%s`: %v", versionconstants.DdevVersion, versions, err)
			return true
		}

		return match
	}

	return true
}

// ShowSponsorshipAppreciation shows a sponsorship appreciation message if data is available
func (c *remoteConfig) ShowSponsorshipAppreciation() {
	// Get sponsorship manager
	sponsorshipMgr := GetGlobalSponsorship()
	if sponsorshipMgr == nil {
		return
	}

	// Check if we should show sponsorship message (same logic as MOTD - once a day)
	if !c.showSponsorshipMessage() {
		return
	}

	sponsorshipData, err := sponsorshipMgr.GetSponsorshipData()
	if err != nil {
		util.Debug("Error getting sponsorship data: %v", err)
		return
	}

	// Only show if we have meaningful data
	if sponsorshipData == nil || sponsorshipData.TotalMonthlyAverageIncome == 0 {
		return
	}

	// Use the AppreciationMessage from the data
	message := sponsorshipData.AppreciationMessage
	if message == "" {
		// Fallback if not present
		message = "💚 DDEV's community sponsorship makes it go! Consider becoming a sponsor at github.com/sponsors/ddev 🤝"
	}

	t := table.NewWriter()
	applyTableStyle(sponsorship, t)
	title := "❤️ DDEV Sponsorship Status"
	t.AppendHeader(table.Row{title})
	t.AppendRow(table.Row{message})

	output.UserOut.Print("\n", t.Render(), "\n")

	// Update state so we don't show this message again today
	c.state.LastSponsorshipAt = time.Now()
	if err := c.state.save(); err != nil {
		util.Debug("Error while saving sponsorship display state: %v", err)
	}
}

// getTicker returns ticker data from the messages structure
func (c *remoteConfig) getTicker() types.Ticker {
	return c.remoteConfig.Messages.Ticker
}

type preset int

const (
	information preset = iota
	warning
	ticker
	sponsorship
)

func applyTableStyle(preset preset, writer table.Writer) {
	styles.SetGlobalTableStyle(writer, true)

	termWidth, _ := nodeps.GetTerminalWidthHeight()
	util.Debug("termWidth: %d", termWidth)
	writer.SetColumnConfigs([]table.ColumnConfig{
		{
			Number:           1,
			WidthMin:         50,
			WidthMax:         int(termWidth) - 5,
			WidthMaxEnforcer: text.WrapSoft,
		},
	})

	style := writer.Style()

	style.Options.SeparateRows = false
	style.Options.SeparateFooter = false
	style.Options.SeparateColumns = false
	style.Options.SeparateHeader = false
	style.Options.DrawBorder = false

	switch preset {
	case information:
		style.Color = table.ColorOptions{
			Header: text.Colors{text.BgHiYellow, text.FgBlack},
			Row:    text.Colors{text.BgHiYellow, text.FgBlack},
		}
	case warning:
		style.Color = table.ColorOptions{
			Header: text.Colors{text.BgHiRed, text.FgBlack},
			Row:    text.Colors{text.BgHiRed, text.FgBlack},
		}
	case ticker:
		style.Color = table.ColorOptions{
			Header: text.Colors{text.BgHiCyan, text.FgBlack},
			Row:    text.Colors{text.BgHiCyan, text.FgBlack},
		}
	case sponsorship:
		style.Color = table.ColorOptions{
			Header: text.Colors{text.BgHiGreen, text.FgBlack},
			Row:    text.Colors{text.BgHiGreen, text.FgBlack},
		}
	}
}
