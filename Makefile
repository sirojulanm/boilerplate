APP  := opencodeproject
PORT ?= 1111
SWAG  = go run github.com/swaggo/swag/cmd/swag@v1.16.6

.PHONY: run build test docs tidy

run:
	PORT=$(PORT) go run ./cmd/api

build:
	go build -o bin/$(APP) ./cmd/api

test:
	go test ./... -cover

docs:
	$(SWAG) init -g cmd/api/main.go -o docs --parseInternal

tidy:
	go mod tidy
