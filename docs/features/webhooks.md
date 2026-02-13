# Webhooks & Callbacks

portblock fires webhooks when mutations happen (POST, PUT, PATCH, DELETE).

## Usage

```bash
portblock serve api.yaml \
  --webhook-target http://localhost:9000/hooks \
  --webhook-delay 500ms
```

## Webhook Payload

```json
{
  "event": "users.created",
  "method": "POST",
  "path": "/users",
  "status": 201,
  "timestamp": "2026-02-13T10:00:00Z",
  "data": { "id": "abc-123", "name": "test" }
}
```

## Event Names

Events are inferred from the HTTP method and resource path:

| Method | Event |
|--------|-------|
| POST | `{resource}.created` |
| PUT/PATCH | `{resource}.updated` |
| DELETE | `{resource}.deleted` |

## Features

- **Retry logic**: 3 attempts with exponential backoff (1s, 2s, 4s)
- **Configurable delay**: simulate async webhook delivery
- **Spec-aware**: picks up callback definitions from your OpenAPI spec
- **Headers**: includes `X-Portblock-Event` and `X-Portblock-Delivery` headers
- **Logged in TUI**: every delivery attempt shows in the terminal
