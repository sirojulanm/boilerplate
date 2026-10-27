APP  := boilerplate
PORT ?= 1111
SWAG  = go run github.com/swaggo/swag/cmd/swag@v1.16.6

.PHONY: run build test docs tidy fmt lint fmt-check ci

run:
	PORT=$(PORT) go run ./cmd/api

build:
	go build -o bin/$(APP) ./cmd/api

test:
	go test ./... -race -cover

docs:
	$(SWAG) init -g cmd/api/main.go -o docs --parseInternal

tidy:
	go mod tidy

fmt:
	gofmt -w .

fmt-check:
	@files="$$(gofmt -l .)"; if [ -n "$$files" ]; then echo "not gofmt-formatted:"; echo "$$files"; exit 1; fi

lint:
	go vet ./...

ci: fmt-check lint test build
