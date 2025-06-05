FROM golang:1.23-alpine AS build_base

RUN apk add build-base ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /tmp/go-simba-app

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY cmd/*.go /tmp/go-simba-app/cmd/
COPY *.go /tmp/go-simba-app/

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-linkmode external -extldflags -static" -o ./out/app cmd/*.go

# The scratch base image welcomes us as a blank canvas for our prod stage.
FROM scratch

WORKDIR /

# We copy the passwd file, essential for our non-root user 
COPY --from=build_base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build_base /tmp/go-simba-app/out/app /app/app

# Run the binary program produced by `go install`
ENTRYPOINT ["/app/app"]

