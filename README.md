# Argowil Backend

REST + WebSocket API for the Argowil mobile app. Written in Go, backed by MySQL, with optional Shiftbase roster integration.

---

## Stack

| Layer | Technology |
|---|---|
| Language | Go 1.25 |
| Router | [chi v5](https://github.com/go-chi/chi) |
| Database | MySQL 8 / sqlx |
| Migrations | Plain SQL (auto-applied on startup) |
| Auth | JWT (HS256) — 15 min access · 30 day refresh |
| Real-time | WebSocket (gorilla/websocket) |
| Storage | S3-compatible (Hetzner Object Storage) |
| Roster | Shiftbase Public API v2 |

---

## Project layout

```
backend/
├── cmd/server/         # Entrypoint — wires everything together
├── config/             # Env-var loading
├── migrations/         # SQL migrations (applied in filename order)
└── internal/
    ├── auth/           # JWT minting, middleware, role guards
    ├── admin/          # Employee management (teamleader/admin)
    ├── community/      # Group chat — REST history + WebSocket hub
    ├── news/           # News posts & comments
    ├── openshift/      # Open shift listing, claiming, creation
    ├── schedule/       # Roster & time-registration proxy
    ├── shiftbase/      # Shiftbase API client
    ├── storage/        # S3 client (image uploads)
    ├── user/           # User CRUD
    └── database/       # MySQL connection + migration runner
```

---

## Getting started

### Prerequisites

- Go 1.25+
- MySQL 8 running locally (or via Docker)
- (Optional) A Shiftbase subscription + API key

### 1. Database

```sql
CREATE DATABASE argowil CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'argowil_user'@'localhost' IDENTIFIED BY 'changeme';
GRANT ALL PRIVILEGES ON argowil.* TO 'argowil_user'@'localhost';
```

### 2. Environment

```bash
cp .env.example .env
# Edit .env — fill in DB credentials, JWT_SECRET, and optionally Shiftbase keys
```

Generate a secure JWT secret:

```bash
openssl rand -hex 32
```

### 3. Run

```bash
go run ./cmd/server
# or build first:
go build -o argowil-backend ./cmd/server && ./argowil-backend
```

Migrations run automatically on startup. The server listens on `:8080` by default.

---

## Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `PORT` | no | `8080` | HTTP listen port |
| `DB_HOST` | no | `127.0.0.1` | MySQL host |
| `DB_PORT` | no | `3306` | MySQL port |
| `DB_NAME` | **yes** | — | Database name |
| `DB_USER` | **yes** | — | Database user |
| `DB_PASSWORD` | **yes** | — | Database password |
| `JWT_SECRET` | **yes** | — | HMAC signing key (min 32 chars) |
| `JWT_ACCESS_TTL_MINUTES` | no | `15` | Access token lifetime |
| `JWT_REFRESH_TTL_DAYS` | no | `30` | Refresh token lifetime |
| `SERVER_BASE_URL` | no | `http://localhost:8080` | Public base URL (used in image URLs) |
| `SHIFTBASE_API_KEY` | no | — | Disables Shiftbase integration when empty |
| `SHIFTBASE_BASE_URL` | no | `https://api.shiftbase.com` | Shiftbase API base |
| `SHIFTBASE_DEFAULT_DEPARTMENT_ID` | no | — | Auto-assigned department for new employees |
| `S3_ENDPOINT` | no | — | S3-compatible endpoint URL |
| `S3_ACCESS_KEY` | no | — | S3 access key |
| `S3_SECRET_KEY` | no | — | S3 secret key |
| `S3_BUCKET` | no | — | S3 bucket name |
| `S3_REGION` | no | `eu-central` | S3 region |

---

## API reference

All endpoints (except login, refresh, and the WebSocket upgrade) require a `Bearer` token.

### Auth

| Method | Path | Description |
|---|---|---|
| `POST` | `/auth/login` | Email + password → token pair |
| `POST` | `/auth/refresh` | Refresh token → new token pair |
| `GET` | `/auth/me` | Current user info |
| `PUT` | `/auth/change-password` | Change own password |

### News

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/news` | employee+ | List posts |
| `GET` | `/news/:id` | employee+ | Single post + comments |
| `POST` | `/news/:id/comments` | employee+ | Add comment |
| `POST` | `/news` | teamleader+ | Create post |
| `DELETE` | `/news/:id` | teamleader+ | Delete post |
| `POST` | `/news/upload` | teamleader+ | Upload image |

### Roster (Shiftbase proxy)

| Method | Path | Description |
|---|---|---|
| `GET` | `/schedules` | Shifts for current week (`?from=&to=`) |
| `GET` | `/time-registrations` | Clock-in/out records |

### Open shifts

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/open-shifts` | employee+ | List open shifts (`?from=&to=`) |
| `POST` | `/open-shifts/:id/claim` | employee+ | Claim a shift |
| `GET` | `/shift-templates` | teamleader+ | Available shift templates |
| `POST` | `/open-shifts` | teamleader+ | Create open shift |
| `DELETE` | `/open-shifts/:id` | teamleader+ | Remove open shift |

### Community chat

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/community/messages` | employee+ | Last 50 messages |
| `PATCH` | `/community/messages/:id` | owner | Edit own message |
| `DELETE` | `/community/messages/:id` | owner | Delete own message |
| `GET` | `/community/members` | employee+ | All active users |
| `GET` | `/community/ws?token=` | token in query | WebSocket connection |

#### WebSocket events

Incoming (client → server):
```json
{ "content": "hello", "reply_to_id": 42 }
```

Outgoing (server → client):
```json
{ "type": "new",    "message": { ... } }
{ "type": "edit",   "id": 42, "message": { "content": "...", "edited": true } }
{ "type": "delete", "id": 42 }
```

### Admin panel

All require `teamleader` or `admin` role.

| Method | Path | Description |
|---|---|---|
| `GET` | `/admin/employees` | List all employees |
| `POST` | `/admin/employees` | Create employee (+ optional Shiftbase) |
| `GET` | `/admin/employees/:id` | Single employee |
| `PATCH` | `/admin/employees/:id` | Update employee |
| `DELETE` | `/admin/employees/:id` | Deactivate + remove from Shiftbase |

---

## Roles

| Role | Access |
|---|---|
| `employee` | Read-only on news, roster, open shifts, chat |
| `teamleader` | + admin panel, create/delete news, manage open shifts |
| `admin` | + user management |

---

## Shiftbase integration

When `SHIFTBASE_API_KEY` is set the backend:

- Proxies roster and time-registration data from Shiftbase
- Creates / deletes Shiftbase employee records alongside local accounts
- Fetches open shifts with `min_date` / `max_date` filtering
- Auto-resolves the default team ID from the first team in the account on startup

Leave `SHIFTBASE_API_KEY` empty to run without Shiftbase — schedule endpoints return empty arrays and employee management works locally only.
