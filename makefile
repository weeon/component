.DEFAULT_GOAL := all

all: test

test:
	go test ./... -v -coverprofile .coverage.txt
	go tool cover -func .coverage.txt
coverage: test
	go tool cover -html=.coverage.txt

lint:
	golangci-lint run

update:
	go mod tidy
