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
	gofmt -w $$(find cmd internal -name '*.go' -type f)

fmt-check:
	test -z "$$(gofmt -l $$(find cmd internal -name '*.go' -type f))"

run:
	go run ./cmd/server

simulate:
	go run ./cmd/simulator -scenario scenarios/balanced.json

docker-build:
	docker build -t cluster-process-manager .

