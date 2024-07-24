FROM golang:1.22-alpine3.20 as builder

WORKDIR /app

# download go modules
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# copy source files
COPY . /app

# build project
ENV GOOS="linux"
ENV GOARCH="amd64"
ENV CGO_ENABLED="0"

RUN go build -ldflags="-s -w" -o service cmd/main.go \
      && chmod 0755 /app/service

FROM alpine:3.20 as dev

ARG RELEASE
ENV IMG_VERSION="${RELEASE}"

COPY --from=builder /app/service /usr/bin/

# hadolint ignore=DL3018
RUN set -eux; \
    apk add --no-progress --quiet --no-cache --upgrade \
        tzdata

USER nobody

CMD ["/usr/bin/service", "--config", "/config/application.yaml"]

LABEL org.opencontainers.image.title="Card-Manager Service" \
      org.opencontainers.image.description="Application that helps you to manage your card collection" \
      org.opencontainers.image.version="$IMG_VERSION" \
      org.opencontainers.image.source="https://github.com/konstantinfoerster/card-service-go.git" \
      org.opencontainers.image.vendor="Konstantin Förster" \
      org.opencontainers.image.authors="Konstantin Förster"
