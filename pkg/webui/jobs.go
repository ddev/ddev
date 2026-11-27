package webui

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// maxJobLines bounds the output a job keeps for late subscribers.
const maxJobLines = 5000

// maxFinishedJobs is how many finished jobs are kept for the activity list.
const maxFinishedJobs = 30

// job is a ddev subcommand started from the dashboard.
type job struct {
	ID      int       `json:"id"`
	Title   string    `json:"title"`
	Project string    `json:"project"`
	Started time.Time `json:"started"`

	mu      sync.Mutex
	lines   []string
	dropped int
	// partial is the last line while it has no newline yet, such as
	// "Building project images...." while the dots arrive.
	partial string
	done    bool
	err     string
	cmd     *exec.Cmd
	// changed is closed and replaced on each new line and at the end.
	changed chan struct{}
}

// jobSummary is what the status stream reports about a job.
type jobSummary struct {
	ID      int       `json:"id"`
	Title   string    `json:"title"`
	Project string    `json:"project"`
	Started time.Time `json:"started"`
	Done    bool      `json:"done"`
	Err     string    `json:"err,omitempty"`
}

func (j *job) summary() jobSummary {
	j.mu.Lock()
	defer j.mu.Unlock()
	return jobSummary{ID: j.ID, Title: j.Title, Project: j.Project, Started: j.Started, Done: j.done, Err: j.err}
}

func (j *job) addLine(line string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.lines = append(j.lines, line)
	if len(j.lines) > maxJobLines {
		n := len(j.lines) - maxJobLines/2
		j.dropped += n
		j.lines = slices.Clone(j.lines[n:])
	}
	close(j.changed)
	j.changed = make(chan struct{})
}

func (j *job) setPartial(text string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if text == j.partial {
		return
	}
	j.partial = text
	close(j.changed)
	j.changed = make(chan struct{})
}

func (j *job) finish(err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.done = true
	if err != nil {
		j.err = err.Error()
	}
	close(j.changed)
	j.changed = make(chan struct{})
}

// jobView is a job's output after a given line.
type jobView struct {
	lines   []string
	next    int // the index to continue from
	partial string
	done    bool
	err     string
	changed <-chan struct{}
}

// since returns the output after the first `from` lines the job ever had.
func (j *job) since(from int) jobView {
	j.mu.Lock()
	defer j.mu.Unlock()
	v := jobView{next: j.dropped + len(j.lines), partial: j.partial, done: j.done, err: j.err, changed: j.changed}
	if start := max(from-j.dropped, 0); start < len(j.lines) {
		v.lines = slices.Clone(j.lines[start:])
	}
	return v
}

// jobs tracks running and recent jobs, one running job per project at most.
type jobs struct {
	mu       sync.Mutex
	nextID   int
	all      []*job
	onChange func()
}

func newJobs(onChange func()) *jobs {
	return &jobs{onChange: onChange}
}

// errBusy is returned when the project already has a running job.
type errBusy struct{ title string }

func (e errBusy) Error() string { return "already running: " + e.title }

// start runs cmd as a job for project ("" for global actions, which
// conflict with every project's jobs).
func (js *jobs) start(title, project string, cmd *exec.Cmd) (*job, error) {
	js.mu.Lock()
	for _, j := range js.all {
		s := j.summary()
		if !s.Done && (s.Project == project || s.Project == "" || project == "") {
			js.mu.Unlock()
			return nil, errBusy{s.Title}
		}
	}
	js.nextID++
	j := &job{ID: js.nextID, Title: title, Project: project, Started: time.Now(), cmd: cmd, changed: make(chan struct{})}

	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	j.addLine("$ ddev " + strings.Join(cmd.Args[1:], " "))
	if err := cmd.Start(); err != nil {
		js.mu.Unlock()
		return nil, err
	}
	js.all = append(js.all, j)
	js.prune()
	js.mu.Unlock()

	scanned := make(chan struct{})
	go func() {
		defer close(scanned)
		readOutput(pr, j.addLine, j.setPartial)
	}()
	go func() {
		err := cmd.Wait()
		_ = pw.Close()
		<-scanned
		j.finish(err)
		js.onChange()
	}()
	js.onChange()
	return j, nil
}

// remove drops a finished job, and reports whether it did.
func (js *jobs) remove(id int) bool {
	js.mu.Lock()
	n := len(js.all)
	js.all = slices.DeleteFunc(js.all, func(j *job) bool { return j.ID == id && j.summary().Done })
	removed := len(js.all) < n
	js.mu.Unlock()
	if removed {
		js.onChange()
	}
	return removed
}

// clearFinished drops every finished job.
func (js *jobs) clearFinished() {
	js.mu.Lock()
	js.all = slices.DeleteFunc(js.all, func(j *job) bool { return j.summary().Done })
	js.mu.Unlock()
	js.onChange()
}

// prune drops the oldest finished jobs. The caller holds js.mu.
func (js *jobs) prune() {
	finished := 0
	for _, j := range js.all {
		if j.summary().Done {
			finished++
		}
	}
	js.all = slices.DeleteFunc(js.all, func(j *job) bool {
		if finished > maxFinishedJobs && j.summary().Done {
			finished--
			return true
		}
		return false
	})
}

func (js *jobs) get(id int) *job {
	js.mu.Lock()
	defer js.mu.Unlock()
	for _, j := range js.all {
		if j.ID == id {
			return j
		}
	}
	return nil
}

func (js *jobs) summaries() []jobSummary {
	js.mu.Lock()
	defer js.mu.Unlock()
	out := make([]jobSummary, 0, len(js.all))
	for _, j := range js.all {
		out = append(out, j.summary())
	}
	return out
}

func (js *jobs) killAll() {
	js.mu.Lock()
	defer js.mu.Unlock()
	for _, j := range js.all {
		if !j.summary().Done && j.cmd.Process != nil {
			_ = j.cmd.Process.Kill()
		}
	}
}

// scanLines calls add with each line of r, cleaned by cleanLine.
func scanLines(r io.Reader, add func(string)) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		add(cleanLine(scanner.Text()))
	}
	// A line too long to scan must not block the subprocess.
	_, _ = io.Copy(io.Discard, r)
}

// maxPartial bounds an unfinished line, such as a progress bar that only
// ever returns the cursor.
const maxPartial = 64 * 1024

// readOutput is scanLines that also reports the last line as it grows,
// so output that arrives without a newline shows at once.
func readOutput(r io.Reader, add func(string), partial func(string)) {
	buf := make([]byte, 32*1024)
	var pending []byte
	for {
		n, err := r.Read(buf)
		pending = append(pending, buf[:n]...)
		for {
			i := bytes.IndexByte(pending, '\n')
			if i < 0 {
				break
			}
			add(cleanLine(string(pending[:i])))
			pending = pending[i+1:]
		}
		if len(pending) > maxPartial {
			if i := bytes.LastIndexByte(pending, '\r'); i >= 0 {
				pending = pending[i+1:]
			}
			if len(pending) > maxPartial {
				add(cleanLine(string(pending)))
				pending = nil
			}
		}
		if err != nil {
			if len(pending) > 0 {
				add(cleanLine(string(pending)))
			}
			partial("")
			return
		}
		partial(cleanLine(string(trimIncompleteRune(pending))))
	}
}

// trimIncompleteRune cuts a UTF-8 sequence that the next read completes.
func trimIncompleteRune(b []byte) []byte {
	for i := len(b) - 1; i >= 0 && i >= len(b)-utf8.UTFMax; i-- {
		if utf8.RuneStart(b[i]) {
			if !utf8.FullRune(b[i:]) {
				return b[:i]
			}
			break
		}
	}
	return b
}

// escape matches an escape sequence: CSI, such as a color or a cursor
// movement, OSC, such as a hyperlink, or a two-character one. Last come a
// CSI cut short at the end and a lone escape.
var escape = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\)|[@-Z\\-_]|\[[0-?]*[ -/]*$|)`)

// cleanLine keeps what a terminal would show of a line: the text after its
// last carriage return, with its colors but without cursor movement or
// hyperlinks.
func cleanLine(line string) string {
	line = strings.TrimRight(line, "\r\n")
	if i := strings.LastIndexByte(line, '\r'); i >= 0 {
		line = line[i+1:]
	}
	return escape.ReplaceAllStringFunc(line, func(seq string) string {
		if strings.HasPrefix(seq, "\x1b[") && strings.HasSuffix(seq, "m") {
			return seq
		}
		return ""
	})
}

// ddevCommand prepares a ddev subcommand in dir, or the current directory
// when dir is empty. It cannot prompt or start a TUI.
func (s *Server) ddevCommand(ctx context.Context, dir string, args ...string) *exec.Cmd {
	c := exec.CommandContext(ctx, s.ddevBinary, args...)
	// The page shows ddev's colors, which ddev leaves out on a pipe.
	c.Env = append(slices.Clone(s.baseEnv), "DDEV_NO_TUI=true", "DDEV_NONINTERACTIVE=true", "CLICOLOR_FORCE=1")
	c.Dir = dir
	return c
}

// startJob runs a ddev subcommand as a job and replies with its id.
func (s *Server) startJob(w http.ResponseWriter, title, project, dir string, args ...string) {
	// Jobs outlive the request that started them.
	j, err := s.jobs.start(title, project, s.ddevCommand(context.Background(), dir, args...))
	if err != nil {
		status := http.StatusInternalServerError
		if _, ok := err.(errBusy); ok {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, http.StatusAccepted, j.summary())
}

func (s *Server) handleRemoveJob(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	if !s.jobs.remove(id) {
		http.Error(w, "no finished job with that id", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleClearJobs(w http.ResponseWriter, _ *http.Request) {
	s.jobs.clearFinished()
	w.WriteHeader(http.StatusNoContent)
}

// handleJobStream streams a job's output as `lines` events, its unfinished
// last line as `partial` events, then an `end` event with its error, if any.
// A `lines` event ends the partial line the client shows.
func (s *Server) handleJobStream(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	j := s.jobs.get(id)
	if j == nil {
		http.NotFound(w, r)
		return
	}
	sse, ok := newSSE(w)
	if !ok {
		return
	}
	next, shown := 0, ""
	for {
		v := j.since(next)
		next = v.next
		if len(v.lines) > 0 {
			sse.send("lines", v.lines)
			shown = ""
		}
		if v.partial != shown {
			sse.send("partial", v.partial)
			shown = v.partial
		}
		if v.done {
			sse.send("end", map[string]string{"err": v.err})
			return
		}
		select {
		case <-v.changed:
		case <-r.Context().Done():
			return
		}
	}
}

// sse writes Server-Sent Events.
type sse struct {
	w http.ResponseWriter
	f http.Flusher
}

func newSSE(w http.ResponseWriter) (*sse, bool) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	f.Flush()
	return &sse{w: w, f: f}, true
}

func (s *sse) send(event string, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, b)
	s.f.Flush()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
