# Auto-Rename - File UUID Renamer with Web Interface

A Go application that renames files to UUID-based names with SQLite database logging and web interface for monitoring.

## Overview

This project provides a comprehensive file renaming solution with:
- 🔄 Renames files using UUID while preserving extensions
- 📊 SQLite database to track all rename operations
- 🌐 Web interface to view rename history and statistics
- ⏰ Cron-based scheduling with customizable expressions
- 🐳 Docker container support for easy deployment
- 🧪 Dry-run mode for testing without actual changes

## Features

### Core Functionality
- **File Renaming**: Converts filenames to UUID format (e.g., `document.pdf` → `550e8400-e29b-41d4-a716-446655440000.pdf`)
- **Database Logging**: Records all operations with file metadata, timestamps, and success/failure status
- **Web Dashboard**: Real-time interface to view rename history, search records, and view statistics
- **Dry Run Mode**: Preview changes without actually renaming files
- **Cron Scheduling**: Automatically scan and rename files on a schedule using cron expressions
  - Every minute, hourly, daily, or custom intervals
  - Only processes files not yet renamed
  - Skips files already in UUID format

---

### Styling Instruction

This project uses [Tailwind CSS](https://tailwindcss.com/) for all styling. **Do not use regular CSS files or custom CSS.** All UI components and pages must be styled using Tailwind utility classes only.

---

### Web Interface
- 📋 **Dashboard**: Overview with statistics and quick navigation
- 📊 **Records View**: Detailed table of all rename operations with search functionality
- 🔍 **Search**: Find records by original filename
- 📈 **Statistics**: Total operations, success rate, recent activity
- 🎨 **Responsive Design**: Clean, modern interface that works on desktop and mobile

## Project Structure

```
auto-rename/
├── cmd/
│   └── auto-rename/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration parsing
│   ├── delivery/
│   │   └── webserver.go         # Web interface and API endpoints
│   ├── domain/
│   │   └── file.go              # Domain entities
│   ├── infrastructure/
│   │   ├── database.go          # SQLite database operations
│   │   └── filesystem.go        # File system helpers
│   └── usecase/
│       └── rename.go            # Business logic for file renaming
├── template/                     # HTML templates
├── static/                       # Static assets (CSS, JS)
├── go.mod                        # Go module definition
├── go.sum                        # Go dependencies checksums
├── Dockerfile                    # Multi-stage Docker build
├── docker-compose.yml            # Docker Compose configuration
├── .env.example                  # Example environment configuration
├── QUICK_START.md               # Quick start guide
└── README.md                     # This documentation
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

### 2. Configure with .env File (Recommended)
```bash
# Copy the example .env file
cp .env.example .env

# Edit .env and set your configuration
# For example:
# DIR=/path/to/files
# WEB_PORT=8080
# DRY_RUN=false
```

Then simply run:
```bash
./auto-rename
```

### 3. Or Use Command Line Options

**Start Web Interface Only**:
```bash
./auto-rename -web-only -db=./renames.db -web-port=8080
```
Then visit: http://localhost:8080

**Rename Files with Web Interface**:
```bash
./auto-rename -dir=/path/to/files -db=./renames.db -web-port=8080
```

**Dry Run Mode**:
```bash
./auto-rename -dir=/path/to/files -dry-run -db=./renames.db -web-port=8080
```

**Continuous Monitoring (Cron Mode)**:
Run a scan using cron expressions to catch new files dropped into the directory:
```bash
./auto-rename -dir=/path/to/watch -db=./renames.db -web-port=8080 -cron -cron-expr="*/30 * * * * *"
```
The application will:
- Perform an initial rename pass
- Run scans based on the cron expression schedule
- Only rename files that have not been processed yet
- Skip files whose names already look like UUIDs

**Cron Expression Examples**:
- `*/1 * * * *` - Every minute (default)
- `*/5 * * * *` - Every 5 minutes
- `*/30 * * * *` - Every 30 minutes
- `0 * * * *` - Every hour
- `0 0 * * *` - Daily at midnight
- `0 */6 * * *` - Every 6 hours
- `0 9-17 * * 1-5` - Weekdays 9 AM to 5 PM

Use `-cron-expr="EXPRESSION"` flag or set `CRON_EXPR` environment variable.

## Configuration

The application supports three ways to configure settings (in order of precedence):

1. **Command-line flags** (highest priority)
2. **Environment variables**
3. **.env file** (lowest priority)

### Environment Variables

You can set these environment variables directly or use a `.env` file:

| Variable | Description | Default |
|----------|-------------|---------|
| `DIR` | Directory containing files to rename | (none) |
| `DRY_RUN` | Preview changes without renaming (`true`/`false`) | `false` |
| `WEB_PORT` | Port for web interface | `8080` |
| `WEB_ONLY` | Start web server without renaming (`true`/`false`) | `false` |
| `DB_PATH` | SQLite database file path | `./file_renames.db` |
| `CRON` | Continuously scan directory (`true`/`false`) | `false` |
| `CRON_EXPR` | Cron expression for scan schedule | `*/1 * * * *` (every minute) |

### Using .env File

Create a `.env` file in the project root:

```bash
# Example .env configuration
DIR=/Users/username/Documents/files
DRY_RUN=false
WEB_PORT=8080
DB_PATH=./renames.db
CRON=true
```

Then run the application without any flags:
```bash
./auto-rename
```

### Configuration Priority Example

```bash
# .env file has: DIR=/default/path
# Environment variable: export DIR=/env/path
# Command line: ./auto-rename -dir=/flag/path

# Result: Uses /flag/path (command-line flags have highest priority)
```

## Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-dir` | Directory containing files to rename | Required (unless -web-only) |
| `-dry-run` | Preview changes without renaming | `false` |
| `-web-port` | Port for web interface | `8080` |
| `-web-only` | Start web server without renaming | `false` |
| `-db` | SQLite database file path | `./file_renames.db` |
| `-cron` | Continuously rescan directory | `false` |
| `-cron-expr` | Cron expression for scan schedule | `*/1 * * * *` (every minute) |

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
