# Build stage
FROM golang:1.23-alpine AS build

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY cmd/*.go ./cmd/
COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o simba ./cmd

# Production stage: distroless
FROM gcr.io/distroless/base-debian12

# Optional: non-root user (UID 10001 is standard for distroless)
USER 10001:0

WORKDIR /app

COPY --from=build /app/simba /app/simba
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/app/simba"]
