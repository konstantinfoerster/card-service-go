##### BUILDER #####

FROM golang:1.22-alpine3.19 as builder

## Task: copy source files
COPY . /app
WORKDIR /app

## Task: fetch project deps
RUN go mod download && go mod verify

## Task: build project
ENV GOOS="linux"
ENV GOARCH="amd64"
ENV CGO_ENABLED="0"

RUN go build -ldflags="-s -w" -o card-service cmd/main.go && chmod 0755 /app/card-service

##### TARGET #####

FROM alpine:3.19

ARG RELEASE
ENV IMG_VERSION="${RELEASE}"

COPY --from=builder /app/card-service /usr/bin/

# hadolint ignore=DL3018
RUN set -eux; \
    apk add --no-progress --quiet --no-cache --upgrade \
        tzdata

USER nobody

CMD ["/usr/bin/card-service", "--config", "/config/application.yaml"]

LABEL org.opencontainers.image.title="Card-Manager Service" \
      org.opencontainers.image.description="Application that helps you to manage your card collection" \
      org.opencontainers.image.version="$IMG_VERSION" \
      org.opencontainers.image.source="https://github.com/konstantinfoerster/card-service-go.git" \
      org.opencontainers.image.vendor="Konstantin Förster" \
      org.opencontainers.image.authors="Konstantin Förster"
