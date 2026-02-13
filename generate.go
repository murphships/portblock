package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var defaultProbePaths = []string{
	"/users", "/posts", "/comments", "/todos", "/products",
	"/orders", "/items", "/articles", "/categories", "/tags",
	"/events", "/messages", "/notifications", "/settings",
	"/health", "/status", "/api/v1", "/api",
}

func runGenerate(cmd *cobra.Command, args []string) error {
	target, _ := cmd.Flags().GetString("target")
	pathsStr, _ := cmd.Flags().GetString("paths")
	output, _ := cmd.Flags().GetString("output")

	target = strings.TrimRight(target, "/")

	// determine paths to probe
	var probePaths []string
	if pathsStr != "" {
		for _, p := range strings.Split(pathsStr, ",") {
			p = strings.TrimSpace(p)
			if !strings.HasPrefix(p, "/") {
				p = "/" + p
			}
			probePaths = append(probePaths, p)
		}
	} else {
		probePaths = defaultProbePaths
	}

	client := &http.Client{Timeout: 15 * time.Second}

	fmt.Println(renderGenerateBanner(target, len(probePaths)))

	var results []probeResult

	for _, path := range probePaths {
		reqURL := target + path
		resp, err := client.Get(reqURL)
		if err != nil {
			logGenerateProbe(path, 0, "error: "+err.Error())
			continue
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 400 {
			logGenerateProbe(path, resp.StatusCode, "skipped")
			continue
		}

		var body interface{}
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			logGenerateProbe(path, resp.StatusCode, "non-JSON response")
			continue
		}

		isArray := false
		if _, ok := body.([]interface{}); ok {
			isArray = true
		}

		logGenerateProbe(path, resp.StatusCode, "ok")
		results = append(results, probeResult{
			path:       path,
			statusCode: resp.StatusCode,
			body:       body,
			isArray:    isArray,
		})
	}

	if len(results) == 0 {
		return fmt.Errorf("no successful probes — couldn't generate a spec")
	}

	// build OpenAPI spec
	spec := buildSpec(target, results)

	// marshal to YAML
	yamlBytes, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal spec: %w", err)
	}

	if output == "" || output == "-" {
		fmt.Println(string(yamlBytes))
	} else {
		if err := os.WriteFile(output, yamlBytes, 0644); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		fmt.Printf("\nspec written to %s\n", output)
	}

	return nil
}

type probeResult struct {
	path       string
	statusCode int
	body       interface{}
	isArray    bool
}

func buildSpec(target string, results []probeResult) map[string]interface{} {
	parsedURL, _ := url.Parse(target)
	title := parsedURL.Host
	if title == "" {
		title = target
	}

	paths := map[string]interface{}{}
	for _, r := range results {
		schema := inferSchema(r.body)
		responseSchema := schema

		if r.isArray {
			// for arrays, the schema describes the items
			if items, ok := schema["items"]; ok {
				// also generate a single-item path
				singlePath := r.path + "/{id}"
				paths[singlePath] = map[string]interface{}{
					"get": map[string]interface{}{
						"summary":     "Get single " + pathToResource(r.path),
						"operationId": "get" + capitalize(pathToResource(r.path)),
						"parameters": []interface{}{
							map[string]interface{}{
								"name":     "id",
								"in":       "path",
								"required": true,
								"schema":   map[string]interface{}{"type": "string"},
							},
						},
						"responses": map[string]interface{}{
							fmt.Sprintf("%d", r.statusCode): map[string]interface{}{
								"description": "successful response",
								"content": map[string]interface{}{
									"application/json": map[string]interface{}{
										"schema": items,
									},
								},
							},
						},
					},
				}
			}
		}

		paths[r.path] = map[string]interface{}{
			"get": map[string]interface{}{
				"summary":     "List " + pathToResource(r.path),
				"operationId": "list" + capitalize(pathToResource(r.path)),
				"responses": map[string]interface{}{
					fmt.Sprintf("%d", r.statusCode): map[string]interface{}{
						"description": "successful response",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": responseSchema,
							},
						},
					},
				},
			},
		}
	}

	spec := map[string]interface{}{
		"openapi": "3.1.0",
		"info": map[string]interface{}{
			"title":       title + " API",
			"version":     "1.0.0",
			"description": fmt.Sprintf("auto-generated spec from %s by portblock generate", target),
		},
		"servers": []interface{}{
			map[string]interface{}{"url": target},
		},
		"paths": paths,
	}

	return spec
}

func inferSchema(data interface{}) map[string]interface{} {
	if data == nil {
		return map[string]interface{}{"type": "string"}
	}

	switch v := data.(type) {
	case map[string]interface{}:
		return inferObjectSchema(v)
	case []interface{}:
		itemSchema := map[string]interface{}{"type": "object"}
		if len(v) > 0 {
			// merge schemas from multiple items
			schemas := make([]map[string]interface{}, 0, len(v))
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					schemas = append(schemas, inferObjectSchema(m))
				}
			}
			if len(schemas) > 0 {
				itemSchema = mergeObjectSchemas(schemas)
			}
		}
		return map[string]interface{}{
			"type":  "array",
			"items": itemSchema,
		}
	case string:
		return inferStringSchema(v)
	case float64:
		if v == float64(int64(v)) {
			return map[string]interface{}{"type": "integer"}
		}
		return map[string]interface{}{"type": "number"}
	case bool:
		return map[string]interface{}{"type": "boolean"}
	default:
		return map[string]interface{}{"type": "string"}
	}
}

func inferObjectSchema(obj map[string]interface{}) map[string]interface{} {
	properties := map[string]interface{}{}
	required := []interface{}{}

	// sort keys for deterministic output
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := obj[k]
		properties[k] = inferSchema(v)
		if v != nil {
			required = append(required, k)
		}
	}

	result := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}

func mergeObjectSchemas(schemas []map[string]interface{}) map[string]interface{} {
	if len(schemas) == 0 {
		return map[string]interface{}{"type": "object"}
	}
	if len(schemas) == 1 {
		return schemas[0]
	}

	// merge properties from all schemas
	allProps := map[string]map[string]interface{}{}
	propCount := map[string]int{}

	for _, s := range schemas {
		props, ok := s["properties"].(map[string]interface{})
		if !ok {
			continue
		}
		for name, schema := range props {
			if _, exists := allProps[name]; !exists {
				if sm, ok := schema.(map[string]interface{}); ok {
					allProps[name] = sm
				}
			}
			propCount[name]++
		}
	}

	properties := map[string]interface{}{}
	required := []interface{}{}
	for name, schema := range allProps {
		properties[name] = schema
		// required if present in all samples
		if propCount[name] == len(schemas) {
			required = append(required, name)
		}
	}

	// sort required for determinism
	sort.Slice(required, func(i, j int) bool {
		return required[i].(string) < required[j].(string)
	})

	result := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}

var (
	emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	uuidRe  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	dateRe  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	dtRe    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)
	urlRe   = regexp.MustCompile(`^https?://`)
	ipv4Re  = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
)

func inferStringSchema(val string) map[string]interface{} {
	s := map[string]interface{}{"type": "string"}

	switch {
	case emailRe.MatchString(val):
		s["format"] = "email"
	case uuidRe.MatchString(val):
		s["format"] = "uuid"
	case dtRe.MatchString(val):
		s["format"] = "date-time"
	case dateRe.MatchString(val):
		s["format"] = "date"
	case urlRe.MatchString(val):
		s["format"] = "uri"
	case ipv4Re.MatchString(val):
		s["format"] = "ipv4"
	}

	return s
}

func pathToResource(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "resource"
	}
	return parts[len(parts)-1]
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
