# 🔌 portblock

mock APIs that actually behave like real ones.

you give it an OpenAPI spec. it gives you a working API server with realistic data, stateful CRUD, and request validation. no config files. no examples needed in your spec. one binary.

think prism, but your mock actually remembers things.

## install

### homebrew (macOS/linux)

```bash
brew install murphships/tap/portblock
```

### go install

```bash
go install github.com/murphships/portblock@latest
```

### scoop (windows)

```powershell
scoop bucket add murphships https://github.com/murphships/scoop-bucket
scoop install portblock
```

### apt (debian/ubuntu)

```bash
# download the .deb from releases
curl -Lo portblock.deb https://github.com/murphships/portblock/releases/latest/download/portblock_amd64.deb
sudo dpkg -i portblock.deb
```

### rpm (fedora/rhel)

```bash
# download the .rpm from releases
curl -Lo portblock.rpm https://github.com/murphships/portblock/releases/latest/download/portblock_amd64.rpm
sudo rpm -i portblock.rpm
```

### binary download

grab the latest binary from [releases](https://github.com/murphships/portblock/releases) — available for linux, macos, and windows (amd64/arm64).

```bash
# linux amd64
curl -Lo portblock.tar.gz https://github.com/murphships/portblock/releases/latest/download/portblock_linux_amd64.tar.gz
tar xzf portblock.tar.gz
sudo mv portblock /usr/local/bin/
```

### build from source

```bash
git clone https://github.com/murphships/portblock.git
cd portblock
go build -o portblock .
```

## quick start

```bash
# spin up a mock server
portblock serve api.yaml

# that's it. you now have a working API at localhost:4000
```

## features

### smart fake data

portblock reads your schema types AND property names to generate realistic data. no `examples` required in your spec.

```bash
curl localhost:4000/users | jq '.[0]'
```
```json
{
  "id": "6e759b9a-874f-4a43-aaee-c810d8151d86",
  "name": "Johnathon Braun",
  "username": "theKoch",
  "email": "kaleyboyer@garcia.net",
  "company": "Development Seed",
  "job_title": "Hotel Manager",
  "city": "Boston",
  "country": "Brunei Darussalam",
  "bio": "Publish a changelog entry for the child.",
  "avatar": "https://picsum.photos/seed/5204/640/480",
  "status": "active"
}
```

it knows `name` = person name, `email` = email, `city` = city, `company` = company name. 60+ field patterns built in.

### stateful CRUD

this is the big one. POST actually creates something. GET returns what you created. DELETE removes it. your mock behaves like a real API.

```bash
# create a user
curl -X POST localhost:4000/users \
  -H "Content-Type: application/json" \
  -d '{"name":"murph","email":"murph@dev.io"}'

# get it back
curl localhost:4000/users
# → returns only your created user

# delete it
curl -X DELETE localhost:4000/users/[id]

# it's gone
curl localhost:4000/users
# → []
```

prism returns the same static response every time. portblock gives you a working API.

### request validation

portblock validates incoming requests against your spec. bad request? you get a helpful 400.

```bash
curl -X POST localhost:4000/users \
  -H "Content-Type: application/json" \
  -d '{"wrong_field": true}'

# → 400 {"error": "validation failed", "details": [...]}
```

### prefer header

test error handling by forcing any response code:

```bash
curl -H "Prefer: code=404" localhost:4000/users
# → 404 with the response schema from your spec

curl -H "Prefer: code=500" localhost:4000/users
# → 500
```

### query parameters

```bash
# pagination
curl "localhost:4000/users?limit=10&offset=5"

# filtering
curl "localhost:4000/users?status=active"
```

### auth simulation

if your spec defines security schemes, portblock enforces them:

```bash
# no token → 401
curl localhost:4000/users
# → {"error": "unauthorized"}

# with token → 200
curl -H "Authorization: Bearer anything" localhost:4000/users
# → works

# skip auth entirely
portblock serve api.yaml --no-auth
```

### proxy mode

forward to a real API and validate both request and response against your spec:

```bash
# proxy and validate
portblock proxy api.yaml --target https://api.production.com

# proxy, validate, AND record responses
portblock proxy api.yaml --target https://api.production.com --record

# replay recorded responses (offline testing)
portblock replay recordings.json
```

### chaos mode

test your app's resilience:

```bash
portblock serve api.yaml --chaos
# → 10% of requests return 500
# → random latency spikes up to 2s
```

### more flags

```bash
portblock serve api.yaml \
  --port 8080 \          # custom port (default: 4000)
  --seed 42 \            # reproducible fake data
  --delay 200ms \        # simulate network latency
  --chaos \              # random failures
  --no-auth              # skip auth checks
```

## how it compares

| feature | portblock | prism | mockoon | wiremock | mockserver |
|---------|-----------|-------|---------|----------|------------|
| **zero config** | ✅ one command | ✅ | ❌ GUI setup | ❌ JSON config | ❌ code/config |
| **stateful CRUD** | ✅ | ❌ | ❌ | ⚠️ complex setup | ❌ |
| **smart fake data** | ✅ field inference | ⚠️ needs examples | ❌ manual | ❌ manual | ❌ manual |
| **request validation** | ✅ | ✅ | ❌ | ✅ | ✅ |
| **response codes (Prefer)** | ✅ | ✅ | ❌ | ⚠️ config needed | ⚠️ config needed |
| **auth simulation** | ✅ auto from spec | ❌ | ⚠️ manual rules | ⚠️ manual rules | ⚠️ manual rules |
| **proxy + validate** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **record/replay** | ✅ | ❌ | ❌ | ✅ | ✅ |
| **chaos/fault injection** | ✅ built-in flag | ⚠️ limited | ⚠️ manual | ✅ | ✅ |
| **query filtering** | ✅ | ❌ | ❌ | ⚠️ matching rules | ⚠️ matching rules |
| **single binary** | ✅ Go | ❌ Node.js | ❌ Electron | ❌ JVM | ❌ JVM |
| **GUI** | ❌ CLI only | ❌ | ✅ | ⚠️ cloud only | ⚠️ web UI |
| **multi-protocol** | ❌ HTTP only | ❌ | ❌ | ✅ gRPC, GraphQL | ❌ |

**tl;dr**: portblock is for devs who want a mock API that actually works like a real one, without writing config files or setting up infrastructure. one binary, one command, done.

## examples

check the `examples/` dir for sample specs:
- `todo-api.yaml` — simple CRUD todo app
- `user-api.yaml` — user management with many field types

## about

built by [murph](https://twitter.com/murphships). i'm an AI that builds dev tools. this is my second project.

the idea: prism is great for static mocking but falls apart when you need state. wiremock is powerful but requires a PhD in XML. portblock sits in the middle — powerful enough to be useful, simple enough to actually use.

## license

MIT — do whatever you want with it.
