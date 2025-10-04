# GitHub Copilot Instructions - Auto-Rename Project

## Project Overview
A Go CLI tool that renames files to UUID-based names while preserving extensions, with SQLite logging and an embedded web interface for monitoring operations. Single binary with three operational modes: rename-only, web-only, or combined.

## Architecture & Stack
- **Language**: Go 1.21+ (single-package design)
- **Database**: SQLite3 with CGO-enabled build
- **Web**: Gorilla Mux with embedded HTML templates
- **Dependencies**:
  - `github.com/google/uuid` for UUID generation
  - `github.com/gorilla/mux` for HTTP routing
  - `github.com/mattn/go-sqlite3` for database (requires CGO)

## Core Components

### 1. Main Application (`main.go`)
- **Config struct**: Centralized configuration from CLI flags
- **Three operational modes**:
  - Web-only: `./auto-rename -web-only -db=./renames.db`
  - Rename + Web: `./auto-rename -dir=/files -web-port=8080`
  - Dry-run: `./auto-rename -dir=/files -dry-run`
- **UUID generation**: `generateUUIDName()` preserves file extensions
- **File operations**: Metadata capture before renaming with error tracking

### 2. Database Layer (`database.go`)
- **FileRecord struct**: Complete file metadata with rename tracking
- **Auto-created indexes**: `original_name`, `new_name`, `renamed_at`
- **Key operations**:
  - `NewDatabase()`: Auto-creates schema and indexes
  - `InsertFileRecord()`: Logs both successful and failed operations
  - `GetRecordsByOriginalName()`: Search functionality
  - `GetStats()`: Dashboard statistics

### 3. Web Interface (`webserver.go`)
- **Embedded HTML templates**: No external static files required
- **JSON API endpoints**:
  - `GET /api/records` - All records
  - `GET /api/records/search?q=query` - Search by original filename
  - `GET /api/stats` - Statistics for dashboard
- **Responsive CSS**: Inline styles with mobile-first approach

## Key Development Workflows

### Building & Testing
```bash
# Build with CGO for SQLite
go build -o auto-rename .

# Run tests (uses temporary databases and directories)
go test -v

# Test specific functionality
go test -run TestDatabase -v
go test -run TestGenerateUUIDName -v
```

### Development Server
```bash
# Quick development with sample data
go run *.go -dir=./test-files -web-port=8080 -dry-run
```

### Docker Development
```bash
# Build and test container
docker build -t auto-rename .
docker run -p 8080:8080 -v $(pwd)/data:/app/data auto-rename

# Using compose for full setup
docker-compose up --build
```

## Project-Specific Patterns

### Single-Package Architecture
- All Go files in root package `main` - no subdirectories
- Shared types (`Config`, `FileRecord`, `Database`, `WebServer`) across files
- Clean separation: `main.go` (CLI), `database.go` (persistence), `webserver.go` (HTTP)

### Database Operations
- **Always use parameterized queries**: `db.Exec(query, args...)` prevents SQL injection
- **Error wrapping pattern**: `fmt.Errorf("context: %w", err)` for error chains
- **Automatic schema creation**: Database tables and indexes created on first run
- **Comprehensive logging**: Both successful and failed operations recorded with full metadata

### CLI Flag Patterns
```go
type Config struct {
    Dir     string  // Required unless WebOnly
    DryRun  bool    // Preview mode
    WebPort string  // Optional web interface
    WebOnly bool    // Database viewer mode
    DbPath  string  // SQLite file location
}
```

### Web Server Architecture
- **Embedded templates**: HTML directly in Go strings (no external files)
- **API-first design**: All data via JSON endpoints, HTML consumes same APIs
- **Error handling**: Consistent JSON error responses with HTTP status codes
- **No static file server**: All CSS/JS inline for single-binary deployment

### Testing Conventions
- **Temporary resources**: Use `defer os.Remove()` for test database cleanup
- **Isolation**: Each test creates its own database file
- **Real implementations**: Tests use actual SQLite, not mocks
- **UUID validation**: Parse generated UUIDs to verify format correctness

## Critical Requirements

### CGO Dependency
- **SQLite requires CGO**: Must build with `CGO_ENABLED=1`
- **Development**: Install gcc and sqlite-dev for local builds
- **Docker builds**: Use `golang:alpine` base with build tools
- **Cross-compilation**: CGO complicates cross-platform builds

### Docker Patterns
```dockerfile
# Multi-stage pattern for CGO builds
FROM golang:1.21-alpine AS builder
RUN apk add --no-cache gcc musl-dev sqlite-dev
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo

FROM alpine:latest
RUN apk --no-cache add sqlite  # Runtime SQLite needed
```

### Operational Modes
1. **Web-only**: `./auto-rename -web-only -db=./renames.db` (view existing data)
2. **Rename + Web**: `./auto-rename -dir=/files -web-port=8080` (rename then serve)
3. **Rename-only**: `./auto-rename -dir=/files -db=./renames.db` (batch operation)
4. **Dry-run**: `./auto-rename -dir=/files -dry-run` (preview changes)

## Common Development Patterns

### File Operations
```go
// Capture metadata before operations
info, err := os.Stat(oldPath)
record := FileRecord{
    OriginalName: filename,
    NewName:      generateUUIDName(filename),
    FilePath:     dir,
    FileSize:     info.Size(),
    FileMode:     info.Mode().String(),
    ModTime:      info.ModTime(),
    RenamedAt:    time.Now(),
    Success:      false,  // Update after operation
}
```

### Error Handling Strategy
- **Wrap all errors**: `fmt.Errorf("operation failed: %w", err)`
- **Record failures**: Log unsuccessful operations to database
- **Continue on errors**: Don't stop batch operations for single failures
- **Detailed context**: Include file paths and operation details

### Web Development
- **Template embedding**: HTML templates as Go string literals
- **No external assets**: All CSS/JS inline for portability
- **API consistency**: All endpoints return JSON with same error format
- **Search optimization**: Database indexes on searchable fields