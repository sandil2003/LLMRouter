# LLMRouter

LLMRouter is a cross-platform desktop application for intelligently routing requests across multiple Large Language Model (LLM) API providers.

The primary goal is to provide a single local API endpoint that applications can use while LLMRouter automatically selects an available provider and falls back to other providers when a provider reaches its rate limit, quota, becomes unavailable, or encounters an error.

The application is designed to run on:

* Windows
* macOS

The architecture should remain extensible for future platforms such as Linux and potentially mobile clients.

> 📖 **Architecture Deep-Dive**: For detailed pipeline flows, diagrams, and algorithmic specifications, check out the [Intelligent Routing Pipeline Architecture](docs/architecture.md).

---

# 1. Core Objective

LLMRouter acts as a local LLM API gateway.

Instead of an application directly communicating with:

```text
Application
    ↓
Gemini
```

the application communicates with:

```text
Application
    ↓
LLMRouter
    ↓
┌──────────┬──────────┬────────────┐
│ Gemini   │ Groq     │ OpenRouter │
└──────────┴──────────┴────────────┘
```

If one provider becomes unavailable:

```text
Request
   ↓
Provider A
   ↓
Rate Limited ❌
   ↓
Provider B
   ↓
Success ✅
```

The user should not need to manually change providers.

---

# 2. Technology Stack

## Frontend

* React
* TypeScript
* Vite
* Bootstrap
* React Router

Responsibilities:

* User interface
* Dashboard
* Provider management
* Routing configuration
* Usage analytics
* Logs
* Settings
* Application status
* Communication with the local Go API
* Communication with Tauri when native functionality is required

---

## Desktop Layer

* Tauri
* Rust

Responsibilities:

* Desktop application lifecycle
* Native OS integration
* Starting and stopping the Go backend
* Native notifications
* System tray integration
* Window management
* Secure credential/key storage integration
* Native file dialogs
* Platform-specific functionality

Rust should remain a thin native layer.

Do NOT move the core LLM routing logic into Rust.

---

## Backend

* Go

The Go application is the core LLM gateway.

Responsibilities:

* HTTP API server
* LLM provider abstraction
* Provider communication
* Request routing
* Provider selection
* Rate-limit detection
* Retry handling
* Fallback handling
* Circuit breakers
* Health checks
* Streaming responses
* Usage tracking
* Logging
* SQLite persistence

---

## Database

* SQLite

SQLite is the default local database.

The application is primarily a desktop application, therefore a local database is preferred over PostgreSQL for the initial architecture.

The database should store application state such as:

* Providers
* Models
* Routing rules
* Usage statistics
* Request logs
* Provider health state
* Configuration metadata

API keys and other sensitive credentials should NOT be stored as plain text in SQLite.

Sensitive credentials should use the operating system's secure credential/keychain mechanism through Tauri/Rust.

---

# 3. High-Level Architecture

```text
                         LLMRouter
                            │
              ┌─────────────┴─────────────┐
              │                           │
              ▼                           ▼
        React + TypeScript          Tauri + Rust
              │                           │
              │                           │
              └─────────────┬─────────────┘
                            │
                     Local HTTP API
                            │
                            ▼
                    ┌──────────────┐
                    │ Go Gateway   │
                    └──────┬───────┘
                           │
             ┌─────────────┼─────────────┐
             │             │             │
             ▼             ▼             ▼
          Gemini          Groq       OpenRouter
             │             │             │
             └─────────────┼─────────────┘
                           │
                           ▼
                        Internet
```

---

# 4. Repository Structure

```text
LLMRouter/
│
├── frontend/
│   │
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── services/
│   │   ├── hooks/
│   │   ├── types/
│   │   ├── utils/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   └── index.css
│   │
│   ├── src-tauri/
│   │   ├── src/
│   │   │   ├── main.rs
│   │   │   ├── commands/
│   │   │   ├── backend/
│   │   │   └── services/
│   │   │
│   │   ├── binaries/
│   │   ├── icons/
│   │   ├── Cargo.toml
│   │   ├── Cargo.lock
│   │   └── tauri.conf.json
│   │
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
│
├── backend/
│   │
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   │
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handlers/
│   │   │   ├── middleware/
│   │   │   └── routes.go
│   │   │
│   │   ├── router/
│   │   ├── providers/
│   │   ├── ratelimit/
│   │   ├── retry/
│   │   ├── circuitbreaker/
│   │   ├── models/
│   │   ├── database/
│   │   ├── config/
│   │   └── logging/
│   │
│   ├── tests/
│   ├── go.mod
│   └── go.sum
│
├── database/
│   └── migrations/
│
├── docs/
│   ├── architecture.md
│   ├── api.md
│   └── routing.md
│
├── scripts/
│
├── .gitignore
├── README.md
└── LICENSE
```

---

# 5. Frontend Architecture

The frontend should follow a component-based architecture.

```text
src/
├── components/
├── pages/
├── services/
├── hooks/
├── types/
├── utils/
├── App.tsx
└── main.tsx
```

## Pages

The initial pages should be:

```text
Dashboard
Providers
Routing
Usage
Logs
Settings
```

### Dashboard

Display:

* Gateway status
* Number of active providers
* Provider health
* Requests today
* Successful requests
* Failed requests
* Current routing activity
* Recent requests
* Provider usage

---

### Providers

Allow users to:

* Add provider
* Remove provider
* Enable/disable provider
* Configure models
* Configure priority
* Test provider connection
* View provider status
* View rate-limit information

Example:

```text
Gemini
Status: Healthy
Model: Gemini model
Priority: 1
Requests: 125
```

---

### Routing

Allow users to configure:

* Provider priority
* Model priority
* Fallback order
* Retry behavior
* Rate-limit behavior
* Enabled providers

Example:

```text
Priority

1. Gemini
2. Groq
3. OpenRouter
4. OpenAI
```

---

### Usage

Display:

* Requests
* Successful requests
* Failed requests
* Provider usage
* Model usage
* Token usage when available
* Rate-limit events
* Fallback events

Charts should be clear and lightweight.

---

### Logs

Display:

* Timestamp
* Request ID
* Provider
* Model
* Status
* Latency
* Error
* Fallback information

---

### Settings

Include:

* Gateway configuration
* Default model
* Logging configuration
* Startup behavior
* Theme
* Application information

---

# 6. Go Backend Architecture

The Go backend is the most important part of the system.

It should be designed around clean separation of responsibilities.

```text
backend/
└── internal/
    ├── api/
    ├── router/
    ├── providers/
    ├── ratelimit/
    ├── retry/
    ├── circuitbreaker/
    ├── models/
    ├── database/
    ├── config/
    └── logging/
```

---

# 7. Provider Abstraction

Every LLM provider must implement a common interface.

Example:

```go
type Provider interface {
    Name() string

    Chat(
        ctx context.Context,
        req ChatRequest,
    ) (*ChatResponse, error)

    HealthCheck(
        ctx context.Context,
    ) error
}
```

The router should depend on the interface rather than specific providers.

Example:

```text
Provider Interface
       │
       ├── Gemini
       ├── Groq
       ├── OpenRouter
       └── OpenAI
```

Adding a new provider should NOT require rewriting the router.

---

# 8. Routing Engine

The router is responsible for deciding which provider should receive a request.

Basic flow:

```text
Incoming Request
       ↓
Validate Request
       ↓
Determine Model
       ↓
Get Available Providers
       ↓
Check Provider Health
       ↓
Check Rate Limit
       ↓
Select Highest Priority Provider
       ↓
Send Request
       ↓
Success?
   ┌───┴───┐
   │       │
  YES      NO
   │       │
   ▼       ▼
Response  Retry/Fallback
           │
           ▼
      Next Provider
```

---

# 9. Provider Selection

The initial routing strategy should support priority-based routing.

Example:

```text
Provider       Priority
-----------------------
Gemini             1
Groq               2
OpenRouter         3
OpenAI             4
```

The router attempts providers in priority order.

Later the routing system can support additional strategies:

* Priority
* Round-robin
* Least-used
* Lowest latency
* Cost-aware
* Health-aware
* Weighted routing

The architecture should allow new strategies without rewriting the entire router.

---

# 10. Rate Limiting

The application must detect provider rate-limit responses.

Examples:

```text
HTTP 429
Quota exceeded
Too many requests
Rate limit exceeded
Provider-specific quota errors
```

When a provider is rate limited:

```text
Gemini
  ↓
429
  ↓
Mark temporarily unavailable
  ↓
Try Groq
```

The application should track:

* Rate-limit timestamp
* Provider
* Model
* Retry-after information when available
* Current availability state

Do not continuously retry a provider that is known to be rate limited.

---

# 11. Retry System

Retries should use controlled exponential backoff.

Example:

```text
Attempt 1
   ↓
Wait
   ↓
Attempt 2
   ↓
Wait longer
   ↓
Attempt 3
```

Avoid infinite retries.

The retry system must have:

* Maximum attempts
* Maximum delay
* Exponential backoff
* Optional jitter
* Context cancellation

---

# 12. Circuit Breaker

Providers that repeatedly fail should temporarily enter an unavailable state.

Example:

```text
Healthy
   ↓
Failures increase
   ↓
Open Circuit
   ↓
Stop sending requests
   ↓
Wait
   ↓
Half Open
   ↓
Health test
   ↓
Healthy OR Open
```

This prevents repeatedly sending requests to an unavailable provider.

---

# 13. Streaming

The gateway should support streaming responses where the provider supports them.

The architecture should eventually support an OpenAI-compatible interface such as:

```text
POST /v1/chat/completions
```

with streaming support.

The goal is for applications to interact with LLMRouter similarly to a normal LLM API.

---

# 14. Local API

The Go backend should expose a local HTTP API.

Example:

```text
http://127.0.0.1:<port>
```

Example endpoints:

```text
GET    /api/health

GET    /api/providers
POST   /api/providers
PUT    /api/providers/:id
DELETE /api/providers/:id

GET    /api/models

GET    /api/routing
PUT    /api/routing

GET    /api/usage
GET    /api/logs

POST   /api/providers/:id/test

POST   /v1/chat/completions
```

The exact port should be configurable.

Prefer binding to:

```text
127.0.0.1
```

rather than:

```text
0.0.0.0
```

unless there is an explicit requirement for network access.

---

# 15. Security

Security is important because the application manages LLM API credentials.

## Never:

* Hardcode API keys
* Commit API keys
* Store API keys in Git
* Store API keys as plain text in SQLite
* Log API keys
* Return API keys through normal API responses

## API keys

Use the operating system's secure credential storage through Tauri/Rust.

The frontend should never directly manage raw API credentials unnecessarily.

Preferred flow:

```text
React
  ↓
Tauri Command
  ↓
OS Credential Store
```

The Go backend should receive credentials securely when needed.

---

# 16. SQLite Database

The database should be local.

Initial conceptual schema:

```text
providers
---------
id
name
enabled
priority
base_url
created_at
updated_at


models
------
id
provider_id
name
enabled
created_at


routing_rules
-------------
id
strategy
enabled
created_at
updated_at


usage
-----
id
provider_id
model
request_id
input_tokens
output_tokens
latency
status
created_at


request_logs
------------
id
request_id
provider_id
model
status
latency
error
fallback_used
created_at
```

Do not over-engineer the schema initially.

Use migrations for database changes.

---

# 17. Error Handling

Errors should be structured.

Do not return random strings throughout the application.

Errors should contain useful information such as:

```text
provider
error type
HTTP status
retryable
rate limited
message
request ID
```

The frontend should receive safe, user-friendly errors.

Internal provider errors should not expose sensitive information.

---

# 18. Logging

Use structured logging in Go.

Every request should have a request ID.

Example:

```text
Request ID: abc123

Provider: Gemini
Model: model-name
Status: Rate Limited
Fallback: Groq
Latency: 1.2s
```

Never log:

* API keys
* Authorization headers
* Sensitive user content unless explicitly required
* Secrets

---

# 19. Tauri/Rust Responsibilities

Rust should remain small and focused.

Primary responsibilities:

```text
Application lifecycle
        │
        ├── Start Go backend
        ├── Stop Go backend
        ├── Monitor Go process
        ├── Native notifications
        ├── System tray
        ├── Secure credential storage
        └── Native OS integration
```

Do not implement:

```text
LLM routing
Provider logic
Rate limiting
Retry logic
Database business logic
```

in Rust.

Those belong in Go.

---

# 20. Go Backend Process

The Go backend should be packaged with the desktop application.

Conceptually:

```text
LLMRouter Desktop
│
├── React UI
├── Tauri
├── Rust
└── Go Gateway
```

When the application starts:

```text
User opens LLMRouter
        ↓
Tauri starts
        ↓
Rust starts Go process
        ↓
Go initializes SQLite
        ↓
Go starts HTTP server
        ↓
React connects to Go
        ↓
Application Ready
```

When the application closes:

```text
Application closes
        ↓
Tauri shutdown
        ↓
Stop Go process
        ↓
Exit
```

The implementation must work on both Windows and macOS.

---

# 21. Cross-Platform Requirements

The application must support:

```text
Windows
macOS
```

Avoid unnecessary platform-specific code.

Platform-specific functionality should be isolated inside:

```text
src-tauri/
```

React code should remain platform-independent.

Go code should remain platform-independent wherever possible.

Do not assume:

```text
Windows paths
Windows environment variables
Windows shell commands
```

inside shared application logic.

Likewise, do not assume macOS-specific paths or commands.

---

# 22. Frontend ↔ Backend Communication

The normal flow is:

```text
React
   ↓ HTTP
Go Gateway
```

Example:

```text
React
GET /api/providers
       ↓
Go
       ↓
SQLite
       ↓
JSON
       ↓
React
```

Tauri commands should only be used when native functionality is required.

Do not unnecessarily route normal API requests through Rust.

---

# 23. Coding Standards

The coding agent must prioritize:

1. Clean architecture
2. Maintainability
3. Separation of concerns
4. Reusability
5. Type safety
6. Testability
7. Security
8. Cross-platform compatibility
9. Simplicity

Avoid premature optimization.

Avoid unnecessary abstractions.

Do not create a complex architecture just for the sake of architecture.

---

# 24. React Standards

Use:

* TypeScript
* Functional components
* React hooks
* Strong typing
* Reusable components
* Clear component boundaries

Avoid:

* Large monolithic components
* Duplicated UI logic
* `any` unless genuinely necessary
* Business logic inside UI components

Business/API logic should live in:

```text
services/
hooks/
utils/
```

---

# 25. Go Standards

Follow idiomatic Go.

Use:

* Small interfaces
* Dependency injection where useful
* Explicit error handling
* Context propagation
* Structured logging
* Unit tests
* Clear package boundaries

Avoid:

* Global mutable state
* Huge functions
* Circular dependencies
* Unnecessary interfaces
* Over-engineering

Use interfaces primarily at important boundaries such as providers.

---

# 26. Rust Standards

Follow idiomatic Rust.

Use:

* Strong types
* Result-based error handling
* Small Tauri commands
* Clear modules
* Safe process management
* Minimal unsafe code

Rust should not become a second backend.

---

# 27. Testing

Testing should be introduced throughout development.

## Go

Test:

* Provider clients
* Router
* Fallback
* Retry
* Rate-limit handling
* Circuit breaker
* Database repositories
* API handlers

Example:

```text
Provider A → rate limited
Provider B → success

Expected:
Provider B receives request
```

---

## Frontend

Test:

* Components
* Provider configuration
* Routing configuration
* API service behavior
* Error states
* Loading states

---

## Integration

Eventually test:

```text
React
 ↓
Tauri
 ↓
Go
 ↓
Provider
```

without relying exclusively on real API calls.

Mock providers should be available for testing.

---

# 28. Development Phases

Do not implement the entire application at once.

## Phase 1 — Desktop Shell

Build:

```text
React
+
Tauri
+
Rust
```

Requirements:

* Desktop window
* Basic navigation
* Basic layout
* Windows development working

---

## Phase 2 — UI

Build:

```text
Dashboard
Providers
Routing
Usage
Logs
Settings
```

Use mocked data initially.

Do not connect to real LLM providers yet.

---

## Phase 3 — Go Gateway

Build:

```text
Go HTTP Server
       ↓
Health endpoint
       ↓
Provider abstraction
```

---

## Phase 4 — SQLite

Add:

```text
SQLite
+
Migrations
+
Repositories
```

---

## Phase 5 — First Provider

Implement one provider completely.

Recommended initial approach:

```text
Provider Interface
       ↓
One Provider
       ↓
Chat Request
       ↓
Response
```

---

## Phase 6 — Multiple Providers

Add:

```text
Gemini
Groq
OpenRouter
```

The exact provider list can change.

Providers must be implemented independently.

---

## Phase 7 — Routing

Implement:

```text
Priority Routing
+
Fallback
+
Retry
+
Rate-limit handling
```

---

## Phase 8 — Monitoring

Add:

```text
Usage
Logs
Latency
Provider health
Fallback statistics
```

---

## Phase 9 — Tauri Integration

Implement:

```text
Rust
 ↓
Start Go
 ↓
Monitor Go
 ↓
Stop Go
```

Add secure credential storage.

---

## Phase 10 — Packaging

Build and test:

```text
Windows
macOS
```

Verify:

* Installation
* Startup
* Backend startup
* Backend shutdown
* SQLite persistence
* Provider configuration
* API requests
* Updates/configuration
* Uninstallation

---

# 29. MVP Definition

The MVP is complete when a user can:

1. Install LLMRouter.
2. Open the desktop application.
3. Add multiple LLM providers.
4. Configure API credentials securely.
5. Configure provider priority.
6. Send an OpenAI-compatible chat request.
7. Have the router select a provider.
8. Automatically fall back when a provider is unavailable.
9. Detect rate limits.
10. Retry appropriate failures.
11. View request logs.
12. View basic usage statistics.
13. Close and reopen the application without losing configuration.
14. Run the application on Windows.
15. Run the application on macOS.

---

# 30. Future Features

Potential future features include:

* More LLM providers
* Automatic model selection
* Cost-aware routing
* Latency-aware routing
* Token-aware routing
* Weighted routing
* Load balancing
* Advanced analytics
* Provider performance comparison
* Custom routing rules
* Prompt transformation
* Request caching
* Response caching
* Local model support
* Ollama integration
* LM Studio integration
* Plugin system
* Cloud synchronization
* Remote gateway
* Linux support
* Mobile client

These should NOT be implemented until the MVP architecture is stable.

---

# 31. Important Architectural Principle

Always preserve this separation:

```text
┌───────────────────────────┐
│ React                     │
│ Presentation              │
└─────────────┬─────────────┘
              │
┌─────────────▼─────────────┐
│ Tauri + Rust              │
│ Native desktop layer      │
└─────────────┬─────────────┘
              │
┌─────────────▼─────────────┐
│ Go                        │
│ Application/backend logic │
└─────────────┬─────────────┘
              │
┌─────────────▼─────────────┐
│ SQLite                    │
│ Persistence               │
└───────────────────────────┘
```

Do not mix responsibilities between these layers without a strong reason.

---

# 32. Coding Agent Instructions

When modifying this project, the coding agent MUST:

### Before coding

1. Inspect the existing project structure.
2. Understand the current architecture.
3. Reuse existing abstractions.
4. Avoid creating duplicate functionality.
5. Check whether the requested feature belongs to React, Rust, or Go.

### While coding

1. Follow the existing architecture.
2. Keep modules small.
3. Prefer reusable components.
4. Use strong typing.
5. Handle errors explicitly.
6. Add loading states to asynchronous UI.
7. Add error states to asynchronous UI.
8. Validate user input.
9. Never expose secrets.
10. Avoid hardcoded provider-specific logic in the router.
11. Maintain Windows/macOS compatibility.
12. Add tests for important backend behavior.

### Before finishing

1. Run formatting.
2. Run linting.
3. Run type checking.
4. Run relevant tests.
5. Verify the application builds.
6. Check for accidental secrets.
7. Check that existing functionality has not been broken.

---

# 33. Do Not Over-Engineer

This project should evolve incrementally.

Do NOT introduce:

* Microservices
* Kubernetes
* Redis
* PostgreSQL
* Message queues
* Cloud infrastructure

unless a real requirement appears.

The initial application is a local desktop application.

Prefer:

```text
React
+
Tauri
+
Rust
+
Go
+
SQLite
```

over introducing unnecessary infrastructure.

---

# 34. Primary Design Goal

The most important property of LLMRouter is:

> Applications should be able to communicate with one stable LLM endpoint while LLMRouter transparently manages provider availability, rate limits, failures, retries, and fallback routing.

The user should not have to think about which provider is currently available.

The complexity should be handled by the routing engine.

```text
                USER APPLICATION
                       │
                       ▼
              ┌─────────────────┐
              │    LLMRouter    │
              │                 │
              │ Smart Routing   │
              │ Rate Limits     │
              │ Retry           │
              │ Fallback        │
              │ Health          │
              └────────┬────────┘
                       │
             ┌─────────┼─────────┐
             ▼         ▼         ▼
          Provider   Provider   Provider
             A         B          C
```

LLMRouter should make multiple LLM APIs behave like **one reliable API**.
