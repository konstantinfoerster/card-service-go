[![CI](https://github.com/konstantinfoerster/card-service-go/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/konstantinfoerster/card-service-go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/konstantinfoerster/card-service-go/graph/badge.svg?token=I0TRRY5SZE)](https://codecov.io/gh/konstantinfoerster/card-service-go)

# Card-Manager in Go

A web application that helps you to manage your card collection.

## Features

- Search for cards
- Add cards to your card-collection
- Remove cards from your card-collection
- Detect cards based on an uploaded card image

## TODOs

- compress assets
- fingerprint assets
- pin image version to digest and use dependabot to update it

## Requirements

- go version >= 1.27
- postgres
- make (optional)
- docker (optional)

## Run locally

Run `go run cmd/main.go` to start the web application with the default configuration file (`configs/application.yaml`).

Flags:

| Flag       | Usage                               | Default Value            | Description                                      |
| ---------- | ----------------------------------- | ------------------------ | ------------------------------------------------ |
| `--config` | `--config configs/application.yaml` | configs/application.yaml | path to the configuration files, can be repeated |

You can also run `make run-local` to start the application (uses `configs/application-local.yaml`).

## Test

- Run **all** tests with `go test -v ./...` or `make test`
- Run **unit tests** `go test -v -short ./...` or `make test-unit`

Running **all** tests requires **docker** to be installed.

## Build

Build it with `go build -o card-service cmd/main.go` or `make build`.

### Docker

The **dev** docker image can be built via:

- `docker build --target dev -t card-service:local-dev -f build/Dockerfile .`
  or `make docker-build`.

The **prod** image can be built with target `prod`.

Hint: The config file is expected to be mounted to `/opts/app/application.yaml`.

## Misc

### Linting

To run all linters just run `make lint` or check the steps below.

#### Code

The lint aggregator [golangci-lint](https://golangci-lint.run/) is used to apply best practices and find errors in this project.

Just run `docker run --pull always --rm -v $(pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v`
inside the root dir of the project to start the linting process.

#### Dockerfile

The linter [Hadolint](https://github.com/hadolint/hadolint) can be used to apply best practices on your Dockerfile.

Just run `docker run --pull always --rm -i hadolint/hadolint < build/Dockerfile` to check your Dockerfile.
