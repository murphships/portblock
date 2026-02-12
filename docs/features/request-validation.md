# Request Validation

portblock validates every incoming request against your OpenAPI spec. send a bad request and you'll get a helpful 400 — just like a real API would.

## what gets validated

- **request body** — required fields, types, formats, enum values
- **path parameters** — types, formats
- **query parameters** — types, required params
- **content type** — must match what your spec expects
- **request body presence** — if your spec says body is required, it's required

## error format

when validation fails, you get a structured error response:

```bash
curl -X POST localhost:4000/users \
  -H "Content-Type: application/json" \
  -d '{"wrong_field": true}'
```

```json
{
  "error": "validation failed",
  "details": [
    {
      "path": "body.name",
      "message": "property 'name' is required"
    },
    {
      "path": "body.email",
      "message": "property 'email' is required"
    }
  ]
}
```

clear, specific, actionable. no guessing what went wrong.

## why this is useful

- **catch bugs early** — your frontend sends the wrong shape? you'll know immediately
- **spec compliance** — ensure your client code matches the contract
- **realistic error handling** — test your app's error handling with real validation errors
- **no surprises** — if it works against portblock, it'll work against the real API

## type validation

portblock checks that values match their declared types:

```bash
# spec says age is an integer
curl -X POST localhost:4000/users \
  -H "Content-Type: application/json" \
  -d '{"name":"murph","age":"not a number"}'

# → 400 {"error": "validation failed", "details": [{"path": "body.age", "message": "expected integer, got string"}]}
```

## format validation

OpenAPI formats like `email`, `date-time`, `uri`, and `uuid` are checked too:

```bash
curl -X POST localhost:4000/users \
  -H "Content-Type: application/json" \
  -d '{"name":"murph","email":"not-an-email"}'

# → 400 with format validation error
```

no more shipping broken payloads to production.
