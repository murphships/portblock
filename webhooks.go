package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

var (
	webhookTarget string
	webhookDelay  time.Duration
)

// WebhookManager handles webhook/callback dispatching
type WebhookManager struct {
	target   string
	delay    time.Duration
	doc      *openapi3.T
	client   *http.Client
}

func NewWebhookManager(target string, delayDur time.Duration, doc *openapi3.T) *WebhookManager {
	if target == "" {
		return nil
	}
	return &WebhookManager{
		target: strings.TrimRight(target, "/"),
		delay:  delayDur,
		doc:    doc,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// FireWebhook sends a webhook after a mutating request (POST/PUT/PATCH/DELETE)
func (wm *WebhookManager) FireWebhook(method, path string, statusCode int, responseBody interface{}) {
	if wm == nil {
		return
	}

	// build webhook payload
	event := inferEventName(method, path)
	payload := map[string]interface{}{
		"event":     event,
		"method":    method,
		"path":      path,
		"status":    statusCode,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	if responseBody != nil {
		payload["data"] = responseBody
	}

	// check callbacks on matching operations in the spec
	if wm.doc != nil && wm.doc.Paths != nil {
		for _, pathItem := range wm.doc.Paths.Map() {
			op := getOperationForMethod(pathItem, method)
			if op != nil && len(op.Callbacks) > 0 {
				for cbName := range op.Callbacks {
					payload["callback"] = cbName
					break
				}
			}
		}
	}

	go func() {
		if wm.delay > 0 {
			time.Sleep(wm.delay)
		}
		wm.deliverWithRetry(payload, event)
	}()
}

func (wm *WebhookManager) deliverWithRetry(payload interface{}, event string) {
	body, err := json.Marshal(payload)
	if err != nil {
		logWebhookDelivery(event, wm.target, 0, err)
		return
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			time.Sleep(backoff)
		}

		req, err := http.NewRequest("POST", wm.target, bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Portblock-Event", event)
		req.Header.Set("X-Portblock-Delivery", fmt.Sprintf("%d", time.Now().UnixNano()))

		resp, err := wm.client.Do(req)
		if err != nil {
			lastErr = err
			logWebhookDelivery(event, wm.target, 0, fmt.Errorf("attempt %d: %w", attempt+1, err))
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			logWebhookDelivery(event, wm.target, resp.StatusCode, nil)
			return
		}
		lastErr = fmt.Errorf("status %d", resp.StatusCode)
		logWebhookDelivery(event, wm.target, resp.StatusCode, fmt.Errorf("attempt %d: %s", attempt+1, lastErr))
	}

	logWebhookDelivery(event, wm.target, 0, fmt.Errorf("all retries exhausted: %w", lastErr))
}

func inferEventName(method, path string) string {
	resource := extractResource(path)
	switch strings.ToUpper(method) {
	case "POST":
		return resource + ".created"
	case "PUT", "PATCH":
		return resource + ".updated"
	case "DELETE":
		return resource + ".deleted"
	default:
		return resource + ".action"
	}
}

func matchesWebhook(webhookName, method, path string) bool {
	name := strings.ToLower(webhookName)
	resource := strings.ToLower(extractResource(path))
	m := strings.ToLower(method)

	if strings.Contains(name, resource) {
		return true
	}
	if m == "post" && (strings.Contains(name, "create") || strings.Contains(name, "new")) {
		return true
	}
	if (m == "put" || m == "patch") && strings.Contains(name, "update") {
		return true
	}
	if m == "delete" && strings.Contains(name, "delete") {
		return true
	}
	return false
}

func getWebhookOperation(pathItem *openapi3.PathItem) *openapi3.Operation {
	if pathItem.Post != nil {
		return pathItem.Post
	}
	if pathItem.Put != nil {
		return pathItem.Put
	}
	if pathItem.Get != nil {
		return pathItem.Get
	}
	return nil
}

func getOperationForMethod(pathItem *openapi3.PathItem, method string) *openapi3.Operation {
	return getOperation(pathItem, method)
}

func isMutatingMethod(method string) bool {
	switch strings.ToUpper(method) {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	}
	return false
}
