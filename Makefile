OPENAPI_DIR ?= ../lock

.PHONY: generate sync-openapi test check

generate:
	go generate ./...

sync-openapi:
	cp "$(OPENAPI_DIR)/openapi.admin.json" openapi/admin.json
	cp "$(OPENAPI_DIR)/openapi.auth.json" openapi/auth.json
	cp "$(OPENAPI_DIR)/openapi.management.json" openapi/management.json
	$(MAKE) generate

test:
	go test -race ./...

check:
	go vet ./...
	go test -race ./...
	test -z "$$(gofmt -l .)"
