FROM golang:1.23-alpine AS build_base

RUN apk add git

# Set the Current Working Directory inside the container
WORKDIR /tmp/go-simba-app

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY cmd/*.go /tmp/go-simba-app/cmd/
COPY *.go /tmp/go-simba-app/

RUN go build -o ./out/app cmd/*.go

# Start fresh from a smaller image
FROM alpine:3

RUN apk add ca-certificates

COPY --from=build_base /tmp/go-simba-app/out/app /app/app

# Run the binary program produced by `go install`
ENTRYPOINT ["/app/app"]

