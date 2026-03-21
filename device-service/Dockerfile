FROM golang:1.25-alpine

WORKDIR /app
COPY shared/ ./shared/
COPY device-service/ ./device-service/
WORKDIR /app/device-service
RUN go build -o bin/device-service ./cmd/main.go
CMD ["./bin/device-service"]