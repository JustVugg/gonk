# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/bin/gonk /usr/local/bin/gonk

EXPOSE 8080

ENTRYPOINT ["gonk"]
CMD ["-config", "/etc/gonk/gonk.yaml"]
