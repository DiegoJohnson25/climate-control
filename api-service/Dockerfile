FROM golang:1.25-alpine

WORKDIR /app
COPY shared/ ./shared/
COPY api-service/ ./api-service/
WORKDIR /app/api-service
RUN go build -o bin/api-service ./cmd/main.go
CMD ["./bin/api-service"]