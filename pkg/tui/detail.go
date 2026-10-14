package tui

import (
	"cmp"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/Masterminds/semver/v3"
	"github.com/charmbracelet/x/ansi"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
)

const addonRegistryURL = "https://addons.ddev.com"

// labelWidth is the width of the labels in the detail view.
const labelWidth = 13

// detailView shows one project, as `ddev describe` reports it.
type detailView struct {
	root, name string
	project    *ProjectDetail
	loading    bool
	// xdebug is nil until known. It is read once, after the rest of the
	// detail, and kept until the project's status changes or it reloads.
	xdebug *bool
	// latest maps an add-on's lowercased repository to its latest release,
	// once read.
	latest map[string]string
	vp     viewport.Model
}

// newDetailView shows p with what its configuration and the project list
// say until the rest of its detail loads.
func newDetailView(p ProjectInfo) detailView {
	v := detailView{root: p.AppRoot, name: p.Name, loading: true, vp: newViewport()}
	if p.Detail != nil {
		d := *p.Detail
		d.Status, d.PrimaryURL = p.Status, p.URL
		v.project = &d
	}
	return v
}

// loaded shows d, then reads the latest releases of its add-ons and the
// Xdebug state of a running project.
func (v detailView) loaded(d ProjectDetail) (detailView, tea.Cmd) {
	// A start or stop resets Xdebug to the project's configuration.
	if v.project != nil && v.project.Status != d.Status {
		v.xdebug = nil
	}
	v.loading = false
	v.project = &d
	var cmds []tea.Cmd
	if len(d.Addons) > 0 {
		cmds = append(cmds, latestAddons(v.root))
	}
	if d.Status == ddevapp.SiteRunning && v.xdebug == nil {
		cmds = append(cmds, xdebugStatus(v.root, "status"))
	}
	return v, tea.Batch(cmds...)
}

func (v detailView) xdebugLoaded(msg xdebugStatusMsg) (detailView, action) {
	if msg.appRoot != v.root {
		return v, nil
	}
	if msg.err != nil {
		v.xdebug = nil
		if msg.toggled {
			return v, statusAction{text: fmt.Sprintf("Xdebug toggle failed: %v", msg.err), isErr: true}
		}
		return v, nil
	}
	v.xdebug = &msg.enabled
	switch {
	case !msg.toggled:
		return v, nil
	case msg.enabled:
		return v, statusAction{text: "Xdebug enabled"}
	}
	return v, statusAction{text: "Xdebug disabled"}
}

// statusChanged reports whether projects has this project in another status
// than its detail.
func (v detailView) statusChanged(projects []ProjectInfo) bool {
	for _, p := range projects {
		if p.AppRoot == v.root {
			return p.Status != v.project.Status
		}
	}
	return false
}

// keys returns k with the bindings that cannot act on the project disabled.
func (v detailView) keys(k KeyMap) KeyMap {
	d := v.project
	running := d != nil && d.Status == ddevapp.SiteRunning
	for _, b := range []*key.Binding{&k.Start, &k.Stop, &k.Restart, &k.Delete, &k.Launch, &k.Mailpit} {
		b.SetEnabled(d != nil)
	}
	k.XHGui.SetEnabled(d != nil && usesXHGui(d.XHProfMode))
	for _, b := range []*key.Binding{&k.SSH, &k.Logs} {
		b.SetEnabled(running)
	}
	k.Xdebug.SetEnabled(running && v.xdebug != nil)
	k.CopyURL.SetEnabled(d != nil && d.PrimaryURL != "")
	// A snapshot would start a stopped project, which only a confirmed
	// restore does here.
	k.Snapshot.SetEnabled(running && !d.DBOmitted)
	k.Restore.SetEnabled(d != nil && !d.DBOmitted && len(d.Snapshots) > 0)
	k.Update.SetEnabled(len(v.outdated()) > 0)
	return k
}

// newer returns the latest release of add-on a if it is newer than the one
// installed. The registry is cached for a day, so a release installed since
// can be newer than its latest. A branch or commit counts as outdated, since
// `ddev add-on update` replaces it with the latest release.
func (v detailView) newer(a AddonInfo) string {
	latest := v.latest[strings.ToLower(a.Repository)]
	if latest == "" || latest == a.Version {
		return ""
	}
	if l, err := semver.NewVersion(latest); err == nil {
		if installed, err := semver.NewVersion(a.Version); err == nil && !l.GreaterThan(installed) {
			return ""
		}
	}
	return latest
}

// outdated lists the add-ons that have a newer release.
func (v detailView) outdated() []AddonInfo {
	var out []AddonInfo
	if v.project == nil {
		return nil
	}
	for _, a := range v.project.Addons {
		if v.newer(a) != "" {
			out = append(out, a)
		}
	}
	return out
}

func (v detailView) help(k KeyMap, helpKey key.Binding) helpKeys {
	k = v.keys(k)
	vp := v.vp.KeyMap
	return helpKeys{
		short: []key.Binding{k.Back, k.Start, k.Stop, k.Launch, k.SSH, k.Logs, helpKey, k.Quit},
		full: [][]key.Binding{
			{vp.Up, vp.Down, vp.PageUp, vp.PageDown, k.Home, k.End, k.Back, k.Refresh, helpKey, k.Quit},
			{k.Start, k.Stop, k.Restart, k.Delete, k.Launch, k.Mailpit, k.XHGui, k.OpenDir, k.OpenVSC, k.OpenPhpS},
			{k.SSH, k.Logs, k.Xdebug, k.CopyURL, k.Snapshot, k.Restore, k.Update},
		},
	}
}

func (v detailView) update(msg tea.KeyPressMsg, k KeyMap) (detailView, tea.Cmd, action) {
	k = v.keys(k)
	d := v.project
	switch {
	case key.Matches(msg, k.Back):
		return v, nil, backAction{}
	case key.Matches(msg, k.Start):
		return v, nil, operation("Starting "+d.Name, d.AppRoot, "start")
	case key.Matches(msg, k.Stop):
		return v, nil, operation("Stopping "+d.Name, d.AppRoot, "stop")
	case key.Matches(msg, k.Restart):
		return v, nil, operation("Restarting "+d.Name, d.AppRoot, "restart")
	case key.Matches(msg, k.Delete):
		return v, nil, confirmDelete(d.Name)
	case key.Matches(msg, k.Launch, k.Mailpit, k.XHGui):
		return v, nil, browse(k, msg, d.Name, d.AppRoot)
	case key.Matches(msg, k.SSH):
		ssh := operation("SSH into "+d.Name, d.AppRoot, "ssh")
		ssh.done, ssh.shell = "SSH session ended", true
		return v, nil, ssh
	case key.Matches(msg, k.Logs):
		return v, nil, logsAction{d.Name, d.AppRoot}
	case key.Matches(msg, k.Xdebug):
		arg := "on"
		if *v.xdebug {
			arg = "off"
		}
		return v, nil, statusAction{text: "Toggling Xdebug...", cmd: xdebugStatus(d.AppRoot, arg)}
	case key.Matches(msg, k.CopyURL):
		return v, copyToClipboard(d.PrimaryURL), nil
	case key.Matches(msg, k.OpenDir, k.OpenVSC, k.OpenPhpS):
		return v, openProjectCmd(k, msg, v.root), nil
	case key.Matches(msg, k.Snapshot):
		return v, nil, operation("Making a snapshot of "+d.Name, d.AppRoot, "snapshot")
	case key.Matches(msg, k.Restore):
		latest := d.Snapshots[0]
		prompt := fmt.Sprintf("Replace the database of '%s' with snapshot %s from %s?", d.Name, latest.Label, latest.Created.Format("2006-01-02"))
		return v, nil, confirmAction{kind: confirmRestore, target: d.Name, dir: d.AppRoot, args: []string{latest.Path}, prompt: prompt}
	case key.Matches(msg, k.Update):
		var updates, repos []string
		for _, a := range v.outdated() {
			updates = append(updates, a.Repository+" "+a.Version+" → "+v.newer(a))
			repos = append(repos, a.Repository)
		}
		prompt := fmt.Sprintf("Update the add-ons of '%s'? %s", d.Name, strings.Join(updates, ", "))
		return v, nil, confirmAction{kind: confirmUpdate, target: d.Name, dir: d.AppRoot, args: repos, prompt: prompt}
	case key.Matches(msg, k.Refresh):
		return v, nil, refreshAction{}
	case key.Matches(msg, k.Home):
		v.vp.GotoTop()
		return v, nil, nil
	case key.Matches(msg, k.End):
		v.vp.GotoBottom()
		return v, nil, nil
	}
	var cmd tea.Cmd
	v.vp, cmd = v.vp.Update(msg)
	return v, cmd, nil
}

func (v detailView) title(u ui) string {
	title := u.styles.Title.Render(v.name)
	if v.project != nil {
		title += "  " + u.styles.status(v.project.Status)
	}
	return title
}

func (v detailView) body(u ui) string {
	if v.project == nil {
		return message(u.width, u.spinner+"Loading project details...")
	}
	return scrollable(u.styles, v.vp)
}

// resize fits the view to width and height, rendering the detail for width.
func (v *detailView) resize(s Styles, width, height int) {
	v.vp.SetWidth(width)
	v.vp.SetHeight(height)
	v.vp.SetContent(v.content(s, width))
}

// content renders the project detail for width w.
func (v detailView) content(s Styles, w int) string {
	d := v.project
	if d == nil {
		return ""
	}
	var lines []string
	add := func(line string) {
		lines = append(lines, ansi.Truncate(line, w, "…"))
	}
	section := func(title, note string) {
		title = s.Section.Render(title) + note
		rule := s.Muted.Render(" " + strings.Repeat("─", max(0, w-ansi.StringWidth(title)-3)))
		lines = append(lines, "", " "+title+rule)
	}
	link := func(target, text string) string {
		return s.URL.Render(output.Hyperlink(target, text))
	}
	label := func(l string) string {
		return " " + fit(s.Label.Render(l), labelWidth)
	}
	none := s.Muted.Render("–")

	// The left column is what the project runs and the right one how it is
	// set up. Xdebug has a fixed width: it arrives later and must not move
	// anything.
	perf := cmp.Or(d.PerformanceMode, "none")
	if d.MutagenStatus != "" {
		perf += " (" + d.MutagenStatus + ")"
	}
	value := func(v string) string { return cmp.Or(v, none) }
	left := []string{
		label("Type") + value(d.Type), label("PHP") + value(d.PHPVersion),
		label("Webserver") + value(d.WebserverType), label("Node.js") + value(d.NodeJSVersion),
	}
	right := []string{
		label("Docroot") + cmp.Or(d.Docroot, "(project root)"), label("Perf mode") + perf,
		label("Xdebug") + fit(value(v.xdebugText(s)), 3), label("XHProf mode") + value(d.XHProfMode),
	}
	leftW, rightW := 0, 0
	for i := range left {
		leftW = max(leftW, ansi.StringWidth(left[i])+4)
		rightW = max(rightW, ansi.StringWidth(right[i]))
	}
	if leftW+rightW <= w {
		for i := range left {
			add(fit(left[i], leftW) + right[i])
		}
	} else {
		for _, cell := range append(left, right...) {
			add(cell)
		}
	}

	// flow puts items after a label, wrapping onto unlabeled lines.
	flow := func(l string, items ...string) {
		sep := s.Muted.Render(" · ")
		line := ""
		for _, item := range items {
			if line != "" && 1+labelWidth+ansi.StringWidth(line+sep+item) > w {
				add(label(l) + line)
				l, line = "", ""
			}
			if line != "" {
				line += sep
			}
			line += item
		}
		if line != "" {
			add(label(l) + line)
		}
	}

	// The database and the path keep their height when the detail loads or
	// the project stops, so they come before the URLs, which grow.
	switch {
	case d.DBOmitted:
		lines = append(lines, "")
		add(label("Database") + s.Muted.Render("none"))
	case d.DatabaseType != "":
		lines = append(lines, "")
		flow("Database", strings.TrimSpace(d.DatabaseType+" "+d.DatabaseVersion), "db/db"+s.Muted.Render(" or ")+"root/root")
		// The host address has its own line, so that its length, which
		// changes as the project loads or stops, never rewraps the others.
		from := func(addr, where string) string { return addr + s.Muted.Render(" from "+where) }
		switch {
		case d.DBHostPort != "":
			add(label("") + from(d.DBHostPort, "host"))
		case d.FromConfig:
			add(label("") + s.Muted.Render(statusLoading+" from host"))
		default:
			add(label("") + s.Muted.Render("no host port while stopped"))
		}
		var containers []string
		if d.DBInternal != "" {
			containers = append(containers, from(d.DBInternal, "this project"))
		}
		if d.DBShared != "" {
			containers = append(containers, from(d.DBShared, "other projects"))
		}
		flow("", containers...)
	}
	switch {
	case d.DatabaseType == "" || d.DBOmitted:
	case d.FromConfig:
		add(label("Snapshots") + s.Muted.Render(statusLoading))
	case len(d.Snapshots) == 0:
		add(label("Snapshots") + s.Muted.Render("none"))
	default:
		latest := d.Snapshots[0]
		// One line, so a long name cannot move the rows below. The name is cut
		// first, since the count and date say more.
		count := label("Snapshots") + strconv.Itoa(len(d.Snapshots))
		date := s.Muted.Render(" · " + latest.Created.Format("2006-01-02"))
		room := w - ansi.StringWidth(count+date) - len(" · latest ")
		if room >= 6 {
			date = s.Muted.Render(" · latest ") + ansi.Truncate(latest.Label, room, "…") + date
		}
		add(count + date)
	}

	lines = append(lines, "")
	add(label("Path") + link(output.FileURL(d.AppRoot), formatProjectPath(d.AppRoot)))
	add(label("Open in") + link(editorURL("vscode", d.AppRoot), "VS Code") + s.Muted.Render(" · ") + link(editorURL("phpstorm", d.AppRoot), "PhpStorm"))

	var urls []string
	for _, u := range d.URLs {
		urls = append(urls, link(u, u))
	}
	if d.FromConfig {
		urls = []string{s.Muted.Render(statusLoading)}
		// The list already has the primary URL of a running project.
		if d.PrimaryURL != "" {
			urls = []string{link(d.PrimaryURL, d.PrimaryURL)}
		}
	}
	if len(urls) > 0 || d.MailpitURL != "" || d.XHGuiURL != "" {
		lines = append(lines, "")
	}
	if len(urls) > 0 {
		flow("URLs", urls...)
	}
	if d.MailpitURL != "" {
		add(label("Mailpit") + link(d.MailpitURL, d.MailpitURL))
	}
	if d.XHGuiURL != "" {
		add(label("XHGui") + link(d.XHGuiURL, d.XHGuiURL))
	}

	loading := s.Muted.Render(" Loading...")
	section("Services", "")
	switch {
	case d.FromConfig:
		add(loading)
	case len(d.Services) == 0:
		add(s.Muted.Render(" None running"))
	default:
		lines = append(lines, servicesTable(s, d, w)...)
	}

	section("Add-ons", s.Muted.Render(" · find more at ")+link(addonRegistryURL, addonRegistryURL))
	switch {
	case d.FromConfig:
		add(loading)
	case len(d.Addons) == 0:
		add(s.Muted.Render(" None installed"))
	default:
		nameW := 0
		for _, a := range d.Addons {
			nameW = max(nameW, ansi.StringWidth(cmp.Or(a.Repository, a.Name)))
		}
		for _, a := range d.Addons {
			name := a.Name
			if a.Repository != "" {
				name = link("https://github.com/"+a.Repository, a.Repository)
			}
			row := " " + fit(name, nameW) + "  " + s.Muted.Render(a.Version)
			if latest := v.newer(a); latest != "" {
				row += s.Paused.Render(" → " + latest)
			}
			add(row)
		}
	}
	return strings.Join(lines, "\n")
}

// servicesTable lists the services with their URL and ports. A port is shown
// as container→host when it is published on the Docker host, or alone when
// it is reachable only from the other containers.
func servicesTable(s Styles, d *ProjectDetail, w int) []string {
	legend := "Ports (container→host)"
	type service struct {
		name, status, url, ports string
		portList                 []string
		info                     string
	}
	var rows []service
	nameW, statusW, urlW, portsW, infoW := len("Service"), len("Status"), 0, len(legend), 0
	for _, svc := range d.Services {
		var ports []string
		for _, p := range svc.Ports {
			if p.Host != "" {
				ports = append(ports, p.Container+"→"+net.JoinHostPort(d.DockerHost, p.Host))
			} else {
				ports = append(ports, p.Container+s.Muted.Render(" (containers only)"))
			}
		}
		r := service{name: s.Name.Render(svc.Name), status: s.status(svc.Status), ports: strings.Join(ports, ", "), portList: ports, info: svc.Info}
		if svc.URL != "" {
			r.url = s.URL.Render(output.Hyperlink(svc.URL, svc.URL))
		}
		rows = append(rows, r)
		nameW = max(nameW, ansi.StringWidth(svc.Name))
		statusW = max(statusW, ansi.StringWidth(r.status))
		urlW = max(urlW, ansi.StringWidth(svc.URL))
		portsW = max(portsW, ansi.StringWidth(r.ports))
		infoW = max(infoW, ansi.StringWidth(svc.Info))
	}
	if urlW > 0 {
		urlW = max(urlW, len("URL"))
	}
	indent := " " + strings.Repeat(" ", nameW+2)
	var lines []string
	add := func(line string) {
		lines = append(lines, ansi.Truncate(line, w, "…"))
	}

	// A table when every column fits, otherwise each service's URL and
	// ports go on lines of their own under it.
	tableW := 1 + nameW + 2 + statusW + 2 + urlW + 2 + portsW
	if tableW <= w {
		infoColumn := infoW > 0 && tableW+2+infoW <= w
		row := func(name, status, link, ports, info string) string {
			line := " " + fit(name, nameW) + "  " + fit(status, statusW) + "  "
			if urlW > 0 {
				line += fit(link, urlW) + "  "
			}
			if infoColumn {
				return line + fit(ports, portsW) + "  " + info
			}
			return line + ports
		}
		add(s.Muted.Render(row("Service", "Status", "URL", legend, "Info")))
		for _, r := range rows {
			add(row(r.name, r.status, r.url, r.ports, s.Muted.Render(r.info)))
			if r.info != "" && !infoColumn {
				add(indent + s.Muted.Render(r.info))
			}
		}
		return lines
	}
	add(s.Muted.Render(" " + legend))
	for _, r := range rows {
		add(" " + fit(r.name, nameW) + "  " + r.status)
		for _, v := range append(append([]string{r.url}, r.portList...), s.Muted.Render(r.info)) {
			if ansi.StringWidth(v) > 0 {
				add(indent + v)
			}
		}
	}
	return lines
}

// xdebugText is the Xdebug state, which loads after the rest of the detail.
func (v detailView) xdebugText(s Styles) string {
	switch {
	case v.project == nil:
		return ""
	case v.project.FromConfig:
		return s.Muted.Render("…")
	case v.project.Status == ddevapp.SiteStopped:
		return "off"
	case v.project.Status != ddevapp.SiteRunning:
		return ""
	case v.xdebug == nil:
		return s.Muted.Render("…")
	case *v.xdebug:
		return s.Running.Render("on")
	}
	return "off"
}

// openProjectCmd opens appRoot in the file manager or an editor, depending
// on which open key was pressed.
func openProjectCmd(k KeyMap, msg tea.KeyPressMsg, appRoot string) tea.Cmd {
	switch {
	case key.Matches(msg, k.OpenVSC):
		return openTargetCmd("VS Code", editorURL("vscode", appRoot))
	case key.Matches(msg, k.OpenPhpS):
		return openInPhpStormCmd(appRoot)
	}
	return openTargetCmd("directory", appRoot)
}

// editorURL returns a URL that opens dir through an editor's registered URL
// scheme, or "" for an unknown editor. Under WSL the editors run on Windows,
// so they get a WSL remote URI or a \\wsl.localhost path instead of a Linux path.
func editorURL(editor, dir string) string {
	p := filepath.ToSlash(dir)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	distro := os.Getenv("WSL_DISTRO_NAME")
	switch editor {
	case "vscode":
		u := &url.URL{Scheme: "vscode", Host: "file", Path: p, RawQuery: "windowId=_blank"}
		if distro != "" {
			u.Host = "vscode-remote"
			u.Path = "/wsl+" + distro + p
		}
		return u.String()
	case "phpstorm":
		if distro != "" {
			p = "//wsl.localhost/" + distro + p
		}
		// EscapedPath leaves & and + alone, but they would end the file= value
		return "phpstorm://open?file=" + strings.NewReplacer("&", "%26", "+", "%2B").Replace((&url.URL{Path: p}).EscapedPath())
	}
	return ""
}
