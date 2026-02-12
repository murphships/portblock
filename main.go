package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	stdlog "log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/spf13/cobra"
)

var (
	port    int
	seed    int64
	delay   time.Duration
	chaos   bool
	noAuth  bool
	version = "0.2.0"
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "portblock",
		Short:   "mock APIs that actually behave like real ones",
		Version: version,
	}

	serveCmd := &cobra.Command{
		Use:   "serve [spec-file]",
		Short: "start a mock server from an OpenAPI spec",
		Args:  cobra.ExactArgs(1),
		RunE:  runServe,
	}

	serveCmd.Flags().IntVarP(&port, "port", "p", 4000, "port to listen on")
	serveCmd.Flags().Int64Var(&seed, "seed", 0, "random seed for reproducible data (0 = random)")
	serveCmd.Flags().DurationVar(&delay, "delay", 0, "simulated latency per request (e.g. 200ms)")
	serveCmd.Flags().BoolVar(&chaos, "chaos", false, "chaos mode — random 500s and latency spikes")
	serveCmd.Flags().BoolVar(&noAuth, "no-auth", false, "skip auth simulation")

	proxyCmd := &cobra.Command{
		Use:   "proxy [spec-file]",
		Short: "proxy to a real API and validate against the spec",
		Args:  cobra.ExactArgs(1),
		RunE:  runProxy,
	}
	var proxyTarget string
	var proxyRecord string
	proxyCmd.Flags().IntVarP(&port, "port", "p", 4000, "port to listen on")
	proxyCmd.Flags().StringVar(&proxyTarget, "target", "", "target base URL to proxy to (required)")
	proxyCmd.Flags().StringVar(&proxyRecord, "record", "", "file to record responses to")
	proxyCmd.MarkFlagRequired("target")

	replayCmd := &cobra.Command{
		Use:   "replay [recordings-file]",
		Short: "replay recorded responses",
		Args:  cobra.ExactArgs(1),
		RunE:  runReplay,
	}
	replayCmd.Flags().IntVarP(&port, "port", "p", 4000, "port to listen on")

	rootCmd.AddCommand(serveCmd, proxyCmd, replayCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServe(cmd *cobra.Command, args []string) error {
	specFile := args[0]

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(specFile)
	if err != nil {
		return fmt.Errorf("failed to load spec: %w", err)
	}

	if err := doc.Validate(context.Background()); err != nil {
		stdlog.Printf("warning: spec validation issues: %v", err)
	}

	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		stdlog.Printf("warning: could not build validation router: %v", err)
	}

	server := &MockServer{
		doc:    doc,
		store:  NewStore(),
		seed:   seed,
		router: router,
		noAuth: noAuth,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleRequest)

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{Addr: addr, Handler: mux}

	// render banner
	fmt.Println(renderBanner("serve", specFile, port, seed, delay, chaos, noAuth))

	// collect and render routes
	var routes []routeInfo
	for path, pathItem := range doc.Paths.Map() {
		methods := []string{}
		if pathItem.Get != nil {
			methods = append(methods, "GET")
		}
		if pathItem.Post != nil {
			methods = append(methods, "POST")
		}
		if pathItem.Put != nil {
			methods = append(methods, "PUT")
		}
		if pathItem.Patch != nil {
			methods = append(methods, "PATCH")
		}
		if pathItem.Delete != nil {
			methods = append(methods, "DELETE")
		}
		routes = append(routes, routeInfo{path: path, methods: methods})
	}
	fmt.Print(renderRoutes(routes))
	fmt.Print(renderReady(port))

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Print(renderShutdown())
		srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// --------------- Store ---------------

type Store struct {
	mu      sync.RWMutex
	data    map[string]map[string]interface{}
	written map[string]bool
}

func NewStore() *Store {
	return &Store{
		data:    make(map[string]map[string]interface{}),
		written: make(map[string]bool),
	}
}

func (s *Store) HasBeenWritten(resource string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.written[resource]
}

func (s *Store) Get(resource, id string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	col, ok := s.data[resource]
	if !ok {
		return nil, false
	}
	obj, ok := col[id]
	return obj, ok
}

func (s *Store) List(resource string) []interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	col := s.data[resource]
	result := make([]interface{}, 0, len(col))
	for _, v := range col {
		result = append(result, v)
	}
	return result
}

func (s *Store) Put(resource, id string, obj interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[resource] == nil {
		s.data[resource] = make(map[string]interface{})
	}
	s.data[resource][id] = obj
	s.written[resource] = true
}

func (s *Store) Delete(resource, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	col := s.data[resource]
	if col == nil {
		return false
	}
	if _, ok := col[id]; !ok {
		return false
	}
	delete(col, id)
	return true
}

// --------------- MockServer ---------------

type MockServer struct {
	doc    *openapi3.T
	store  *Store
	seed   int64
	router routers.Router
	noAuth bool
}

var pathParamRe = regexp.MustCompile(`\{([^}]+)\}`)

func (s *MockServer) findRoute(reqPath, reqMethod string) (*openapi3.PathItem, *openapi3.Operation, map[string]string) {
	for pattern, pathItem := range s.doc.Paths.Map() {
		params := matchPath(pattern, reqPath)
		if params == nil {
			continue
		}
		op := getOperation(pathItem, reqMethod)
		if op != nil {
			return pathItem, op, params
		}
	}
	return nil, nil, nil
}

func matchPath(pattern, actual string) map[string]string {
	regexStr := "^" + pathParamRe.ReplaceAllString(pattern, `([^/]+)`) + "$"
	re, err := regexp.Compile(regexStr)
	if err != nil {
		return nil
	}
	matches := re.FindStringSubmatch(actual)
	if matches == nil {
		return nil
	}
	paramNames := pathParamRe.FindAllStringSubmatch(pattern, -1)
	params := make(map[string]string)
	for i, name := range paramNames {
		params[name[1]] = matches[i+1]
	}
	return params
}

func getOperation(item *openapi3.PathItem, method string) *openapi3.Operation {
	switch strings.ToUpper(method) {
	case "GET":
		return item.Get
	case "POST":
		return item.Post
	case "PUT":
		return item.Put
	case "PATCH":
		return item.Patch
	case "DELETE":
		return item.Delete
	case "OPTIONS":
		return item.Options
	case "HEAD":
		return item.Head
	}
	return nil
}

// --------------- Content Negotiation ---------------

func negotiateContentType(r *http.Request) string {
	accept := r.Header.Get("Accept")
	if accept == "" || accept == "*/*" {
		return "application/json"
	}
	for _, part := range strings.Split(accept, ",") {
		mt := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		switch mt {
		case "application/json", "*/*":
			return "application/json"
		case "application/xml", "text/xml":
			return "application/xml"
		}
	}
	return "" // unsupported
}

func writeResponse(w http.ResponseWriter, contentType string, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)
	if data == nil {
		return
	}
	switch contentType {
	case "application/xml", "text/xml":
		writeXML(w, data)
	default:
		json.NewEncoder(w).Encode(data)
	}
}

// XMLMap is a wrapper for encoding maps as XML
type XMLMap struct {
	XMLName xml.Name
	Items   []XMLEntry
}

type XMLEntry struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
	Items   []XMLEntry
}

func writeXML(w io.Writer, data interface{}) {
	w.Write([]byte(xml.Header))
	switch v := data.(type) {
	case map[string]interface{}:
		w.Write([]byte("<root>"))
		writeXMLMap(w, v)
		w.Write([]byte("</root>"))
	case []interface{}:
		w.Write([]byte("<root>"))
		for _, item := range v {
			w.Write([]byte("<item>"))
			if m, ok := item.(map[string]interface{}); ok {
				writeXMLMap(w, m)
			} else {
				fmt.Fprintf(w, "%v", item)
			}
			w.Write([]byte("</item>"))
		}
		w.Write([]byte("</root>"))
	default:
		fmt.Fprintf(w, "<root>%v</root>", v)
	}
}

func writeXMLMap(w io.Writer, m map[string]interface{}) {
	for k, v := range m {
		safe := xml.EscapeText
		_ = safe
		fmt.Fprintf(w, "<%s>", k)
		switch val := v.(type) {
		case map[string]interface{}:
			writeXMLMap(w, val)
		case []interface{}:
			for _, item := range val {
				fmt.Fprintf(w, "<item>")
				if m2, ok := item.(map[string]interface{}); ok {
					writeXMLMap(w, m2)
				} else {
					buf := &bytes.Buffer{}
					xml.EscapeText(buf, []byte(fmt.Sprintf("%v", item)))
					w.Write(buf.Bytes())
				}
				fmt.Fprintf(w, "</item>")
			}
		default:
			buf := &bytes.Buffer{}
			xml.EscapeText(buf, []byte(fmt.Sprintf("%v", val)))
			w.Write(buf.Bytes())
		}
		fmt.Fprintf(w, "</%s>", k)
	}
}

// --------------- Auth Simulation ---------------

func (s *MockServer) checkAuth(w http.ResponseWriter, r *http.Request, op *openapi3.Operation) bool {
	if s.noAuth {
		return true
	}

	// collect security requirements: operation-level or global
	secReqs := op.Security
	if secReqs == nil {
		secReqs = &s.doc.Security
	}
	if secReqs == nil || len(*secReqs) == 0 {
		return true
	}

	schemes := s.doc.Components
	if schemes == nil {
		return true
	}

	// any one security requirement set must pass (OR logic)
	for _, reqSet := range *secReqs {
		allPassed := true
		for schemeName := range reqSet {
			schemeRef, ok := schemes.SecuritySchemes[schemeName]
			if !ok || schemeRef.Value == nil {
				allPassed = false
				break
			}
			ss := schemeRef.Value
			switch ss.Type {
			case "http":
				auth := r.Header.Get("Authorization")
				if auth == "" {
					allPassed = false
				} else if strings.ToLower(ss.Scheme) == "bearer" && !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
					allPassed = false
				}
			case "apiKey":
				switch ss.In {
				case "header":
					if r.Header.Get(ss.Name) == "" {
						allPassed = false
					}
				case "query":
					if r.URL.Query().Get(ss.Name) == "" {
						allPassed = false
					}
				case "cookie":
					if _, err := r.Cookie(ss.Name); err != nil {
						allPassed = false
					}
				}
			case "oauth2", "openIdConnect":
				if r.Header.Get("Authorization") == "" {
					allPassed = false
				}
			}
			if !allPassed {
				break
			}
		}
		if allPassed {
			return true
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(401)
	json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	return false
}

// --------------- Request Validation ---------------

func (s *MockServer) validateRequest(w http.ResponseWriter, r *http.Request, body []byte) bool {
	if s.router == nil {
		return true
	}

	route, pathParams, err := s.router.FindRoute(r)
	if err != nil {
		return true // can't find route in validator, skip validation
	}

	input := &openapi3filter.RequestValidationInput{
		Request:    r,
		PathParams: pathParams,
		Route:      route,
		Options: &openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
	}

	// restore body for validation
	if len(body) > 0 {
		r.Body = io.NopCloser(bytes.NewReader(body))
	}

	err = openapi3filter.ValidateRequest(context.Background(), input)
	if err != nil {
		details := parseValidationError(err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "validation failed",
			"details": details,
		})
		return false
	}

	// restore body again for handler
	if len(body) > 0 {
		r.Body = io.NopCloser(bytes.NewReader(body))
	}

	return true
}

func parseValidationError(err error) []map[string]string {
	details := []map[string]string{}
	errStr := err.Error()

	// extract only the meaningful error lines (property errors)
	lines := strings.Split(errStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// only include lines with actual error info
		if !strings.Contains(line, "property") && !strings.Contains(line, "error") && !strings.Contains(line, "must") && !strings.Contains(line, "invalid") {
			continue
		}
		detail := map[string]string{}

		// extract field name from "property "X" is missing" patterns
		if strings.Contains(line, "property") {
			parts := strings.SplitN(line, "property", 2)
			if len(parts) > 1 {
				fieldPart := strings.TrimSpace(parts[1])
				if idx := strings.IndexByte(fieldPart, '"'); idx >= 0 {
					end := strings.IndexByte(fieldPart[idx+1:], '"')
					if end >= 0 {
						detail["field"] = fieldPart[idx+1 : idx+1+end]
					}
				}
			}
			// simplify message
			if field, ok := detail["field"]; ok {
				if strings.Contains(line, "missing") {
					detail["message"] = "required field missing"
				} else {
					detail["message"] = "invalid value for field " + field
				}
			} else {
				detail["message"] = line
			}
		} else {
			detail["message"] = line
		}
		details = append(details, detail)
	}

	if len(details) == 0 {
		details = append(details, map[string]string{"message": errStr})
	}
	return details
}

// --------------- Prefer Header ---------------

func parsePreferCode(r *http.Request) int {
	prefer := r.Header.Get("Prefer")
	if prefer == "" {
		return 0
	}
	for _, part := range strings.Split(prefer, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "code=") {
			code, err := strconv.Atoi(strings.TrimPrefix(part, "code="))
			if err == nil {
				return code
			}
		}
	}
	return 0
}

func (s *MockServer) handlePreferCode(w http.ResponseWriter, r *http.Request, op *openapi3.Operation, contentType string) bool {
	code := parsePreferCode(r)
	if code == 0 {
		return false
	}

	codeStr := strconv.Itoa(code)
	schema := s.getResponseSchema(op, codeStr)
	if schema != nil {
		rng := seededRng(s.seed, r.URL.Path+codeStr)
		fake := generateFromSchema(schema, rng, 0)
		writeResponse(w, contentType, code, fake)
	} else {
		w.WriteHeader(code)
	}
	return true
}

// --------------- Query Param Filtering ---------------

func applyQueryParams(items []interface{}, query url.Values) []interface{} {
	// filter
	for key, vals := range query {
		if key == "limit" || key == "offset" {
			continue
		}
		if len(vals) == 0 {
			continue
		}
		filterVal := vals[0]
		filtered := make([]interface{}, 0)
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				if v, exists := m[key]; exists {
					if fmt.Sprintf("%v", v) == filterVal {
						filtered = append(filtered, item)
					}
				}
			}
		}
		items = filtered
	}

	// offset
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset > 0 {
			if offset >= len(items) {
				items = []interface{}{}
			} else {
				items = items[offset:]
			}
		}
	}

	// limit
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit >= 0 {
			if limit < len(items) {
				items = items[:limit]
			}
		}
	}

	return items
}

// --------------- Main Request Handler ---------------

func (s *MockServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Prefer, Accept")
	if r.Method == "OPTIONS" {
		w.WriteHeader(204)
		return
	}

	// simulated delay
	if delay > 0 {
		time.Sleep(delay)
	}

	// chaos mode
	if chaos {
		chaosRng := rand.New(rand.NewSource(time.Now().UnixNano()))
		if chaosRng.Float64() < 0.1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]string{"error": "chaos mode struck 💥"})
			logChaos(r.Method, r.URL.Path, time.Since(start))
			return
		}
		if chaosRng.Float64() < 0.2 {
			spike := time.Duration(chaosRng.Intn(2000)) * time.Millisecond
			time.Sleep(spike)
		}
	}

	// content negotiation
	contentType := negotiateContentType(r)
	if contentType == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(406)
		json.NewEncoder(w).Encode(map[string]string{"error": "not acceptable — supported: application/json, application/xml"})
		logRequest(r.Method, r.URL.Path, 406, time.Since(start))
		return
	}

	_, op, params := s.findRoute(r.URL.Path, r.Method)
	if op == nil {
		writeResponse(w, contentType, 404, map[string]string{"error": "route not found"})
		logRequest(r.Method, r.URL.Path, 404, time.Since(start))
		return
	}

	// auth check
	if !s.checkAuth(w, r, op) {
		logRequest(r.Method, r.URL.Path, 401, time.Since(start))
		return
	}

	// read body for validation
	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// request validation
	if len(bodyBytes) > 0 {
		if !s.validateRequest(w, r, bodyBytes) {
			logRequestValidationError(r.Method, r.URL.Path, time.Since(start))
			return
		}
	}

	// restore body for handlers
	if len(bodyBytes) > 0 {
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// Prefer header
	if s.handlePreferCode(w, r, op, contentType) {
		logRequest(r.Method, r.URL.Path, parsePreferCode(r), time.Since(start))
		return
	}

	resource := extractResource(r.URL.Path)

	switch strings.ToUpper(r.Method) {
	case "POST":
		s.handlePost(w, r, op, resource, contentType)
	case "GET":
		if id, ok := params["id"]; ok {
			s.handleGetOne(w, r, op, resource, id, contentType)
		} else {
			s.handleGetList(w, r, op, resource, contentType)
		}
	case "PUT", "PATCH":
		if id, ok := params["id"]; ok {
			s.handlePut(w, r, op, resource, id, contentType)
		} else {
			writeResponse(w, contentType, 400, map[string]string{"error": "missing id"})
		}
	case "DELETE":
		if id, ok := params["id"]; ok {
			s.handleDelete(w, r, resource, id)
		} else {
			writeResponse(w, contentType, 400, map[string]string{"error": "missing id"})
		}
	default:
		s.handleGeneric(w, r, op, contentType)
	}

	logRequest(r.Method, r.URL.Path, 200, time.Since(start))
}

func extractResource(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "root"
	}
	return parts[0]
}

func (s *MockServer) handlePost(w http.ResponseWriter, r *http.Request, op *openapi3.Operation, resource, contentType string) {
	var body map[string]interface{}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&body)
	}
	if body == nil {
		body = make(map[string]interface{})
	}

	if _, ok := body["id"]; !ok {
		body["id"] = gofakeit.UUID()
	}

	id := fmt.Sprintf("%v", body["id"])
	s.store.Put(resource, id, body)

	writeResponse(w, contentType, 201, body)
}

func (s *MockServer) handleGetOne(w http.ResponseWriter, r *http.Request, op *openapi3.Operation, resource, id, contentType string) {
	obj, ok := s.store.Get(resource, id)
	if ok {
		writeResponse(w, contentType, 200, obj)
		return
	}

	schema := s.getResponseSchema(op, "200")
	if schema == nil {
		schema = s.getResponseSchema(op, "201")
	}
	if schema != nil {
		rng := seededRng(s.seed, r.URL.Path)
		fake := generateFromSchema(schema, rng, 0)
		if m, ok := fake.(map[string]interface{}); ok {
			m["id"] = id
		}
		writeResponse(w, contentType, 200, fake)
		return
	}

	writeResponse(w, contentType, 200, map[string]interface{}{"id": id})
}

func (s *MockServer) handleGetList(w http.ResponseWriter, r *http.Request, op *openapi3.Operation, resource, contentType string) {
	items := s.store.List(resource)
	if s.store.HasBeenWritten(resource) {
		items = applyQueryParams(items, r.URL.Query())
		writeResponse(w, contentType, 200, items)
		return
	}

	if len(items) == 0 {
		schema := s.getResponseSchema(op, "200")
		if schema != nil {
			rng := seededRng(s.seed, r.URL.Path)
			fake := generateFromSchema(schema, rng, 0)
			// if fake data is an array, apply query params
			if arr, ok := fake.([]interface{}); ok {
				arr = applyQueryParams(arr, r.URL.Query())
				writeResponse(w, contentType, 200, arr)
				return
			}
			writeResponse(w, contentType, 200, fake)
			return
		}
	}

	items = applyQueryParams(items, r.URL.Query())
	writeResponse(w, contentType, 200, items)
}

func (s *MockServer) handlePut(w http.ResponseWriter, r *http.Request, op *openapi3.Operation, resource, id, contentType string) {
	var body map[string]interface{}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&body)
	}
	if body == nil {
		body = make(map[string]interface{})
	}
	body["id"] = id

	existing, ok := s.store.Get(resource, id)
	if ok {
		if existingMap, ok := existing.(map[string]interface{}); ok {
			for k, v := range body {
				existingMap[k] = v
			}
			body = existingMap
		}
	}

	s.store.Put(resource, id, body)
	writeResponse(w, contentType, 200, body)
}

func (s *MockServer) handleDelete(w http.ResponseWriter, r *http.Request, resource, id string) {
	s.store.Delete(resource, id)
	w.WriteHeader(204)
}

func (s *MockServer) handleGeneric(w http.ResponseWriter, r *http.Request, op *openapi3.Operation, contentType string) {
	schema := s.getResponseSchema(op, "200")
	if schema != nil {
		rng := seededRng(s.seed, r.URL.Path)
		fake := generateFromSchema(schema, rng, 0)
		writeResponse(w, contentType, 200, fake)
		return
	}
	writeResponse(w, contentType, 200, map[string]string{"status": "ok"})
}

func (s *MockServer) getResponseSchema(op *openapi3.Operation, statusCode string) *openapi3.SchemaRef {
	if op.Responses == nil {
		return nil
	}
	resp := op.Responses.Value(statusCode)
	if resp == nil {
		return nil
	}
	if resp.Value == nil {
		return nil
	}
	ct := resp.Value.Content.Get("application/json")
	if ct == nil {
		return nil
	}
	return ct.Schema
}

// --------------- Proxy Mode ---------------

func runProxy(cmd *cobra.Command, args []string) error {
	specFile := args[0]
	target, _ := cmd.Flags().GetString("target")
	recordFile, _ := cmd.Flags().GetString("record")

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(specFile)
	if err != nil {
		return fmt.Errorf("failed to load spec: %w", err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		stdlog.Printf("warning: spec validation issues: %v", err)
	}

	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		stdlog.Printf("warning: could not build validation router: %v", err)
	}

	targetURL, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	var recorder *Recorder
	if recordFile != "" {
		recorder = NewRecorder(recordFile)
		defer recorder.Close()
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = targetURL.Host
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		// validate response
		if router != nil {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			route, pathParams, err := router.FindRoute(resp.Request)
			if err == nil {
				input := &openapi3filter.RequestValidationInput{
					Request:    resp.Request,
					PathParams: pathParams,
					Route:      route,
					Options:    &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
				}
				respInput := &openapi3filter.ResponseValidationInput{
					RequestValidationInput: input,
					Status:                 resp.StatusCode,
					Header:                 resp.Header,
					Body:                   io.NopCloser(bytes.NewReader(bodyBytes)),
				}
				respInput.SetBodyBytes(bodyBytes)
				if err := openapi3filter.ValidateResponse(context.Background(), respInput); err != nil {
					logProxyValidation("RESPONSE", resp.Request.Method, resp.Request.URL.Path, err)
				}
			}

			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			// record
			if recorder != nil {
				recorder.Record(resp.Request.Method, resp.Request.URL.Path, resp.StatusCode, bodyBytes)
			}
		}
		return nil
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// validate request
		if router != nil {
			bodyBytes, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			route, pathParams, err := router.FindRoute(r)
			if err == nil {
				input := &openapi3filter.RequestValidationInput{
					Request:    r,
					PathParams: pathParams,
					Route:      route,
					Options:    &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
				}
				r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				if err := openapi3filter.ValidateRequest(context.Background(), input); err != nil {
					logProxyValidation("REQUEST", r.Method, r.URL.Path, err)
				}
			}
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		proxy.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{Addr: addr, Handler: handler}

	fmt.Println(renderProxyBanner(specFile, target, port, recordFile))
	fmt.Print(renderReady(port))

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Print(renderShutdown())
		srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed { return err }; return nil
}

// --------------- Recorder ---------------

type Recording struct {
	Method string          `json:"method"`
	Path   string          `json:"path"`
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

type Recorder struct {
	mu         sync.Mutex
	file       *os.File
	recordings []Recording
}

func NewRecorder(path string) *Recorder {
	f, err := os.Create(path)
	if err != nil {
		stdlog.Printf("warning: could not create recording file: %v", err)
		return &Recorder{}
	}
	return &Recorder{file: f}
}

func (r *Recorder) Record(method, path string, status int, body []byte) {
	if r.file == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	rec := Recording{
		Method: method,
		Path:   path,
		Status: status,
		Body:   json.RawMessage(body),
	}
	r.recordings = append(r.recordings, rec)
}

func (r *Recorder) Close() {
	if r.file == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	json.NewEncoder(r.file).Encode(r.recordings)
	r.file.Close()
}

// --------------- Replay Mode ---------------

func runReplay(cmd *cobra.Command, args []string) error {
	recordFile := args[0]
	data, err := os.ReadFile(recordFile)
	if err != nil {
		return fmt.Errorf("failed to read recordings: %w", err)
	}

	var recordings []Recording
	if err := json.Unmarshal(data, &recordings); err != nil {
		return fmt.Errorf("failed to parse recordings: %w", err)
	}

	// build lookup: method+path -> recording
	lookup := make(map[string]Recording)
	for _, rec := range recordings {
		key := rec.Method + " " + rec.Path
		lookup[key] = rec
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		rec, ok := lookup[key]
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(map[string]string{"error": "no recording for " + key})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rec.Status)
		w.Write(rec.Body)
	})

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{Addr: addr, Handler: handler}

	fmt.Println(renderReplayBanner(recordFile, port, len(recordings)))
	fmt.Print(renderReady(port))

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Print(renderShutdown())
		srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed { return err }; return nil
}

// --------------- Fake Data Generation ---------------

func seededRng(baseSeed int64, path string) *rand.Rand {
	h := baseSeed
	for _, c := range path {
		h = h*31 + int64(c)
	}
	return rand.New(rand.NewSource(h))
}

func generateFromSchema(ref *openapi3.SchemaRef, rng *rand.Rand, depth int) interface{} {
	if ref == nil {
		return nil
	}
	schema := ref.Value
	if schema == nil {
		return nil
	}

	if depth > 5 {
		return nil
	}

	if len(schema.AllOf) > 0 {
		result := make(map[string]interface{})
		for _, sub := range schema.AllOf {
			v := generateFromSchema(sub, rng, depth+1)
			if m, ok := v.(map[string]interface{}); ok {
				for k, val := range m {
					result[k] = val
				}
			}
		}
		return result
	}

	if len(schema.OneOf) > 0 {
		return generateFromSchema(schema.OneOf[0], rng, depth+1)
	}
	if len(schema.AnyOf) > 0 {
		return generateFromSchema(schema.AnyOf[0], rng, depth+1)
	}

	if schema.Example != nil {
		return schema.Example
	}

	types := schema.Type.Slice()
	if len(types) == 0 {
		// no type specified, try to infer from properties
		if len(schema.Properties) > 0 {
			return generateObject(schema, rng, depth)
		}
		return "unknown"
	}

	switch types[0] {
	case "object":
		return generateObject(schema, rng, depth)
	case "array":
		return generateArray(schema, rng, depth)
	case "string":
		return generateString(schema, rng)
	case "integer":
		return generateInteger(schema, rng)
	case "number":
		return generateNumber(schema, rng)
	case "boolean":
		return rng.Intn(2) == 1
	default:
		return "unknown"
	}
}

func generateObject(schema *openapi3.Schema, rng *rand.Rand, depth int) interface{} {
	result := make(map[string]interface{})
	for name, prop := range schema.Properties {
		result[name] = generateFromSchemaWithName(prop, rng, depth+1, name)
	}
	return result
}

func generateFromSchemaWithName(ref *openapi3.SchemaRef, rng *rand.Rand, depth int, propName string) interface{} {
	if ref == nil {
		return nil
	}
	schema := ref.Value
	if schema == nil || depth > 5 {
		return nil
	}

	types := schema.Type.Slice()
	if len(types) > 0 && types[0] == "string" && schema.Format == "" && len(schema.Enum) == 0 {
		if v, ok := generateStringByName(propName, rng); ok {
			return v
		}
	}

	if schema.Example != nil {
		return schema.Example
	}

	return generateFromSchema(ref, rng, depth)
}

func generateArray(schema *openapi3.Schema, rng *rand.Rand, depth int) interface{} {
	count := 2 + rng.Intn(4)
	items := make([]interface{}, count)
	for i := range items {
		items[i] = generateFromSchema(schema.Items, rng, depth+1)
	}
	return items
}

func generateStringByName(propName string, rng *rand.Rand) (string, bool) {
	faker := gofakeit.New(uint64(rng.Int63()))
	name := strings.ToLower(propName)

	switch {
	case name == "name" || name == "full_name" || name == "fullname":
		return faker.Name(), true
	case name == "first_name" || name == "firstname" || name == "given_name":
		return faker.FirstName(), true
	case name == "last_name" || name == "lastname" || name == "surname" || name == "family_name":
		return faker.LastName(), true
	case name == "username" || name == "user_name" || name == "handle" || name == "login":
		return faker.Username(), true
	case name == "email" || name == "email_address" || strings.HasSuffix(name, "_email"):
		return faker.Email(), true
	case name == "phone" || name == "phone_number" || name == "mobile" || name == "tel":
		return faker.Phone(), true
	case name == "address" || name == "street" || name == "street_address":
		return faker.Street(), true
	case name == "city":
		return faker.City(), true
	case name == "state" || name == "province" || name == "region":
		return faker.State(), true
	case name == "country":
		return faker.Country(), true
	case name == "zip" || name == "zip_code" || name == "postal_code" || name == "zipcode":
		return faker.Zip(), true
	case name == "latitude" || name == "lat":
		return fmt.Sprintf("%.6f", faker.Latitude()), true
	case name == "longitude" || name == "lng" || name == "lon":
		return fmt.Sprintf("%.6f", faker.Longitude()), true
	case name == "title" || name == "subject" || name == "headline":
		return faker.Sentence(3 + rng.Intn(4)), true
	case name == "description" || name == "summary" || name == "bio" || name == "about":
		return faker.Sentence(8 + rng.Intn(8)), true
	case name == "body" || name == "content" || name == "text" || name == "message":
		return faker.Paragraph(1, 3, 5, " "), true
	case name == "comment" || name == "note" || name == "notes":
		return faker.Sentence(5 + rng.Intn(6)), true
	case name == "url" || name == "website" || name == "link" || name == "homepage":
		return faker.URL(), true
	case name == "image" || name == "avatar" || name == "photo" || name == "picture" || name == "image_url" || name == "avatar_url":
		return fmt.Sprintf("https://picsum.photos/seed/%d/640/480", rng.Intn(10000)), true
	case name == "domain" || name == "hostname":
		return faker.DomainName(), true
	case name == "ip" || name == "ip_address":
		return faker.IPv4Address(), true
	case name == "slug":
		return strings.ToLower(strings.ReplaceAll(faker.BuzzWord()+" "+faker.BuzzWord(), " ", "-")), true
	case name == "sku" || name == "code" || name == "product_code":
		return faker.LetterN(3) + "-" + fmt.Sprintf("%04d", rng.Intn(10000)), true
	case name == "color" || name == "colour":
		return faker.Color(), true
	case name == "company" || name == "company_name" || name == "organization" || name == "org":
		return faker.Company(), true
	case name == "job" || name == "job_title" || name == "role" || name == "position":
		return faker.JobTitle(), true
	case name == "industry" || name == "sector":
		return faker.JobDescriptor(), true
	case name == "currency" || name == "currency_code":
		return faker.CurrencyShort(), true
	case name == "language" || name == "lang" || name == "locale":
		return faker.Language(), true
	case name == "status":
		statuses := []string{"active", "inactive", "pending", "completed", "archived"}
		return statuses[rng.Intn(len(statuses))], true
	case name == "type" || name == "kind" || name == "category":
		return faker.Word(), true
	case name == "tag" || name == "label":
		return faker.Word(), true
	}

	return "", false
}

func generateString(schema *openapi3.Schema, rng *rand.Rand) string {
	faker := gofakeit.New(uint64(rng.Int63()))

	if len(schema.Enum) > 0 {
		return fmt.Sprintf("%v", schema.Enum[rng.Intn(len(schema.Enum))])
	}

	switch schema.Format {
	case "email":
		return faker.Email()
	case "date-time":
		return faker.Date().Format(time.RFC3339)
	case "date":
		return faker.Date().Format("2006-01-02")
	case "uri", "url":
		return faker.URL()
	case "uuid":
		return faker.UUID()
	case "ipv4":
		return faker.IPv4Address()
	case "ipv6":
		return faker.IPv6Address()
	case "hostname":
		return faker.DomainName()
	case "password":
		return faker.Password(true, true, true, false, false, 12)
	}

	maxLen := 100
	if schema.MaxLength != nil {
		maxLen = int(*schema.MaxLength)
	}

	s := faker.Sentence(5 + rng.Intn(8))
	s = strings.TrimSuffix(s, ".")
	if len(s) > maxLen {
		s = s[:maxLen]
		if idx := strings.LastIndex(s, " "); idx > 0 {
			s = s[:idx]
		}
	}
	return s
}

func generateInteger(schema *openapi3.Schema, rng *rand.Rand) int64 {
	min := int64(1)
	max := int64(1000)
	if schema.Min != nil {
		min = int64(*schema.Min)
	}
	if schema.Max != nil {
		max = int64(*schema.Max)
	}
	if max <= min {
		max = min + 100
	}
	return min + rng.Int63n(max-min)
}

func generateNumber(schema *openapi3.Schema, rng *rand.Rand) float64 {
	min := 0.0
	max := 1000.0
	if schema.Min != nil {
		min = *schema.Min
	}
	if schema.Max != nil {
		max = *schema.Max
	}
	return min + rng.Float64()*(max-min)
}
