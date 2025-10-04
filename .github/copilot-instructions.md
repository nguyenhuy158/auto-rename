# GitHub Copilot Instructions - Auto-Rename Project

## Project Overview
This is a Go application that renames files to UUID-based names with comprehensive database logging and web interface monitoring. The project uses Docker for deployment and includes both command-line and web interfaces.

## Architecture & Stack
- **Language**: Go 1.21+
- **Database**: SQLite3 with go-sqlite3 driver
- **Web Framework**: Gorilla Mux router
- **Frontend**: Vanilla HTML/CSS/JavaScript (responsive design)
- **Container**: Docker with multi-stage builds
- **Dependencies**: 
  - `github.com/google/uuid` for UUID generation
  - `github.com/gorilla/mux` for web routing
  - `github.com/mattn/go-sqlite3` for database operations

## Core Components

### 1. Database Operations (`database.go`)
- **FileRecord struct**: Complete file metadata with rename tracking
- **Database struct**: SQLite connection and operations
- **Key functions**:
  - `NewDatabase()`: Initialize DB with schema creation
  - `InsertFileRecord()`: Log rename operations
  - `GetAllRecords()`: Retrieve all records with pagination support
  - `GetRecordsByOriginalName()`: Search functionality
  - `GetStats()`: Dashboard statistics

### 2. Web Interface (`webserver.go`)
- **RESTful API**: JSON endpoints for data access
- **Dashboard**: Statistics overview with real-time updates
- **Records View**: Searchable table with file details
- **Responsive Design**: Mobile-friendly interface
- **API Endpoints**:
  - `GET /api/records` - All records
  - `GET /api/records/search?q=query` - Search records
  - `GET /api/stats` - Statistics data

### 3. Main Application (`main.go`)
- **Config struct**: All configuration options
- **Command-line flags**: Flexible operation modes
- **File operations**: UUID renaming with metadata capture
- **Integration**: Database logging + web server coordination

## Docker Configuration

### Multi-stage Dockerfile
```dockerfile
# Build stage with CGO for SQLite
FROM golang:1.21-alpine AS builder
RUN apk add --no-cache gcc musl-dev sqlite-dev
# ... build with CGO_ENABLED=1

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add sqlite
# ... runtime setup
```

### Docker Compose
- **Web Interface**: Port 8080 exposed
- **Volume Mounts**: 
  - `/path/to/files:/app/files` (input files)
  - `./data:/app/data` (persistent database)
- **Flexible Commands**: Web-only, rename+web, dry-run modes

## Usage Patterns

### Command Line Modes
1. **Web-only**: `./auto-rename -web-only -db=./renames.db`
2. **Rename + Web**: `./auto-rename -dir=/files -web-port=8080`
3. **Dry Run**: `./auto-rename -dir=/files -dry-run`

### Database Schema
```sql
CREATE TABLE file_renames (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    original_name TEXT NOT NULL,
    new_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_size INTEGER,
    file_mode TEXT,
    mod_time DATETIME,
    renamed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    success BOOLEAN DEFAULT TRUE,
    error_msg TEXT
);
```

## Development Guidelines

### Code Style & Patterns
- **Error Handling**: Always wrap errors with context
- **Logging**: Use structured logging with operation details
- **Testing**: Unit tests for database operations and file handling
- **Validation**: Input validation for all user-provided data

### Database Best Practices
- **Transactions**: Use for multi-operation consistency
- **Indexing**: Indexes on searchable fields (original_name, renamed_at)
- **Error Recording**: Log both successful and failed operations
- **Metadata Capture**: Store complete file information before renaming

### Web Interface Standards
- **RESTful Design**: Consistent API patterns
- **JSON Responses**: Structured error messages and data formats
- **Progressive Enhancement**: Works without JavaScript for basic functionality
- **Responsive Design**: Mobile-first CSS approach

### Docker Best Practices
- **Multi-stage Builds**: Separate build and runtime environments
- **Security**: Non-root user, minimal base images
- **Persistence**: External volume mounts for data retention
- **Configuration**: Environment variables and command-line flexibility

## Testing Strategy

### Test Coverage Areas
1. **Unit Tests**: Database operations, file operations, UUID generation
2. **Integration Tests**: End-to-end rename operations with database
3. **API Tests**: Web endpoints with various input scenarios
4. **Docker Tests**: Container build and runtime verification

### Test Data Management
- **Temporary Directories**: Use `t.TempDir()` for isolated test environments
- **Database Cleanup**: Defer cleanup for test databases
- **Mock Data**: Realistic file structures and metadata

## Common Development Tasks

### Adding New Features
1. **Database Changes**: Update schema, add migrations if needed
2. **API Endpoints**: Follow RESTful patterns, add comprehensive tests
3. **Web Interface**: Update both HTML templates and JavaScript
4. **Docker Updates**: Modify build process if new dependencies added

### Performance Considerations
- **Database Queries**: Use appropriate indexes, limit result sets
- **File Operations**: Handle large directories efficiently
- **Memory Usage**: Stream file operations for large files
- **Web Assets**: Minimize JavaScript, inline critical CSS

### Security Considerations
- **SQL Injection**: Use parameterized queries exclusively
- **Path Traversal**: Validate all file paths
- **CORS**: Configure appropriately for API access
- **Input Validation**: Sanitize all user inputs

## Deployment Scenarios

### Development
```bash
go run *.go -dir=./test-files -web-port=8080
```

### Production
```bash
docker-compose up -d
# Access at http://localhost:8080
```

### CI/CD Integration
- **Build**: Multi-arch Docker images
- **Test**: Automated test suite with coverage
- **Deploy**: Container registry push and deployment

## Troubleshooting Guide

### Common Issues
1. **CGO Errors**: Ensure gcc and sqlite-dev installed
2. **Permission Issues**: Check file/directory permissions
3. **Database Locks**: Handle concurrent access properly
4. **Port Conflicts**: Configure alternative ports as needed

This project demonstrates modern Go development practices with comprehensive logging, web interfaces, and containerized deployment.