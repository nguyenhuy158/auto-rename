FROM node:22-alpine AS frontend
WORKDIR /app/web
RUN corepack enable
RUN corepack prepare pnpm@10.34.8 --activate
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY openapi.yaml /app/openapi.yaml
COPY web ./
RUN pnpm build

FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -o auto-rename ./cmd/auto-rename

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/auto-rename .
COPY template ./template
COPY static ./static
COPY openapi.yaml ./openapi.yaml
COPY --from=frontend /app/web/dist ./web/dist
RUN mkdir -p /app/data /app/quarantine
EXPOSE 8080
ENTRYPOINT ["./auto-rename"]
