# Quick Start Guide

## Local Development

1. **Clone and setup**:
   ```bash
   git clone <your-repo-url>
   cd auto-rename
   go mod tidy
   ```

2. **Build**:
    ```bash
    go build -o auto-rename ./cmd/auto-rename
    ```

3. **Run locally**:
    ```bash
    # Create test files
    mkdir test-files
    echo "test content" > test-files/document.txt
    echo "more content" > test-files/image.jpg
    
    # Preview changes (dry run)
    ./auto-rename -dir=./test-files -dry-run
    
    # Actually rename files with web interface
    ./auto-rename -dir=./test-files -web-port=8080
    
    # Rename files with cron scanning (every 30 minutes)
    ./auto-rename -dir=./test-files -cron -cron-expr="*/30 * * * *" -web-port=8080
    ```

## Docker Usage

1. **Build image**:
   ```bash
   docker build -t auto-rename:latest .
   ```

2. **Run with mounted directory**:
   ```bash
   # Preview mode
   docker run -v $(pwd)/test-files:/app/files auto-rename:latest -dir=/app/files -dry-run
   
   # Rename files
   docker run -v $(pwd)/test-files:/app/files auto-rename:latest -dir=/app/files
   ```

3. **Using Docker Compose** (Recommended):
    ```bash
    # Start with cron mode (scans every 30 minutes)
    docker compose up -d
    
    # View logs
    docker compose logs -f auto-rename
    
    # Access web interface at http://localhost:8768
    ```
    
    To customize cron expression, edit `docker-compose.yml`:
    ```yaml
    environment:
      - CRON_EXPR=*/30 * * * *  # Change to desired schedule
    command: ["-dir=/app/data", "-cron", "-cron-expr=*/30 * * * *", "-web-port=8080", "-db=/app/data/renames.db"]
    ```

## GitHub Copilot Integration

The project includes `.copilot-instructions.md` which provides context to GitHub Copilot about:

- File renaming patterns using UUIDs
- Error handling best practices
- Docker containerization approach
- Go coding conventions
- Testing strategies

When working on this project with GitHub Copilot, it will understand the project's architecture and suggest appropriate code patterns.

## Common Cron Expressions

- `*/1 * * * *` - Every minute (default)
- `*/5 * * * *` - Every 5 minutes
- `*/30 * * * *` - Every 30 minutes
- `0 * * * *` - Every hour
- `0 0 * * *` - Daily at midnight
- `0 */6 * * *` - Every 6 hours
- `0 9-17 * * 1-5` - Weekdays 9 AM to 5 PM

## Example Output

```
$ ./auto-rename -dir=./test-files -dry-run
Scanning directory: ./test-files
DRY RUN MODE - No files will be renamed
  document.txt -> 550e8400-e29b-41d4-a716-446655440000.txt
  image.jpg -> 6ba7b810-9dad-11d1-80b4-00c04fd430c8.jpg

Would rename 2 files
```

## Cron Mode Example

```
$ ./auto-rename -dir=./test-files -cron -cron-expr="*/30 * * * *" -web-port=8080
[cron] StartCronScanner initialized for dir=./test-files with cron expression=*/30 * * * *
[cron] cron scheduler started
Starting web server on port 8080...
View results at: http://localhost:8080
```

The scanner will automatically run every 30 minutes and:
- Scan the directory for new files
- Rename files not yet processed
- Log operations to the database
- Skip files already renamed (UUID format)

## Troubleshooting

### Permission Issues
```bash
# Make sure you have write permissions
chmod 755 /path/to/your/files
```

### Docker Volume Issues
```bash
# Use absolute paths for volume mounting
docker run -v /absolute/path/to/files:/app/files auto-rename:latest
```

### Module Issues
```bash
# Reset Go modules
go clean -modcache
go mod tidy
```
