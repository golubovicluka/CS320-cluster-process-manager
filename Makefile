.PHONY: build test race vet fmt fmt-check run simulate docker-build

build:
	go build ./...

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(rg --files cmd internal -g '*.go')

fmt-check:
	test -z "$$(gofmt -l $$(rg --files cmd internal -g '*.go'))"

run:
	go run ./cmd/server

simulate:
	go run ./cmd/simulator -scenario scenarios/balanced.json

docker-build:
	docker build -t cluster-process-manager .
