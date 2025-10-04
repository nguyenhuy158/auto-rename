# Quick Start Guide

## Local Development

1. **Clone and setup**:
   ```bash
   git clone <your-repo-url>
   cd auto-rename
   go mod tidy
   ```

2. **Build and test**:
   ```bash
   go build -o auto-rename main.go
   go test -v
   ```

3. **Run locally**:
   ```bash
   # Create test files
   mkdir test-files
   echo "test content" > test-files/document.txt
   echo "more content" > test-files/image.jpg
   
   # Preview changes (dry run)
   ./auto-rename -dir=./test-files -dry-run
   
   # Actually rename files
   ./auto-rename -dir=./test-files
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

3. **Using Docker Compose**:
   ```bash
   # Edit docker-compose.yml to set your directory path
   # Then run:
   docker-compose up
   ```

## GitHub Copilot Integration

The project includes `.copilot-instructions.md` which provides context to GitHub Copilot about:

- File renaming patterns using UUIDs
- Error handling best practices
- Docker containerization approach
- Go coding conventions
- Testing strategies

When working on this project with GitHub Copilot, it will understand the project's architecture and suggest appropriate code patterns.

## Example Output

```
$ ./auto-rename -dir=./test-files -dry-run
Scanning directory: ./test-files
DRY RUN MODE - No files will be renamed
  document.txt -> 550e8400-e29b-41d4-a716-446655440000.txt
  image.jpg -> 6ba7b810-9dad-11d1-80b4-00c04fd430c8.jpg

Would rename 2 files
```

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
