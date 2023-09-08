[![Lint](https://github.com/konstantinfoerster/card-service-go/actions/workflows/check.yml/badge.svg)](https://github.com/konstantinfoerster/card-service-go/actions/workflows/check.yml)
[![codecov](https://codecov.io/gh/konstantinfoerster/card-service-go/graph/badge.svg?token=I0TRRY5SZE)](https://codecov.io/gh/konstantinfoerster/card-service-go)

# Card-Manager in Go

Web application that help you to manage your card collection.

## Run locally

Run `go run cmd/main.go` to start the web application with the default configuration file (configs/application.yaml).

Flags:

| Flag            | Usage                              | Default Value            | Description                    |
|-----------------|------------------------------------|--------------------------|--------------------------------|
| `-c`,`--config` | `-c configs/application-prod.yaml` | configs/application.yaml | path to the configuration file |

## Test

* Run **all** tests with `go test -v ./...`
* Run **unit tests** `go test -v -short ./...`
* Run **integration tests** `go test -v -run Integration ./...`

**Integration tests** require **docker** to be installed.

## Build

Build it with `go build -o card-service cmd/main.go`

## Dependencies

Update all dependencies with `go get -u ./...`. Run `go mod tidy` afterwards to update and cleanup the `go.mod` file.
For mor information check: https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies

## Misc

### Docker linting

The linter [Hadolint](https://github.com/hadolint/hadolint) can be used to apply best practice on your Dockerfile.

Just run `docker run --pull always --rm -i hadolint/hadolint < Dockerfile` to check your Dockerfile.

### Golang linting

The lint aggregator [golangci-lint](https://golangci-lint.run/) can be used to apply best practice and find errors in
your golang code.

Just run `docker run --pull always --rm -v $(pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v`
inside the root dir of the project to start the linting process.