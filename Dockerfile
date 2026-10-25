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
RUN mkdir -p /app/data /app/quarantine
EXPOSE 8080
ENTRYPOINT ["./auto-rename"]
