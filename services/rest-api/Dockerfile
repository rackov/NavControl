FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o rest-api ./services/rest-api/cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

RUN mkdir -p /log

COPY --from=builder /app/rest-api .
COPY --from=builder /app/NavControl/cfg ./NavControl/cfg

EXPOSE 8081
EXPOSE 9091

CMD ["./rest-api", "-config", "NavControl/cfg/restapi.toml"]