# Build stage
FROM golang:1.23-alpine AS build

RUN apk add --no-cache build-base

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY cmd/*.go ./cmd/
COPY *.go ./

RUN CGO_ENABLED=1 GOOS=linux go build  -ldflags="-linkmode external -extldflags -static" -o simba ./cmd

# Production stage: distroless
FROM ghcr.io/distroless/static


# Optional: non-root user (UID 10001 is standard for distroless)
USER 10001:0

WORKDIR /app

COPY --from=build /app/simba /app/simba
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/app/simba"]
