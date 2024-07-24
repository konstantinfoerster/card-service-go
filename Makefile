BINARY_NAME=card-service
CURRENT_DIR=$(shell pwd)

build:
	go build -o ${BINARY_NAME} cmd/main.go
utest:
	go test --short --count=1 ./...
test:
	go test --count=1 ./...
run:
	go run main.go
update:
	go get -u ./...
	go mod tidy
glint:
	docker run --pull always --rm -v ${CURRENT_DIR}\:/app -w /app golangci/golangci-lint\:latest golangci-lint run -v
dlint:
	docker run --pull always --rm -i hadolint/hadolint < build/Dockerfile
	docker run --pull always --rm -i hadolint/hadolint < build/opencv.Dockerfile
lint:
	docker run --pull always --rm -v ${CURRENT_DIR}\:/app -w /app golangci/golangci-lint\:latest golangci-lint run -v
	docker run --pull always --rm -i hadolint/hadolint < build/Dockerfile
	docker run --pull always --rm -i hadolint/hadolint < build/opencv.Dockerfile


