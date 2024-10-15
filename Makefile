BINARY_NAME=card-service
CURRENT_DIR=$(shell pwd)

run:
	go run cmd/main.go -c application-local.yaml
build:
	go build -o ${BINARY_NAME} cmd/main.go
docker:
	docker build -t card-service:local -f build/opencv.Dockerfile .
test-unit:
	go test --short --count=1 ./...
test:
	go test --count=1 ./...
update:
	go get -u ./...
	go mod tidy
lint:
	docker run --pull always --rm -v ${CURRENT_DIR}\:/app -w /app golangci/golangci-lint\:latest golangci-lint run -v
	docker run --pull always --rm -i hadolint/hadolint < build/Dockerfile
	docker run --pull always --rm -i hadolint/hadolint < build/opencv.Dockerfile


