#!/usr/bin/env bash

# Build script for auto-rename Docker image
# Optimized for faster builds with Docker layer caching

set -e

# Configuration
IMAGE_NAME="${1:-auto-rename}"
IMAGE_TAG="${2:-latest}"
FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"
BUILD_CONTEXT="${3:-.}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print header
echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}Building Docker Image${NC}"
echo -e "${GREEN}================================${NC}"
echo -e "Image: ${YELLOW}${FULL_IMAGE}${NC}"
echo -e "Build context: ${YELLOW}${BUILD_CONTEXT}${NC}"
echo ""

# Build with optimizations
echo -e "${GREEN}Running docker build...${NC}"
docker build \
  --tag "${FULL_IMAGE}" \
  --cache-from "${FULL_IMAGE}" \
  --build-arg BUILDKIT_INLINE_CACHE=1 \
  --progress=plain \
  "${BUILD_CONTEXT}"

# Check build result
if [ $? -eq 0 ]; then
  echo ""
  echo -e "${GREEN}================================${NC}"
  echo -e "${GREEN}Build completed successfully${NC}"
  echo -e "${GREEN}================================${NC}"
  echo ""
  
  # Show image info
  echo -e "${GREEN}Image Information:${NC}"
  docker image ls "${IMAGE_NAME}" | tail -1
  
  echo ""
  echo -e "${GREEN}Quick start:${NC}"
  echo "  docker run -p 8080:8080 ${FULL_IMAGE}"
  echo ""
else
  echo ""
  echo -e "${RED}Build failed${NC}"
  exit 1
fi
