package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// TestFile represents a test definition file
type TestFile struct {
	Tests []TestCase `yaml:"tests"`
}

// TestCase represents a single test
type TestCase struct {
	Name    string                 `yaml:"name"`
	Method  string                 `yaml:"method"`
	Path    string                 `yaml:"path"`
	Headers map[string]string      `yaml:"headers"`
	Body    map[string]interface{} `yaml:"body"`
	Expect  TestExpect             `yaml:"expect"`
	Save    map[string]string      `yaml:"save"`
}

// TestExpect represents expected assertions
type TestExpect struct {
	Status    int              `yaml:"status"`
	IsArray   bool             `yaml:"is_array"`
	MinLength *int             `yaml:"min_length"`
	MaxLength *int             `yaml:"max_length"`
	Body      []FieldAssertion `yaml:"body"`
}

// FieldAssertion represents a single field assertion
type FieldAssertion struct {
	Field   string      `yaml:"field"`
	Exists  *bool       `yaml:"exists"`
	Equals  interface{} `yaml:"equals"`
	Matches string      `yaml:"matches"`
	Type    string      `yaml:"type"`
}

// TestResult holds the outcome of a single test
type TestResult struct {
	Name     string
	Passed   bool
	Skipped  bool
	Failures []string
	Duration time.Duration
	Request  *testRequestInfo
	Response *testResponseInfo
}

type testRequestInfo struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

type testResponseInfo struct {
	Status  int
	Headers http.Header
	Body    string
}

var testVerbose bool

func runTest(cmd *cobra.Command, args []string) error {
	specFile := args[0]
	testFile := args[1]
	target, _ := cmd.Flags().GetString("target")
	testVerbose, _ = cmd.Flags().GetBool("verbose")

	// load test file
	testData, err := os.ReadFile(testFile)
	if err != nil {
		return fmt.Errorf("failed to read test file: %w", err)
	}
	var tf TestFile
	if err := yaml.Unmarshal(testData, &tf); err != nil {
		return fmt.Errorf("failed to parse test file: %w", err)
	}

	if len(tf.Tests) == 0 {
		return fmt.Errorf("no tests found in %s", testFile)
	}

	// if no target, spin up internal mock server
	var cleanup func()
	if target == "" {
		addr, cleanupFn, err := startInternalMock(specFile)
		if err != nil {
			return fmt.Errorf("failed to start mock server: %w", err)
		}
		cleanup = cleanupFn
		target = "http://" + addr
		defer cleanup()
	}

	target = strings.TrimRight(target, "/")

	// render banner
	fmt.Println(renderTestBanner(specFile, testFile, target, len(tf.Tests)))

	// run tests sequentially
	savedVars := make(map[string]string)
	var results []TestResult
	passed, failed, skipped := 0, 0, 0

	for _, tc := range tf.Tests {
		result := executeTest(tc, target, savedVars)
		results = append(results, result)

		if result.Skipped {
			skipped++
		} else if result.Passed {
			passed++
		} else {
			failed++
		}

		renderTestResult(result)
	}

	// summary
	fmt.Println()
	renderTestSummary(passed, failed, skipped)

	if failed > 0 {
		os.Exit(1)
	}
	return nil
}

func startInternalMock(specFile string) (string, func(), error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(specFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load spec: %w", err)
	}

	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		router = nil
	}

	mockSeed := time.Now().UnixNano()
	server := &MockServer{
		doc:    doc,
		store:  NewStore(),
		seed:   mockSeed,
		router: router,
		noAuth: true, // disable auth for testing
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleRequest)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}

	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)

	addr := listener.Addr().String()
	cleanup := func() {
		srv.Shutdown(context.Background())
	}

	// give the server a moment to start
	time.Sleep(50 * time.Millisecond)

	return addr, cleanup, nil
}

func executeTest(tc TestCase, target string, savedVars map[string]string) TestResult {
	start := time.Now()
	result := TestResult{Name: tc.Name}

	// interpolate variables in path
	path := interpolateVars(tc.Path, savedVars)
	url := target + path

	// build request body
	var bodyReader io.Reader
	var bodyStr string
	if tc.Body != nil {
		// interpolate vars in body values
		interpolatedBody := interpolateBodyVars(tc.Body, savedVars)
		bodyBytes, _ := json.Marshal(interpolatedBody)
		bodyStr = string(bodyBytes)
		bodyReader = bytes.NewReader(bodyBytes)
	}

	method := strings.ToUpper(tc.Method)
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		result.Failures = append(result.Failures, fmt.Sprintf("failed to create request: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	// set default content type for requests with body
	if tc.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	// set custom headers
	for k, v := range tc.Headers {
		req.Header.Set(k, interpolateVars(v, savedVars))
	}

	result.Request = &testRequestInfo{
		Method:  method,
		URL:     url,
		Headers: tc.Headers,
		Body:    bodyStr,
	}

	// execute
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		result.Failures = append(result.Failures, fmt.Sprintf("request failed: %v", err))
		result.Duration = time.Since(start)
		return result
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	result.Response = &testResponseInfo{
		Status:  resp.StatusCode,
		Headers: resp.Header,
		Body:    string(respBody),
	}

	// assert status
	if tc.Expect.Status != 0 && resp.StatusCode != tc.Expect.Status {
		result.Failures = append(result.Failures,
			fmt.Sprintf("expected status %d, got %d", tc.Expect.Status, resp.StatusCode))
	}

	// parse response body for assertions
	var respData interface{}
	if len(respBody) > 0 {
		json.Unmarshal(respBody, &respData)
	}

	// array assertions
	if tc.Expect.IsArray {
		arr, ok := respData.([]interface{})
		if !ok {
			result.Failures = append(result.Failures, "expected response to be an array")
		} else {
			if tc.Expect.MinLength != nil && len(arr) < *tc.Expect.MinLength {
				result.Failures = append(result.Failures,
					fmt.Sprintf("expected array length >= %d, got %d", *tc.Expect.MinLength, len(arr)))
			}
			if tc.Expect.MaxLength != nil && len(arr) > *tc.Expect.MaxLength {
				result.Failures = append(result.Failures,
					fmt.Sprintf("expected array length <= %d, got %d", *tc.Expect.MaxLength, len(arr)))
			}
		}
	}

	// field assertions
	for _, fa := range tc.Expect.Body {
		failures := assertField(respData, fa)
		result.Failures = append(result.Failures, failures...)
	}

	// save variables
	if tc.Save != nil && respData != nil {
		for varName, fieldPath := range tc.Save {
			val := getFieldValue(respData, fieldPath)
			if val != nil {
				savedVars[varName] = fmt.Sprintf("%v", val)
			}
		}
	}

	result.Passed = len(result.Failures) == 0
	result.Duration = time.Since(start)
	return result
}

func assertField(data interface{}, fa FieldAssertion) []string {
	var failures []string
	val := getFieldValue(data, fa.Field)

	// exists check
	if fa.Exists != nil {
		if *fa.Exists && val == nil {
			failures = append(failures, fmt.Sprintf("field '%s' expected to exist but not found", fa.Field))
			return failures
		}
		if !*fa.Exists && val != nil {
			failures = append(failures, fmt.Sprintf("field '%s' expected to not exist but found", fa.Field))
			return failures
		}
	}

	if val == nil && (fa.Equals != nil || fa.Matches != "" || fa.Type != "") {
		failures = append(failures, fmt.Sprintf("field '%s' not found", fa.Field))
		return failures
	}

	// equals check
	if fa.Equals != nil {
		actual := fmt.Sprintf("%v", val)
		expected := fmt.Sprintf("%v", fa.Equals)
		if actual != expected {
			failures = append(failures, fmt.Sprintf("field '%s': expected '%v', got '%v'", fa.Field, fa.Equals, val))
		}
	}

	// matches check (regex)
	if fa.Matches != "" {
		actual := fmt.Sprintf("%v", val)
		re, err := regexp.Compile(fa.Matches)
		if err != nil {
			failures = append(failures, fmt.Sprintf("field '%s': invalid regex '%s': %v", fa.Field, fa.Matches, err))
		} else if !re.MatchString(actual) {
			failures = append(failures, fmt.Sprintf("field '%s': value '%s' doesn't match pattern '%s'", fa.Field, actual, fa.Matches))
		}
	}

	// type check
	if fa.Type != "" {
		typeOk := false
		switch fa.Type {
		case "string":
			_, typeOk = val.(string)
		case "number":
			_, typeOk = val.(float64)
		case "boolean":
			_, typeOk = val.(bool)
		case "array":
			_, typeOk = val.([]interface{})
		case "object":
			_, typeOk = val.(map[string]interface{})
		}
		if !typeOk {
			failures = append(failures, fmt.Sprintf("field '%s': expected type '%s', got %T", fa.Field, fa.Type, val))
		}
	}

	return failures
}

// getFieldValue retrieves a value from nested data using dot notation
func getFieldValue(data interface{}, path string) interface{} {
	if data == nil || path == "" {
		return data
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil
			}
			current = val
		case []interface{}:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(v) {
				return nil
			}
			current = v[idx]
		default:
			return nil
		}
	}
	return current
}

func interpolateVars(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}

func interpolateBodyVars(body map[string]interface{}, vars map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range body {
		if s, ok := v.(string); ok {
			result[k] = interpolateVars(s, vars)
		} else if m, ok := v.(map[string]interface{}); ok {
			result[k] = interpolateBodyVars(m, vars)
		} else {
			result[k] = v
		}
	}
	return result
}

// ---- Test TUI rendering ----

func renderTestBanner(specFile, testFile, target string, testCount int) string {
	var b strings.Builder
	title := styleTitle.Render("⬛ portblock test") + " " + styleVersion.Render("v"+version)
	b.WriteString(title + "\n")
	b.WriteString(styleLabel.Render("spec") + styleValue.Render(specFile) + "\n")
	b.WriteString(styleLabel.Render("tests") + styleValue.Render(testFile) + "\n")
	b.WriteString(styleLabel.Render("target") + styleURL.Render(target) + "\n")
	b.WriteString(styleLabel.Render("count") + styleValue.Render(fmt.Sprintf("%d tests", testCount)) + "\n")
	return styleBanner.Render(b.String())
}

func renderTestResult(r TestResult) {
	var icon, name string
	dur := lipgloss.NewStyle().Foreground(colorDim).Render(r.Duration.Round(time.Millisecond).String())

	if r.Skipped {
		icon = lipgloss.NewStyle().Foreground(colorDim).Render("⏭️ ")
		name = lipgloss.NewStyle().Foreground(colorDim).Render(r.Name)
	} else if r.Passed {
		icon = lipgloss.NewStyle().Foreground(colorGreen).Render("✅")
		name = lipgloss.NewStyle().Foreground(colorWhite).Render(r.Name)
	} else {
		icon = lipgloss.NewStyle().Foreground(colorRed).Render("❌")
		name = lipgloss.NewStyle().Foreground(colorWhite).Render(r.Name)
	}

	fmt.Printf("  %s %s %s\n", icon, name, dur)

	// show failures
	for _, f := range r.Failures {
		detail := lipgloss.NewStyle().Foreground(colorRed).Render("     " + f)
		fmt.Println(detail)
	}

	// verbose output
	if testVerbose && r.Request != nil {
		reqStyle := lipgloss.NewStyle().Foreground(colorDim)
		fmt.Println(reqStyle.Render(fmt.Sprintf("     → %s %s", r.Request.Method, r.Request.URL)))
		if r.Request.Body != "" {
			fmt.Println(reqStyle.Render("     → body: " + r.Request.Body))
		}
		if r.Response != nil {
			fmt.Println(reqStyle.Render(fmt.Sprintf("     ← %d", r.Response.Status)))
			if r.Response.Body != "" {
				body := r.Response.Body
				if len(body) > 500 {
					body = body[:500] + "..."
				}
				fmt.Println(reqStyle.Render("     ← body: " + body))
			}
		}
	}
}

func renderTestSummary(passed, failed, skipped int) {
	total := passed + failed + skipped
	sep := styleSeparator.Render(strings.Repeat("─", 44))
	fmt.Println("  " + sep)

	parts := []string{}
	if passed > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render(fmt.Sprintf("%d passed", passed)))
	}
	if failed > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render(fmt.Sprintf("%d failed", failed)))
	}
	if skipped > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(colorDim).Render(fmt.Sprintf("%d skipped", skipped)))
	}

	summary := strings.Join(parts, lipgloss.NewStyle().Foreground(colorDim).Render(" · "))
	totalStr := lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf(" (%d total)", total))

	fmt.Printf("  %s%s\n", summary, totalStr)
}
