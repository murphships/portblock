# Hot Reload

portblock watches your spec file and automatically reloads when it changes. No restart needed.

## How It Works

When you run `portblock serve`, it watches the spec file using filesystem events. Edit your spec, save it, and the mock server updates instantly — routes, schemas, everything.

```bash
portblock serve api.yaml
# edit api.yaml in another terminal...
# ↻ reloaded api.yaml
```

Your stored data (from POST/PUT) survives reloads. Only the routing and schema definitions refresh.

## Disabling Hot Reload

```bash
portblock serve api.yaml --watch=false
```

Or in `.portblock.yaml`:

```yaml
watch: false
```

## How It's Implemented

- Uses `fsnotify` for filesystem events
- 200ms debounce to avoid rapid-fire reloads
- Validates the new spec before swapping (bad specs are rejected with an error log)
- Thread-safe: uses a read-write mutex so in-flight requests complete safely
