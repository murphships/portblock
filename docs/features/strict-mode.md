# Strict Mode

Strict mode validates everything — specs, requests, and generated responses.

## Usage

```bash
portblock serve api.yaml --strict
portblock proxy api.yaml --target https://api.example.com --strict
```

Or in `.portblock.yaml`:

```yaml
strict: true
```

## What It Validates

### Spec Validation
Rejects specs with validation warnings that would normally be logged as warnings.

### Request Validation
- All required fields must be present, non-null, and non-empty
- Recursive validation for nested objects
- Standard OpenAPI request validation (types, formats, patterns)

### Response Validation
Validates generated mock responses against schema constraints:

- `minLength` / `maxLength` on strings
- `pattern` (regex) on strings
- `minimum` / `maximum` on numbers
- `exclusiveMinimum` / `exclusiveMaximum`
- `multipleOf`
- `minItems` / `maxItems` on arrays
- `enum` values
- `required` fields

Violations are logged as warnings in the TUI output.
