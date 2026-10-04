FROM alpine:3.24 AS assets

# hadolint ignore=DL3018 # only used at build time
RUN apk add --no-cache brotli gzip

WORKDIR /assets

COPY internal/api/web/assets/ .

# pre-compress static files, original files are unchanged
RUN find . -type f \( -name '*.js' -o -name '*.css' \) \
      -exec gzip -k -9 {} + \
      -exec brotli -k -q 11 {} +

FROM golang:1.27-alpine3.24 AS builder

WORKDIR /app

# download go modules
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# copy source files
COPY . .
# add pre-compressed assets
COPY --from=assets /assets/ internal/api/web/assets/

# build project
ENV GOOS="linux" \
    GOARCH="amd64" \
    CGO_ENABLED="0"

RUN go build -trimpath -tags timetzdata -ldflags="-s -w" -o service cmd/main.go

FROM alpine:3.24 AS dev

ARG RELEASE

# hadolint ignore=DL3018 # we always want the latest certificates
RUN apk add --no-cache ca-certificates

RUN addgroup -g 10001 nonroot \
  && adduser -D -u 10001 -G nonroot nonroot

COPY --from=builder --chown=nonroot:nonroot /app/service /usr/bin/

USER 10001:10001

ENTRYPOINT ["/usr/bin/service"]
CMD ["--config", "/opts/app/application.yaml"]

LABEL org.opencontainers.image.title="Card-Manager Service" \
      org.opencontainers.image.description="Application that helps you to manage your card collection" \
      org.opencontainers.image.version="$RELEASE" \
      org.opencontainers.image.source="https://github.com/konstantinfoerster/card-service-go.git" \
      org.opencontainers.image.vendor="Konstantin Förster" \
      org.opencontainers.image.authors="Konstantin Förster"

FROM scratch AS prod

ARG RELEASE

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/service /usr/bin/service

USER 10001:10001

ENTRYPOINT ["/usr/bin/service"]
CMD ["--config", "/opts/app/application.yaml"]

LABEL org.opencontainers.image.title="Card-Manager Service" \
      org.opencontainers.image.description="Application that helps you to manage your card collection" \
      org.opencontainers.image.version="$RELEASE" \
      org.opencontainers.image.source="https://github.com/konstantinfoerster/card-service-go.git" \
      org.opencontainers.image.vendor="Konstantin Förster" \
      org.opencontainers.image.authors="Konstantin Förster"
