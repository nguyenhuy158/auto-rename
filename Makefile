IMAGE_NAME=auto-rename
REGISTRY=nguyenhuy158
IMAGE_TAG=latest
FULL_IMAGE=$(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: build build-fast build-no-cache build-advanced run push clean help

help:
	@echo "Available targets:"
	@echo "  make build              - Build image with cache (optimized)"
	@echo "  make build-fast         - Build image using build.sh script"
	@echo "  make build-advanced     - Build with advanced options (--clean, --push)"
	@echo "  make build-no-cache     - Force rebuild without cache"
	@echo "  make run                - Stop and start containers with docker-compose"
	@echo "  make push               - Push image to registry"
	@echo "  make clean              - Remove dangling images"

build:
	docker build -t $(IMAGE_NAME) -t $(FULL_IMAGE) .

build-fast:
	./build.sh $(IMAGE_NAME) $(IMAGE_TAG)

build-advanced:
	./build-advanced.sh --name $(IMAGE_NAME) --tag $(IMAGE_TAG) --registry $(REGISTRY) --clean

build-no-cache:
	docker build --no-cache -t $(IMAGE_NAME) -t $(FULL_IMAGE) .

run:
	docker compose down ; docker compose up -d

push:
	docker push $(FULL_IMAGE)

clean:
	docker image prune -f --filter "dangling=true"