package testcommon

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/nodeps"
)

// RequestOption configures a local HTTP request.
type RequestOption func(*requestConfig)

// LocalHTTPResponse holds the status and headers we return to callers. The body
// is returned separately as a string, so there is no open response body to leak
// (unlike *http.Response).
type LocalHTTPResponse struct {
	StatusCode int
	Header     http.Header
}

// GetLocalHTTPResponse requests rawURL on the local Docker IP and returns the
// body and the response status/headers. A status other than expected (default
// 200, see WithExpectStatus) comes back as an error, with the body still
// returned. Like assert, it never fails the test. By default, it makes a single
// request; use WithMaxAttempts/WithBackoff/WithTimeout/WithExpectStatus to tune it.
func GetLocalHTTPResponse(t *testing.T, rawURL string, opts ...RequestOption) (string, *LocalHTTPResponse, error) {
	t.Helper()
	return getLocalHTTPResponse(t, rawURL, newRequestConfig(opts...), nil)
}

// RequireLocalHTTPContent checks that rawURL returns the expected status (default
// 200, see WithExpectStatus) and a body containing wantContent, stopping the test
// (t.FailNow) if not, like require.Contains. By default, it makes a single request;
// use WithMaxAttempts to keep trying until the content shows up (handy when it
// appears a moment later, e.g. a xhprof run written after the response). To look
// at the response yourself, use GetLocalHTTPResponse instead.
func RequireLocalHTTPContent(t *testing.T, rawURL string, wantContent string, opts ...RequestOption) {
	t.Helper()
	checkLocalHTTPContent(t, true, rawURL, wantContent, opts...)
}

// AssertLocalHTTPContent is the non-fatal version of RequireLocalHTTPContent (like
// assert.Contains): it marks the test failed but lets it keep running, and returns
// whether the content was found. Prefer it in loops where you want to check every
// URL instead of stopping at the first failure.
func AssertLocalHTTPContent(t *testing.T, rawURL string, wantContent string, opts ...RequestOption) bool {
	t.Helper()
	return checkLocalHTTPContent(t, false, rawURL, wantContent, opts...)
}

// WithTimeout sets the per-request timeout. Zero or negative disables it.
func WithTimeout(d time.Duration) RequestOption {
	return func(c *requestConfig) {
		if d < 0 {
			d = 0
		}
		c.timeout = d
	}
}

// WithMaxAttempts sets the total number of requests to make (the first try plus
// any retries); values below 1 become 1. Use it to keep trying until the expected
// content shows up or a temporary error goes away.
func WithMaxAttempts(n int) RequestOption {
	return func(c *requestConfig) {
		if n < 1 {
			n = 1
		}
		c.maxAttempts = n
	}
}

// WithBackoff doubles the delay after each attempt (starting at initial, capped at
// the timeout) instead of the fixed 1s delay. Zero or negative initial keeps the
// default 1s start delay.
func WithBackoff(initial time.Duration) RequestOption {
	return func(c *requestConfig) {
		if initial > 0 {
			c.tick = initial
		}
		c.backoff = true
	}
}

// WithExpectStatus sets the status code treated as success; any other code is an
// error (and triggers a retry). Defaults to 200 (http.StatusOK).
func WithExpectStatus(code int) RequestOption {
	return func(c *requestConfig) {
		c.expectStatus = code
	}
}

// WithAlsoContains adds more substrings the body must contain, on top of the
// wantContent passed to Assert/RequireLocalHTTPContent. All must be present, and
// retries continue until they are. No effect on GetLocalHTTPResponse.
func WithAlsoContains(substrings ...string) RequestOption {
	return func(c *requestConfig) {
		c.alsoContains = append(c.alsoContains, substrings...)
	}
}

// WithMessagef adds a printf-style message shown when an Assert/RequireLocalHTTPContent
// check fails. No effect on GetLocalHTTPResponse.
func WithMessagef(format string, args ...any) RequestOption {
	return func(c *requestConfig) {
		c.msgFormat = format
		c.msgArgs = args
	}
}

// requestConfig holds the resolved options for a local HTTP request.
type requestConfig struct {
	// timeout is the per-request HTTP client timeout (zero means no timeout).
	timeout time.Duration
	// maxAttempts is the total number of requests to make (>= 1).
	maxAttempts int
	// tick is the delay between attempts.
	tick time.Duration
	// backoff doubles the delay after each attempt (capped at timeout) when set.
	backoff bool
	// expectStatus is the status code treated as success.
	expectStatus int
	// alsoContains are extra substrings the body must contain, on top of the
	// wantContent argument (see WithAlsoContains).
	alsoContains []string
	// msgFormat/msgArgs are an optional message shown on a content-check failure.
	msgFormat string
	msgArgs   []any
}

// Default request settings, each overridden by the matching With... option.
const (
	defaultHTTPTimeout  = 60 * time.Second // per-request timeout; see WithTimeout
	defaultHTTPTick     = time.Second      // initial delay between attempts; see WithBackoff
	defaultHTTPAttempts = 1                // total attempts; see WithMaxAttempts
	defaultHTTPStatus   = http.StatusOK    // status treated as success; see WithExpectStatus

	// macOSMinAttempts is the attempt floor getLocalHTTPResponse applies on a
	// brief 5xx on macOS (php-fpm SIGBUS), so even a single-shot request retries
	// once.
	macOSMinAttempts = 2
)

// newRequestConfig starts from the defaults above and applies opts.
func newRequestConfig(opts ...RequestOption) requestConfig {
	c := requestConfig{
		timeout:      defaultHTTPTimeout,
		tick:         defaultHTTPTick,
		maxAttempts:  defaultHTTPAttempts,
		expectStatus: defaultHTTPStatus,
	}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// checkLocalHTTPContent fetches rawURL (retrying per WithMaxAttempts until every
// wanted substring appears) and checks the body contains them all. The wanted
// substrings are wantContent plus any from WithAlsoContains. fatal picks require
// (t.FailNow) vs assert behavior; it returns whether all were found.
func checkLocalHTTPContent(t *testing.T, fatal bool, rawURL string, wantContent string, opts ...RequestOption) bool {
	t.Helper()
	cfg := newRequestConfig(opts...)
	// wantContent is the main substring; WithAlsoContains adds more. An empty
	// wantContent is skipped, so we only check the extras (or just the status if
	// there are none).
	var wants []string
	if wantContent != "" {
		wants = append(wants, wantContent)
	}
	wants = append(wants, cfg.alsoContains...)
	missingFrom := func(body string) []string {
		var missing []string
		for _, w := range wants {
			if !strings.Contains(body, w) {
				missing = append(missing, w)
			}
		}
		return missing
	}
	contentOK := func(body string) bool { return len(missingFrom(body)) == 0 }

	body, resp, err := getLocalHTTPResponse(t, rawURL, cfg, contentOK)

	missing := missingFrom(body)
	if err == nil && len(missing) == 0 {
		return true
	}
	userMsg := ""
	if cfg.msgFormat != "" {
		userMsg = fmt.Sprintf(cfg.msgFormat, cfg.msgArgs...)
	}
	t.Errorf("%s", describeHTTPFailure(rawURL, resp, body, err, missing, userMsg))
	if fatal {
		t.FailNow()
	}
	return false
}

// getLocalHTTPResponse makes up to cfg.maxAttempts requests (waiting cfg.tick
// between them) until one succeeds. A request succeeds when there is no error
// and, if contentOK is given, contentOK(body) is true - so callers can wait for
// expected content, not just a working request. Each retry is logged via t.Logf.
func getLocalHTTPResponse(t *testing.T, rawURL string, cfg requestConfig, contentOK func(body string) bool) (string, *LocalHTTPResponse, error) {
	t.Helper()
	address, host, err := localDockerAddress(rawURL)
	if err != nil {
		return "", nil, err
	}

	var body string
	var resp *LocalHTTPResponse
	delay := cfg.tick
	for attempt := 1; ; attempt++ {
		body, resp, err = httpGetLocal(address, host, cfg)
		success := err == nil && (contentOK == nil || contentOK(body))
		limit := cfg.maxAttempts
		// macOS php-fpm sometimes crashes (SIGBUS) and returns a brief 502/503;
		// give it one extra try even for a single-shot request.
		if nodeps.IsMacOS() && resp != nil && resp.StatusCode >= 500 && limit < macOSMinAttempts {
			limit = macOSMinAttempts
		}
		if success || attempt >= limit {
			return body, resp, err
		}
		reason := "expected content not present yet"
		if err != nil {
			reason = err.Error()
		}
		t.Logf("GET %s: attempt %d/%d failed, retrying in %s: %s", rawURL, attempt, limit, delay, reason)
		time.Sleep(delay)
		// With backoff on, wait longer next time, but never past the timeout.
		if cfg.backoff {
			if delay *= 2; delay > cfg.timeout {
				delay = cfg.timeout
			}
		}
	}
}

// httpGetLocal makes a single GET to address, sending host as the Host header and
// TLS server name, and does not follow redirects. A status other than expectStatus
// is returned as an error, with the body still returned. The response body is read
// and closed here, so the caller gets a plain string and nothing to close.
func httpGetLocal(address, host string, cfg requestConfig) (string, *LocalHTTPResponse, error) {
	// ServerName makes TLS verify the cert for host; ErrUseLastResponse stops the
	// client from following redirects. Refs:
	// https://stackoverflow.com/a/47169975/215713 and
	// https://stackoverflow.com/a/38150816/215713
	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{TLSClientConfig: &tls.Config{ServerName: host}},
		Timeout:   cfg.timeout,
	}

	req, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create GET request for %s: %w", address, err)
	}
	req.Host = host

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("unable to read response body from %s: %w", address, err)
	}
	body := string(bodyBytes)
	result := &LocalHTTPResponse{StatusCode: resp.StatusCode, Header: resp.Header}
	if resp.StatusCode != cfg.expectStatus {
		return body, result, fmt.Errorf("status code for %s was %d, not %d", address, resp.StatusCode, cfg.expectStatus)
	}
	return body, result, nil
}

// localDockerAddress points rawURL at the local Docker IP but keeps the original
// hostname for the Host header and TLS server name.
func localDockerAddress(rawURL string) (address, host string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse URL %s: %w", rawURL, err)
	}
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return "", "", fmt.Errorf("failed to get Docker IP: %w", err)
	}
	host = u.Hostname()
	// Read the port before overwriting u.Host, because u.Port() reads it from there.
	port := u.Port()
	u.Host = dockerIP
	if port != "" {
		u.Host = dockerIP + ":" + port
	}
	return u.String(), host, nil
}

// describeHTTPFailure builds the failure message: a Fail line with the caller's
// message, then the request, status, headers, body, and the reason(s) it failed
// (missing substrings and/or a request error).
func describeHTTPFailure(rawURL string, resp *LocalHTTPResponse, body string, err error, missing []string, userMsg string) string {
	status := 0
	var header http.Header
	if resp != nil {
		status = resp.StatusCode
		header = resp.Header
	}
	var reasons []string
	if err != nil {
		reasons = append(reasons, err.Error())
	}
	if len(missing) > 0 {
		reasons = append(reasons, fmt.Sprintf("body does not contain %q", missing))
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "unexpected response")
	}
	if userMsg == "" {
		userMsg = "HTTP request failed"
	}
	parts := []string{
		fmt.Sprintf("Fail: %s", userMsg),
		fmt.Sprintf("GET: %s", rawURL),
		fmt.Sprintf("Status: %d", status),
		fmt.Sprintf("Error: %s", strings.Join(reasons, "; ")),
		fmt.Sprintf("Headers: %q", header),
		fmt.Sprintf("Body: %q", body),
	}
	return strings.Join(parts, "\n")
}
