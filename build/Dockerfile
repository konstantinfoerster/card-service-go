FROM golang:1.25-alpine3.23 AS builder

WORKDIR /app

# download go modules
COPY go.mod go.sum ./
RUN go mod download && go mod verify 

# copy source files
COPY . /app

# build project
ENV GOOS="linux" \
    GOARCH="amd64" \
    CGO_ENABLED="0" 

RUN go build -tags timetzdata -ldflags="-s -w" -o service cmd/main.go \
      && chmod 0755 /app/service \
      && go clean -modcache -cache 

FROM alpine:3.23 AS dev

ARG RELEASE
ENV IMG_VERSION="${RELEASE}"

RUN addgroup -g 10001 nonroot \
  && adduser -D -u 10001 -G nonroot nonroot 

COPY --from=builder --chown=nonroot:nonroot /app/service /usr/bin/

USER nonroot

ENTRYPOINT ["/usr/bin/service"]
CMD ["--config", "/opts/app/application.yaml"]

LABEL org.opencontainers.image.title="Card-Manager Service" \
      org.opencontainers.image.description="Application that helps you to manage your card collection" \
      org.opencontainers.image.version="$IMG_VERSION" \
      org.opencontainers.image.source="https://github.com/konstantinfoerster/card-service-go.git" \
      org.opencontainers.image.vendor="Konstantin Förster" \
      org.opencontainers.image.authors="Konstantin Förster"
