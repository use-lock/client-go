OPENAPI_SOURCE ?= ../lock/openapi.json

.PHONY: generate sync-openapi test check

generate:
	go generate ./...

sync-openapi:
	cp "$(OPENAPI_SOURCE)" openapi/lock.json
	$(MAKE) generate

test:
	go test -race ./...

check:
	go vet ./...
	go test -race ./...
	test -z "$$(gofmt -l .)"
