BINARY_NAME=card-service

build:
	go build -o ${BINARY_NAME} cmd/main.go

test:
	go test --count=1 ./...

run:
	go run main.go

update:
	go get -u ./...
	go mod tidy

