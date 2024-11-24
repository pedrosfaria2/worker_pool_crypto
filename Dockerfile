FROM golang:1.23.3-alpine AS builder

WORKDIR /build

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o worker cmd/worker/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /build/worker .

RUN chmod +x worker

EXPOSE 8080

CMD ["./worker"]
