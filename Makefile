APP     ?= boilerplate
PORT    ?= 1111
VERSION ?= 1.0.0
SWAG     = go run github.com/swaggo/swag/cmd/swag@v1.16.6
LDFLAGS  = -s -w -X main.version=$(VERSION)

.PHONY: run build test docs tidy fmt fmt-check lint ci docker docker-run

run:
	PORT=$(PORT) go run ./cmd/api

build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(APP) ./cmd/api

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

docker:
	docker build --build-arg VERSION=$(VERSION) -t $(APP):$(VERSION) .

docker-run:
	docker run --rm -p $(PORT):1111 -v "$(APP)-logs:/app/logs" $(APP):$(VERSION)
