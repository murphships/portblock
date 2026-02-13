# Generate (Reverse-Engineer a Spec)

`portblock generate` probes a running API and generates an OpenAPI spec from the responses.

## Usage

```bash
# auto-probe common paths
portblock generate --target https://jsonplaceholder.typicode.com

# specify paths
portblock generate --target https://api.example.com --paths /users,/posts,/comments

# write to file
portblock generate --target https://api.example.com --output spec.yaml
```

## What It Does

1. Sends GET requests to each path
2. Inspects response JSON to infer schemas
3. Detects formats (email, UUID, date-time, URI, IPv4)
4. Identifies required fields (present in all array items)
5. Generates single-item endpoints for array responses (`/users/{id}`)

## Default Probe Paths

When `--paths` is not specified, portblock probes these common API paths:

`/users`, `/posts`, `/comments`, `/todos`, `/products`, `/orders`, `/items`, `/articles`, `/categories`, `/tags`, `/events`, `/messages`, `/notifications`, `/settings`, `/health`, `/status`, `/api/v1`, `/api`

## Options

| Flag | Description |
|------|-------------|
| `--target` | Base URL of the API (required) |
| `--paths` | Comma-separated paths to probe |
| `--output` | Output file (default: stdout) |
