.PHONY: generate
generate: generate-proto-go

.PHONY: generate-proto-go
generate-proto-go:
	# update buf mod
	buf mod update
	# buf build
	buf build
	# generate proto
	buf generate -v --path=api/v1
	# swagger
	go-bindata -nometadata --nocompress -pkg bindata -o internal/pkg/bindata/swagger-json.go api/api.swagger.json

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags='-w -s' -v -o bin/kafka-webinars-rebrain ./cmd

.PHONY: lint
lint:
	golangci-lint run --fix ./...
