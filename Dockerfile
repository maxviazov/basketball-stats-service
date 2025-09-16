# syntax=docker/dockerfile:1
FROM golang:1.25 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

FROM gcr.io/distroless/base
WORKDIR /app
COPY --from=builder /app/server /app/server
# Ship runtime assets needed by the server
COPY --from=builder /app/config.yaml /app/config.yaml
COPY --from=builder /app/api /app/api
EXPOSE 8080
USER 10001
ENTRYPOINT ["/app/server"]
