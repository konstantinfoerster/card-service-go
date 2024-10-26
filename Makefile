BINARY_NAME=card-service
CURRENT_DIR=$(shell pwd)
ifndef VERSION
override VERSION = local-dev
endif

run:
	go run cmd/main.go -c application-local.yaml
.PHONY: build
build:
	go build -o $(BINARY_NAME) cmd/main.go
docker:
	echo $(VERSION)
	docker build --build-arg RELEASE="$(VERSION)" -t card-service:$(VERSION) -f build/opencv.Dockerfile .
test-unit:
	go test --short --count=1 ./...
.PHONY: test
test:
	go test --count=1 ./...
.PHONY: update
update:
	go get -u ./...
	go mod tidy
lint:
	docker run --pull always --rm -v $(CURRENT_DIR)\:/app -w /app golangci/golangci-lint\:latest golangci-lint run -v
	docker run --pull always --rm -i hadolint/hadolint < build/Dockerfile
	docker run --pull always --rm -i hadolint/hadolint < build/opencv.Dockerfile


