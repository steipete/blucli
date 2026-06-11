# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.23

FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/blu ./cmd/blu

FROM alpine:${ALPINE_VERSION}
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 -h /data blu
ENV HOME=/data \
    XDG_CONFIG_HOME=/data/config \
    XDG_CACHE_HOME=/data/cache
VOLUME ["/data"]
WORKDIR /data
COPY --from=build /out/blu /usr/local/bin/blu
USER blu
ENTRYPOINT ["blu"]
CMD ["--help"]
