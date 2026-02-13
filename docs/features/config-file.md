# Config File

Avoid repeating CLI flags by creating a `.portblock.yaml` config file.

## Quick Setup

```bash
portblock init
```

This creates `.portblock.yaml` in your current directory.

## Config Format

```yaml
# .portblock.yaml
port: 8080
seed: 42
delay: 200ms
chaos: false
no-auth: false
watch: true
strict: false
webhook-target: http://localhost:9000/hooks
webhook-delay: 500ms
```

Also supports `.portblock.yml` and `.portblock.json`.

## Lookup Order

1. Current working directory
2. Home directory (`~/.portblock.yaml`)

CLI flags always override config file values.

## JSON Format

```json
{
  "port": 8080,
  "seed": 42,
  "delay": "200ms",
  "chaos": false,
  "strict": true
}
```
