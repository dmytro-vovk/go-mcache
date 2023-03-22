default: lint test

lint:
	golangci-lint run ./...

test:
	go test -v -race -shuffle on -coverprofile=coverage.out ./...
