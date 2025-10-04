IMAGE_NAME=auto-rename

build:
	docker build -t $(IMAGE_NAME) .

run:
	docker compose down ; docker compose up -d