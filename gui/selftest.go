package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed selftest.js
var selfTestJS string

// selfTest runs selftest.js in the app's own window. The page cannot be
// scripted from outside, so the app adds the script to the dashboard page
// and the script posts its result back.
type selfTest struct {
	// root holds dir and DDEV's global configuration for the test.
	root string
	// dir is an empty directory the script configures as a project.
	dir    string
	result chan selfTestResult
}

type selfTestResult struct {
	OK     bool   `json:"ok"`
	Step   string `json:"step"`
	Error  string `json:"error"`
	Output string `json:"output"`
}

// newSelfTest isolates DDEV's global configuration in a temporary directory,
// so the test's project never reaches the user's project list. It must run
// before ddev gui starts, which inherits the setting.
func newSelfTest() (*selfTest, error) {
	root, err := os.MkdirTemp("", "ddev-gui-selftest-")
	if err != nil {
		return nil, err
	}
	st := &selfTest{root: root, dir: filepath.Join(root, "project"), result: make(chan selfTestResult, 1)}
	for _, d := range []string{st.dir, filepath.Join(root, "xdg")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			st.cleanup()
			return nil, err
		}
	}
	_ = os.Setenv("DDEV_XDG_CONFIG_HOME", filepath.Join(root, "xdg"))
	return st, nil
}

func (st *selfTest) cleanup() {
	_ = os.RemoveAll(st.root)
}

func (st *selfTest) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/selftest.js":
			dir, _ := json.Marshal(st.dir)
			w.Header().Set("Content-Type", "text/javascript")
			_, _ = fmt.Fprintf(w, "const DDEV_SELFTEST_DIR = %s;\n%s", dir, selfTestJS)
		case r.Method == http.MethodPost && r.URL.Path == "/selftest/result":
			var res selfTestResult
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &res); err != nil {
				res = selfTestResult{Step: "report", Error: err.Error()}
			}
			select {
			case st.result <- res:
			default:
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/":
			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, r)
			for k, v := range rec.Header() {
				if k != "Content-Length" {
					w.Header()[k] = v
				}
			}
			w.WriteHeader(rec.Code)
			_, _ = io.WriteString(w, strings.Replace(rec.Body.String(), "</body>", `<script src="/selftest.js"></script></body>`, 1))
		default:
			next.ServeHTTP(w, r)
		}
	})
}

// check adds what only the app can see to the page's result.
func (st *selfTest) check(res selfTestResult) selfTestResult {
	if res.OK {
		if _, err := os.Stat(filepath.Join(st.dir, ".ddev", "config.yaml")); err != nil {
			return selfTestResult{Step: "config.yaml", Error: err.Error(), Output: res.Output}
		}
	}
	return res
}

// runSelfTest waits for selftest.js to report, prints the result, and exits
// with its status. Wails' Quit ends the process without returning from Run.
// tray checks the tray menu once the page has passed.
func runSelfTest(srv *serveClient, st *selfTest, tray func() error) {
	code := 1
	select {
	case res := <-st.result:
		res = st.check(res)
		if res.OK {
			if err := tray(); err != nil {
				res = selfTestResult{Step: "tray menu", Error: err.Error()}
			}
		}
		if res.OK {
			fmt.Println("SELFTEST PASS")
			code = 0
		} else {
			fmt.Printf("SELFTEST FAIL at %s: %s\n%s\n", res.Step, res.Error, res.Output)
		}
	case <-time.After(3 * time.Minute):
		fmt.Println("SELFTEST FAIL: no result from the window within 3 minutes")
	}
	srv.Close()
	st.cleanup()
	os.Exit(code)
}
