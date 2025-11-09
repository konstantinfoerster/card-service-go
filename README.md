[![CI](https://github.com/konstantinfoerster/card-service-go/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/konstantinfoerster/card-service-go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/konstantinfoerster/card-service-go/graph/badge.svg?token=I0TRRY5SZE)](https://codecov.io/gh/konstantinfoerster/card-service-go)

# Card-Manager in Go

A web application that help you to manage your card collection.

## Features

- Search for cards
- Add cards to your card-collection
- Remove cards from you card-collection
- Detect cards based on an uploaded card image

## Requirements

- go version >= 1.24
- postgres
- opencv4
- make (optional)
- docker (optional)

## Run locally

Run `go run cmd/main.go` to start the web application with the default configuration file (configs/application.yaml).

Flags:

| Flag            | Usage                         | Default Value            | Description                    |
| --------------- | ----------------------------- | ------------------------ | ------------------------------ |
| `-c`,`--config` | `-c configs/application.yaml` | configs/application.yaml | path to the configuration file |

## Test

- Run **all** tests with `go test -v ./...` or `make test`
- Run **unit tests** `go test -v -short ./...` or `make test-unit`
- Run **integration tests** `go test -v -run Integration ./...` or `make test-it`

**Integration tests** require **docker** to be installed.

## Build

Build it with `go build -o card-service cmd/main.go` (without detect functionality). To enable opencv integration you will need
to install opencv4 on your local machine and then execute `CGO_ENABLED=1; go build -o card-service -tags opencv cmd/main.go` to build the application.

## Misc

### Linting

To run all linter just run `make lint` or check the steps below.

#### Code

The lint aggregator [golangci-lint](https://golangci-lint.run/) can is used to apply best practice and find errors in this project.

Just run `docker run --pull always --rm -v $(pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v`
inside the root dir of the project to start the linting process.

#### Dockerfile

The linter [Hadolint](https://github.com/hadolint/hadolint) can be used to apply best practice on your Dockerfile.

Just run `docker run --pull always --rm -i hadolint/hadolint < build/Dockerfile` to check your Dockerfile.
