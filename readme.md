# Finance API

A backend service for managing users, roles, financial records, and dashboard summaries.

This project was built to demonstrate:

- role-based access control
- financial record CRUD
- dashboard aggregation APIs
- validation and error handling
- database-backed persistence
- thoughtful backend design decisions

## Stack

- Go
- Fiber
- GORM
- PostgreSQL (Supabase)
- Redis (Upstash)

## Project Structure

```text
config/         configuration loading
controllers/    request handlers
database/       PostgreSQL and Redis connection setup
docs/           API documentation
logic/          shared logic helpers
middlewares/    auth, RBAC, rate limiting, idempotency
migrations/     SQL schema files
models/         database models and query logic
routes/         route registration
```

## Features Implemented

- user signup and login
- logout with JWT blacklisting
- role-based authorization using permissions stored on roles
- active and inactive user checks during authenticated requests
- create, list, update, and delete financial records
- soft delete for records
- filtering by type, category, and date range
- search by note text
- pagination for record listing
- dashboard summary API
- dashboard trend API
- request rate limiting
- idempotency support for record creation
- API documentation
- OpenAPI and Swagger UI documentation
- unit tests for core controller helper logic

## Core Requirements Coverage

### 1. User and Role Management

Implemented:

- users can sign up and log in
- users are assigned roles
- user status is tracked with an `active` flag
- protected routes reject inactive users
- route access is enforced using role permissions

Role behavior is permission-driven and supports the assignment’s intent:

- `Viewer`
  Can view dashboard data
- `Analyst`
  Can view records and insights
- `Admin`
  Can create, update, and delete records and has broader management permissions through the permission model

Note:

- roles are stored in the database and resolved at request time
- permissions are stored as JSON on the `roles` table
- this project currently focuses on role enforcement rather than a full admin user-management API surface

### 2. Financial Records Management

Implemented record fields:

- amount
- type: `income` or `expense`
- category
- date
- note

Implemented operations:

- create record
- view records
- update record
- delete record
- filter records by:
  - type
  - category
  - date range
- search records by note text
- paginate record listing

### 3. Dashboard Summary APIs

Implemented:

- total income
- total expenses
- net balance
- category-wise totals
- recent activity
- monthly trends
- weekly trends

The dashboard functionality is intentionally more than CRUD and focuses on aggregated views of financial data.

### 4. Access Control Logic

Implemented at backend level through middleware:

- JWT authentication middleware
- active-user validation
- token blacklist validation
- permission-based authorization middleware

Examples of enforced behavior:

- viewers cannot create or modify records
- analysts can read data they are permitted to access
- only roles with the required permission can create, update, or delete records

### 5. Validation and Error Handling

Implemented:

- request body validation for auth endpoints
- password length validation
- duplicate email protection
- token format and expiry checks
- record list query validation for:
  - page
  - per-page
  - record type
  - category ID
  - date filters
- meaningful HTTP status codes such as `400`, `401`, `403`, `404`, `409`, `429`, and `500`
- conflict protection for idempotency key reuse with a different payload

### 6. Data Persistence

Persistence choice:

- PostgreSQL on Supabase is used as the main relational database
- Redis is used for token blacklist and idempotency storage when available
- if Redis is unavailable, supported features fall back to in-memory storage where implemented

Schema is managed through SQL files in:

- `migrations/postgres/roles.sql`
- `migrations/postgres/users.sql`
- `migrations/postgres/categories.sql`
- `migrations/postgres/financial.sql`

## Optional Enhancements Implemented

Implemented from the optional list:

- authentication using JWT
- pagination for record listing
- search support
- soft delete functionality
- rate limiting
- unit tests
- API documentation
- OpenAPI / Swagger documentation

Additional thoughtfulness included:

- JWT blacklist on logout
- Redis fallback behavior
- idempotency support on record creation

## API Overview

Auth:

- `POST /auth/signup`
- `POST /auth/login`
- `POST /auth/logout`

Records:

- `POST /records/`
- `GET /records/`
- `PUT /records/:id`
- `DELETE /records/:id`

Dashboard:

- `GET /dashboard/summary`
- `GET /dashboard/trends`

Detailed request and response examples are documented in:

[API.md](/home/kai/code/finance/docs/API.md)

OpenAPI and Swagger are available at:

- `GET /docs/openapi.yaml`
- `GET /docs`

An Insomnia collection is included here:

[finance-api-insomnia.json](/home/kai/code/finance/finance-api-insomnia.json)

## Setup

### 1. Prerequisites

- Go installed
- PostgreSQL database running
- Redis running

### 2. Configuration

The app reads configuration from environment variables or local config loading logic in [config.go](/home/kai/code/finance/config/config.go).

Relevant values include:

- `DB_HOST`
- `DB_PORT`
- `DB_NAME`
- `DB_USER`
- `DB_PASSWORD`
- `DB_SSLMODE`
- `DATABASE_URL`
- `REDIS_URL`
- `REDIS_PASS`
- `SECRET`
- `FN_ENV`

### 3. Run Migrations

Execute the SQL files in `migrations/postgres/` against your PostgreSQL database.

### 4. Start the App

```bash
go run main.go
```

The server starts on:

```text
http://localhost:3000
```

## Running Tests

Run all tests:

```bash
env GOCACHE=/tmp/go-build go test ./...
```

Run only controller tests:

```bash
env GOCACHE=/tmp/go-build go test ./controllers -v
```

Current tests are focused on:

- pagination parsing
- filter parsing
- record query helper behavior
- auth cookie helper behavior

## Assumptions

- roles are seeded in the database before normal usage
- newly signed up users receive a default role
- this project emphasizes backend API behavior and access control more than a full admin UI or advanced user-management workflows
- Redis is optional but recommended for better logout revocation and idempotency behavior

## Design Choices And Why

### Fiber instead of `net/http`

I chose Fiber because it provides convenient routing, middleware composition, and rapid REST API development with low boilerplate.

Why this:

- route grouping is clean
- middleware chaining is straightforward
- JSON response handling is fast to build

Why not `net/http` only:

- `net/http` is more minimal and standard
- but this project benefits from quicker route and middleware composition

Tradeoff:

- Fiber introduces framework dependency, while `net/http` stays closer to the standard library

### GORM with PostgreSQL instead of raw SQL everywhere

I chose GORM with PostgreSQL for the application data layer because it speeds up CRUD work and keeps model-related logic centralized.

Why this:

- faster implementation for common database flows
- model methods are easy to organize
- soft delete support is built in

Why not raw SQL everywhere:

- raw SQL offers maximum control
- but it increases boilerplate for a CRUD-heavy backend

Tradeoff:

- some queries are less explicit than handwritten SQL

### Supabase PostgreSQL and hosted Redis instead of managing AWS database infrastructure

I chose Supabase for PostgreSQL and hosted Redis for cache-like features because it kept the project practical and affordable without adding AWS infrastructure cost and ops overhead.

Why this:

- faster setup for a portfolio-style backend
- managed Postgres and Redis reduce infrastructure work
- lower cost than spinning up comparable AWS services for this project

Why not self-manage on AWS:

- AWS is powerful and flexible
- but for this project it would be more expensive and add more deployment and networking setup than needed

Tradeoff:

- managed platform connection details and network constraints need to be handled carefully during deployment

### JWT authentication instead of server sessions

I chose JWTs so the API can support both cookie-based and bearer-token-based clients without maintaining traditional session state for every request.

Why this:

- practical for API clients like Insomnia
- easy to apply at middleware level
- works well for stateless auth flows

Why not server sessions:

- sessions make revocation simpler
- but they require more server-managed session infrastructure from the start

Tradeoff:

- logout becomes more complex, which is why token blacklisting was added

### Redis blacklist instead of token-expiry-only logout

I chose Redis-backed blacklisting so logout immediately revokes a JWT before its normal expiry.

Why this:

- improves security
- revoked tokens can expire automatically using their remaining TTL
- fallback behavior exists when Redis is unavailable

Why not rely on JWT expiry only:

- expiry alone does not support immediate revocation

Tradeoff:

- more moving parts than purely stateless auth

### Permission-based RBAC instead of hardcoded role checks

I chose permission-based authorization because it scales better than scattered role-name comparisons.

Why this:

- route authorization stays flexible
- permissions are easier to evolve than hardcoded role branching
- handlers remain cleaner

Why not hardcoded role checks:

- simpler initially
- but less maintainable once the permission model grows

Tradeoff:

- roles and permissions need proper seeding and maintenance

### Soft delete instead of hard delete for records

I chose soft delete because financial records are sensitive and accidental permanent deletion is costly.

Why this:

- protects against irreversible mistakes
- keeps deleted rows out of normal reads
- supports safer reporting logic

Why not hard delete:

- simpler implementation
- but permanently removes potentially important financial data

Tradeoff:

- all record queries must consistently ignore deleted rows

### Pagination and filtering on `GET /records/` instead of separate specialized routes

I chose to keep listing, pagination, search, and filtering in one endpoint.

Why this:

- smaller API surface
- more REST-like behavior
- one consistent record-query experience

Why not separate search routes:

- separate routes can be useful for more advanced search systems
- but the current use case is still filtered listing

Tradeoff:

- more query validation is required on one endpoint

### Idempotency on record creation

I chose idempotency for create operations because duplicate financial entries are a correctness issue.

Why this:

- protects against retries creating duplicate records
- especially useful for unstable networks or impatient clients

Why not skip it:

- simpler to implement
- but financially risky for duplicate submissions

Tradeoff:

- adds cache and request-body comparison logic

### Application-level rate limiting

I chose to enforce rate limits in middleware rather than assume infrastructure-level protection exists.

Why this:

- protects auth endpoints immediately
- works even in simple local deployments
- keeps the service safer by default

Why not rely only on reverse proxies or gateways:

- infrastructure throttling is excellent in production
- but may not always exist in smaller deployments

Tradeoff:

- rate limits may need coordination later if infrastructure-level throttling is also added

### Unit tests first instead of full integration coverage first

I chose to add targeted unit tests first so the project would have real automated coverage without introducing heavy database test infrastructure.

Why this:

- fast feedback
- low setup cost
- immediate coverage for parsing and controller helper logic

Why not begin with integration tests only:

- integration tests give stronger end-to-end confidence
- but need more environment setup and test isolation

Tradeoff:

- route and database integration coverage can still be improved further

## Evaluation-Oriented Notes

### Backend Design

The codebase separates concerns into routes, controllers, models, middleware, and database setup to keep responsibilities clear.

### Logical Thinking

Business rules such as active-user checks, RBAC, blacklisting, idempotency, and dashboard aggregation are implemented explicitly in middleware and query logic.

### Functionality

The main required APIs are present and optional improvements were added to make the system more realistic.

### Code Quality

The project aims for readable package boundaries, simple handler flow, and clear middleware-based security rules.

### Database and Data Modeling

A relational model was chosen because users, roles, categories, and financial records map naturally to relational tables.

### Validation and Reliability

The API returns structured errors and validates common invalid states such as bad credentials, bad query params, duplicate emails, and invalid idempotency key reuse.

### Documentation

This README explains setup, requirements coverage, design choices, assumptions, and tradeoffs. Detailed endpoint examples are documented separately in [API.md](/home/kai/code/finance/docs/API.md).

### Additional Thoughtfulness

Redis-backed revocation, idempotency handling, pagination, search, and test coverage were added beyond the core baseline.

## Current Limitations

- there is no full admin API yet for creating or editing users and roles through dedicated management endpoints
- tests are still lightweight and not full database-backed integration tests
- refresh-token rotation is not implemented
- migrations are raw SQL files rather than being managed by a dedicated migration tool
- search is limited to note text plus structured filters, not full-text indexing
