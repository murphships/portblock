# Comparison

how portblock stacks up against the alternatives.

## the quick version

**portblock** — zero config, stateful CRUD, smart data, single binary. for devs who want a mock that works like a real API.

**prism** — good for static mocking and validation, but no state. returns the same response every time.

**mockoon** — GUI-based, good for simple mocks, but requires manual setup for everything.

**wiremock** — powerful but complex. JVM-based. config files everywhere.

**mockserver** — similar to wiremock. powerful, complex, JVM.

## detailed comparison

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

## when to use portblock

- you want a mock that behaves like a real API (stateful CRUD)
- you don't want to write config files or example blocks
- you want realistic data without manual setup
- you need a single binary with no runtime dependencies
- you're building a frontend and need a working backend in 30 seconds

## when to use something else

- **you need a GUI** → mockoon
- **you need gRPC/GraphQL mocking** → wiremock
- **you need static-only mocking with great validation** → prism
- **you need enterprise-grade features and don't mind JVM** → wiremock or mockserver

## the philosophy

most mock tools make you do the work upfront — write examples, configure responses, set up state management. portblock inverts this. give it a spec, get a working API. configure only when you need to.

less config, more building.
