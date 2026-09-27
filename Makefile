BINARY_NAME=card-service
CURRENT_DIR=$(shell pwd)
ifndef VERSION
override VERSION = local-dev
endif

.PHONY: run-local
run-local:
	go run cmd/main.go --config configs/application-local.yaml
.PHONY: build
build:
	go build -o $(BINARY_NAME) cmd/main.go
.PHONY: docker-build
docker-build:
	@echo "Build image version $(VERSION)"
	docker build --target dev --build-arg RELEASE="$(VERSION)" -t card-service:$(VERSION) -f build/Dockerfile .
.PHONY: test
test:
	go test --count=1 ./...
.PHONY: test-race
test-race:
	go test -race --count=1 ./...
.PHONY: test-unit
test-unit:
	go test --short --count=1 ./...
.PHONY: update
update:
	go get github.com/anthonynsimon/bild
	go get github.com/corona10/goimagehash 
	go get github.com/gofiber/fiber/v2
	go get github.com/gofiber/template/html/v2 
	go get github.com/jackc/pgx/v5 
	go get github.com/stretchr/testify 
	go get github.com/testcontainers/testcontainers-go 
	go get golang.org/x/sync 
	go get go.yaml.in/yaml/v3
	go mod tidy
.PHONY: lint
lint:
	docker run --pull always --rm -v $(CURRENT_DIR)\:/app -w /app golangci/golangci-lint\:latest golangci-lint config verify
	docker run --pull always --rm -v $(CURRENT_DIR)\:/app -w /app golangci/golangci-lint\:latest golangci-lint run -v
	docker run --pull always --rm -i hadolint/hadolint < build/Dockerfile
