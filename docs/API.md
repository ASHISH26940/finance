# Finance API Documentation

## Base URL

`http://localhost:3000`

## Authentication

- Public endpoints:
1. `POST /auth/signup`
2. `POST /auth/login`

- Protected endpoints require a valid JWT:
1. Cookie: `token` (set by login/signup)
2. Header: `Authorization: Bearer <token>`

## Role Permissions

Permissions are evaluated from `roles.permissions` JSON.

- `CanViewDashboard` required for `GET /dashboard/summary`
- `CanViewInsights` required for `GET /dashboard/trends`
- `CanViewRecords` required for `GET /records/`
- `CanCreateRecords` required for `POST /records/`
- `CanUpdateRecords` required for `PUT /records/:id`
- `CanDeleteRecords` required for `DELETE /records/:id`

## Standard Error Shape

```json
{
  "status": "error",
  "message": "..."
}
```

## Request Guards

- Global API rate limit: `120` requests per IP per minute for non-auth routes.
- `POST /auth/login`: `10` requests per IP per 10 minutes.
- `POST /auth/signup`: `5` requests per IP per 15 minutes.
- `POST /records/` supports the `Idempotency-Key` header.
- Reusing an `Idempotency-Key` with the same authenticated user and identical request body returns the cached response for 24 hours.
- Reusing the same `Idempotency-Key` with a different request body returns `409 Conflict`.

---

## Auth APIs

### POST `/auth/signup`

Creates a new user with default `viewer` role.

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

Possible errors:

- `400` invalid body/required fields/password length
- `400` duplicate email
- `429` too many attempts
- `500` secret/server issues

### POST `/auth/login`

Authenticates user and sets `token` cookie.

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

Possible errors:

- `400` invalid body
- `401` invalid credentials
- `429` too many attempts

---

## Record APIs

### POST `/records/`

Creates a financial record for authenticated user.

Permission: `CanCreateRecords`

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

Optional headers:

```http
Idempotency-Key: create-record-001
```

Success response (`200`): created record object.

Notes:

- `date` must be RFC3339 format.
- `category_id` can be `null`.
- A matching `Idempotency-Key` replays the original response instead of creating a duplicate record.
- A duplicate request arriving while the original one is still running returns `409`.

### GET `/records/`

Returns all records for authenticated user.

Permission: `CanViewRecords`

Success response (`200`): array of record objects.

### PUT `/records/:id`

Updates an existing record by ID.

Permission: `CanUpdateRecords`

Body: record fields (same shape as create model).

### DELETE `/records/:id`

Soft deletes a record by ID.

Permission: `CanDeleteRecords`

Success response (`200`):

```json
{
  "status": "deleted"
}
```

Common errors for record APIs:

- `401` missing/invalid token
- `403` permission denied
- `404` record not found
- `500` server/database error

Soft delete behavior:

- Soft-deleted records are excluded from `GET /records/`.
- Soft-deleted records are excluded from dashboard totals, recent activity, and trends.

---

## Dashboard APIs

### GET `/dashboard/summary`

Aggregated summary for current user.

Permission: `CanViewDashboard`

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

### GET `/dashboard/trends?period=monthly&points=6`

Returns trend-only series.

Permission: `CanViewInsights`

Query params:

- `period`: `monthly` (default) or `weekly`
- `points`: positive integer, max `24` (default `6`)

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

## Testing with Insomnia

Use either export file:

1. `finance-api-insomnia.json`
2. `insomnia/finance-api-insomnia.json`

These include all current routes:

1. Auth (`signup`, `login`)
2. Records CRUD
3. Dashboard summary/trends
