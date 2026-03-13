FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Копируем папку с миграциями, чтобы они были доступны при запуске
COPY internal/migrations /app/internal/migrations
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /server .
# Копируем миграции из builder в финальный образ
COPY --from=builder /app/internal/migrations ./internal/migrations
EXPOSE 4200
CMD ["./server"]
