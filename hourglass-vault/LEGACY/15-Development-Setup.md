# Development Setup

Complete guide to setting up local development environment for Hourglass.

## Prerequisites

- **Go**: 1.26.1 or later
- **Node.js**: 18+ with npm
- **PostgreSQL**: 13+ or Docker
- **Git**: For version control

---

## Database Setup

### Option 1: Docker (Recommended)

Easiest for local development — PostgreSQL runs in a container.

```bash
cd /Users/stefanoprivitera/Projects/hourglass

# Start PostgreSQL (and any other services)
docker-compose up -d

# Verify it's running
docker ps
# Should show: hourglass-postgres (or similar)

# Check logs
docker-compose logs -f postgres
```

**Default Credentials** (from `docker-compose.yml`):
- Host: `localhost`
- Port: `5432`
- Username: `hourglass`
- Password: `hourglass`
- Database: `hourglass`

**Migrations Auto-Run:** Docker mounts `/migrations` into PostgreSQL's init directory, so schema loads automatically on first start.

### Option 2: Local PostgreSQL

If PostgreSQL is installed locally:

```bash
# Create database
createdb -U postgres hourglass

# Create user
createuser -U postgres -P hourglass  # Password: hourglass

# Run migrations manually
# (Note: cmd/migrate doesn't exist in current snapshot)
# For now, manually execute files in /migrations in order:
psql -U hourglass -d hourglass -f migrations/001_init.up.sql
psql -U hourglass -d hourglass -f migrations/002_contracts_projects.up.sql
# ... continue with 003, 004, etc.
```

---

## Backend Setup

### 1. Environment Variables

Create `.env` file in project root (or set in shell):

```bash
# Database connection
DATABASE_URL=postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable

# JWT secret (change in production!)
JWT_SECRET=dev-secret-change-in-production

# Port (optional, defaults to 8080)
PORT=8080
```

Or export in shell:
```bash
export DATABASE_URL=postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable
export JWT_SECRET=dev-secret-change-in-production
```

### 2. Install Dependencies

```bash
cd /Users/stefanoprivitera/Projects/hourglass

# Download Go modules
go mod download

# Verify dependencies
go mod tidy
```

### 3. Build (Optional)

```bash
# Build to bin/hourglass
make build

# Or run directly (no binary)
go run ./cmd/server
```

### 4. Run Server

```bash
# Option A: Direct execution
go run ./cmd/server

# Option B: Using built binary
./bin/hourglass

# Output should show:
# Listening on :8080
```

**Verify it's running:**
```bash
curl http://localhost:8080/health
# Should return: {"status":"ok"}
```

---

## Frontend Setup

### 1. Install Dependencies

```bash
cd /Users/stefanoprivitera/Projects/hourglass/web

npm install
# or
bun install  # If using Bun instead of npm
```

### 2. Environment Variables

Create `web/.env.local` (or use defaults):

```bash
# API base URL (optional, defaults to http://localhost:8080)
VITE_API_URL=http://localhost:8080
```

Vite dev server proxies `/api/*` requests to backend automatically.

### 3. Start Dev Server

```bash
cd web

npm run dev
# or
bun run dev
```

**Output:**
```
  Local:    http://localhost:5173/
  Press q + enter to quit
```

**Access app:**
- Frontend: http://localhost:5173
- Backend API: http://localhost:8080

### 4. Build for Production

```bash
npm run build

# Output: web/dist/
# Ready to deploy or serve via `npm run preview`
```

---

## Testing

### Backend Tests

```bash
cd /Users/stefanoprivitera/Projects/hourglass

# Run all tests
make test

# Or directly:
go test ./...

# Run specific test:
go test ./internal/handlers -v

# Run with coverage:
go test -cover ./...
```

### Frontend Tests

```bash
cd web

# Run tests (if configured)
npm run test

# Or run linter
npm run lint
```

---

## Development Workflow

### Terminal 1: Database

```bash
cd /Users/stefanoprivitera/Projects/hourglass
docker-compose up
```

Keep this running throughout development.

### Terminal 2: Backend

```bash
cd /Users/stefanoprivitera/Projects/hourglass
go run ./cmd/server

# Auto-reload with entr or similar:
# go install github.com/cosmtrek/air@latest
# air
```

### Terminal 3: Frontend

```bash
cd /Users/stefanoprivitera/Projects/hourglass/web
npm run dev
```

### Terminal 4: Working Directory

```bash
cd /Users/stefanoprivitera/Projects/hourglass
# Edit code, run git commands, etc.
```

---

## Common Tasks

### Add a New API Endpoint

1. **Backend:**
   - Add handler method to `internal/handlers/*.go`
   - Register in `cmd/server/main.go`
   - Test with `curl` or Postman

2. **Frontend:**
   - Add query/mutation in `web/src/api/*.ts`
   - Create route or component in `web/src/routes/`
   - Fetch data with React Query

See [[04-Backend-Patterns]] and [[07-Frontend-Architecture]] for patterns.

### Run Database Migration

Currently manual (cmd/migrate not in snapshot):

```bash
# To apply a new migration:
psql -U hourglass -d hourglass -f migrations/00X_new_feature.up.sql

# To rollback:
psql -U hourglass -d hourglass -f migrations/00X_new_feature.down.sql
```

See [[16-Database-Migrations]] for migration structure.

### Debug Backend

Use VS Code or your IDE with Go extension:

```bash
# Launch configuration in .vscode/launch.json:
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Go: Attach",
      "type": "go",
      "mode": "debug",
      "request": "attach",
      "processId": "${command:pickGoProcess}"
    }
  ]
}
```

Or use `delve` debugger:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug ./cmd/server
```

### Debug Frontend

Use Chrome DevTools:
- Open DevTools: F12 or Cmd+Option+I
- Check React Dev Tools extension
- Use `debugger` statements or breakpoints

---

## Troubleshooting

### Backend: Database Connection Refused

**Error:** `connect: connection refused`

**Solutions:**
1. Ensure Docker is running: `docker-compose up -d`
2. Check PostgreSQL is accessible: `psql -U hourglass -d hourglass`
3. Verify DATABASE_URL is correct
4. Check logs: `docker-compose logs postgres`

### Frontend: CORS Issues

**Error:** `No 'Access-Control-Allow-Origin' header`

**Solutions:**
1. Ensure backend is running on :8080
2. Verify VITE_API_URL is set correctly
3. Check that requests are to `/api/*` (proxied)
4. In production, backend must handle CORS

### Port Already in Use

```bash
# Find process using port
lsof -i :8080  # Backend
lsof -i :5173  # Frontend

# Kill process (get PID from above)
kill -9 <PID>
```

### Node Modules Issues

```bash
# Clear and reinstall
rm -rf node_modules package-lock.json
npm install
```

---

## Docker Commands

```bash
# Start services
docker-compose up -d

# Stop services
docker-compose down

# View logs
docker-compose logs -f postgres

# Connect to database container
docker-compose exec postgres psql -U hourglass -d hourglass

# Rebuild images
docker-compose build --no-cache
```

---

## IDE Setup

### VS Code (Recommended)

**Extensions:**
- Go (golang.go)
- TypeScript Vue Plugin (Vue.volar)
- Tailwind CSS IntelliSense
- Prettier

**Workspace Settings** (`.vscode/settings.json`):
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "[typescript]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode",
    "editor.formatOnSave": true
  },
  "tailwindCSS.experimental.classRegex": [
    ["cva\\(([^)]*)\\)", "(?:'|\"|`)([^']*)(?:'|\"|`)"]
  ]
}
```

### GoLand / IntelliJ

- Open project folder
- Run → Edit Configurations → Add Go configuration
- Set working directory to `/Users/stefanoprivitera/Projects/hourglass`
- Set run kind to `Directory` with `./cmd/server`

---

## Next Steps

1. **Follow the tutorial**: Create a simple time entry
   - POST to `/time-entries` from backend
   - Fetch and display in frontend
   
2. **Explore the codebase**: Start with [[02-Architecture]]

3. **Write a test**: See [[17-Testing]] for patterns

---

**Next**: [[16-Database-Migrations]] for schema understanding.
