FROM ghcr.io/hybridgroup/opencv:4.10.0

ARG RELEASE
ENV IMG_VERSION="${RELEASE}"

## Task: copy source files
COPY . /app
WORKDIR /app

## Task: fetch project deps
RUN go mod download && go mod verify

## Task: build project
ENV GOOS="linux"
ENV GOARCH="amd64"
ENV CGO_ENABLED="1"

RUN go build -tags opencv -ldflags="-s -w" -o card-service cmd/main.go \
      && chmod 0755 /app/card-service \
      && cp /app/card-service /usr/bin/card-service

USER nobody

CMD ["/usr/bin/card-service", "--config", "/config/application-service.yaml"]

LABEL org.opencontainers.image.title="Card-Manager Service" \
      org.opencontainers.image.description="Application that helps you to manage your card collection" \
      org.opencontainers.image.version="$IMG_VERSION" \
      org.opencontainers.image.source="https://github.com/konstantinfoerster/card-service-go.git" \
      org.opencontainers.image.vendor="Konstantin Förster" \
      org.opencontainers.image.authors="Konstantin Förster"
