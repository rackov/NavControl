FROM golang:1.24.6-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o rest-api ./services/rest-api/cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

# Установим рабочую директорию в /app как в builder stage
WORKDIR /app

# Создаем директорию для логов, которая будет точкой монтирования
RUN mkdir -p /logs

COPY --from=builder /app/rest-api .

# Удаление: COPY --from=builder /app/NavControl/cfg ./NavControl/cfg

EXPOSE 8082
EXPOSE 9990

# Указываем точку монтирования для логов
VOLUME ["/logs"]

CMD ["./rest-api", "-config", "NavControl/cfg/restapi.toml"]