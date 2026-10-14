package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/sahilm/fuzzy"
)

// dashboardView lists the projects, one per row of a viewport, which
// scrolls a row at a time to keep the selection in view.
type dashboardView struct {
	items []projectItem
	// visible is what the filter lets through, best match first.
	visible []match
	cursor  int
	filter  textinput.Model
	vp      viewport.Model
	styles  Styles
	cols    columns
	// countLine is set when the line above the list fits.
	countLine bool
	cwd       string
	// current is the root of the project the TUI was started in, if any.
	current string
	// listed is set once projects are shown, and loaded once their status is
	// known; until then the header shows the spinner.
	listed, loaded bool
	// selectCurrent is set after `ddev config` succeeds. A load already
	// running may not list the new project yet, so it stays set until one does.
	selectCurrent bool
	// err is why loading failed while there was no project to show; other
	// failures go to the status line.
	err     error
	loading bool
	// queued is set when a reload was asked for while loading ran.
	queued bool
}

// match is a visible item, and the rune offsets in its FilterValue that the
// filter matched.
type match struct {
	item  int
	runes []int
}

func newDashboardView(styles Styles, cwd string) dashboardView {
	filter := textinput.New()
	filter.Prompt = "Filter: "
	v := dashboardView{filter: filter, vp: newViewport(), cwd: cwd, loading: true}
	v.setStyles(styles)
	return v
}

func (v *dashboardView) setStyles(styles Styles) {
	v.styles = styles
	v.filter.SetStyles(styles.filter())
}

// waiting reports whether the status of the projects is still unknown. A
// failed first load shows its error instead, without the spinner.
func (v dashboardView) waiting() bool {
	return !v.loaded && v.err == nil
}

// filtering reports whether the filter is being typed.
func (v dashboardView) filtering() bool {
	return v.filter.Focused()
}

func (v dashboardView) filtered() bool {
	return v.filtering() || v.filter.Value() != ""
}

// setProjects replaces the projects, keeping the selected project selected.
// On the first load, and after `ddev config`, it selects the working
// directory's project.
func (v *dashboardView) setProjects(projects []ProjectInfo) {
	selected := ""
	if p := v.selected(); p != nil {
		selected = p.AppRoot
	}
	v.current = currentProjectRoot(projects, v.cwd)
	if !v.listed || (v.selectCurrent && v.current != "") {
		selected = v.current
		v.listed, v.selectCurrent = true, false
	}
	v.items = make([]projectItem, len(projects))
	for i, p := range projects {
		v.items[i] = projectItem{ProjectInfo: p, path: formatProjectPath(p.AppRoot)}
		if p.AppRoot == v.current {
			v.items[i].mark = "*"
		}
	}
	v.refilter()
	v.selectProject(selected)
}

// markUnknown shows the projects still loading as unknown.
func (v *dashboardView) markUnknown() {
	for i := range v.items {
		if v.items[i].Status == statusLoading {
			v.items[i].Status = statusUnknown
		}
	}
}

// refilter matches the items against the filter.
func (v *dashboardView) refilter() {
	v.visible = nil
	term := v.filter.Value()
	if term == "" {
		for i := range v.items {
			v.visible = append(v.visible, match{item: i})
		}
		return
	}
	targets := make([]string, len(v.items))
	for i, it := range v.items {
		targets[i] = it.FilterValue()
	}
	for _, m := range filterProjects(term, targets) {
		v.visible = append(v.visible, match{item: m.Index, runes: m.MatchedIndexes})
	}
}

// clearFilter shows every project again, keeping the selection.
func (v *dashboardView) clearFilter() {
	selected := ""
	if p := v.selected(); p != nil {
		selected = p.AppRoot
	}
	v.filter.Reset()
	v.filter.Blur()
	v.refilter()
	v.selectProject(selected)
}

// selectProject selects the visible project at appRoot, or keeps the cursor
// in range when it is gone.
func (v *dashboardView) selectProject(appRoot string) {
	for i, m := range v.visible {
		if v.items[m.item].AppRoot == appRoot {
			v.cursor = i
			return
		}
	}
	v.moveTo(v.cursor)
}

// moveTo selects visible row i, kept in range.
func (v *dashboardView) moveTo(i int) {
	v.cursor = max(0, min(i, len(v.visible)-1))
}

// projects returns all projects, filtered or not.
func (v dashboardView) projects() []ProjectInfo {
	projects := make([]ProjectInfo, len(v.items))
	for i, it := range v.items {
		projects[i] = it.ProjectInfo
	}
	return projects
}

// visibleItems returns the projects the filter lets through.
func (v dashboardView) visibleItems() []projectItem {
	items := make([]projectItem, len(v.visible))
	for i, m := range v.visible {
		items[i] = v.items[m.item]
	}
	return items
}

// selected returns the selected project, or nil.
func (v dashboardView) selected() *ProjectInfo {
	if v.cursor < 0 || v.cursor >= len(v.visible) {
		return nil
	}
	return &v.items[v.visible[v.cursor].item].ProjectInfo
}

func (v dashboardView) runningCount() int {
	n := 0
	for _, it := range v.items {
		if it.Status == ddevapp.SiteRunning {
			n++
		}
	}
	return n
}

// keys returns k with the bindings that cannot act on the current selection
// disabled, which also hides them from the help.
func (v dashboardView) keys(k KeyMap) KeyMap {
	p := v.selected()
	for _, b := range []*key.Binding{&k.Detail, &k.Start, &k.Stop, &k.Restart, &k.Delete, &k.Launch, &k.Mailpit, &k.OpenDir, &k.OpenVSC, &k.OpenPhpS} {
		b.SetEnabled(p != nil)
	}
	k.XHGui.SetEnabled(p != nil && p.Detail != nil && usesXHGui(p.Detail.XHProfMode))
	hasProjects := len(v.items) > 0
	k.StartAll.SetEnabled(hasProjects)
	k.StopAll.SetEnabled(hasProjects)
	k.Filter.SetEnabled(hasProjects)
	k.ClearFilter.SetEnabled(v.filter.Value() != "")
	return k
}

func (v dashboardView) help(k KeyMap, helpKey key.Binding) helpKeys {
	if v.filtering() {
		return helpKeys{
			short: []key.Binding{k.AcceptFilter, k.CancelFilter},
			full:  [][]key.Binding{{k.AcceptFilter, k.CancelFilter}},
		}
	}
	k = v.keys(k)
	return helpKeys{
		short: []key.Binding{k.Detail, k.Start, k.Stop, k.Restart, k.Launch, k.Filter, k.ClearFilter, helpKey, k.Quit},
		full: [][]key.Binding{
			{k.Detail, k.Up, k.Down, k.PageUp, k.PageDown, k.Home, k.End, k.Filter, k.ClearFilter},
			{k.Start, k.Stop, k.Restart, k.Delete, k.Launch, k.Mailpit, k.XHGui, k.OpenDir, k.OpenVSC, k.OpenPhpS},
			{k.StartAll, k.StopAll, k.Poweroff, k.Config, k.Refresh, helpKey, k.Quit},
		},
	}
}

func (v dashboardView) update(msg tea.KeyPressMsg, k KeyMap) (dashboardView, tea.Cmd, action) {
	if v.filtering() {
		v, cmd := v.updateFiltering(msg, k)
		return v, cmd, nil
	}
	k = v.keys(k)
	p := v.selected()
	switch {
	case key.Matches(msg, k.Detail):
		return v, nil, openDetailAction{*p}
	case key.Matches(msg, k.Start):
		return v, nil, operation("Starting "+p.Name, "", "start", p.Name)
	case key.Matches(msg, k.Stop):
		return v, nil, operation("Stopping "+p.Name, "", "stop", p.Name)
	case key.Matches(msg, k.Restart):
		return v, nil, operation("Restarting "+p.Name, "", "restart", p.Name)
	case key.Matches(msg, k.Delete):
		return v, nil, confirmDelete(p.Name)
	case key.Matches(msg, k.Launch, k.Mailpit, k.XHGui):
		return v, nil, browse(k, msg, p.Name, p.AppRoot)
	case key.Matches(msg, k.OpenDir, k.OpenVSC, k.OpenPhpS):
		return v, openProjectCmd(k, msg, p.AppRoot), nil
	case key.Matches(msg, k.StartAll):
		return v, nil, confirmAction{kind: confirmStartAll, prompt: fmt.Sprintf("Start all %d projects?", len(v.items))}
	case key.Matches(msg, k.StopAll):
		return v, nil, confirmAction{kind: confirmStopAll, prompt: fmt.Sprintf("Stop all %d projects?", len(v.items))}
	case key.Matches(msg, k.Poweroff):
		return v, nil, confirmAction{kind: confirmPoweroff, prompt: "Power off all DDEV projects and containers?"}
	case key.Matches(msg, k.Config):
		config := operation("ddev config", "", "config")
		config.done, config.selectCurrent = "Configured the current directory", true
		return v, nil, config
	case key.Matches(msg, k.Refresh):
		return v, nil, refreshAction{}
	case key.Matches(msg, k.Up):
		v.moveTo(v.cursor - 1)
	case key.Matches(msg, k.Down):
		v.moveTo(v.cursor + 1)
	case key.Matches(msg, k.Home):
		v.moveTo(0)
	case key.Matches(msg, k.End):
		v.moveTo(len(v.visible) - 1)
	case key.Matches(msg, k.PageDown):
		return v.page(v.vp.Height()), nil, nil
	case key.Matches(msg, k.PageUp):
		return v.page(-v.vp.Height()), nil, nil
	case key.Matches(msg, k.Filter):
		v.filter.CursorEnd()
		v.moveTo(0)
		return v, v.filter.Focus(), nil
	case key.Matches(msg, k.ClearFilter):
		v.clearFilter()
	}
	return v, nil, nil
}

// updateFiltering types msg into the filter, or ends typing it. A filter
// that matches nothing is dropped rather than applied.
func (v dashboardView) updateFiltering(msg tea.KeyPressMsg, k KeyMap) (dashboardView, tea.Cmd) {
	switch {
	case key.Matches(msg, k.CancelFilter):
		v.clearFilter()
	case key.Matches(msg, k.AcceptFilter):
		v.filter.Blur()
		if v.filter.Value() == "" || len(v.visible) == 0 {
			v.clearFilter()
		}
	default:
		return v.updateFilter(msg)
	}
	return v, nil
}

// updateFilter passes msg to the filter input, such as a key, a paste or the
// cursor's blink, and matches the projects again if the filter changed.
func (v dashboardView) updateFilter(msg tea.Msg) (dashboardView, tea.Cmd) {
	if !v.filtering() {
		return v, nil
	}
	before := v.filter.Value()
	var cmd tea.Cmd
	v.filter, cmd = v.filter.Update(msg)
	if v.filter.Value() != before {
		v.refilter()
		v.moveTo(0)
	}
	return v, cmd
}

// page moves the selection and the view by rows.
func (v dashboardView) page(rows int) dashboardView {
	v.moveTo(v.cursor + rows)
	v.vp.SetYOffset(v.vp.YOffset() + rows)
	return v
}

// resize fits the view to width and height, scrolling to the selection.
func (v *dashboardView) resize(width, height int) {
	// A filter keeps the line above the list, since it holds the input.
	v.countLine = height >= 5 || v.filtered()
	if v.countLine {
		height--
	}
	v.cols = projectColumns(v.items, width)
	v.vp.SetWidth(width)
	v.vp.SetHeight(max(1, height))
	v.vp.SetContent(v.rows(width))
	switch {
	case v.cursor < v.vp.YOffset():
		v.vp.SetYOffset(v.cursor)
	case v.cursor >= v.vp.YOffset()+v.vp.Height():
		v.vp.SetYOffset(v.cursor - v.vp.Height() + 1)
	}
}

// rows renders the visible projects, one per line of width.
func (v dashboardView) rows(width int) string {
	if len(v.visible) == 0 {
		return v.styles.Muted.Render("  No matching projects.")
	}
	lines := make([]string, len(v.visible))
	for i, m := range v.visible {
		lines[i] = ansi.Truncate(v.row(v.items[m.item], m.runes, i == v.cursor), width, "…")
	}
	return strings.Join(lines, "\n")
}

func (v dashboardView) title(u ui) string {
	return u.styles.Title.Render("DDEV") + " " + u.styles.Muted.Render(versionconstants.DdevVersion)
}

// body renders the list, or the loading, error or empty state.
func (v dashboardView) body(u ui) string {
	switch {
	case len(v.items) > 0:
		rows := scrollable(u.styles, v.vp)
		if v.countLine {
			return v.count(u.styles) + "\n" + rows
		}
		return rows
	case v.err != nil:
		return message(u.width, u.styles.Error.Render("Error: "+v.err.Error()))
	case !v.loaded:
		return message(u.width, u.spinner+"Loading projects...")
	}
	return message(u.width, "No DDEV projects found.",
		"Press "+u.keys.Config.Help().Key+" to run ddev config in the current directory, or run it in a project directory.")
}

// count counts the projects, or holds the filter input and counts the
// matches. Aligned with the names, it stays in place when the filter is
// applied, so the list does not move.
func (v dashboardView) count(s Styles) string {
	total, shown := len(v.items), len(v.visible)
	if v.filtered() {
		count := fmt.Sprintf("%d of %d projects", shown, total)
		w := max(0, v.vp.Width()-2-len(count)-1)
		input := ansi.Truncate(v.filter.View(), w, "")
		return "  " + input + strings.Repeat(" ", max(0, w-ansi.StringWidth(input))) + " " + s.Muted.Render(count)
	}
	text := fmt.Sprintf("%d projects", total)
	if total == 1 {
		text = "1 project"
	}
	if running := v.runningCount(); running > 0 {
		text += fmt.Sprintf(" · %d running", running)
	}
	if v.current != "" {
		text += " · * current directory"
	}
	return s.Muted.Render("  " + text)
}

// hint is shown in the status line when nothing else is.
func (v dashboardView) hint(k KeyMap) string {
	if len(v.items) == 0 {
		return ""
	}
	for _, it := range v.items {
		if it.Status != ddevapp.SiteStopped {
			return ""
		}
	}
	if v.selected() == nil {
		return fmt.Sprintf("All projects are stopped. Press %s to start all.", k.StartAll.Help().Key)
	}
	return fmt.Sprintf("All projects are stopped. Press %s to start the selected one, %s to start all.",
		k.Start.Help().Key, k.StartAll.Help().Key)
}

// confirmDelete asks before deleting a project. O skips the snapshot, which
// otherwise starts a stopped project and fails if it can't start; the prompt
// leaves O out when omit_snapshot_on_delete already skips it.
func confirmDelete(name string) confirmAction {
	prompt := fmt.Sprintf("Delete project '%s' and its database? A database snapshot will be made first.", name)
	if globalconfig.DdevGlobalConfig.OmitSnapshotOnDelete {
		prompt = fmt.Sprintf("Delete project '%s' and its database? No snapshot will be made (omit_snapshot_on_delete).", name)
	}
	return confirmAction{kind: confirmDeleteProject, target: name, prompt: prompt}
}

// projectItem is a project in the dashboard list.
type projectItem struct {
	ProjectInfo
	path string
	// mark follows the name of the project the TUI was started in, so it
	// stays visible after the selection moves away.
	mark string
}

// FilterValue is what the fuzzy filter matches against. It has only the
// project directory's own name, since every full path would match most
// short queries. The row highlights matches by their offsets in it.
func (i projectItem) FilterValue() string {
	return i.Name + " " + i.Type + " " + i.Status + " " + filepath.Base(i.path)
}

// filterProjects fuzzy-matches term against targets, best first, except that
// projects whose name starts with or contains term come first. Fuzzy ranking
// alone can put "gamma laravel" above "alpha" for "al".
func filterProjects(term string, targets []string) fuzzy.Matches {
	matches := fuzzy.Find(term, targets)
	term = strings.ToLower(term)
	score := func(m fuzzy.Match) int {
		name, _, _ := strings.Cut(strings.ToLower(m.Str), " ")
		switch {
		case strings.HasPrefix(name, term):
			return 2
		case strings.Contains(name, term):
			return 1
		}
		return 0
	}
	sort.SliceStable(matches, func(i, j int) bool { return score(matches[i]) > score(matches[j]) })
	// The fuzzy matcher reports byte offsets, and lipgloss.StyleRunes takes
	// rune offsets, which differ in a non-ASCII directory name.
	for i, m := range matches {
		runes := make([]int, len(m.MatchedIndexes))
		for j, b := range m.MatchedIndexes {
			runes[j] = utf8.RuneCountInString(m.Str[:b])
		}
		matches[i].MatchedIndexes = runes
	}
	return matches
}

// columns holds the width of each column of the project list. A zero width
// leaves the column out.
type columns struct {
	name, status, kind, path, url int
}

// columnGap separates the columns of the project list.
const columnGap = 2

// projectColumns fits the columns to width, giving any room left after the
// name, status and type to the path and then the URL.
func projectColumns(items []projectItem, width int) columns {
	// The status column fits every status a start or stop passes through,
	// so the columns after it stay put while one runs.
	c := columns{status: ansi.StringWidth(statusSymbol + ddevapp.SiteUnhealthy)}
	maxName, maxPath, maxURL := 0, 0, 0
	for _, p := range items {
		maxName = max(maxName, ansi.StringWidth(p.Name+p.mark))
		c.status = max(c.status, ansi.StringWidth(statusSymbol+statusLabel(p.Status)))
		c.kind = max(c.kind, ansi.StringWidth(p.Type))
		maxPath = max(maxPath, ansi.StringWidth(p.path))
		maxURL = max(maxURL, ansi.StringWidth(p.URL))
	}
	avail := width - 2
	// One long name would otherwise widen the column for every project.
	c.name = min(maxName, max(8, min(20, avail/3)))
	c.kind = min(c.kind, 12)
	rest := avail - c.name - columnGap - c.status - columnGap - c.kind
	if rest > 10 {
		c.path = min(maxPath, rest-columnGap, max(16, avail/2))
		rest -= c.path + columnGap
	}
	if rest > 12 {
		c.url = rest - columnGap
	}
	// A cut name takes back the room the URLs leave empty, so it is cut only
	// when the path or a URL needs that room.
	if grow := min(maxName-c.name, c.url-maxURL); grow > 0 {
		c.name += grow
		c.url -= grow
	}
	return c
}

// row renders project p as one line of aligned columns, highlighting the
// filter's matches, at rune offsets in its FilterValue.
func (v dashboardView) row(p projectItem, matches []int, selected bool) string {
	c := v.cols
	// highlight styles the filter matches in field, which starts at offset in
	// FilterValue and at start in text.
	highlight := func(text, field string, offset, start int, style lipgloss.Style) string {
		var local []int
		n := len([]rune(field))
		for _, i := range matches {
			if i >= offset && i < offset+n {
				local = append(local, start+i-offset)
			}
		}
		if len(local) == 0 {
			return style.Render(text)
		}
		return lipgloss.StyleRunes(text, local, style.Inherit(v.styles.Match), style)
	}
	dir := filepath.Base(p.path)
	typeOffset := len([]rune(p.Name)) + 1
	dirOffset := typeOffset + len([]rune(p.Type)) + 1 + len([]rune(p.Status)) + 1

	cursor, nameStyle := "  ", v.styles.Name
	if selected {
		cursor, nameStyle = v.styles.Selected.Render("> "), v.styles.Selected
	}
	name := ansi.Truncate(highlight(p.Name, p.Name, 0, 0, nameStyle), c.name-len(p.mark), "…") + v.styles.Muted.Render(p.mark)
	gap := strings.Repeat(" ", columnGap)
	row := cursor + fit(name, c.name) + gap +
		fit(v.styles.status(p.Status), c.status) + gap +
		fit(highlight(p.Type, p.Type, typeOffset, 0, v.styles.Muted), c.kind)
	if c.path > 0 {
		path := highlight(p.path, dir, dirOffset, len([]rune(p.path))-len([]rune(dir)), v.styles.URL)
		row += gap + fit(output.Hyperlink(output.FileURL(p.AppRoot), path), c.path)
	}
	if c.url > 0 && p.URL != "" {
		row += gap + ansi.Truncate(v.styles.URL.Render(output.Hyperlink(p.URL, p.URL)), c.url, "…")
	}
	return row
}

// fit truncates or pads s to exactly width cells. Truncating keeps escape
// sequences, so a cut hyperlink is still terminated.
func fit(s string, width int) string {
	s = ansi.Truncate(s, width, "…")
	return s + strings.Repeat(" ", max(0, width-ansi.StringWidth(s)))
}

func formatProjectPath(path string) string {
	if path == "" {
		return ""
	}
	formatted := filepath.ToSlash(path)
	home, err := os.UserHomeDir()
	if err != nil {
		return formatted
	}
	home = filepath.ToSlash(home)
	if formatted == home {
		return "~"
	}
	if strings.HasPrefix(formatted, home+"/") {
		return "~" + strings.TrimPrefix(formatted, home)
	}
	return formatted
}
