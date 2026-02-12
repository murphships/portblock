# Auth Simulation

if your OpenAPI spec defines security schemes, portblock enforces them automatically. no extra config.

## how it works

portblock reads the `securityDefinitions` / `components/securitySchemes` from your spec and enforces them on every request that requires auth.

```yaml
# in your spec
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
```

```bash
# no token → 401
curl localhost:4000/users
# → {"error": "unauthorized"}

# with token → 200 (any token value works)
curl -H "Authorization: Bearer anything-goes-here" localhost:4000/users
# → your data
```

portblock checks that the auth header is *present* and *formatted correctly*. it doesn't validate the actual token value — it's a mock, not a security audit.

## supported schemes

- **Bearer token** (`type: http, scheme: bearer`)
- **API key** (header, query, or cookie)
- **Basic auth** (`type: http, scheme: basic`)
- **OAuth2** (checks for Bearer token presence)

## skipping auth

sometimes you just want data and don't want to deal with auth headers:

```bash
portblock serve api.yaml --no-auth
```

all security checks are disabled. every endpoint is open. great for quick prototyping.

## per-endpoint security

if your spec applies security to specific endpoints only, portblock respects that. public endpoints stay public, protected endpoints require auth.

```yaml
paths:
  /public/health:
    get:
      security: []  # no auth needed
      # ...
  /users:
    get:
      security:
        - BearerAuth: []  # auth required
      # ...
```

## testing auth flows

this is great for testing your app's auth error handling:

```bash
# test what happens with no token
curl localhost:4000/users
# → 401

# test with an expired/bad format
curl -H "Authorization: NotBearer oops" localhost:4000/users
# → 401

# test with correct format
curl -H "Authorization: Bearer mytoken" localhost:4000/users
# → 200
```

your frontend's auth error handling gets tested against realistic responses without touching a real auth service.
