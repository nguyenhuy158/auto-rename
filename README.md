# Auto-Rename - File UUID Renamer with Web Interface

A Go application that renames files to UUID-based names with SQLite database logging and web interface for monitoring.

## Overview

This project provides a comprehensive file renaming solution with:
- 🔄 Renames files using UUID while preserving extensions
- 📊 SQLite database to track all rename operations
- 🌐 Web interface to view rename history and statistics
- 🐳 Docker container support for easy deployment
- 🧪 Dry-run mode for testing without actual changes

## Features

### Core Functionality
- **File Renaming**: Converts filenames to UUID format (e.g., `document.pdf` → `550e8400-e29b-41d4-a716-446655440000.pdf`)
- **Database Logging**: Records all operations with file metadata, timestamps, and success/failure status
- **Web Dashboard**: Real-time interface to view rename history, search records, and view statistics
- **Dry Run Mode**: Preview changes without actually renaming files

### Web Interface
- 📋 **Dashboard**: Overview with statistics and quick navigation
- 📊 **Records View**: Detailed table of all rename operations with search functionality
- 🔍 **Search**: Find records by original filename
- 📈 **Statistics**: Total operations, success rate, recent activity
- 🎨 **Responsive Design**: Clean, modern interface that works on desktop and mobile

## Project Structure

```
auto-rename/
├── main.go              # Main application logic
├── database.go          # SQLite database operations
├── webserver.go         # Web interface and API endpoints
├── main_test.go         # Unit tests for main functionality
├── database_test.go     # Tests for database operations
├── go.mod              # Go module definition
├── go.sum              # Go dependencies checksums
├── Dockerfile          # Multi-stage Docker build
├── docker-compose.yml   # Docker Compose with web interface
└── README.md           # This documentation
```

## Prerequisites

- Go 1.21 or higher
- Docker (for containerized deployment)
- Docker Compose (optional)

## Quick Start

### 1. Clone and Build
```bash
git clone <repository-url>
cd auto-rename
go mod download
go build -o auto-rename .
```

### 2. Start Web Interface Only
```bash
./auto-rename -web-only -db=./renames.db -web-port=8080
```
Then visit: http://localhost:8080

### 3. Rename Files with Web Interface
```bash
./auto-rename -dir=/path/to/files -db=./renames.db -web-port=8080
```

### 4. Dry Run Mode
```bash
./auto-rename -dir=/path/to/files -dry-run -db=./renames.db -web-port=8080
```

## Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-dir` | Directory containing files to rename | Required (unless -web-only) |
| `-dry-run` | Preview changes without renaming | `false` |
| `-web-port` | Port for web interface | `8080` |
| `-web-only` | Start web server without renaming | `false` |
| `-db` | SQLite database file path | `./file_renames.db` |

## Docker Deployment

### Method 1: Docker Compose (Recommended)

1. **Create data directory**:
```bash
mkdir -p ./data
```

2. **Edit docker-compose.yml** to set your file path:
```yaml
volumes:
  - /your/actual/path:/app/files  # Change this line
  - ./data:/app/data
```

3. **Start the service**:
```bash
docker-compose up -d
```

4. **Access web interface**: http://localhost:8080

### Method 2: Docker Run

```bash
# Web interface only
docker run -d -p 8080:8080 -v ./data:/app/data auto-rename:latest

# Rename files and start web interface
docker run -d -p 8080:8080 \
  -v /path/to/your/files:/app/files \
  -v ./data:/app/data \
  auto-rename:latest \
  -dir=/app/files -db=/app/data/renames.db -web-port=8080
```

## API Endpoints

The web server provides these REST API endpoints:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Web dashboard |
| `/records` | GET | Records view page |
| `/api/records` | GET | JSON list of all records |
| `/api/records/search?q=filename` | GET | Search records by filename |
| `/api/stats` | GET | Statistics (total, success, failed, recent) |

### API Example Usage

```bash
# Get all records
curl http://localhost:8080/api/records

# Search for files containing "document"
curl http://localhost:8080/api/records/search?q=document

# Get statistics
curl http://localhost:8080/api/stats
```

## GitHub Copilot Instructions

When working with this project, GitHub Copilot should understand:

1. **File Renaming Logic**: Generate UUID-based filenames while preserving extensions
2. **Error Handling**: Implement robust error handling for file operations
3. **Docker Best Practices**: Use multi-stage builds and minimal base images
4. **Go Conventions**: Follow Go naming conventions and project structure
5. **CLI Interface**: Use flag package for command-line argument parsing

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License
