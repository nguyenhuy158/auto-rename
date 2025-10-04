IMAGE_NAME=auto-rename

build:
# 	docker build --no-cache -t $(IMAGE_NAME) .
	docker build -t $(IMAGE_NAME) -t nguyenhuy158/auto-rename:latest .

run:
	docker compose down ; docker compose up -d

push:
	docker push nguyenhuy158/auto-rename:latest