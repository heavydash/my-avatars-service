# Dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Копия зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Собираем два бинарника
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /worker ./cmd/worker

# Финальный минимальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /server /server
COPY --from=builder /worker /worker

# Порты
EXPOSE 8085

# По умолчанию запускаем server
CMD ["./server"]