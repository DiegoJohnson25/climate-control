FROM golang:1.25-alpine

WORKDIR /app
COPY simulator-service/ ./simulator-service/
WORKDIR /app/simulator-service
RUN go build -o bin/simulator-service ./cmd/main.go
CMD ["./bin/simulator-service"]