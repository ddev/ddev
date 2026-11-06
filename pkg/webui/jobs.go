package webui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/x/ansi"
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

// since returns the lines after the first `from` the job ever had, the index
// to continue from, and whether the job is done.
func (j *job) since(from int) (lines []string, next int, done bool, errText string, changed <-chan struct{}) {
	j.mu.Lock()
	defer j.mu.Unlock()
	start := max(from-j.dropped, 0)
	if start < len(j.lines) {
		lines = slices.Clone(j.lines[start:])
	}
	return lines, j.dropped + len(j.lines), j.done, j.err, j.changed
}

// jobs tracks running and recent jobs, one running job per project at most.
type jobs struct {
	mu       sync.Mutex
	nextID   int
	all      []*job
	onChange func()
	onDone   func(jobSummary)
}

func newJobs(onChange func(), onDone func(jobSummary)) *jobs {
	return &jobs{onChange: onChange, onDone: onDone}
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
		scanLines(pr, j.addLine)
	}()
	go func() {
		err := cmd.Wait()
		_ = pw.Close()
		<-scanned
		j.finish(err)
		js.onChange()
		if js.onDone != nil {
			js.onDone(j.summary())
		}
	}()
	js.onChange()
	return j, nil
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

// scanLines calls add with each line of r as plain text.
func scanLines(r io.Reader, add func(string)) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		add(cleanLine(scanner.Text()))
	}
	// A line too long to scan must not block the subprocess.
	_, _ = io.Copy(io.Discard, r)
}

// cleanLine keeps what a terminal would show of a line: the text after its
// last carriage return, without colors or cursor movement.
func cleanLine(line string) string {
	line = strings.TrimRight(line, "\r\n")
	if i := strings.LastIndexByte(line, '\r'); i >= 0 {
		line = line[i+1:]
	}
	return ansi.Strip(line)
}

// ddevCommand prepares a ddev subcommand in dir, or the current directory
// when dir is empty. It cannot prompt or start a TUI.
func (s *Server) ddevCommand(ctx context.Context, dir string, args ...string) *exec.Cmd {
	c := exec.CommandContext(ctx, s.ddevBinary, args...)
	c.Env = append(slices.Clone(s.baseEnv), "DDEV_NO_TUI=true", "DDEV_NONINTERACTIVE=true")
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

// handleJobStream streams a job's output as `line` events, then an `end`
// event with its error, if any.
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
	next := 0
	for {
		lines, n, done, errText, changed := j.since(next)
		next = n
		if len(lines) > 0 {
			sse.send("lines", lines)
		}
		if done {
			sse.send("end", map[string]string{"err": errText})
			return
		}
		select {
		case <-changed:
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
