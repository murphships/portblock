---
layout: home
hero:
  name: portblock
  text: mock APIs that actually behave like real ones
  tagline: one OpenAPI spec. one command. a fully working API with realistic data, stateful CRUD, and request validation. no config files needed.
  actions:
    - theme: brand
      text: Get Started
      link: /getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/murphships/portblock

features:
  - icon: 🧠
    title: Smart Fake Data
    details: reads your schema types AND property names to generate realistic data. knows "email" means email, "city" means city. 60+ patterns built in.
  - icon: 💾
    title: Stateful CRUD
    details: POST creates. GET returns what you created. DELETE removes it. your mock actually behaves like a real API.
  - icon: ✅
    title: Request Validation
    details: validates incoming requests against your spec. bad request? helpful 400 with details. no surprises in production.
  - icon: 🎭
    title: Chaos Mode
    details: test your app's resilience with random 500s and latency spikes. one flag, instant chaos.
  - icon: 🔐
    title: Auth Simulation
    details: security schemes from your spec are enforced automatically. no token? 401. skip it all with --no-auth.
  - icon: 🔄
    title: Proxy + Record + Replay
    details: forward to a real API, validate against your spec, record responses, and replay them offline.
---

## quick demo

```bash
# install
go install github.com/murphships/portblock@latest

# run it
portblock serve api.yaml

# that's it. working API at localhost:4000
curl localhost:4000/users | jq '.[0]'
```

```json
{
  "id": "6e759b9a-874f-4a43-aaee-c810d8151d86",
  "name": "Johnathon Braun",
  "email": "kaleyboyer@garcia.net",
  "company": "Development Seed",
  "city": "Boston",
  "status": "active"
}
```

no examples in your spec. no config files. just realistic data, instantly.
