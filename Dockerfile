# Build stage
FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compile the app
RUN go build -o server ./main.go

# Runtime stage
FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/server .

RUN mkdir -p ./uploads

EXPOSE 8080

CMD ["./server"]
