package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
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
	"github.com/spf13/cobra"
)

var (
	port    int
	seed    int64
	delay   time.Duration
	chaos   bool
	version = "0.1.0"
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

	rootCmd.AddCommand(serveCmd)

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
		log.Printf("warning: spec validation issues: %v", err)
	}

	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	server := &MockServer{
		doc:   doc,
		store: NewStore(),
		seed:  seed,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleRequest)

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{Addr: addr, Handler: mux}

	fmt.Printf("\n  ⬛ portblock v%s\n", version)
	fmt.Printf("  spec:  %s\n", specFile)
	fmt.Printf("  port:  %d\n", port)
	fmt.Printf("  seed:  %d\n", seed)
	if delay > 0 {
		fmt.Printf("  delay: %s\n", delay)
	}
	if chaos {
		fmt.Printf("  chaos: enabled 💥\n")
	}
	fmt.Printf("\n  ready at http://localhost:%d\n\n", port)

	// print registered routes
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
		fmt.Printf("  %s %s\n", strings.Join(methods, ","), path)
	}
	fmt.Println()

	// graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("\nshutting down...")
		srv.Shutdown(context.Background())
	}()

	return srv.ListenAndServe()
}

// Store is the in-memory CRUD store
type Store struct {
	mu      sync.RWMutex
	data    map[string]map[string]interface{} // resource type -> id -> object
	written map[string]bool                   // tracks if a resource has ever been written to
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

// MockServer handles all incoming requests
type MockServer struct {
	doc   *openapi3.T
	store *Store
	seed  int64
}

// path param regex: {param}
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
	// convert /users/{id} to regex
	regexStr := "^" + pathParamRe.ReplaceAllString(pattern, `([^/]+)`) + "$"
	re, err := regexp.Compile(regexStr)
	if err != nil {
		return nil
	}
	matches := re.FindStringSubmatch(actual)
	if matches == nil {
		return nil
	}

	// extract param names
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

func (s *MockServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
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
			log.Printf("💥 CHAOS %s %s → 500 (%s)", r.Method, r.URL.Path, time.Since(start))
			return
		}
		if chaosRng.Float64() < 0.2 {
			spike := time.Duration(chaosRng.Intn(2000)) * time.Millisecond
			time.Sleep(spike)
		}
	}

	_, op, params := s.findRoute(r.URL.Path, r.Method)
	if op == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]string{"error": "route not found"})
		log.Printf("❌ %s %s → 404 (%s)", r.Method, r.URL.Path, time.Since(start))
		return
	}

	// determine resource name from path
	resource := extractResource(r.URL.Path)

	// handle stateful CRUD
	switch strings.ToUpper(r.Method) {
	case "POST":
		s.handlePost(w, r, op, resource)
	case "GET":
		if id, ok := params["id"]; ok {
			s.handleGetOne(w, r, op, resource, id)
		} else {
			s.handleGetList(w, r, op, resource)
		}
	case "PUT", "PATCH":
		if id, ok := params["id"]; ok {
			s.handlePut(w, r, op, resource, id)
		} else {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": "missing id"})
		}
	case "DELETE":
		if id, ok := params["id"]; ok {
			s.handleDelete(w, r, resource, id)
		} else {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": "missing id"})
		}
	default:
		s.handleGeneric(w, r, op)
	}

	log.Printf("✅ %s %s (%s)", r.Method, r.URL.Path, time.Since(start))
}

func extractResource(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "root"
	}
	// use first path segment as resource name
	return parts[0]
}

func (s *MockServer) handlePost(w http.ResponseWriter, r *http.Request, op *openapi3.Operation, resource string) {
	w.Header().Set("Content-Type", "application/json")

	var body map[string]interface{}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&body)
	}
	if body == nil {
		body = make(map[string]interface{})
	}

	// generate an ID if not provided
	if _, ok := body["id"]; !ok {
		body["id"] = gofakeit.UUID()
	}

	id := fmt.Sprintf("%v", body["id"])
	s.store.Put(resource, id, body)

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(body)
}

func (s *MockServer) handleGetOne(w http.ResponseWriter, r *http.Request, op *openapi3.Operation, resource, id string) {
	w.Header().Set("Content-Type", "application/json")

	obj, ok := s.store.Get(resource, id)
	if ok {
		json.NewEncoder(w).Encode(obj)
		return
	}

	// generate fake response from schema
	schema := s.getResponseSchema(op, "200")
	if schema == nil {
		schema = s.getResponseSchema(op, "201")
	}
	if schema != nil {
		// seed based on path + id for consistency
		rng := seededRng(s.seed, r.URL.Path)
		fake := generateFromSchema(schema, rng, 0)
		// set the id field to match
		if m, ok := fake.(map[string]interface{}); ok {
			m["id"] = id
		}
		json.NewEncoder(w).Encode(fake)
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id})
}

func (s *MockServer) handleGetList(w http.ResponseWriter, r *http.Request, op *openapi3.Operation, resource string) {
	w.Header().Set("Content-Type", "application/json")

	// if any POST has been made to this resource, only return stored items (even if empty)
	items := s.store.List(resource)
	if s.store.HasBeenWritten(resource) {
		json.NewEncoder(w).Encode(items)
		return
	}

	// no writes yet — generate fake seed data
	if len(items) == 0 {
		schema := s.getResponseSchema(op, "200")
		if schema != nil {
			rng := seededRng(s.seed, r.URL.Path)
			fake := generateFromSchema(schema, rng, 0)
			json.NewEncoder(w).Encode(fake)
			return
		}
	}

	json.NewEncoder(w).Encode(items)
}

func (s *MockServer) handlePut(w http.ResponseWriter, r *http.Request, op *openapi3.Operation, resource, id string) {
	w.Header().Set("Content-Type", "application/json")

	var body map[string]interface{}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&body)
	}
	if body == nil {
		body = make(map[string]interface{})
	}
	body["id"] = id

	// merge with existing if PATCH-like
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
	json.NewEncoder(w).Encode(body)
}

func (s *MockServer) handleDelete(w http.ResponseWriter, r *http.Request, resource, id string) {
	w.Header().Set("Content-Type", "application/json")
	s.store.Delete(resource, id)
	w.WriteHeader(204)
}

func (s *MockServer) handleGeneric(w http.ResponseWriter, r *http.Request, op *openapi3.Operation) {
	w.Header().Set("Content-Type", "application/json")
	schema := s.getResponseSchema(op, "200")
	if schema != nil {
		rng := seededRng(s.seed, r.URL.Path)
		fake := generateFromSchema(schema, rng, 0)
		json.NewEncoder(w).Encode(fake)
		return
	}
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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

	// handle allOf
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

	// handle oneOf/anyOf — pick first
	if len(schema.OneOf) > 0 {
		return generateFromSchema(schema.OneOf[0], rng, depth+1)
	}
	if len(schema.AnyOf) > 0 {
		return generateFromSchema(schema.AnyOf[0], rng, depth+1)
	}

	// use example if available
	if schema.Example != nil {
		return schema.Example
	}

	switch schema.Type.Slice()[0] {
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

	// for strings, use property-name-aware generation
	if len(schema.Type.Slice()) > 0 && schema.Type.Slice()[0] == "string" && schema.Format == "" && len(schema.Enum) == 0 {
		if v, ok := generateStringByName(propName, rng); ok {
			return v
		}
	}

	// also check examples on the schema
	if schema.Example != nil {
		return schema.Example
	}

	return generateFromSchema(ref, rng, depth)
}

func generateArray(schema *openapi3.Schema, rng *rand.Rand, depth int) interface{} {
	count := 2 + rng.Intn(4) // 2-5 items
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
	// names
	case name == "name" || name == "full_name" || name == "fullname":
		return faker.Name(), true
	case name == "first_name" || name == "firstname" || name == "given_name":
		return faker.FirstName(), true
	case name == "last_name" || name == "lastname" || name == "surname" || name == "family_name":
		return faker.LastName(), true
	case name == "username" || name == "user_name" || name == "handle" || name == "login":
		return faker.Username(), true

	// contact
	case name == "email" || name == "email_address" || strings.HasSuffix(name, "_email"):
		return faker.Email(), true
	case name == "phone" || name == "phone_number" || name == "mobile" || name == "tel":
		return faker.Phone(), true

	// location
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

	// text content
	case name == "title" || name == "subject" || name == "headline":
		return faker.Sentence(3 + rng.Intn(4)), true
	case name == "description" || name == "summary" || name == "bio" || name == "about":
		return faker.Sentence(8 + rng.Intn(8)), true
	case name == "body" || name == "content" || name == "text" || name == "message":
		return faker.Paragraph(1, 3, 5, " "), true
	case name == "comment" || name == "note" || name == "notes":
		return faker.Sentence(5 + rng.Intn(6)), true

	// web
	case name == "url" || name == "website" || name == "link" || name == "homepage":
		return faker.URL(), true
	case name == "image" || name == "avatar" || name == "photo" || name == "picture" || name == "image_url" || name == "avatar_url":
		return fmt.Sprintf("https://picsum.photos/seed/%d/640/480", rng.Intn(10000)), true
	case name == "domain" || name == "hostname":
		return faker.DomainName(), true
	case name == "ip" || name == "ip_address":
		return faker.IPv4Address(), true

	// identifiers
	case name == "slug":
		return strings.ToLower(strings.ReplaceAll(faker.BuzzWord()+" "+faker.BuzzWord(), " ", "-")), true
	case name == "sku" || name == "code" || name == "product_code":
		return faker.LetterN(3) + "-" + fmt.Sprintf("%04d", rng.Intn(10000)), true
	case name == "color" || name == "colour":
		return faker.Color(), true

	// business
	case name == "company" || name == "company_name" || name == "organization" || name == "org":
		return faker.Company(), true
	case name == "job" || name == "job_title" || name == "role" || name == "position":
		return faker.JobTitle(), true
	case name == "industry" || name == "sector":
		return faker.JobDescriptor(), true

	// misc
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

	// check enum
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

	// generate a realistic string based on length constraints
	maxLen := 100
	if schema.MaxLength != nil {
		maxLen = int(*schema.MaxLength)
	}

	// generate a full sentence
	s := faker.Sentence(5 + rng.Intn(8))
	// trim trailing period for cleaner look
	s = strings.TrimSuffix(s, ".")
	if len(s) > maxLen {
		// truncate at last word boundary
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

// used by handleGetOne to try to parse path param as resource identifier
func init() {
	// suppress unused import
	_ = strconv.Itoa
}
