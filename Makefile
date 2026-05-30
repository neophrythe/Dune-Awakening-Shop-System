# Dune Awakening Shop — build tooling.

BINARY := dune-shop
WEB_DIR := web
EMBED_DIST := internal/web/dist

.PHONY: all build build-web build-server run test vet fmt clean dev-web

## build: compile the SPA, embed it, and build the production binary
build: build-web
	rm -rf $(EMBED_DIST)
	cp -r $(WEB_DIR)/dist $(EMBED_DIST)
	go build -tags embed -o $(BINARY) ./cmd/dune-shop

## build-web: compile the React dashboard to web/dist (uses npm)
build-web:
	cd $(WEB_DIR) && npm install && npm run build

## build-server: build the Go binary WITHOUT the embedded SPA (API-only / dev)
build-server:
	go build -o $(BINARY) ./cmd/dune-shop

## run: build everything and run
run: build
	./$(BINARY) -config config.yaml

## test: run Go tests (set DUNE_SHOP_TEST_DB for store integration tests)
test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

## dev-web: run the Vite dev server (proxies /api to :8091)
dev-web:
	cd $(WEB_DIR) && npm run dev

clean:
	rm -f $(BINARY)
	rm -rf $(EMBED_DIST) $(WEB_DIR)/dist
