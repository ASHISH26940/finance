# Finance API Documentation

## Base URL

`http://localhost:3000`

## OpenAPI / Swagger

- OpenAPI spec: `GET /docs/openapi.yaml`
- Swagger UI: `GET /docs`

## Authentication

Public endpoints:

- `POST /auth/signup`
- `POST /auth/login`

Protected endpoints accept either:

- cookie: `token`
- header: `Authorization: Bearer <token>`

`POST /auth/logout` revokes the current JWT by blacklisting it until expiry.

## Permissions

Permissions are read from `roles.permissions`.

- `can_view_dashboard` for `GET /dashboard/summary`
- `can_view_insights` for `GET /dashboard/trends`
- `can_view_records` for `GET /records/`
- `can_create_records` for `POST /records/`
- `can_update_records` for `PUT /records/:id`
- `can_delete_records` for `DELETE /records/:id`
- `can_manage_users` for all `/admin/*` routes

## Standard Error Shape

```json
{
  "status": "error",
  "message": "..."
}
```

## Request Guards

- Global API rate limit: `120` requests per IP per minute for non-auth routes
- `POST /auth/login`: `10` requests per IP per 10 minutes
- `POST /auth/signup`: `5` requests per IP per 15 minutes
- `POST /records/` supports `Idempotency-Key`
- Reusing an idempotency key with the same user and same payload returns the cached response for 24 hours
- Reusing the same key with a different payload returns `409 Conflict`

---

## Auth Endpoints

### POST `/auth/signup`

Creates a user and assigns the default `viewer` role.

Request body:

```json
{
  "name": "Kai User",
  "email": "kai@example.com",
  "password": "Password@123"
}
```

Success response (`201`):

```json
{
  "status": "success",
  "message": "Signup successful",
  "user": {
    "id": 6,
    "name": "Kai User",
    "email": "kai@example.com",
    "role_id": 1,
    "active": true,
    "password_updated_at": 1775378276
  }
}
```

Common errors:

- `400` invalid body, missing required fields, short password, duplicate email
- `429` too many attempts
- `500` hashing, role lookup, or server config failure

### POST `/auth/login`

Authenticates a user and sets the `token` cookie.

Request body:

```json
{
  "email": "kai@example.com",
  "password": "Password@123"
}
```

Success response (`200`):

```json
{
  "status": "success",
  "message": "Login successful",
  "user": {
    "id": 6,
    "name": "Kai User",
    "email": "kai@example.com",
    "role_id": 1,
    "active": true,
    "password_updated_at": 1775378276
  }
}
```

Common errors:

- `400` invalid body or missing credentials
- `401` invalid credentials
- `429` too many attempts

### POST `/auth/logout`

Revokes the current JWT and clears the `token` cookie.

Success response (`200`):

```json
{
  "status": "success",
  "message": "Logout successful"
}
```

Notes:

- blacklist storage uses Redis when available
- if Redis is unavailable, supported blacklist behavior falls back to in-memory storage

---

## Record Endpoints

### POST `/records/`

Creates a financial record for the authenticated user.

Permission: `can_create_records`

Request body:

```json
{
  "amount": 1250.75,
  "type": "income",
  "category_id": 1,
  "date": "2026-04-05T10:00:00Z",
  "note": "salary credit"
}
```

Optional header:

```http
Idempotency-Key: create-record-001
```

Success response (`200`):

```json
{
  "id": 12,
  "user_id": 6,
  "amount": 1250.75,
  "type": "income",
  "category_id": 1,
  "date": "2026-04-05T10:00:00Z",
  "note": "salary credit",
  "created_at": "2026-04-05T10:01:00Z",
  "updated_at": "2026-04-05T10:01:00Z",
  "deleted_at": null
}
```

Common errors:

- `400` invalid body, invalid type, invalid category, invalid date, non-positive amount
- `401` missing or invalid token
- `403` permission denied
- `409` idempotency conflict
- `500` server or database error

### GET `/records/`

Returns paginated records for the authenticated user.

Permission: `can_view_records`

Query params:

- `page` optional, default `1`
- `per_page` optional, default `20`, max `100`
- `search` optional
- `type` optional, `income` or `expense`
- `category_id` optional
- `from` optional, `YYYY-MM-DD` or RFC3339
- `to` optional, `YYYY-MM-DD` or RFC3339

Example:

```http
GET /records/?page=1&per_page=10&search=salary&type=income&category_id=1&from=2026-04-01&to=2026-04-30
```

Success response (`200`):

```json
{
  "status": "success",
  "data": [
    {
      "id": 12,
      "user_id": 6,
      "amount": 1250.75,
      "type": "income",
      "category_id": 1,
      "date": "2026-04-05T10:00:00Z",
      "note": "salary credit",
      "created_at": "2026-04-05T10:01:00Z",
      "updated_at": "2026-04-05T10:01:00Z",
      "deleted_at": null
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

### PUT `/records/:id`

Updates a record owned by the authenticated user.

Permission: `can_update_records`

Request body:

```json
{
  "amount": 1500.25,
  "type": "income",
  "category_id": 1,
  "date": "2026-04-06T10:00:00Z",
  "note": "updated note"
}
```

All fields are optional, but at least one must be provided.

Success response (`200`): updated record object.

### DELETE `/records/:id`

Soft deletes a record owned by the authenticated user.

Permission: `can_delete_records`

Success response (`200`):

```json
{
  "status": "deleted"
}
```

Notes:

- soft-deleted records are excluded from record listings and dashboard calculations

---

## Dashboard Endpoints

### GET `/dashboard/summary`

Returns aggregated dashboard data for the authenticated user.

Permission: `can_view_dashboard`

Success response (`200`):

```json
{
  "status": "success",
  "data": {
    "total_income": 65000,
    "total_expenses": 3400.5,
    "net_balance": 61600.5,
    "category_totals": [
      {
        "category_id": 1,
        "category_name": "Salary",
        "type": "income",
        "total": 50000
      }
    ],
    "recent_activity": [
      {
        "id": 10,
        "amount": 1200.5,
        "type": "expense",
        "category_id": 3,
        "category_name": "Food",
        "date": "2026-04-05T10:00:00Z",
        "note": "Lunch + groceries"
      }
    ],
    "monthly_trends": [
      {
        "period": "2026-04",
        "income": 65000,
        "expense": 3400.5,
        "net": 61600.5
      }
    ],
    "generated_at": "2026-04-05T16:45:22+05:30"
  }
}
```

### GET `/dashboard/trends`

Returns trend-only data for the authenticated user.

Permission: `can_view_insights`

Query params:

- `period` optional, `monthly` or `weekly`, default `monthly`
- `points` optional, positive integer, max `24`, default `6`

Example:

```http
GET /dashboard/trends?period=monthly&points=6
```

Success response (`200`):

```json
{
  "status": "success",
  "data": {
    "period": "monthly",
    "points": 6,
    "trends": [
      {
        "period": "2026-04",
        "income": 65000,
        "expense": 3400.5,
        "net": 61600.5
      }
    ],
    "generated_at": "2026-04-05T16:45:22+05:30"
  }
}
```

---

## Admin Endpoints

All admin routes require authentication plus `can_manage_users`.

### GET `/admin/users`

Returns all users.

Success response (`200`):

```json
{
  "status": "success",
  "data": [
    {
      "id": 1,
      "name": "Kai User",
      "email": "kai@example.com",
      "role_id": 1,
      "active": true,
      "created_at": "2026-04-05T10:00:00Z",
      "updated_at": "2026-04-05T10:00:00Z"
    }
  ]
}
```

### GET `/admin/users/:id`

Returns one user by ID.

### PUT `/admin/users/:id`

Updates user fields.

Request body:

```json
{
  "name": "Updated User",
  "role_id": 2,
  "active": true
}
```

All fields are optional, but at least one is required.

### GET `/admin/roles`

Returns all roles.

Success response (`200`):

```json
{
  "status": "success",
  "data": [
    {
      "id": 1,
      "name": "viewer",
      "permissions": {
        "can_view_dashboard": true,
        "can_view_records": false,
        "can_view_insights": false,
        "can_create_records": false,
        "can_update_records": false,
        "can_delete_records": false,
        "can_manage_users": false
      }
    }
  ]
}
```

### GET `/admin/roles/:id`

Returns one role by ID.

### POST `/admin/roles`

Creates a new role.

Request body:

```json
{
  "name": "auditor",
  "permissions": {
    "can_view_dashboard": true,
    "can_view_records": true,
    "can_view_insights": true,
    "can_create_records": false,
    "can_update_records": false,
    "can_delete_records": false,
    "can_manage_users": false
  }
}
```

Success response (`201`): created role object.

### PUT `/admin/roles/:id`

Updates a role name and permissions.

Request body:

```json
{
  "name": "analyst",
  "permissions": {
    "can_view_dashboard": true,
    "can_view_records": true,
    "can_view_insights": true,
    "can_create_records": false,
    "can_update_records": false,
    "can_delete_records": false,
    "can_manage_users": false
  }
}
```

Common admin errors:

- `400` invalid body, invalid ID, missing fields, duplicate role name
- `401` missing or invalid token
- `403` permission denied
- `404` user or role not found
- `500` server or database error

---

## Testing with Insomnia

Use [finance-api-insomnia.json](/home/kai/code/finance/finance-api-insomnia.json) to test the current routes:

- auth
- records
- dashboard
- admin
