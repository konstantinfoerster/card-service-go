BINARY_NAME=card-service
CURRENT_DIR=$(shell pwd)
ifndef VERSION
override VERSION = local-dev
endif

.PHONY: run
run:
	go run cmd/main.go -c configs/application-local.yaml
.PHONY: build
build:
	go build -o $(BINARY_NAME) cmd/main.go
.PHONY: docker
docker-build:
	docker build --build-arg RELEASE="$(VERSION)" -t card-service:$(VERSION) -f build/opencv.Dockerfile .
.PHONY: docker-run
docker-run: docker
	docker run -it --rm -v ./configs:/config card-service:$(VERSION)
.PHONY: test-unit
test-unit:
	go test --short --count=1 ./...
.PHONY: test
test:
	go test -race --count=1 ./...
.PHONY: update
update:
	go get -u ./...
	go mod tidy
.PHONY: lint
lint:
	docker run --pull always --rm -v $(CURRENT_DIR)\:/app -w /app golangci/golangci-lint\:latest golangci-lint run -v
	docker run --pull always --rm -i hadolint/hadolint < build/Dockerfile
	docker run --pull always --rm -i hadolint/hadolint < build/opencv.Dockerfile


