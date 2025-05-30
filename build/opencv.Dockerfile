FROM ghcr.io/hybridgroup/opencv:4.11.0

ARG RELEASE
ENV IMG_VERSION="${RELEASE}"

WORKDIR /app

# download go modules
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# copy source files
COPY . /app

# build project
ENV GOOS="linux"
ENV GOARCH="amd64"
ENV CGO_ENABLED="1"

RUN go build -tags opencv -ldflags="-s -w" -o service cmd/main.go \
      && chmod 0755 /app/service \
      && cp /app/service /usr/bin/service \
      && go clean -modcache -cache

USER nobody

ENTRYPOINT ["/usr/bin/service"]
CMD ["--config", "/config/application.yaml"]

LABEL org.opencontainers.image.title="Card-Manager Service" \
      org.opencontainers.image.description="Application that helps you to manage your card collection" \
      org.opencontainers.image.version="$IMG_VERSION" \
      org.opencontainers.image.source="https://github.com/konstantinfoerster/card-service-go.git" \
      org.opencontainers.image.vendor="Konstantin Förster" \
      org.opencontainers.image.authors="Konstantin Förster"
