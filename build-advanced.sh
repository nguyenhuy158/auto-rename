#!/usr/bin/env bash

# Advanced build script with multiple options
# Usage: ./build-advanced.sh [OPTIONS]
#
# Options:
#   -n, --name NAME           Image name (default: auto-rename)
#   -t, --tag TAG             Image tag (default: latest)
#   -c, --context PATH        Build context (default: current dir)
#   --no-cache                Build without using cache
#   --push                    Push to registry after build
#   --registry URL            Registry URL for push
#   --clean                   Remove dangling images after build
#   -h, --help                Show this help message

set -e

# Defaults
IMAGE_NAME="auto-rename"
IMAGE_TAG="latest"
BUILD_CONTEXT="."
USE_CACHE=true
PUSH_IMAGE=false
REGISTRY=""
CLEAN_AFTER=false

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Help function
show_help() {
  grep "^# " "$0" | tail -n +2
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -n|--name)
      IMAGE_NAME="$2"
      shift 2
      ;;
    -t|--tag)
      IMAGE_TAG="$2"
      shift 2
      ;;
    -c|--context)
      BUILD_CONTEXT="$2"
      shift 2
      ;;
    --no-cache)
      USE_CACHE=false
      shift
      ;;
    --push)
      PUSH_IMAGE=true
      shift
      ;;
    --registry)
      REGISTRY="$2"
      shift 2
      ;;
    --clean)
      CLEAN_AFTER=true
      shift
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    *)
      echo -e "${RED}Unknown option: $1${NC}"
      show_help
      exit 1
      ;;
  esac
done

# Setup image name
FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"
if [ -n "$REGISTRY" ]; then
  FULL_IMAGE="${REGISTRY}/${FULL_IMAGE}"
fi

# Header
echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║      Docker Image Build Script         ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Configuration:${NC}"
echo "  Image:           ${YELLOW}${FULL_IMAGE}${NC}"
echo "  Build context:   ${YELLOW}${BUILD_CONTEXT}${NC}"
echo "  Use cache:       ${YELLOW}$([ "$USE_CACHE" = true ] && echo "Yes" || echo "No")${NC}"
echo "  Push after:      ${YELLOW}$([ "$PUSH_IMAGE" = true ] && echo "Yes" || echo "No")${NC}"
echo ""

# Verify build context exists
if [ ! -d "$BUILD_CONTEXT" ]; then
  echo -e "${RED}Error: Build context '$BUILD_CONTEXT' not found${NC}"
  exit 1
fi

# Start timing
START_TIME=$(date +%s)

# Build command
BUILD_ARGS=("--tag" "${FULL_IMAGE}")

if [ "$USE_CACHE" = true ]; then
  BUILD_ARGS+=("--cache-from" "${FULL_IMAGE}")
fi

if [ "$USE_CACHE" = false ]; then
  BUILD_ARGS+=("--no-cache")
fi

BUILD_ARGS+=("--progress=plain" "${BUILD_CONTEXT}")

# Run build
echo -e "${GREEN}Starting build...${NC}"
echo ""
docker build "${BUILD_ARGS[@]}"

# Check result
if [ $? -eq 0 ]; then
  END_TIME=$(date +%s)
  DURATION=$((END_TIME - START_TIME))
  
  echo ""
  echo -e "${GREEN}════════════════════════════════════════${NC}"
  echo -e "${GREEN}✓ Build completed successfully${NC}"
  echo -e "${GREEN}════════════════════════════════════════${NC}"
  echo ""
  
  # Show image info
  echo -e "${GREEN}Image Information:${NC}"
  docker image inspect "${FULL_IMAGE}" \
    --format="  Size: {{.Size | printf \"%.1f MB\"}} ({{.Size}} bytes)
  Created: {{.Created}}
  Architecture: {{.Architecture}}/{{.Os}}"
  echo ""
  
  # Show layers
  echo -e "${GREEN}Image Layers:${NC}"
  docker history "${FULL_IMAGE}" --no-trunc --human | head -8
  echo ""
  
  # Build time
  echo -e "${GREEN}Build Time: ${YELLOW}${DURATION}s${NC}"
  echo ""
  
  # Push if requested
  if [ "$PUSH_IMAGE" = true ]; then
    echo -e "${GREEN}Pushing image...${NC}"
    docker push "${FULL_IMAGE}"
    echo -e "${GREEN}✓ Push completed${NC}"
    echo ""
  fi
  
  # Clean up if requested
  if [ "$CLEAN_AFTER" = true ]; then
    echo -e "${GREEN}Cleaning up dangling images...${NC}"
    docker image prune -f --filter "dangling=true" > /dev/null 2>&1 || true
    echo -e "${GREEN}✓ Cleanup completed${NC}"
    echo ""
  fi
  
  # Quick start
  echo -e "${GREEN}Quick Start:${NC}"
  echo "  docker run -p 8080:8080 ${FULL_IMAGE}"
  echo ""
  echo -e "${GREEN}Compose:${NC}"
  echo "  docker-compose up -d"
  echo ""
else
  echo ""
  echo -e "${RED}Build failed${NC}"
  exit 1
fi
