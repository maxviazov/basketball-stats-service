# syntax=docker/dockerfile:1
FROM golang:1.25 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o basketball-stats ./cmd/server

FROM gcr.io/distroless/base
COPY --from=builder /app/basketball-stats /basketball-stats
ENTRYPOINT ["/basketball-stats"]
