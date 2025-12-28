# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o smtp-router .

# Run Stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/smtp-router .
COPY config.yaml .
# Copy the Starlark script if it's not embedded or if you want it to be editable
COPY round_robin.star .

EXPOSE 1025

CMD ["./smtp-router"]
