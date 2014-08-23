// Package honeybadger provides basic exception reporting to Honeybadger.io.
// Adapted from hk's Rollbar implementation:
//   https://github.com/heroku/hk/blob/master/rollbar/rollbar.go
package honeybadger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

const (
	DefaultEndpoint     = "https://api.honeybadger.io/v1/notices"
	DefaultNotifierName = "honeybadger-go"
	DefaultNotifierURL  = "https://github.com/hashrabbit/honeybadger-go"
)

type Client struct {
	// APIKey is the Honeybadger API key for a given project by which notices
	// are for.
	APIKey string

	// ProjectRoot is the root path of the project. Automatically detected as the
	// directory name of first method's location in the app (main).
	ProjectRoot string

	// Context maps arbitrary keys and values containing additional details about
	// the state of an app prior to a notice being sent.
	Context Context

	// Endpoint is the URL to send notices to.
	Endpoint string

	// Notifier references the library responsible for sending notices.
	NotifierName string

	// NotifierURL references the homepage of the library responsible for
	// sending notices.
	NotifierURL string
}

// New returns a new honeybadger.Client with apiKey for sending notices.
func New(apiKey string) *Client {
	return &Client{
		APIKey:       apiKey,
		ProjectRoot:  detectProjectRoot(),
		Context:      make(Context),
		Endpoint:     DefaultEndpoint,
		NotifierName: DefaultNotifierName,
		NotifierURL:  DefaultNotifierURL,
	}
}

func detectProjectRoot() string {
	files := []string{}

	for i := 1; ; i++ {
		_, file, _, ok := runtime.Caller(i)
		if !ok {
			break
		}
		files = append(files, file)
	}

	if len(files) < 3 {
		return ""
	}

	return filepath.Dir(files[len(files)-3])
}

// Report sends a message to Honeybadger along with additional information
// (stacktrace, Go version, architecture, and operating system) and Context.
// Returns Honeybadger error ID for informing end-users.
func (c *Client) Report(e interface{}) (string, error) {
	msg := ""
	switch e := e.(type) {
	case error:
		msg = e.Error()
	default:
		msg = fmt.Sprintf("%v", e)
	}

	notice := c.buildNotice(msg, 2)

	jsonBody, err := json.Marshal(notice)
	if err != nil {
		return "", err
	}

	client := &http.Client{}

	req, err := http.NewRequest("POST", c.Endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)

	res, err := client.Do(req)
	defer res.Body.Close()
	if err != nil {
		return "", err
	}

	if res.StatusCode/100 != 2 { // 200, 201, 202, etc
		return "", fmt.Errorf("unexpected status code %d", res.StatusCode)
	}

	return extractErrorID(res)
}

// Reportf formats according to a format specifier before sending a message to
// Honeybadger through Report.
func (c *Client) Reportf(format string, params ...interface{}) (string, error) {
	return c.Report(fmt.Sprintf(format, params...))
}

func (c *Client) buildNotice(message string, skip int) map[string]interface{} {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}

	return map[string]interface{}{
		"notifier": map[string]interface{}{
			"name":     c.NotifierName,
			"url":      c.NotifierURL,
			"language": "go",
		},
		"error": map[string]interface{}{
			"class":     "",
			"message":   message,
			"backtrace": c.stacktraceFrames(3 + skip),
		},
		"request": map[string]interface{}{
			"cgi_data": map[string]interface{}{
				"GOARCH": runtime.GOARCH,
				"GOOS":   runtime.GOOS,
				"GOVER":  runtime.Version(),
			},
			"context": c.Context,
		},
		"server": map[string]interface{}{
			"environment_name": "production",
			"hostname":         hostname,
			"project_root":     c.ProjectRoot,
		},
	}
}

var rootFilter = regexp.MustCompile("^" + regexp.QuoteMeta(runtime.GOROOT()))

func (c *Client) filterPath(file string) string {
	file = rootFilter.ReplaceAllString(file, "[GO_ROOT]")

	if c.ProjectRoot != "" {
		projectPat := regexp.MustCompile("^" + regexp.QuoteMeta(c.ProjectRoot))
		file = projectPat.ReplaceAllString(file, "[PROJECT_ROOT]")
	}

	return file
}

func (c *Client) stacktraceFrames(skip int) []map[string]interface{} {
	frames := []map[string]interface{}{}

	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		file = c.filterPath(file)

		f := runtime.FuncForPC(pc)
		fname := "unknown"
		if f != nil {
			fname = f.Name()
		}

		frames = append(frames, map[string]interface{}{
			"file":   file,
			"number": line,
			"method": fname,
		})
	}
	return frames
}

func extractErrorID(res *http.Response) (string, error) {
	jsonBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	notice := struct {
		ID string `json:"id"`
	}{}

	err = json.Unmarshal(jsonBody, &notice)
	if err != nil {
		return "", err
	}

	return notice.ID, nil
}
